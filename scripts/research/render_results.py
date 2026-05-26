#!/usr/bin/env python3
import argparse
import csv
import html
from collections import defaultdict
from pathlib import Path


VARIANT_LABELS = {
    "without_secondary_indexes": "без вторичных индексов",
    "with_migration_btree_indexes": "с B-tree индексами миграций",
    "redis_off": "Redis отключён",
    "redis_on_warm": "Redis включён, прогретый кэш",
}


def read_csv(path):
    with Path(path).open(encoding="utf-8") as f:
        return list(csv.DictReader(f))


def write_markdown_table(path, rows, headers):
    with Path(path).open("w", encoding="utf-8") as f:
        f.write("| " + " | ".join(headers) + " |\n")
        f.write("| " + " | ".join("---" for _ in headers) + " |\n")
        for row in rows:
            f.write("| " + " | ".join(str(row.get(h, "")) for h in headers) + " |\n")


def bar_svg(path, title, labels, series, ylabel):
    width = 1200
    height = 680
    left = 110
    right = 40
    top = 70
    bottom = 155
    plot_w = width - left - right
    plot_h = height - top - bottom
    colors = ["#2b6cb0", "#dd6b20", "#2f855a", "#805ad5"]
    max_v = max([v for values in series.values() for v in values] + [1])
    y_max = max_v * 1.15
    group_w = plot_w / max(1, len(labels))
    bar_gap = 8
    keys = list(series.keys())
    bar_w = max(10, (group_w - 24 - bar_gap * (len(keys) - 1)) / max(1, len(keys)))

    parts = [
        f'<svg xmlns="http://www.w3.org/2000/svg" width="{width}" height="{height}" viewBox="0 0 {width} {height}">',
        '<rect width="100%" height="100%" fill="#ffffff"/>',
        f'<text x="{width / 2}" y="34" text-anchor="middle" font-family="Arial" font-size="24" font-weight="700">{html.escape(title)}</text>',
        f'<text x="24" y="{top + plot_h / 2}" text-anchor="middle" transform="rotate(-90 24 {top + plot_h / 2})" font-family="Arial" font-size="16">{html.escape(ylabel)}</text>',
        f'<line x1="{left}" y1="{top}" x2="{left}" y2="{top + plot_h}" stroke="#333"/>',
        f'<line x1="{left}" y1="{top + plot_h}" x2="{left + plot_w}" y2="{top + plot_h}" stroke="#333"/>',
    ]

    for i in range(6):
        value = y_max * i / 5
        y = top + plot_h - (value / y_max) * plot_h
        parts.append(f'<line x1="{left}" y1="{y:.1f}" x2="{left + plot_w}" y2="{y:.1f}" stroke="#e5e7eb"/>')
        parts.append(f'<text x="{left - 12}" y="{y + 5:.1f}" text-anchor="end" font-family="Arial" font-size="13">{value:.1f}</text>')

    for li, label in enumerate(labels):
        base_x = left + li * group_w + 12
        for si, key in enumerate(keys):
            value = series[key][li]
            bar_h = (value / y_max) * plot_h
            x = base_x + si * (bar_w + bar_gap)
            y = top + plot_h - bar_h
            color = colors[si % len(colors)]
            parts.append(f'<rect x="{x:.1f}" y="{y:.1f}" width="{bar_w:.1f}" height="{bar_h:.1f}" fill="{color}"/>')
            parts.append(f'<text x="{x + bar_w / 2:.1f}" y="{max(top + 14, y - 6):.1f}" text-anchor="middle" font-family="Arial" font-size="11">{value:.1f}</text>')
        label_x = base_x + (len(keys) * bar_w + (len(keys) - 1) * bar_gap) / 2
        safe_label = html.escape(label.replace("_", " "))
        parts.append(f'<text x="{label_x:.1f}" y="{top + plot_h + 28}" text-anchor="middle" font-family="Arial" font-size="12">{safe_label}</text>')

    legend_x = left
    legend_y = height - 48
    for si, key in enumerate(keys):
        x = legend_x + si * 310
        parts.append(f'<rect x="{x}" y="{legend_y}" width="18" height="18" fill="{colors[si % len(colors)]}"/>')
        parts.append(f'<text x="{x + 26}" y="{legend_y + 14}" font-family="Arial" font-size="14">{html.escape(VARIANT_LABELS.get(key, key))}</text>')

    parts.append("</svg>\n")
    Path(path).write_text("\n".join(parts), encoding="utf-8")


def render_db(out_dir, rows):
    display_rows = []
    grouped = defaultdict(dict)
    for row in rows:
        query = row["query"]
        variant = row["variant"]
        grouped[query][variant] = float(row["avg_execution_ms"])
        display_rows.append(
            {
                "Запрос": query,
                "Вариант": VARIANT_LABELS.get(variant, variant),
                "Планирование, мс": row["avg_planning_ms"],
                "Выполнение, мс": row["avg_execution_ms"],
                "Мин., мс": row["min_execution_ms"],
                "Макс., мс": row["max_execution_ms"],
                "Использованные индексы": row["indexes_used"],
            }
        )
    write_markdown_table(
        out_dir / "db_index_summary.md",
        display_rows,
        ["Запрос", "Вариант", "Планирование, мс", "Выполнение, мс", "Мин., мс", "Макс., мс", "Использованные индексы"],
    )

    labels = list(grouped.keys())
    series = {
        "without_secondary_indexes": [grouped[label].get("without_secondary_indexes", 0) for label in labels],
        "with_migration_btree_indexes": [grouped[label].get("with_migration_btree_indexes", 0) for label in labels],
    }
    bar_svg(out_dir / "db_index_execution_ms.svg", "Время выполнения SQL-запросов", labels, series, "мс")
    return display_rows


def render_http(out_dir, rows):
    display_rows = []
    for row in rows:
        display_rows.append(
            {
                "Вариант": VARIANT_LABELS.get(row["label"], row["label"]),
                "Запросов": row["requests"],
                "Параллельность": row["concurrency"],
                "Успешно": row["success"],
                "Ошибок": row["errors"],
                "RPS": row["rps"],
                "Среднее, мс": row["avg_ms"],
                "P95, мс": row["p95_ms"],
                "P99, мс": row["p99_ms"],
                "Макс., мс": row["max_ms"],
            }
        )
    write_markdown_table(
        out_dir / "http_cache_summary.md",
        display_rows,
        ["Вариант", "Запросов", "Параллельность", "Успешно", "Ошибок", "RPS", "Среднее, мс", "P95, мс", "P99, мс", "Макс., мс"],
    )

    labels = ["avg_ms", "p95_ms", "p99_ms"]
    series = {row["label"]: [float(row[k]) for k in labels] for row in rows}
    bar_svg(out_dir / "http_cache_latency_ms.svg", "HTTP latency карточки товара", labels, series, "мс")
    return display_rows


def html_table(rows):
    if not rows:
        return ""
    headers = list(rows[0].keys())
    out = ["<table>", "<thead><tr>"]
    out.extend(f"<th>{html.escape(h)}</th>" for h in headers)
    out.append("</tr></thead><tbody>")
    for row in rows:
        out.append("<tr>")
        out.extend(f"<td>{html.escape(str(row[h]))}</td>" for h in headers)
        out.append("</tr>")
    out.append("</tbody></table>")
    return "\n".join(out)


def write_tex(out_dir, db_rows, http_rows):
    lines = [
        "% Generated by scripts/research/render_results.py",
        "\\begin{table}[H]",
        "\\centering",
        "\\caption{Среднее время выполнения запросов при различном наборе индексов}",
        "\\begin{tabular}{|l|l|r|}",
        "\\hline",
        "Запрос & Вариант & Время, мс \\\\ \\hline",
    ]
    for row in db_rows:
        query = row["Запрос"].replace("_", "\\_")
        variant = row["Вариант"].replace("_", "\\_")
        lines.append(f"{query} & {variant} & {row['Выполнение, мс']} \\\\ \\hline")
    lines.extend(["\\end{tabular}", "\\end{table}", ""])
    lines.extend([
        "\\begin{table}[H]",
        "\\centering",
        "\\caption{Результаты нагрузочного тестирования чтения карточки товара}",
        "\\begin{tabular}{|l|r|r|r|r|}",
        "\\hline",
        "Вариант & RPS & Среднее, мс & P95, мс & P99, мс \\\\ \\hline",
    ])
    for row in http_rows:
        variant = row["Вариант"].replace("_", "\\_")
        lines.append(f"{variant} & {row['RPS']} & {row['Среднее, мс']} & {row['P95, мс']} & {row['P99, мс']} \\\\ \\hline")
    lines.extend(["\\end{tabular}", "\\end{table}", ""])
    (out_dir / "research_tables.tex").write_text("\n".join(lines), encoding="utf-8")


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--out-dir", required=True)
    args = parser.parse_args()

    out_dir = Path(args.out_dir)
    db_rows = render_db(out_dir, read_csv(out_dir / "db_index_summary.csv"))
    http_rows = render_http(out_dir, read_csv(out_dir / "http_summary.csv"))
    write_tex(out_dir, db_rows, http_rows)

    html_doc = f"""<!doctype html>
<html lang="ru">
<head>
  <meta charset="utf-8">
  <title>Результаты исследования marketplace</title>
  <style>
    body {{ font-family: Arial, sans-serif; margin: 32px; color: #1f2933; }}
    h1 {{ font-size: 28px; margin-bottom: 8px; }}
    h2 {{ margin-top: 36px; }}
    table {{ border-collapse: collapse; width: 100%; margin: 16px 0 24px; font-size: 13px; }}
    th, td {{ border: 1px solid #cbd5e1; padding: 8px 10px; text-align: left; }}
    th {{ background: #edf2f7; }}
    img {{ width: 100%; max-width: 1200px; border: 1px solid #cbd5e1; margin: 10px 0 24px; }}
    .note {{ color: #4a5568; }}
  </style>
</head>
<body>
  <h1>Результаты исследования производительности</h1>
  <p class="note">Артефакты сгенерированы автоматически поверх текущей схемы PostgreSQL и HTTP API проекта.</p>
  <h2>Зависимость времени SQL-запросов от индексов</h2>
  <img src="db_index_execution_ms.svg" alt="График времени выполнения SQL-запросов">
  {html_table(db_rows)}
  <h2>Влияние Redis-кэширования на HTTP-ответ</h2>
  <img src="http_cache_latency_ms.svg" alt="График latency HTTP-запросов">
  {html_table(http_rows)}
</body>
</html>
"""
    (out_dir / "research_report.html").write_text(html_doc, encoding="utf-8")


if __name__ == "__main__":
    main()

