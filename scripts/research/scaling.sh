#!/usr/bin/env bash
# Лестница масштабирования для И1/И2: время текстового поиска (Seq Scan vs GIN/pg_trgm)
# при росте числа товаров. Опциональный, более долгий прогон (отдельно от run.sh).
#
#   SIZES="10000 100000 500000" ./scripts/research/scaling.sh
#
# Для каждого размера БД пересоздаётся, наполняется только товарами (заказы/отзывы
# минимальны — текстовый запрос их не затрагивает), затем index_bench --mode scaling
# дописывает строки в db_scaling.csv. В конце перерисовываются графики/таблицы.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
OUT_DIR="${OUT_DIR:-${ROOT_DIR}/DBCourseWork/RPZ/research_results/latest}"
RESEARCH_DB="${RESEARCH_DB:-marketplace_scaling}"
DB_ADMIN_URL="${DB_ADMIN_URL:-postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable}"
DATABASE_URL="${DATABASE_URL:-postgres://postgres:postgres@localhost:5432/${RESEARCH_DB}?sslmode=disable}"
SIZES="${SIZES:-10000 100000 500000}"
INDEX_REPEATS="${INDEX_REPEATS:-10}"

if [[ ! "${RESEARCH_DB}" =~ ^[A-Za-z_][A-Za-z0-9_]*$ ]]; then
  echo "Invalid RESEARCH_DB: ${RESEARCH_DB}" >&2
  exit 2
fi

for c in docker psql goose go python3; do
  command -v "$c" >/dev/null 2>&1 || { echo "Missing command: $c" >&2; exit 2; }
done

mkdir -p "${OUT_DIR}"
rm -f "${OUT_DIR}/db_scaling.csv"

cd "${ROOT_DIR}"
docker compose up -d db >/dev/null 2>&1 || true
for _ in $(seq 1 60); do
  psql "${DB_ADMIN_URL}" -X -q -c "select 1" >/dev/null 2>&1 && break
  sleep 1
done

migrate_with_roles_workaround() {
  local log="$1"
  if ! goose -dir migrations postgres "${DATABASE_URL}" up >"${log}" 2>&1; then
    if grep -q '003_create_roles.sql' "${log}" && grep -q 'already exists' "${log}"; then
      psql "${DATABASE_URL}" -X -q -v ON_ERROR_STOP=1 \
        -c "insert into goose_db_version(version_id, is_applied, tstamp) values (3, true, now());" >>"${log}" 2>&1
      goose -dir migrations postgres "${DATABASE_URL}" up >>"${log}" 2>&1
    else
      cat "${log}" >&2; exit 1
    fi
  fi
}

for n in ${SIZES}; do
  echo "[scaling] size=${n}: recreating database"
  psql "${DB_ADMIN_URL}" -X -q -v ON_ERROR_STOP=1 <<SQL
select pg_terminate_backend(pid) from pg_stat_activity
where datname = '${RESEARCH_DB}' and pid <> pg_backend_pid();
drop database if exists ${RESEARCH_DB};
create database ${RESEARCH_DB};
SQL

  migrate_with_roles_workaround "${OUT_DIR}/scaling_goose.log"
  DATABASE_URL="${DATABASE_URL}" go run ./cmd/seed -config config/config.yaml >"${OUT_DIR}/scaling_seed.log" 2>&1

  echo "[scaling] size=${n}: generating ${n} products"
  psql "${DATABASE_URL}" -X -q -v ON_ERROR_STOP=1 \
    -v product_count="${n}" -v order_count=1000 -v review_count=1000 \
    -f "${SCRIPT_DIR}/prepare_data.sql" >"${OUT_DIR}/scaling_prepare.log" 2>&1

  echo "[scaling] size=${n}: EXPLAIN ANALYZE text search (seq vs gin)"
  python3 "${SCRIPT_DIR}/index_bench.py" \
    --database-url "${DATABASE_URL}" \
    --out-dir "${OUT_DIR}" \
    --mode scaling \
    --scale "${n}" \
    --repeats "${INDEX_REPEATS}"
done

echo "[scaling] rendering"
python3 "${SCRIPT_DIR}/render_results.py" --out-dir "${OUT_DIR}"

if command -v convert >/dev/null 2>&1; then
  for f in "${OUT_DIR}"/*.svg; do
    [[ -e "$f" ]] || continue
    convert -density 150 "$f" "${f%.svg}.png" || true
  done
fi

echo "[scaling] done. db_scaling.csv:"
cat "${OUT_DIR}/db_scaling.csv"
