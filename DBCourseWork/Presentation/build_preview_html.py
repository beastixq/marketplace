from pathlib import Path


ROOT = Path(__file__).resolve().parents[2]
OUT = ROOT / "DBCourseWork" / "Presentation" / "presentation_preview.html"
IMG = ROOT / "DBCourseWork" / "RPZ" / "img"
RESEARCH = ROOT / "DBCourseWork" / "RPZ" / "research_results" / "latest"
ASSETS = ROOT / "DBCourseWork" / "Presentation" / "assets"


def uri(path: Path) -> str:
    return path.resolve().as_uri()


def bullets(items):
    return "<ul>" + "".join(f"<li>{item}</li>" for item in items) + "</ul>"


def img(path, cls="image"):
    return f"<img class='{cls}' src='{uri(path)}'/>"


def table(rows, cls=""):
    html = [f"<table class='{cls}'>"]
    for i, row in enumerate(rows):
        tag = "th" if i == 0 else "td"
        html.append("<tr>" + "".join(f"<{tag}>{cell}</{tag}>" for cell in row) + "</tr>")
    html.append("</table>")
    return "".join(html)


def slide(num, title, body):
    return f"""
    <section class="slide">
        <h1>{title}</h1>
        <main>{body}</main>
        <footer><span>{num}</span></footer>
    </section>
    """


slides = [
    f"""
    <section class="slide cover">
        {img(ASSETS / "bmstu_logo.png", "logo")}
        <div class="ministry">
            Министерство науки и высшего образования Российской Федерации<br/>
            Федеральное государственное автономное образовательное учреждение<br/>
            высшего образования<br/>
            «Московский государственный технический университет<br/>
            имени Н.Э. Баумана<br/>
            (национальный исследовательский университет)»<br/>
            (МГТУ им. Н.Э. Баумана)
        </div>
        <h1>Разработка базы данных<br/>онлайн-маркетплейса</h1>
        <div class="author">Студент: Афанасьев Роман, ИУ7-61Б<br/>Руководитель: Ковтушенко А.П.</div>
        <div class="city">Москва, 2026 г.</div>
    </section>
    """,
    slide(2, "Актуальность", f"""
        <p>Для сравнения с существующими решениями были выбраны следующие критерии:</p>
        <div class="two">
            <div>{bullets([
                "открытость модели данных;",
                "хранение истории изменения цен товара;",
                "наличие встроенной аналитики продаж;",
                "раздельные рейтинги товара и продавца;",
                "выделенная роль аналитика только для чтения;",
                "наличие логистики в предметной области.",
            ])}</div>
            <div>{table([
                ["Критерий", "Ozon", "WB", "AliExpress", "Система"],
                ["Открытость модели данных", "нет", "нет", "нет", "да"],
                ["История цен товара", "не раскрывается", "не раскрывается", "не раскрывается", "да"],
                ["Встроенная аналитика", "частично", "да", "да", "да"],
                ["Рейтинг товара и продавца", "да", "да", "да", "да"],
                ["Роль аналитика", "не указана", "не указана", "не указана", "да"],
                ["Логистика", "да", "да", "да", "нет"],
            ], "small")}</div>
        </div>
        <p>Рассмотренные платформы являются закрытыми коммерческими системами. В проектируемой системе отдельно рассматриваются модель данных, история изменения цен и роль аналитика.</p>
    """),
    slide(3, "Цель и задачи", f"""
        <p><b>Цель:</b> разработать базу данных онлайн-маркетплейса и приложение доступа к ней.</p>
        {bullets([
            "провести анализ предметной области и существующих решений;",
            "выделить роли пользователей, сценарии взаимодействия, сущности и связи;",
            "спроектировать схему БД, ограничения целостности, роли, триггеры и хранимую функцию;",
            "реализовать БД, Redis-кэширование и интерфейс доступа к данным;",
            "исследовать влияние индексов PostgreSQL и Redis-кэша на показатели доступа к данным.",
        ])}
    """),
    slide(4, "Границы проектируемой системы", f"""
        {table([
            ["Данные", "Сценарии", "Не рассматривается"],
            ["пользователи, продавцы, товары, категории, заказы, позиции заказов, адреса, отзывы, история изменения цен",
             "просмотр каталога, управление товарами, оформление заказов, отзывы, администрирование, отчёты продаж",
             "логистика, курьеры, маршруты, графики складов и события фактической передачи товара"],
        ])}
        {bullets([
            "для хранения данных выбрана реляционная модель;",
            "состояние исполнения заказа фиксируется статусами заказа;",
            "количество товара относится к данным товара, которыми управляет продавец;",
            "резерв товара изменяется при обработке заказа.",
        ])}
    """),
    slide(5, "Роли пользователей и сценарии", f"""
        <div class="two roles">
            <div>{bullets([
                "гость — просмотр каталога и карточек товаров;",
                "покупатель — заказы, адреса и отзывы;",
                "продавец — товары, заказы продавца и статистика;",
                "администратор — модерация пользователей, товаров, категорий, отзывов и заказов;",
                "аналитик — чтение данных и формирование отчётов.",
            ])}</div>
            <div>{img(IMG / "UseCases.png", "diagram tall")}</div>
        </div>
    """),
    slide(6, "Диаграмма сущность-связь в нотации Чена", img(IMG / "ER_Chen.png", "diagram wide")),
    slide(7, "Схема базы данных", f"""
        <div class="two schema">
            <div>{img(IMG / "ER.png", "diagram er")}</div>
            <div>{bullets([
                "схема содержит 10 таблиц основной предметной области;",
                "связь многие-ко-многим между товарами и категориями вынесена в product_categories;",
                "корзина представлена заказом со статусом draft;",
                "история изменения цен хранится отдельно от текущей цены товара;",
                "схема приведена к третьей нормальной форме.",
            ])}</div>
        </div>
    """),
    slide(8, "Ограничения целостности и индексы", f"""
        {table([
            ["Группа", "Пример"],
            ["Заказы", "одна draft-корзина на покупателя; price_at_purchase фиксируется при оформлении"],
            ["Остатки", "reserved_quantity >= 0 и reserved_quantity <= stock_quantity"],
            ["Отзывы", "один отзыв покупателя на товар; рейтинг в диапазоне 1..5"],
            ["История цен", "триггер записывает старую и новую цену, время и инициатора"],
            ["Рейтинги", "при изменении отзыва пересчитываются рейтинги товара и продавца"],
            ["Индексы", "выборка заказов, отзывов, товаров продавца, активных записей и фильтрация по цене"],
        ])}
        <p>Ограничения защищают данные от некорректных состояний; индексы влияют только на время доступа.</p>
    """),
    slide(9, "Ролевая модель базы данных", f"""
        {table([
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
        ], "role") }
        <p>C — INSERT, R — SELECT, U — UPDATE, D — DELETE. Ограничения «только свои записи» проверяются приложением.</p>
    """),
    slide(10, "Триггеры и хранимая функция", f"""
        <div class="triggers">
            <div>{img(IMG / "trigger_catch_price_change.png", "diagram trig")}</div>
            <div>{img(IMG / "trigger_update_ratings_on_review.png", "diagram trig")}</div>
            <div>{bullets([
                "catch_price_change формирует журнал изменения цен товара;",
                "update_ratings_on_review поддерживает согласованность рейтингов товара и продавца;",
                "get_seller_statistics принимает продавца и период, возвращает количество заказов, выручку, средний чек и самый продаваемый товар;",
                "в расчёт статистики включаются только заказы paid, shipped и delivered.",
            ])}</div>
        </div>
        <p>Хранимая функция объединяет orders, order_items и products; цена берётся из price_at_purchase, поэтому изменение текущей цены товара не меняет статистику уже оформленных заказов.</p>
    """),
    slide(11, "Реализация и Redis-кэширование", f"""
        <div class="two redis">
            <div>{bullets([
                "PostgreSQL 16.11 — основное хранилище данных;",
                "Go 1.25.0 — REST API, серверный Web-интерфейс и бизнес-операции;",
                "Redis 8.6 — in-memory хранилище для кэша и отозванных JWT-токенов;",
                "кэширование реализовано по схеме Cache-Aside.",
            ])}</div>
            <div>{img(ASSETS / "redis_interaction.png", "diagram redisimg")}</div>
        </div>
        <p>PostgreSQL остаётся источником истины; Redis используется только для ускорения повторных чтений и проверки отозванных токенов.</p>
    """),
    slide(12, "Исследование 1: наличие индексов", f"""
        <p>Проверялись SQL-запросы: выборка каталога по диапазону цены, заказы покупателя, pending-заказы по дате, отзывы товара и вызов get_seller_statistics.</p>
        <div class="two charttext">
            <div>{img(RESEARCH / "db_index_execution_ms.png", "chart")}</div>
            <div class="notes">{bullets([
                "catalog_price_range: 9.351 мс без индекса и 0.073 мс с idx_products_price;",
                "pending_orders_by_date: 11.252 мс без индекса и 0.394 мс с idx_orders_status_created_at;",
                "для seller_statistics_function вторичные индексы миграций ускорения не дали.",
            ])}<p>Вывод: индекс ускоряет выборку только при соответствии запросу.</p></div>
        </div>
    """),
    slide(13, "Исследование 1: тип индекса", f"""
        <p>Сравнивались два случая на основном наборе из 40 000 товаров: заказы покупателя с сортировкой по дате (левый график) и подстрочный поиск товара по условию name ILIKE '%Zephyr%'. На правом графике число товаров меняется от 10 000 до 500 000.</p>
        <div class="two charts">
            <div>{img(RESEARCH / "db_index_type_ms.png", "chart")}</div>
            <div>{img(RESEARCH / "db_scaling_ms.png", "chart")}</div>
        </div>
        <div class="bottom-notes">{bullets([
            "составной индекс (user_id, created_at desc) устраняет Sort;",
            "без индекса по name поиск '%Zephyr%' идёт через Seq Scan по всей таблице (~25 мс на 40 000 товаров);",
            "GIN + pg_trgm (триграммы) обслуживает поиск через Bitmap Heap Scan и почти не зависит от размера таблицы.",
        ])}</div>
    """),
    slide(14, "Исследование 2: влияние Redis-кэша", f"""
        <p>Нагрузка подавалась на четыре GET-маршрута чтения: карточка товара (/products/{{id}}), каталог (/products, 12 товаров на странице), отзывы товара (/products/{{id}}/reviews, 10 отзывов на странице) и список категорий (/categories). На каждый маршрут — 300 запросов, до 12 одновременно.</p>
        <div class="two charttext">
            <div>{img(RESEARCH / "http_cache_latency_ms.png", "chart")}</div>
            <div class="notes">{bullets([
                "сравнивались сценарии Redis отключён и Redis включён с прогретым кэшем;",
                "карточка товара: среднее время 2.256 мс без Redis и 0.965 мс с прогретым кэшем;",
                "при прогретом кэше зафиксировано 1500 попаданий Redis и 0 строк PostgreSQL.",
            ])}<p>Вывод: прогретый кэш снижает время ответа и число обращений к PostgreSQL.</p></div>
        </div>
    """),
    slide(15, "Заключение", f"""
        {bullets([
            "разработана база данных онлайн-маркетплейса и приложение доступа к ней;",
            "спроектированы сущности, связи, ограничения целостности и ролевая модель доступа;",
            "реализованы триггеры журнала изменения цен и пересчёта рейтингов;",
            "реализована хранимая функция статистики продаж продавца;",
            "исследованы индексы PostgreSQL и Redis-кэширование как способы ускорения чтения данных.",
        ])}
        <p><b>Итог исследования:</b> индексы дают эффект при соответствии запросу, а Redis-кэш снижает время HTTP-ответа за счёт обслуживания повторных чтений из памяти.</p>
    """),
]


html = f"""<!doctype html>
<html lang="ru">
<head>
<meta charset="utf-8"/>
<title>Презентация курсовой работы</title>
<style>
@page {{ size: 13.333in 7.5in; margin: 0; }}
* {{ box-sizing: border-box; }}
body {{ margin: 0; background: white; color: black; font-family: "Times New Roman", serif; }}
.slide {{ width: 13.333in; height: 7.5in; padding: .56in .62in .42in .62in; page-break-after: always; position: relative; overflow: hidden; background: white; }}
.top {{ position: absolute; left: 0; right: 0; top: 0; height: .42in; padding: .08in .15in; background: #f5f5f5; color: #606060; font: 10pt "Times New Roman", serif; }}
h1 {{ margin: .08in 0 .28in 0; text-align: center; font-size: 32pt; font-weight: 400; line-height: 1.05; }}
main {{ font-size: 22pt; line-height: 1.22; }}
p {{ margin: 0 0 .22in 0; }}
ul {{ margin: 0; padding-left: .35in; }}
li {{ margin-bottom: .1in; }}
footer {{ position: absolute; left: 0; right: 0; bottom: .15in; display: flex; justify-content: center; color: #606060; font-size: 10pt; }}
table {{ width: 100%; border-collapse: collapse; font-size: 17pt; }}
th, td {{ border: 1px solid black; padding: .06in .08in; vertical-align: middle; }}
th {{ background: #f5f5f5; font-weight: 700; text-align: center; }}
.small {{ font-size: 10.5pt; }}
.role {{ font-size: 13.5pt; }}
.two {{ display: grid; grid-template-columns: 1fr 1fr; gap: .32in; align-items: start; }}
.roles {{ grid-template-columns: 4.25in 7.35in; }}
.schema {{ grid-template-columns: 6.2in 5.3in; }}
.redis {{ grid-template-columns: 4.75in 6.6in; }}
.charttext {{ grid-template-columns: 7.05in 4.25in; }}
.charts {{ grid-template-columns: 1fr 1fr; }}
.diagram, .chart {{ display: block; width: 100%; height: 100%; object-fit: contain; }}
.wide {{ height: 5.78in; }}
.tall {{ height: 5.95in; }}
.er {{ height: 6.02in; }}
.trig {{ height: 2.95in; }}
.triggers {{ display: grid; grid-template-columns: 3.2in 3.2in 4.7in; gap: .25in; align-items: start; }}
.triggers ul {{ font-size: 17pt; }}
.redisimg {{ height: 4.45in; border: 1px solid #d0d0d0; }}
.chart {{ height: 4.0in; }}
.charts .chart {{ height: 3.35in; }}
.notes ul, .notes p {{ font-size: 17pt; line-height: 1.14; }}
.bottom-notes ul {{ font-size: 16pt; line-height: 1.08; margin-top: .08in; }}
.cover {{ padding: 0; }}
.cover .logo {{ position: absolute; left: 2.25in; top: .36in; width: 1.25in; height: 1.25in; object-fit: contain; }}
.cover .ministry {{ position: absolute; left: 3.75in; top: .34in; width: 7.2in; text-align: center; font-size: 16pt; line-height: 1.08; font-weight: 700; }}
.cover h1 {{ position: absolute; left: 1.4in; top: 2.55in; width: 10.6in; font-size: 38pt; line-height: 1.1; font-weight: 700; }}
.cover .author {{ position: absolute; left: 1.15in; top: 5.25in; font-size: 24pt; line-height: 1.35; }}
.cover .city {{ position: absolute; left: 0; right: 0; bottom: .42in; text-align: center; font-size: 18pt; }}
</style>
</head>
<body>
{"".join(slides)}
</body>
</html>
"""


OUT.write_text(html, encoding="utf-8")
print(OUT)
