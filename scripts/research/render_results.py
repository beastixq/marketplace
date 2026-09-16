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
    "text_seq": "Без индекса по name",
    "text_gin_trgm": "GIN-индекс (pg_trgm)",
}

CASE_LABELS = {
    "orders_simple_btree": "Простой (user_id)",
    "orders_composite_btree": "Составной (user_id, created_at)",
    "text_btree_only": "Без индекса по name",
    "text_gin_trgm": "GIN-индекс (pg_trgm)",
}

ENDPOINT_LABELS = {
    "product": "Карточка товара",
    "catalog": "Каталог",
    "reviews": "Отзывы",
    "categories": "Категории",
}


def split_label(label):
    if "__" in label:
        scen, ep = label.split("__", 1)
        return scen, ep
    return label, "product"


def read_csv(path):
    p = Path(path)
    if not p.exists():
        return []
    with p.open(encoding="utf-8") as f:
        return list(csv.DictReader(f))


def write_markdown_table(path, rows, headers):
    with Path(path).open("w", encoding="utf-8") as f:
        f.write("| " + " | ".join(headers) + " |\n")
        f.write("| " + " | ".join("---" for _ in headers) + " |\n")
        for row in rows:
            f.write("| " + " | ".join(str(row.get(h, "")) for h in headers) + " |\n")


def bar_svg(path, title, labels, series, ylabel, errors=None, label_map=None):
    """Сгруппированная столбчатая диаграмма. errors[key][i] — стандартное
    отклонение для усов погрешности (опционально)."""
    width = 1200
    height = 680
    left = 110
    right = 40
    top = 70
    bottom = 170
    plot_w = width - left - right
    plot_h = height - top - bottom
    # Палитра с большим разрывом по яркости: различима и в цвете, и на ЧБ-печати.
    # Тёмный (почти чёрный) против светлого стального — оба видны на белом фоне.
    colors = ["#16304a", "#9fbad4", "#52606d", "#c4d2df"]
    err = errors or {}
    max_v = max(
        [series[k][i] + (err.get(k, [0] * len(labels))[i]) for k in series for i in range(len(labels))]
        + [1]
    )
    y_max = max_v * 1.15
    group_w = plot_w / max(1, len(labels))
    bar_gap = 8
    keys = list(series.keys())
    bar_w = max(10, (group_w - 24 - bar_gap * (len(keys) - 1)) / max(1, len(keys)))

    parts = [
        f'<svg xmlns="http://www.w3.org/2000/svg" width="{width}" height="{height}" viewBox="0 0 {width} {height}">',
        '<rect width="100%" height="100%" fill="#ffffff"/>',
        f'<text x="{width / 2}" y="34" text-anchor="middle" font-family="Arial" font-size="24" font-weight="700">{html.escape(title)}</text>',
        # Подпись единиц оси Y горизонтально над осью: повёрнутый текст теряется
        # при конвертации SVG->PNG через ImageMagick.
        f'<text x="{left}" y="{top - 14}" text-anchor="start" font-family="Arial" font-size="16">{html.escape(ylabel)}</text>',
        f'<line x1="{left}" y1="{top}" x2="{left}" y2="{top + plot_h}" stroke="#333"/>',
        f'<line x1="{left}" y1="{top + plot_h}" x2="{left + plot_w}" y2="{top + plot_h}" stroke="#333"/>',
    ]

    for i in range(6):
        value = y_max * i / 5
        y = top + plot_h - (value / y_max) * plot_h
        parts.append(f'<line x1="{left}" y1="{y:.1f}" x2="{left + plot_w}" y2="{y:.1f}" stroke="#e5e7eb"/>')
        parts.append(f'<text x="{left - 12}" y="{y + 5:.1f}" text-anchor="end" font-family="Arial" font-size="13">{value:.2f}</text>')

    for li, label in enumerate(labels):
        base_x = left + li * group_w + 12
        for si, key in enumerate(keys):
            value = series[key][li]
            bar_h = (value / y_max) * plot_h
            x = base_x + si * (bar_w + bar_gap)
            y = top + plot_h - bar_h
            color = colors[si % len(colors)]
            parts.append(f'<rect x="{x:.1f}" y="{y:.1f}" width="{bar_w:.1f}" height="{bar_h:.1f}" fill="{color}" stroke="#16304a" stroke-width="1"/>')
            e = err.get(key, [0] * len(labels))[li]
            if e > 0:
                cx = x + bar_w / 2
                y_hi = top + plot_h - ((value + e) / y_max) * plot_h
                y_lo = top + plot_h - (max(0.0, value - e) / y_max) * plot_h
                parts.append(f'<line x1="{cx:.1f}" y1="{y_hi:.1f}" x2="{cx:.1f}" y2="{y_lo:.1f}" stroke="#1a202c"/>')
                parts.append(f'<line x1="{cx-4:.1f}" y1="{y_hi:.1f}" x2="{cx+4:.1f}" y2="{y_hi:.1f}" stroke="#1a202c"/>')
                parts.append(f'<line x1="{cx-4:.1f}" y1="{y_lo:.1f}" x2="{cx+4:.1f}" y2="{y_lo:.1f}" stroke="#1a202c"/>')
            parts.append(f'<text x="{x + bar_w / 2:.1f}" y="{max(top + 14, y - 6):.1f}" text-anchor="middle" font-family="Arial" font-size="11">{value:.3g}</text>')
        label_x = base_x + (len(keys) * bar_w + (len(keys) - 1) * bar_gap) / 2
        disp = (label_map or {}).get(label, label).replace("_", " ")
        parts.append(f'<text x="{label_x:.1f}" y="{top + plot_h + 28}" text-anchor="middle" font-family="Arial" font-size="12">{html.escape(disp)}</text>')

    # Легенда не нужна для одиночной серии без собственного смысла (ключ "value"):
    # подписи кейсов уже отображаются под столбцами.
    legend_keys = [k for k in keys if k != "value"]
    legend_x = left
    legend_y = height - 48
    for si, key in enumerate(legend_keys):
        x = legend_x + si * 300
        parts.append(f'<rect x="{x}" y="{legend_y}" width="18" height="18" fill="{colors[si % len(colors)]}"/>')
        parts.append(f'<text x="{x + 26}" y="{legend_y + 14}" font-family="Arial" font-size="14">{html.escape(VARIANT_LABELS.get(key, key))}</text>')

    parts.append("</svg>\n")
    Path(path).write_text("\n".join(parts), encoding="utf-8")


def line_svg(path, title, xlabels, series, ylabel):
    """Линейный график: xlabels — значения оси X (например, размер набора),
    series[key] — список значений Y."""
    width = 1200
    height = 680
    left = 120
    right = 40
    top = 70
    bottom = 150
    plot_w = width - left - right
    plot_h = height - top - bottom
    # Палитра с большим разрывом по яркости: различима и в цвете, и на ЧБ-печати.
    # Тёмный (почти чёрный) против светлого стального — оба видны на белом фоне.
    colors = ["#16304a", "#9fbad4", "#52606d", "#c4d2df"]
    keys = list(series.keys())
    y_max = max([v for k in keys for v in series[k]] + [1]) * 1.15
    n = len(xlabels)
    step = plot_w / max(1, n - 1) if n > 1 else plot_w

    parts = [
        f'<svg xmlns="http://www.w3.org/2000/svg" width="{width}" height="{height}" viewBox="0 0 {width} {height}">',
        '<rect width="100%" height="100%" fill="#ffffff"/>',
        f'<text x="{width / 2}" y="34" text-anchor="middle" font-family="Arial" font-size="24" font-weight="700">{html.escape(title)}</text>',
        # Подпись единиц оси Y горизонтально над осью (см. bar_svg).
        f'<text x="{left}" y="{top - 14}" text-anchor="start" font-family="Arial" font-size="16">{html.escape(ylabel)}</text>',
        f'<line x1="{left}" y1="{top}" x2="{left}" y2="{top + plot_h}" stroke="#333"/>',
        f'<line x1="{left}" y1="{top + plot_h}" x2="{left + plot_w}" y2="{top + plot_h}" stroke="#333"/>',
    ]
    for i in range(6):
        value = y_max * i / 5
        y = top + plot_h - (value / y_max) * plot_h
        parts.append(f'<line x1="{left}" y1="{y:.1f}" x2="{left + plot_w}" y2="{y:.1f}" stroke="#e5e7eb"/>')
        parts.append(f'<text x="{left - 12}" y="{y + 5:.1f}" text-anchor="end" font-family="Arial" font-size="13">{value:.2f}</text>')
    for i, xl in enumerate(xlabels):
        x = left + i * step
        parts.append(f'<text x="{x:.1f}" y="{top + plot_h + 26}" text-anchor="middle" font-family="Arial" font-size="13">{html.escape(str(xl))}</text>')

    for si, key in enumerate(keys):
        color = colors[si % len(colors)]
        pts = []
        for i in range(n):
            x = left + i * step
            y = top + plot_h - (series[key][i] / y_max) * plot_h
            pts.append(f"{x:.1f},{y:.1f}")
            parts.append(f'<circle cx="{x:.1f}" cy="{y:.1f}" r="4" fill="{color}"/>')
        parts.append(f'<polyline points="{" ".join(pts)}" fill="none" stroke="{color}" stroke-width="2.5"/>')

    legend_y = height - 48
    for si, key in enumerate(keys):
        x = left + si * 300
        parts.append(f'<rect x="{x}" y="{legend_y}" width="18" height="18" fill="{colors[si % len(colors)]}"/>')
        parts.append(f'<text x="{x + 26}" y="{legend_y + 14}" font-family="Arial" font-size="14">{html.escape(VARIANT_LABELS.get(key, key))}</text>')

    parts.append("</svg>\n")
    Path(path).write_text("\n".join(parts), encoding="utf-8")


def render_presence(out_dir, rows):
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
    bar_svg(out_dir / "db_index_execution_ms.svg", "Время выполнения SQL-запросов: наличие индексов", labels, series, "мс")
    return display_rows


def yes_no_ru(value):
    normalized = str(value).strip().lower()
    if normalized == "yes":
        return "да"
    if normalized == "no":
        return "нет"
    return str(value)


def normalize_case_label(value):
    return str(value).replace("'%x%'", "'%Zephyr%'")


def render_type(out_dir, rows):
    if not rows:
        return []
    display_rows = []
    labels = []
    values = []
    errors = []
    for row in rows:
        labels.append(row["case"])
        values.append(float(row["avg_execution_ms"]))
        errors.append(float(row.get("std_execution_ms", 0) or 0))
        display_rows.append(
            {
                "Случай": normalize_case_label(row["label"]),
                "Узел Sort": yes_no_ru(row["has_sort"]),
                "Корневой узел плана": row["root_node"],
                "Планирование, мс": row["avg_planning_ms"],
                "Выполнение, мс": row["avg_execution_ms"],
                "Строк": row["actual_rows"],
                "Индексы": row["indexes_used"],
            }
        )
    write_markdown_table(
        out_dir / "db_index_type_summary.md",
        display_rows,
        ["Случай", "Узел Sort", "Корневой узел плана", "Планирование, мс", "Выполнение, мс", "Строк", "Индексы"],
    )
    bar_svg(
        out_dir / "db_index_type_ms.svg",
        "Время выполнения: тип индекса",
        labels,
        {"value": values},
        "мс",
        errors={"value": errors},
        label_map=CASE_LABELS,
    )
    return display_rows


def render_scaling(out_dir, rows):
    if not rows:
        return []
    scales = sorted({int(r["scale"]) for r in rows})
    series = defaultdict(lambda: [0.0] * len(scales))
    idx = {s: i for i, s in enumerate(scales)}
    for r in rows:
        series[r["variant"]][idx[int(r["scale"])]] = float(r["avg_execution_ms"])
    line_svg(
        out_dir / "db_scaling_ms.svg",
        "Текстовый поиск: время vs размер набора",
        [f"{s:,}".replace(",", " ") for s in scales],
        dict(series),
        "мс",
    )
    display_rows = []
    for r in rows:
        display_rows.append(
            {
                "Размер": r["scale"],
                "Вариант": VARIANT_LABELS.get(r["variant"], r["variant"]),
                "Планирование, мс": r["avg_planning_ms"],
                "Выполнение, мс": r["avg_execution_ms"],
                "Строк": r["actual_rows"],
                "Корневой узел": r["root_node"],
            }
        )
    write_markdown_table(
        out_dir / "db_scaling_summary.md",
        display_rows,
        ["Размер", "Вариант", "Планирование, мс", "Выполнение, мс", "Строк", "Корневой узел"],
    )
    return display_rows


def render_http(out_dir, rows):
    display_rows = []
    by_ep_avg = defaultdict(dict)
    by_ep_p99 = defaultdict(dict)
    for row in rows:
        scen, ep = split_label(row["label"])
        by_ep_avg[ep][scen] = float(row["avg_ms"])
        by_ep_p99[ep][scen] = float(row["p99_ms"])
        display_rows.append(
            {
                "Эндпоинт": ENDPOINT_LABELS.get(ep, ep),
                "Сценарий": VARIANT_LABELS.get(scen, scen),
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
        ["Эндпоинт", "Сценарий", "RPS", "Среднее, мс", "P95, мс", "P99, мс", "Макс., мс"],
    )
    eps = list(by_ep_avg.keys())
    series = {
        "redis_off": [by_ep_avg[e].get("redis_off", 0) for e in eps],
        "redis_on_warm": [by_ep_avg[e].get("redis_on_warm", 0) for e in eps],
    }
    bar_svg(
        out_dir / "http_cache_latency_ms.svg",
        "Среднее время ответа HTTP по эндпоинтам: влияние Redis",
        eps,
        series,
        "мс",
        label_map=ENDPOINT_LABELS,
    )
    return display_rows


def render_http_meta(out_dir, rows):
    if not rows:
        return []
    display_rows = []
    for r in rows:
        display_rows.append(
            {
                "Сценарий": VARIANT_LABELS.get(r["label"], r["label"]),
                "Попаданий в кэш": r["keyspace_hits"],
                "Промахов": r["keyspace_misses"],
                "Доля попаданий": r["hit_ratio"],
                "Строк PostgreSQL": r["pg_tup_returned"],
                "Блоков PostgreSQL": r["pg_blocks"],
            }
        )
    write_markdown_table(
        out_dir / "http_cache_meta.md",
        display_rows,
        ["Сценарий", "Попаданий в кэш", "Промахов", "Доля попаданий", "Строк PostgreSQL", "Блоков PostgreSQL"],
    )
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


def tex_escape(s):
    return str(s).replace("_", "\\_").replace("%", "\\%")


def tex_identifier_cell(value, limit=12):
    raw = str(value)
    if raw == "-":
        return "-"
    parts = raw.split("_")
    if len(parts) <= 2 or len(raw) <= limit:
        return tex_escape(raw)

    lines = []
    line = parts[0]
    for part in parts[1:]:
        candidate = f"{line}_{part}"
        if len(candidate) > limit:
            lines.append(line + r"\_")
            line = part
        else:
            line += r"\_" + part
    lines.append(line)
    return r"\makecell[l]{" + r"\\".join(lines) + "}"


def write_tex(out_dir, presence_rows, type_rows, scaling_rows, http_rows, http_meta_rows):
    lines = ["% Generated by scripts/research/render_results.py"]

    lines += [
        "\\begin{table}[H]", "\\centering", "\\scriptsize",
        "\\caption{Среднее время выполнения SQL-запросов при различном наборе индексов}",
        "\\label{tbl:research_presence}",
        "\\begin{tabular}{|p{0.20\\textwidth}|p{0.22\\textwidth}|r|r|p{0.13\\textwidth}|}", "\\hline",
        "Запрос & Вариант & \\makecell{Планирование,\\\\мс} & \\makecell{Выполнение,\\\\мс} & Индекс \\\\ \\hline",
    ]
    for row in presence_rows:
        lines.append(
            f"{tex_identifier_cell(row['Запрос'])} & {tex_escape(row['Вариант'])} & {row['Планирование, мс']} & {row['Выполнение, мс']} & {tex_identifier_cell(row['Использованные индексы'])} \\\\ \\hline"
        )
    lines += ["\\end{tabular}", "\\end{table}", ""]

    if type_rows:
        lines += [
            "\\begin{table}[H]", "\\centering", "\\scriptsize",
            "\\caption{Влияние типа индекса на время выполнения SQL-запроса}",
            "\\label{tbl:research_type}",
            "\\begin{tabular}{|p{0.30\\textwidth}|c|p{0.13\\textwidth}|p{0.15\\textwidth}|r|r|r|}", "\\hline",
            "Случай & Sort & \\makecell{Корневой\\\\узел} & Индекс & Строк & \\makecell{План.,\\\\мс} & \\makecell{Вып.,\\\\мс} \\\\ \\hline",
        ]
        for row in type_rows:
            lines.append(
                f"{tex_escape(row['Случай'])} & {row['Узел Sort']} & {tex_escape(row['Корневой узел плана'])} & {tex_identifier_cell(row['Индексы'])} & {row['Строк']} & {row['Планирование, мс']} & {row['Выполнение, мс']} \\\\ \\hline"
            )
        lines += ["\\end{tabular}", "\\end{table}", ""]

    if scaling_rows:
        lines += [
            "\\begin{table}[H]", "\\centering", "\\small",
            "\\caption{Время текстового поиска в зависимости от числа товаров}",
            "\\label{tbl:research_scaling}",
            "\\begin{tabular}{|r|p{0.22\\textwidth}|r|r|r|p{0.18\\textwidth}|}", "\\hline",
            "Товаров & Вариант & Строк & \\makecell{План.,\\\\мс} & \\makecell{Вып.,\\\\мс} & Узел \\\\ \\hline",
        ]
        for row in scaling_rows:
            lines.append(
                f"{row['Размер']} & {tex_escape(row['Вариант'])} & {row['Строк']} & {row['Планирование, мс']} & {row['Выполнение, мс']} & {tex_escape(row['Корневой узел'])} \\\\ \\hline"
            )
        lines += ["\\end{tabular}", "\\end{table}", ""]

    lines += [
        "\\begin{table}[H]", "\\centering", "\\small",
        "\\caption{Нагрузочное тестирование: время ответа HTTP по эндпоинтам}",
        "\\label{tbl:research_http}",
        "\\begin{tabular}{|p{0.18\\textwidth}|p{0.25\\textwidth}|r|r|r|r|}", "\\hline",
        "Эндпоинт & Сценарий & RPS & \\makecell{Среднее,\\\\мс} & P95, мс & P99, мс \\\\ \\hline",
    ]
    for row in http_rows:
        lines.append(
            f"{tex_escape(row['Эндпоинт'])} & {tex_escape(row['Сценарий'])} & {row['RPS']} & {row['Среднее, мс']} & {row['P95, мс']} & {row['P99, мс']} \\\\ \\hline"
        )
    lines += ["\\end{tabular}", "\\end{table}", ""]

    if http_meta_rows:
        lines += [
            "\\begin{table}[H]", "\\centering", "\\small",
            "\\caption{Контрольные счётчики Redis и PostgreSQL при нагрузочном тестировании}",
            "\\label{tbl:research_cache_meta}",
            "\\begin{tabular}{|p{0.24\\textwidth}|r|r|r|r|r|}", "\\hline",
            "Сценарий & \\makecell{Попаданий\\\\Redis} & \\makecell{Промахов\\\\Redis} & \\makecell{Доля\\\\попаданий} & \\makecell{Строк\\\\PostgreSQL} & \\makecell{Блоков\\\\PostgreSQL} \\\\ \\hline",
        ]
        for row in http_meta_rows:
            lines.append(
                f"{tex_escape(row['Сценарий'])} & {row['Попаданий в кэш']} & {row['Промахов']} & {row['Доля попаданий']} & {row['Строк PostgreSQL']} & {row['Блоков PostgreSQL']} \\\\ \\hline"
            )
        lines += ["\\end{tabular}", "\\end{table}", ""]

    tex = "\n".join(lines)
    (out_dir / "research_tables.tex").write_text(tex, encoding="utf-8")

    if out_dir.parent.name == "research_results" and out_dir.parent.parent.name == "RPZ":
        parts_table = out_dir.parent.parent / "parts" / "research_tables.tex"
        parts_table.write_text(tex, encoding="utf-8")


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--out-dir", required=True)
    args = parser.parse_args()

    out_dir = Path(args.out_dir)
    presence_rows = render_presence(out_dir, read_csv(out_dir / "db_index_summary.csv"))
    type_rows = render_type(out_dir, read_csv(out_dir / "db_index_type_summary.csv"))
    scaling_rows = render_scaling(out_dir, read_csv(out_dir / "db_scaling.csv"))
    http_rows = render_http(out_dir, read_csv(out_dir / "http_summary.csv"))
    http_meta_rows = render_http_meta(out_dir, read_csv(out_dir / "http_cache_meta.csv"))
    write_tex(out_dir, presence_rows, type_rows, scaling_rows, http_rows, http_meta_rows)

    scaling_html = ""
    if scaling_rows:
        scaling_html = (
            '<h2>Текстовый поиск: масштабирование</h2>'
            '<img src="db_scaling_ms.svg" alt="Время текстового поиска от размера набора">'
            + html_table(scaling_rows)
        )
    type_html = ""
    if type_rows:
        type_html = (
            '<h2>Влияние типа индекса</h2>'
            '<img src="db_index_type_ms.svg" alt="Время выполнения по типу индекса">'
            + html_table(type_rows)
        )

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
  <h1>Результаты измерений SQL и HTTP</h1>
  <p class="note">Артефакты сгенерированы автоматически поверх текущей схемы PostgreSQL и HTTP API проекта.</p>
  <h2>Зависимость времени SQL-запросов от наличия индексов</h2>
  <img src="db_index_execution_ms.svg" alt="График времени выполнения SQL-запросов">
  {html_table(presence_rows)}
  {type_html}
  {scaling_html}
  <h2>Влияние Redis-кэширования на HTTP-ответ</h2>
  <img src="http_cache_latency_ms.svg" alt="График latency HTTP-запросов">
  {html_table(http_rows)}
  {html_table(http_meta_rows)}
</body>
</html>
"""
    (out_dir / "research_report.html").write_text(html_doc, encoding="utf-8")


if __name__ == "__main__":
    main()
