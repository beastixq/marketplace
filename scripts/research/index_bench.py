#!/usr/bin/env python3
import argparse
import csv
import json
import statistics
import subprocess
from pathlib import Path


INDEXES = [
    ("idx_products_seller_id", "create index idx_products_seller_id on products (seller_id)"),
    ("idx_orders_user_id", "create index idx_orders_user_id on orders (user_id)"),
    ("idx_orders_status_created_at", "create index idx_orders_status_created_at on orders (status, created_at)"),
    ("idx_product_categories_category_id", "create index idx_product_categories_category_id on product_categories (category_id)"),
    ("idx_reviews_product_id", "create index idx_reviews_product_id on reviews (product_id)"),
    ("idx_products_id_not_deleted", "create index idx_products_id_not_deleted on products (id) where deleted_at is null"),
    ("idx_users_id_not_deleted", "create index idx_users_id_not_deleted on users (id) where deleted_at is null"),
    ("idx_products_price", "create index idx_products_price on products (price)"),
]


def run_psql(db_url: str, sql: str, *, tuples_only: bool = True) -> str:
    cmd = ["psql", db_url, "-X", "-q", "-v", "ON_ERROR_STOP=1"]
    if tuples_only:
        cmd.extend(["-A", "-t"])
    cmd.extend(["-c", sql])
    result = subprocess.run(cmd, check=True, text=True, capture_output=True)
    return result.stdout.strip()


def scalar(db_url: str, sql: str) -> str:
    return run_psql(db_url, sql).splitlines()[0].strip()


def drop_indexes(db_url: str) -> None:
    statements = ";\n".join(f"drop index if exists {name}" for name, _ in INDEXES) + "; analyze;"
    run_psql(db_url, statements)


def create_indexes(db_url: str) -> None:
    statements = []
    for name, definition in INDEXES:
        statements.append(f"drop index if exists {name}")
        statements.append(definition)
    statements.append("analyze")
    run_psql(db_url, ";\n".join(statements) + ";")


def walk_plan(node):
    yield node
    for child in node.get("Plans", []):
        yield from walk_plan(child)


def plan_metrics(plan_json):
    root = plan_json[0]
    plan = root["Plan"]
    nodes = list(walk_plan(plan))
    index_names = sorted({n["Index Name"] for n in nodes if "Index Name" in n})
    shared_hit = sum(int(n.get("Shared Hit Blocks", 0)) for n in nodes)
    shared_read = sum(int(n.get("Shared Read Blocks", 0)) for n in nodes)
    temp_read = sum(int(n.get("Temp Read Blocks", 0)) for n in nodes)
    temp_written = sum(int(n.get("Temp Written Blocks", 0)) for n in nodes)
    return {
        "planning_ms": float(root.get("Planning Time", 0)),
        "execution_ms": float(root.get("Execution Time", 0)),
        "actual_rows": float(plan.get("Actual Rows", 0)),
        "root_node": plan.get("Node Type", ""),
        "shared_hit_blocks": shared_hit,
        "shared_read_blocks": shared_read,
        "temp_read_blocks": temp_read,
        "temp_written_blocks": temp_written,
        "indexes_used": ", ".join(index_names) if index_names else "-",
    }


def explain_json(db_url: str, query: str):
    out = run_psql(db_url, f"explain (analyze, buffers, format json) {query}")
    return json.loads(out)


def explain_text(db_url: str, query: str) -> str:
    return run_psql(db_url, f"explain (analyze, buffers) {query}", tuples_only=False)


def avg(values):
    return statistics.fmean(values) if values else 0.0


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--database-url", required=True)
    parser.add_argument("--out-dir", required=True)
    parser.add_argument("--repeats", type=int, default=5)
    args = parser.parse_args()

    out_dir = Path(args.out_dir)
    plans_dir = out_dir / "plans"
    plans_dir.mkdir(parents=True, exist_ok=True)

    user_id = scalar(
        args.database_url,
        "select user_id from orders group by user_id order by count(*) desc, user_id limit 1",
    )
    product_id = scalar(
        args.database_url,
        "select product_id from reviews group by product_id order by count(*) desc, product_id limit 1",
    )
    seller_id = scalar(
        args.database_url,
        "select seller_id from products group by seller_id order by count(*) desc, seller_id limit 1",
    )

    queries = [
        (
            "catalog_price_range",
            """
            select id, seller_id, name, price
            from products
            where deleted_at is null and price between 1000 and 3000
            order by price asc
            limit 50
            """,
        ),
        (
            "orders_by_user",
            f"""
            select id, status, total_amount, created_at
            from orders
            where user_id = {user_id}
            order by created_at desc
            limit 100
            """,
        ),
        (
            "pending_orders_by_date",
            """
            select id, user_id, seller_id, status, created_at
            from orders
            where status = 'pending'
              and created_at <= timestamp '2026-01-01'
            order by created_at asc
            limit 100
            """,
        ),
        (
            "reviews_by_product",
            f"""
            select id, user_id, product_id, rating, created_at
            from reviews
            where product_id = {product_id}
            order by created_at desc
            limit 100
            """,
        ),
        (
            "seller_statistics_function",
            f"select * from get_seller_statistics({seller_id}, date '2025-01-01', date '2027-01-01')",
        ),
    ]

    detail_rows = []
    variants = [
        ("without_secondary_indexes", drop_indexes),
        ("with_migration_btree_indexes", create_indexes),
    ]

    for variant, setup in variants:
        setup(args.database_url)
        for query_name, query in queries:
            explain_json(args.database_url, query)
            text_plan = explain_text(args.database_url, query)
            (plans_dir / f"{query_name}.{variant}.txt").write_text(text_plan, encoding="utf-8")
            for repeat in range(1, args.repeats + 1):
                plan = explain_json(args.database_url, query)
                (plans_dir / f"{query_name}.{variant}.{repeat}.json").write_text(
                    json.dumps(plan, ensure_ascii=False, indent=2),
                    encoding="utf-8",
                )
                metrics = plan_metrics(plan)
                detail_rows.append(
                    {
                        "query": query_name,
                        "variant": variant,
                        "repeat": repeat,
                        **metrics,
                    }
                )

    create_indexes(args.database_url)

    detail_path = out_dir / "db_index_results.csv"
    with detail_path.open("w", newline="", encoding="utf-8") as f:
        fieldnames = [
            "query",
            "variant",
            "repeat",
            "planning_ms",
            "execution_ms",
            "actual_rows",
            "root_node",
            "shared_hit_blocks",
            "shared_read_blocks",
            "temp_read_blocks",
            "temp_written_blocks",
            "indexes_used",
        ]
        writer = csv.DictWriter(f, fieldnames=fieldnames)
        writer.writeheader()
        writer.writerows(detail_rows)

    summary_rows = []
    for query_name, _ in queries:
        for variant, _ in variants:
            rows = [r for r in detail_rows if r["query"] == query_name and r["variant"] == variant]
            execution = [float(r["execution_ms"]) for r in rows]
            planning = [float(r["planning_ms"]) for r in rows]
            reads = [float(r["shared_read_blocks"]) for r in rows]
            hits = [float(r["shared_hit_blocks"]) for r in rows]
            summary_rows.append(
                {
                    "query": query_name,
                    "variant": variant,
                    "avg_planning_ms": f"{avg(planning):.3f}",
                    "avg_execution_ms": f"{avg(execution):.3f}",
                    "min_execution_ms": f"{min(execution):.3f}",
                    "max_execution_ms": f"{max(execution):.3f}",
                    "avg_shared_hit_blocks": f"{avg(hits):.1f}",
                    "avg_shared_read_blocks": f"{avg(reads):.1f}",
                    "indexes_used": rows[-1]["indexes_used"],
                }
            )

    summary_path = out_dir / "db_index_summary.csv"
    with summary_path.open("w", newline="", encoding="utf-8") as f:
        fieldnames = [
            "query",
            "variant",
            "avg_planning_ms",
            "avg_execution_ms",
            "min_execution_ms",
            "max_execution_ms",
            "avg_shared_hit_blocks",
            "avg_shared_read_blocks",
            "indexes_used",
        ]
        writer = csv.DictWriter(f, fieldnames=fieldnames)
        writer.writeheader()
        writer.writerows(summary_rows)


if __name__ == "__main__":
    main()

