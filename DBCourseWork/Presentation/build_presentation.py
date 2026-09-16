from pathlib import Path

from PIL import Image, ImageDraw, ImageFont
from pptx import Presentation
from pptx.dml.color import RGBColor
from pptx.enum.shapes import MSO_SHAPE
from pptx.enum.text import MSO_VERTICAL_ANCHOR, PP_ALIGN
from pptx.util import Inches, Pt


ROOT = Path(__file__).resolve().parents[2]
OUT_DIR = ROOT / "DBCourseWork" / "Presentation"
PPTX_PATH = OUT_DIR / "presentation.pptx"
IMG = ROOT / "DBCourseWork" / "RPZ" / "img"
RESEARCH = ROOT / "DBCourseWork" / "RPZ" / "research_results" / "latest"
ASSETS = OUT_DIR / "assets"

SLIDE_W = 13.333
SLIDE_H = 7.5

BLACK = RGBColor(0, 0, 0)
GRAY = RGBColor(90, 90, 90)
LIGHT_GRAY = RGBColor(225, 225, 225)
HEADER = RGBColor(245, 245, 245)
WHITE = RGBColor(255, 255, 255)

FONT = "Times New Roman"


def inch(value: float):
    return Inches(value)


def ensure_redis_diagram():
    path = ASSETS / "redis_interaction.png"
    ASSETS.mkdir(parents=True, exist_ok=True)
    if path.exists():
        return path

    image = Image.new("RGB", (1500, 930), "white")
    draw = ImageDraw.Draw(image)
    try:
        font = ImageFont.truetype("DejaVuSans.ttf", 34)
        small = ImageFont.truetype("DejaVuSans.ttf", 27)
    except OSError:
        font = ImageFont.load_default()
        small = ImageFont.load_default()

    def box(x, y, w, h, text):
        draw.rectangle((x, y, x + w, y + h), outline=(0, 0, 0), width=3)
        lines = text.split("\n")
        line_h = 40
        start_y = y + (h - line_h * len(lines)) / 2
        for i, line in enumerate(lines):
            bbox = draw.textbbox((0, 0), line, font=font)
            draw.text((x + (w - (bbox[2] - bbox[0])) / 2, start_y + i * line_h), line, fill=(0, 0, 0), font=font)

    def arrow(start, end, label, dashed=False):
        x1, y1 = start
        x2, y2 = end
        if dashed:
            steps = 18
            for i in range(0, steps, 2):
                xa = x1 + (x2 - x1) * i / steps
                ya = y1 + (y2 - y1) * i / steps
                xb = x1 + (x2 - x1) * (i + 1) / steps
                yb = y1 + (y2 - y1) * (i + 1) / steps
                draw.line((xa, ya, xb, yb), fill=(0, 0, 0), width=3)
        else:
            draw.line((x1, y1, x2, y2), fill=(0, 0, 0), width=3)
        # Simple arrow head.
        if abs(x2 - x1) > abs(y2 - y1):
            sign = 1 if x2 > x1 else -1
            pts = [(x2, y2), (x2 - sign * 22, y2 - 10), (x2 - sign * 22, y2 + 10)]
        else:
            sign = 1 if y2 > y1 else -1
            pts = [(x2, y2), (x2 - 10, y2 - sign * 22), (x2 + 10, y2 - sign * 22)]
        draw.polygon(pts, fill=(0, 0, 0))
        if label:
            lx = (x1 + x2) / 2
            ly = (y1 + y2) / 2
            bbox = draw.textbbox((0, 0), label, font=small)
            draw.rectangle((lx - (bbox[2] - bbox[0]) / 2 - 8, ly - 24, lx + (bbox[2] - bbox[0]) / 2 + 8, ly + 16), fill="white")
            draw.text((lx - (bbox[2] - bbox[0]) / 2, ly - 22), label, fill=(0, 0, 0), font=small)

    box(120, 70, 360, 105, "HTTP-клиент")
    box(120, 365, 420, 115, "Go-приложение\nservice/repository")
    box(965, 365, 350, 115, "Redis\nкэш")
    box(120, 720, 420, 115, "PostgreSQL\nисточник истины")

    arrow((300, 175), (300, 365), "1. HTTP")
    arrow((540, 400), (965, 400), "2. GET")
    arrow((965, 445), (540, 445), "3. hit/miss")
    arrow((520, 365), (990, 365), "6. SET TTL")
    arrow((520, 480), (990, 480), "инвалидация DEL/SCAN", dashed=True)
    arrow((260, 480), (260, 720), "4. SQL")
    arrow((390, 720), (390, 480), "5. данные")

    image.save(path)
    return path


def set_run(run, size=24, bold=False, color=BLACK):
    run.font.name = FONT
    run.font.size = Pt(size)
    run.font.bold = bold
    run.font.color.rgb = color


def add_text(slide, text, left, top, width, height, size=24, bold=False, align=PP_ALIGN.LEFT, color=BLACK):
    box = slide.shapes.add_textbox(inch(left), inch(top), inch(width), inch(height))
    tf = box.text_frame
    tf.clear()
    tf.word_wrap = True
    tf.margin_left = inch(0.03)
    tf.margin_right = inch(0.03)
    tf.margin_top = inch(0.02)
    tf.margin_bottom = inch(0.02)
    p = tf.paragraphs[0]
    p.alignment = align
    p.text = text
    p.font.name = FONT
    p.font.size = Pt(size)
    p.font.bold = bold
    p.font.color.rgb = color
    return box


def add_background(slide):
    bg = slide.shapes.add_shape(MSO_SHAPE.RECTANGLE, 0, 0, inch(SLIDE_W), inch(SLIDE_H))
    bg.fill.solid()
    bg.fill.fore_color.rgb = WHITE
    bg.line.fill.background()


def add_header(slide, number):
    # Только номер слайда по центру снизу; верхняя/нижняя подписи убраны.
    add_text(slide, str(number), 0, 7.17, SLIDE_W, 0.18, 10, False, PP_ALIGN.CENTER, GRAY)


def add_title(slide, title):
    add_text(slide, title, 0.55, 0.72, 12.2, 0.58, 32, False, PP_ALIGN.CENTER)


def add_slide(prs, title, number):
    slide = prs.slides.add_slide(prs.slide_layouts[6])
    add_background(slide)
    add_header(slide, number)
    add_title(slide, title)
    return slide


def add_bullets(slide, items, left, top, width, height, size=22):
    box = slide.shapes.add_textbox(inch(left), inch(top), inch(width), inch(height))
    tf = box.text_frame
    tf.clear()
    tf.word_wrap = True
    tf.margin_left = inch(0.05)
    tf.margin_right = inch(0.03)
    for i, item in enumerate(items):
        p = tf.paragraphs[0] if i == 0 else tf.add_paragraph()
        p.text = f"• {item}"
        p.font.name = FONT
        p.font.size = Pt(size)
        p.font.color.rgb = BLACK
        p.space_after = Pt(8)
        p.line_spacing = 1.08
    return box


def add_numbered(slide, items, left, top, width, height, size=20):
    box = slide.shapes.add_textbox(inch(left), inch(top), inch(width), inch(height))
    tf = box.text_frame
    tf.clear()
    tf.word_wrap = True
    for i, item in enumerate(items, 1):
        p = tf.paragraphs[0] if i == 1 else tf.add_paragraph()
        p.text = f"{i}. {item}"
        p.font.name = FONT
        p.font.size = Pt(size)
        p.font.color.rgb = BLACK
        p.space_after = Pt(5)
    return box


def add_picture_fit(slide, path, left, top, width, height, border=False):
    path = Path(path)
    with Image.open(path) as image:
        img_w, img_h = image.size
    box_ratio = width / height
    img_ratio = img_w / img_h
    if img_ratio > box_ratio:
        pic_w = width
        pic_h = width / img_ratio
    else:
        pic_h = height
        pic_w = height * img_ratio
    pic_left = left + (width - pic_w) / 2
    pic_top = top + (height - pic_h) / 2
    if border:
        frame = slide.shapes.add_shape(MSO_SHAPE.RECTANGLE, inch(left), inch(top), inch(width), inch(height))
        frame.fill.background()
        frame.line.color.rgb = LIGHT_GRAY
        frame.line.width = Pt(1)
    return slide.shapes.add_picture(str(path), inch(pic_left), inch(pic_top), width=inch(pic_w), height=inch(pic_h))


def style_table(table, header=True, size=15):
    for r in range(len(table.rows)):
        for c in range(len(table.columns)):
            cell = table.cell(r, c)
            cell.margin_left = inch(0.04)
            cell.margin_right = inch(0.04)
            cell.margin_top = inch(0.02)
            cell.margin_bottom = inch(0.02)
            cell.fill.solid()
            cell.fill.fore_color.rgb = HEADER if header and r == 0 else WHITE
            for p in cell.text_frame.paragraphs:
                p.alignment = PP_ALIGN.CENTER if r == 0 else PP_ALIGN.LEFT
                for run in p.runs:
                    set_run(run, size=size, bold=(header and r == 0))


def add_table(slide, rows, left, top, width, height, col_widths=None, size=15):
    table = slide.shapes.add_table(len(rows), len(rows[0]), inch(left), inch(top), inch(width), inch(height)).table
    if col_widths:
        for i, w in enumerate(col_widths):
            table.columns[i].width = inch(w)
    for r, row in enumerate(rows):
        for c, value in enumerate(row):
            table.cell(r, c).text = value
    style_table(table, size=size)
    return table


def add_plain_box(slide, title, body, left, top, width, height, size=18):
    shape = slide.shapes.add_shape(MSO_SHAPE.RECTANGLE, inch(left), inch(top), inch(width), inch(height))
    shape.fill.background()
    shape.line.color.rgb = LIGHT_GRAY
    shape.line.width = Pt(1)
    add_text(slide, title, left + 0.1, top + 0.08, width - 0.2, 0.3, size, True)
    add_text(slide, body, left + 0.1, top + 0.45, width - 0.2, height - 0.52, size - 2, False)


def build():
    redis_diagram = ensure_redis_diagram()

    prs = Presentation()
    prs.slide_width = inch(SLIDE_W)
    prs.slide_height = inch(SLIDE_H)

    # 1
    slide = prs.slides.add_slide(prs.slide_layouts[6])
    add_background(slide)
    add_picture_fit(slide, ASSETS / "bmstu_logo.png", 2.25, 0.36, 1.25, 1.25)
    add_text(
        slide,
        "Министерство науки и высшего образования Российской Федерации\n"
        "Федеральное государственное автономное образовательное учреждение\n"
        "высшего образования\n"
        "«Московский государственный технический университет\n"
        "имени Н.Э. Баумана\n"
        "(национальный исследовательский университет)»\n"
        "(МГТУ им. Н.Э. Баумана)",
        3.75,
        0.34,
        7.2,
        1.35,
        16,
        True,
        PP_ALIGN.CENTER,
    )
    add_text(slide, "Разработка базы данных\nонлайн-маркетплейса", 1.4, 2.7, 10.6, 1.1, 38, True, PP_ALIGN.CENTER)
    add_text(slide, "Студент: Афанасьев Роман, ИУ7-61Б\nРуководитель: Ковтушенко А.П.", 1.15, 5.25, 8.2, 0.7, 24, False)
    add_text(slide, "Москва, 2026 г.", 0, 6.82, SLIDE_W, 0.25, 18, False, PP_ALIGN.CENTER)

    # 2
    slide = add_slide(prs, "Актуальность", 2)
    add_text(
        slide,
        "Для сравнения с существующими решениями были выбраны следующие критерии:",
        0.72,
        1.58,
        11.8,
        0.35,
        22,
    )
    add_bullets(
        slide,
        [
            "открытость модели данных;",
            "хранение истории изменения цен товара;",
            "наличие встроенной аналитики продаж;",
            "раздельные рейтинги товара и продавца;",
            "выделенная роль аналитика только для чтения;",
            "наличие логистики в предметной области.",
        ],
        0.72,
        2.0,
        6.0,
        2.25,
        20,
    )
    rows = [
        ["Критерий", "Ozon", "WB", "AliExpress", "Система"],
        ["Открытость модели данных", "нет", "нет", "нет", "да"],
        ["История цен товара", "не раскрывается", "не раскрывается", "не раскрывается", "да"],
        ["Встроенная аналитика", "частично", "да", "да", "да"],
        ["Рейтинг товара и продавца", "да", "да", "да", "да"],
        ["Роль аналитика", "не указана", "не указана", "не указана", "да"],
        ["Логистика", "да", "да", "да", "нет"],
    ]
    add_table(slide, rows, 6.95, 1.76, 5.55, 3.0, [1.75, 0.78, 0.75, 1.1, 1.17], 10.5)
    add_text(
        slide,
        "Рассмотренные платформы являются закрытыми коммерческими системами. "
        "В проектируемой системе отдельно рассматриваются модель данных, история изменения цен и роль аналитика.",
        0.72,
        5.25,
        11.8,
        0.8,
        21,
    )

    # 3
    slide = add_slide(prs, "Цель и задачи", 3)
    add_text(slide, "Цель: разработать базу данных онлайн-маркетплейса и приложение доступа к ней.", 0.82, 1.55, 11.7, 0.45, 23)
    add_numbered(
        slide,
        [
            "провести анализ предметной области и существующих решений;",
            "выделить роли пользователей, сценарии взаимодействия, сущности и связи;",
            "спроектировать схему БД, ограничения целостности, роли, триггеры и хранимую функцию;",
            "реализовать БД, Redis-кэширование и интерфейс доступа к данным;",
            "исследовать влияние индексов PostgreSQL и Redis-кэша на показатели доступа к данным.",
        ],
        0.9,
        2.25,
        11.5,
        3.4,
        22,
    )

    # 4
    slide = add_slide(prs, "Границы проектируемой системы", 4)
    add_plain_box(slide, "Данные", "пользователи, продавцы, товары, категории, заказы, позиции заказов, адреса, отзывы, история изменения цен", 0.75, 1.55, 3.85, 1.35)
    add_plain_box(slide, "Сценарии", "просмотр каталога, управление товарами, оформление заказов, отзывы, администрирование, отчёты продаж", 4.85, 1.55, 3.6, 1.35)
    add_plain_box(slide, "Не рассматривается", "логистика, курьеры, маршруты, графики складов и события фактической передачи товара", 8.7, 1.55, 3.85, 1.35)
    add_bullets(
        slide,
        [
            "Для хранения данных выбрана реляционная модель.",
            "Состояние исполнения заказа фиксируется статусами заказа.",
            "Количество товара относится к данным товара, которыми управляет продавец.",
            "Резерв товара изменяется при обработке заказа.",
        ],
        1.0,
        3.6,
        11.2,
        2.2,
        22,
    )

    # 5
    slide = add_slide(prs, "Роли пользователей и сценарии", 5)
    add_bullets(
        slide,
        [
            "гость — просмотр каталога и карточек товаров;",
            "покупатель — заказы, адреса и отзывы;",
            "продавец — товары, заказы продавца и статистика;",
            "администратор — модерация пользователей, товаров, категорий, отзывов и заказов;",
            "аналитик — чтение данных и формирование отчётов.",
        ],
        0.65,
        1.65,
        4.25,
        3.6,
        18,
    )
    add_picture_fit(slide, IMG / "UseCases.png", 5.05, 1.18, 7.35, 5.95)

    # 6
    slide = add_slide(prs, "Диаграмма сущность-связь в нотации Чена", 6)
    add_picture_fit(slide, IMG / "ER_Chen.png", 0.75, 1.25, 11.85, 5.78)

    # 7
    slide = add_slide(prs, "Схема базы данных", 7)
    add_picture_fit(slide, IMG / "ER.png", 0.65, 1.12, 6.2, 6.02)
    add_bullets(
        slide,
        [
            "схема содержит 10 таблиц основной предметной области;",
            "связь многие-ко-многим между товарами и категориями вынесена в product_categories;",
            "корзина представлена заказом со статусом draft;",
            "история изменения цен хранится отдельно от текущей цены товара;",
            "схема приведена к третьей нормальной форме.",
        ],
        7.1,
        1.65,
        5.1,
        3.4,
        20,
    )

    # 8
    slide = add_slide(prs, "Ограничения целостности и индексы", 8)
    rows = [
        ["Группа", "Пример"],
        ["Заказы", "одна draft-корзина на покупателя; price_at_purchase фиксируется при оформлении"],
        ["Остатки", "reserved_quantity >= 0 и reserved_quantity <= stock_quantity"],
        ["Отзывы", "один отзыв покупателя на товар; рейтинг в диапазоне 1..5"],
        ["История цен", "триггер записывает старую и новую цену, время и инициатора"],
        ["Рейтинги", "при изменении отзыва пересчитываются рейтинги товара и продавца"],
        ["Индексы", "выборка заказов, отзывов, товаров продавца, активных записей и фильтрация по цене"],
    ]
    add_table(slide, rows, 0.85, 1.55, 11.65, 4.3, [2.0, 9.65], 17)
    add_text(slide, "Ограничения защищают данные от некорректных состояний; индексы влияют только на время доступа.", 0.85, 6.15, 11.5, 0.4, 21)

    # 9
    slide = add_slide(prs, "Ролевая модель базы данных", 9)
    rows = [
        ["Таблица", "buyer", "seller", "admin", "analyst"],
        ["users", "—", "—", "CRUD", "R"],
        ["sellers", "R", "CRUD", "CRUD", "R"],
        ["products", "R", "CRUD", "CRUD", "R"],
        ["categories", "R", "R", "CRUD", "R"],
        ["product_categories", "R", "CRUD", "CRUD", "R"],
        ["addresses", "CRUD", "—", "CRUD", "R"],
        ["orders", "CRUD", "RU", "CRUD", "R"],
        ["order_items", "CRUD", "R", "CRUD", "R"],
        ["reviews", "CRUD", "R", "CRUD", "R"],
        ["product_price_history", "R", "R", "CRUD", "R"],
    ]
    add_table(slide, rows, 0.95, 1.35, 11.25, 4.7, [3.35, 1.55, 1.55, 1.55, 1.7], 13.5)
    add_text(slide, "C — INSERT, R — SELECT, U — UPDATE, D — DELETE. Ограничения «только свои записи» проверяются приложением.", 0.95, 6.25, 11.1, 0.35, 18)

    # 10
    slide = add_slide(prs, "Триггеры и хранимая функция", 10)
    add_picture_fit(slide, IMG / "trigger_catch_price_change.png", 0.75, 1.45, 3.2, 2.95)
    add_picture_fit(slide, IMG / "trigger_update_ratings_on_review.png", 4.25, 1.35, 3.2, 3.1)
    add_bullets(
        slide,
        [
            "catch_price_change формирует журнал изменения цен товара;",
            "update_ratings_on_review поддерживает согласованность рейтингов товара и продавца;",
            "get_seller_statistics принимает продавца и период, возвращает количество заказов, выручку, средний чек и самый продаваемый товар;",
            "в расчёт статистики включаются только заказы paid, shipped и delivered.",
        ],
        7.9,
        1.55,
        4.5,
        3.5,
        17,
    )
    add_text(
        slide,
        "Хранимая функция объединяет orders, order_items и products; цена берётся из price_at_purchase, поэтому изменение текущей цены товара не меняет статистику уже оформленных заказов.",
        0.75,
        5.25,
        11.6,
        0.45,
        18,
    )

    # 11
    slide = add_slide(prs, "Реализация и Redis-кэширование", 11)
    add_bullets(
        slide,
        [
            "PostgreSQL 16.11 — основное хранилище данных;",
            "Go 1.25.0 — REST API, серверный Web-интерфейс и бизнес-операции;",
            "Redis 8.6 — in-memory хранилище для кэша и отозванных JWT-токенов;",
            "кэширование реализовано по схеме Cache-Aside.",
        ],
        0.75,
        1.55,
        4.75,
        3.0,
        19,
    )
    add_picture_fit(slide, redis_diagram, 5.75, 1.55, 6.6, 4.45, True)
    add_text(slide, "PostgreSQL остаётся источником истины; Redis используется только для ускорения повторных чтений и проверки отозванных токенов.", 0.8, 5.75, 11.4, 0.5, 20)

    # 12
    slide = add_slide(prs, "Исследование 1: наличие индексов", 12)
    add_text(
        slide,
        "Проверялись SQL-запросы: выборка каталога по диапазону цены, заказы покупателя, pending-заказы по дате, отзывы товара и вызов get_seller_statistics.",
        0.75,
        1.35,
        11.7,
        0.6,
        19,
    )
    add_picture_fit(slide, RESEARCH / "db_index_execution_ms.png", 0.75, 2.1, 7.05, 4.45)
    add_bullets(
        slide,
        [
            "catalog_price_range: 9.351 мс без индекса и 0.073 мс с idx_products_price;",
            "pending_orders_by_date: 11.252 мс без индекса и 0.394 мс с idx_orders_status_created_at;",
            "для seller_statistics_function вторичные индексы миграций ускорения не дали.",
        ],
        8.05,
        2.15,
        4.25,
        2.55,
        15,
    )
    add_text(slide, "Вывод: индекс ускоряет выборку только при соответствии запросу.", 8.05, 5.28, 4.3, 0.45, 16)

    # 13
    slide = add_slide(prs, "Исследование 1: тип индекса", 13)
    add_text(
        slide,
        "Сравнивались два случая на основном наборе из 40 000 товаров: заказы покупателя с сортировкой по дате (левый график) и подстрочный поиск товара по условию name ILIKE '%Zephyr%'. На правом графике число товаров меняется от 10 000 до 500 000.",
        0.75,
        1.28,
        11.7,
        0.72,
        18,
    )
    add_picture_fit(slide, RESEARCH / "db_index_type_ms.png", 0.75, 2.12, 5.75, 3.5)
    add_picture_fit(slide, RESEARCH / "db_scaling_ms.png", 6.75, 2.12, 5.75, 3.5)
    add_bullets(
        slide,
        [
            "составной индекс (user_id, created_at desc) устраняет Sort;",
            "без индекса по name поиск '%Zephyr%' идёт через Seq Scan по всей таблице (~25 мс на 40 000 товаров);",
            "GIN + pg_trgm (триграммы) обслуживает поиск через Bitmap Heap Scan и почти не зависит от размера таблицы.",
        ],
        0.9,
        5.72,
        11.3,
        0.85,
        14,
    )

    # 14
    slide = add_slide(prs, "Исследование 2: влияние Redis-кэша", 14)
    add_text(
        slide,
        "Нагрузка подавалась на четыре GET-маршрута чтения: карточка товара (/products/{id}), "
        "каталог (/products, 12 товаров на странице), отзывы товара (/products/{id}/reviews, 10 отзывов на странице) "
        "и список категорий (/categories). На каждый маршрут — 300 запросов, до 12 одновременно.",
        0.75,
        1.28,
        11.7,
        0.85,
        18,
    )
    add_picture_fit(slide, RESEARCH / "http_cache_latency_ms.png", 0.75, 2.15, 7.05, 4.25)
    add_bullets(
        slide,
        [
            "сравнивались сценарии Redis отключён и Redis включён с прогретым кэшем;",
            "карточка товара: среднее время 2.256 мс без Redis и 0.965 мс с прогретым кэшем;",
            "при прогретом кэше зафиксировано 1500 попаданий Redis и 0 строк PostgreSQL.",
        ],
        8.05,
        2.25,
        4.25,
        2.55,
        15,
    )
    add_text(slide, "Вывод: прогретый кэш снижает время ответа и число обращений к PostgreSQL.", 8.05, 5.25, 4.25, 0.55, 16)

    # 15
    slide = add_slide(prs, "Заключение", 15)
    add_bullets(
        slide,
        [
            "разработана база данных онлайн-маркетплейса и приложение доступа к ней;",
            "спроектированы сущности, связи, ограничения целостности и ролевая модель доступа;",
            "реализованы триггеры журнала изменения цен и пересчёта рейтингов;",
            "реализована хранимая функция статистики продаж продавца;",
            "исследованы индексы PostgreSQL и Redis-кэширование как способы ускорения чтения данных.",
        ],
        0.95,
        1.55,
        11.2,
        3.55,
        22,
    )
    add_text(
        slide,
        "Итог исследования: индексы дают эффект при соответствии запросу, а Redis-кэш снижает время HTTP-ответа за счёт обслуживания повторных чтений из памяти.",
        0.95,
        5.7,
        11.2,
        0.7,
        22,
    )

    OUT_DIR.mkdir(parents=True, exist_ok=True)
    prs.save(PPTX_PATH)
    print(PPTX_PATH)


if __name__ == "__main__":
    build()
