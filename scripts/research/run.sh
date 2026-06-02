#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
OUT_DIR="${OUT_DIR:-${ROOT_DIR}/DBCourseWork/RPZ/research_results/latest}"
RESEARCH_DB="${RESEARCH_DB:-marketplace_research_$(date +%Y%m%d_%H%M%S)}"
DB_ADMIN_URL="${DB_ADMIN_URL:-postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable}"
DATABASE_URL="${DATABASE_URL:-postgres://postgres:postgres@localhost:5432/${RESEARCH_DB}?sslmode=disable}"
API_PORT_OFF="${API_PORT_OFF:-18080}"
API_PORT_ON="${API_PORT_ON:-18081}"
HTTP_REQUESTS="${HTTP_REQUESTS:-600}"
HTTP_CONCURRENCY="${HTTP_CONCURRENCY:-30}"
ADD_RESEARCH_DATA="${ADD_RESEARCH_DATA:-true}"
# Размер синтетического набора для основного прогона (исследования И1/И2/И3).
PRODUCT_COUNT="${PRODUCT_COUNT:-40000}"
ORDER_COUNT="${ORDER_COUNT:-80000}"
REVIEW_COUNT="${REVIEW_COUNT:-50000}"
INDEX_REPEATS="${INDEX_REPEATS:-11}"

if [[ ! "${RESEARCH_DB}" =~ ^[A-Za-z_][A-Za-z0-9_]*$ ]]; then
  echo "Invalid RESEARCH_DB: ${RESEARCH_DB}" >&2
  exit 2
fi

need() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "Missing required command: $1" >&2
    exit 2
  fi
}

need docker
need psql
need goose
need go
need python3
need curl

mkdir -p "${OUT_DIR}"
rm -f "${OUT_DIR}/http_summary.csv"
printf 'label,keyspace_hits,keyspace_misses,hit_ratio,pg_tup_returned,pg_blocks\n' >"${OUT_DIR}/http_cache_meta.csv"

echo "[research] output: ${OUT_DIR}"
echo "[research] database: ${RESEARCH_DB}"

cd "${ROOT_DIR}"

echo "[research] starting PostgreSQL container"
docker compose up -d db

if redis-cli -h localhost -p 6379 ping >/dev/null 2>&1; then
  echo "[research] using existing Redis at localhost:6379"
else
  echo "[research] starting Redis container"
  docker compose up -d cache
fi

echo "[research] waiting for PostgreSQL"
for _ in $(seq 1 60); do
  if psql "${DB_ADMIN_URL}" -X -q -c "select 1" >/dev/null 2>&1; then
    break
  fi
  sleep 1
done
psql "${DB_ADMIN_URL}" -X -q -v ON_ERROR_STOP=1 -c "select 1" >/dev/null

echo "[research] recreating isolated database"
psql "${DB_ADMIN_URL}" -X -q -v ON_ERROR_STOP=1 <<SQL
select pg_terminate_backend(pid)
from pg_stat_activity
where datname = '${RESEARCH_DB}' and pid <> pg_backend_pid();
drop database if exists ${RESEARCH_DB};
create database ${RESEARCH_DB};
SQL

echo "[research] running migrations"
if ! goose -dir migrations postgres "${DATABASE_URL}" up >"${OUT_DIR}/goose.log" 2>&1; then
  if grep -q '003_create_roles.sql' "${OUT_DIR}/goose.log" && grep -q 'already exists' "${OUT_DIR}/goose.log"; then
    {
      echo
      echo "[research] 003_create_roles.sql skipped because PostgreSQL roles already exist in this local cluster."
      echo "[research] Role grants are not used by the benchmark workload."
    } >>"${OUT_DIR}/goose.log"
    psql "${DATABASE_URL}" -X -q -v ON_ERROR_STOP=1 \
      -c "insert into goose_db_version(version_id, is_applied, tstamp) values (3, true, now());" \
      >>"${OUT_DIR}/goose.log" 2>&1
    goose -dir migrations postgres "${DATABASE_URL}" up >>"${OUT_DIR}/goose.log" 2>&1
  else
    cat "${OUT_DIR}/goose.log" >&2
    exit 1
  fi
fi

echo "[research] running base seed command"
DATABASE_URL="${DATABASE_URL}" go run ./cmd/seed -config config/config.yaml >"${OUT_DIR}/seed.log" 2>&1

if [[ "${ADD_RESEARCH_DATA}" == "true" ]]; then
  echo "[research] adding set-based research data (products=${PRODUCT_COUNT}, orders=${ORDER_COUNT}, reviews=${REVIEW_COUNT})"
  psql "${DATABASE_URL}" -X -q -v ON_ERROR_STOP=1 \
    -v product_count="${PRODUCT_COUNT}" \
    -v order_count="${ORDER_COUNT}" \
    -v review_count="${REVIEW_COUNT}" \
    -f "${SCRIPT_DIR}/prepare_data.sql" >"${OUT_DIR}/prepare_data.log" 2>&1
else
  echo "[research] skipping extra set-based data; using project seed dataset"
  echo "Skipped. Set ADD_RESEARCH_DATA=true to add extra synthetic benchmark rows." >"${OUT_DIR}/prepare_data.log"
fi

echo "[research] writing metadata"
{
  echo "# Research Run Metadata"
  echo
  echo "- database: \`${RESEARCH_DB}\`"
  echo "- database_url: \`${DATABASE_URL}\`"
  echo "- generated_at: \`$(date -Is)\`"
  echo "- http_requests: \`${HTTP_REQUESTS}\`"
  echo "- http_concurrency: \`${HTTP_CONCURRENCY}\`"
  echo
  echo "## Table Counts"
  echo
  echo '```text'
  psql "${DATABASE_URL}" -X -A -F $'\t' -q -c "select 'users' as table_name, count(*) from users union all select 'sellers', count(*) from sellers union all select 'products', count(*) from products union all select 'orders', count(*) from orders union all select 'order_items', count(*) from order_items union all select 'reviews', count(*) from reviews order by table_name"
  echo '```'
} >"${OUT_DIR}/metadata.md"

echo "[research] running EXPLAIN ANALYZE index benchmark (наличие индексов)"
python3 "${SCRIPT_DIR}/index_bench.py" \
  --database-url "${DATABASE_URL}" \
  --out-dir "${OUT_DIR}" \
  --mode presence \
  --repeats 5

echo "[research] running EXPLAIN ANALYZE index benchmark (тип индекса: простой/составной, B-tree/GIN)"
python3 "${SCRIPT_DIR}/index_bench.py" \
  --database-url "${DATABASE_URL}" \
  --out-dir "${OUT_DIR}" \
  --mode type \
  --repeats "${INDEX_REPEATS}"

PRODUCT_ID="$(psql "${DATABASE_URL}" -X -A -t -q -c "select id from products where deleted_at is null order by id desc limit 1")"
if [[ -z "${PRODUCT_ID}" ]]; then
  echo "No product found for HTTP benchmark" >&2
  exit 1
fi

write_api_config() {
  local path="$1"
  local addr="$2"
  local redis_enabled="$3"
  cat >"${path}" <<YAML
server:
  addr: "${addr}"

database:
  dsn: "${DATABASE_URL}"

redis:
  enabled: ${redis_enabled}
  addr: "localhost:6379"
  password: ""
  db: 0
  dial_timeout: "2s"
  product_ttl: "5m"
  catalog_ttl: "5m"
  category_ttl: "1h"
  review_ttl: "5m"

auth:
  jwt_secret: "research-secret"
  jwt_ttl: "24h"
  bcrypt_cost: 4

payment:
  ttl: "2m"
  gateway_url: "http://localhost:8080"

orders:
  expiration_check_interval: "1m"

logging:
  level: "error"
  format: "json"
  file: ""
  console: true
  add_source: false
YAML
}

wait_api() {
  local url="$1"
  for _ in $(seq 1 90); do
    if curl -fsS "${url}" >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done
  echo "API did not become ready: ${url}" >&2
  return 1
}

redis_stats() {
  redis-cli -h localhost -p 6379 INFO stats 2>/dev/null | tr -d '\r' \
    | awk -F: '/^keyspace_hits:/{h=$2} /^keyspace_misses:/{m=$2} END{printf "%d %d", h+0, m+0}'
}

pg_db_stats() {
  psql "${DB_ADMIN_URL}" -X -A -t -q -c \
    "select coalesce(tup_returned,0)||' '||coalesce(blks_read+blks_hit,0) from pg_stat_database where datname='${RESEARCH_DB}'" 2>/dev/null
}

# Эндпоинты И3: разные по стоимости запросы. Каждый чувствителен к кэшу по-своему
# (дорогой каталог и отзывы выигрывают сильнее точечной карточки).
api_endpoints() {
  printf '%s\n' \
    "product=/api/v1/products/${PRODUCT_ID}" \
    "catalog=/api/v1/products?limit=12&page=1" \
    "reviews=/api/v1/products/${PRODUCT_ID}/reviews?limit=10&page=1" \
    "categories=/api/v1/categories"
}

run_api_bench() {
  local label="$1"
  local port="$2"
  local redis_enabled="$3"
  local warm="$4"
  local cfg="${OUT_DIR}/api.${label}.yaml"
  local ready="http://127.0.0.1:${port}/api/v1/products/${PRODUCT_ID}"

  write_api_config "${cfg}" ":${port}" "${redis_enabled}"
  echo "[research] starting API scenario ${label}"
  DATABASE_URL="${DATABASE_URL}" go run ./cmd/api -config "${cfg}" >"${OUT_DIR}/api.${label}.log" 2>&1 &
  local api_pid=$!
  trap 'kill ${api_pid} >/dev/null 2>&1 || true' RETURN

  wait_api "${ready}"

  local name path
  if [[ "${warm}" == "warm" ]]; then
    while IFS= read -r ep; do
      path="${ep#*=}"
      curl -fsS "http://127.0.0.1:${port}${path}" >/dev/null || true
    done < <(api_endpoints)
  fi

  local rh0 rm0 pt0 pb0
  read -r rh0 rm0 <<<"$(redis_stats)"; : "${rh0:=0}" "${rm0:=0}"
  read -r pt0 pb0 <<<"$(pg_db_stats)"; : "${pt0:=0}" "${pb0:=0}"

  while IFS= read -r ep; do
    name="${ep%%=*}"
    path="${ep#*=}"
    go run "${SCRIPT_DIR}/http_load.go" \
      -url "http://127.0.0.1:${port}${path}" \
      -label "${label}__${name}" \
      -requests "${HTTP_REQUESTS}" \
      -concurrency "${HTTP_CONCURRENCY}" \
      -csv "${OUT_DIR}/http_${label}__${name}_requests.csv" \
      -summary "${OUT_DIR}/http_summary.csv"
  done < <(api_endpoints)

  local rh1 rm1 pt1 pb1
  read -r rh1 rm1 <<<"$(redis_stats)"; : "${rh1:=0}" "${rm1:=0}"
  read -r pt1 pb1 <<<"$(pg_db_stats)"; : "${pt1:=0}" "${pb1:=0}"
  local dh=$((rh1 - rh0)) dm=$((rm1 - rm0))
  local total=$((dh + dm))
  local ratio="0.000"
  if [[ ${total} -gt 0 ]]; then
    ratio="$(LC_ALL=C awk "BEGIN{printf \"%.3f\", ${dh}/${total}}")"
  fi
  echo "${label},${dh},${dm},${ratio},$((pt1 - pt0)),$((pb1 - pb0))" >>"${OUT_DIR}/http_cache_meta.csv"

  kill "${api_pid}" >/dev/null 2>&1 || true
  wait "${api_pid}" 2>/dev/null || true
  trap - RETURN
}

echo "[research] running HTTP load benchmark without Redis"
run_api_bench "redis_off" "${API_PORT_OFF}" "false" "cold"

echo "[research] flushing Redis DB"
redis-cli -h localhost -p 6379 FLUSHDB >/dev/null

echo "[research] running HTTP load benchmark with warm Redis"
run_api_bench "redis_on_warm" "${API_PORT_ON}" "true" "warm"

echo "[research] rendering tables and charts"
python3 "${SCRIPT_DIR}/render_results.py" --out-dir "${OUT_DIR}"

if command -v convert >/dev/null 2>&1; then
  for f in "${OUT_DIR}"/*.svg; do
    [[ -e "$f" ]] || continue
    convert -density 150 "$f" "${f%.svg}.png" || true
  done
fi

if command -v google-chrome >/dev/null 2>&1; then
  google-chrome --headless --disable-gpu --no-sandbox \
    --window-size=1600,1400 \
    --screenshot="${OUT_DIR}/research_report_screenshot.png" \
    "file://${OUT_DIR}/research_report.html" >/dev/null 2>&1 || true
fi

echo "[research] done"
echo "[research] artifacts:"
find "${OUT_DIR}" -maxdepth 2 -type f | sort
