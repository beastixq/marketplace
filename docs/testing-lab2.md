# Testing Lab2: проверка требований

Задание: `/home/roman/bmstu/7sem/testing/labs/2.pdf`. Это не WebLab#2.
Команды и эксплуатационные ограничения: [testing.md](testing.md#container-test-stand).
Публичный очищенный отчёт после deployment:
[Allure](https://beastixq.github.io/marketplace/).

## Соответствие

| Пункт PDF | Реализация / команда | Проверка |
| --- | --- | --- |
| З1. Тестовый стенд | `docker-compose.test.yml`, `make test-stand-up` | Отдельные API/MPA, PostgreSQL и Redis; UI готов. |
| З2. Integration доступа к данным и бизнес-логики | Компонентные `internal/repository/*_integration_test.go`, `internal/service/*_integration_test.go` | Настоящие repositories, `PgxTxManager` и PostgreSQL; не моки БД. |
| З3. Демонстрационный E2E | `tests/e2e/test_purchase.py`, `make test-e2e` | API-fixtures; login → cart → checkout → mock payment → ship → deliver → read review. |
| З4. ЛР1 и ЛР2 в CI | `.github/workflows/testing.yml`, `make test-all` | Unit, offline/vet, покрытие/отчёты ЛР1, integration и E2E. |
| З5. Запросы и трафик | `purchase.py --curl`, `make test-traffic` | Реальные curl-вызовы того же сценария; PCAP и проверка пар HTTP request/response через tshark. |
| Т1. Отдельный Docker test container | `tests/Dockerfile`, `tests/runner.sh`, `scripts/testing/runner.py` | CI получает точный commit; инструменты и закреплённые Python зависимости в image. Локально — снимок working tree. |
| Т2. Настоящее хранилище | `internal/testsupport/postgres.go`, реальные SQL-наборы | Оба integration-пакета обязательны; пропуски отвергаются. |
| Т3. Отдельный инстанс | Compose project по `STAND_ID` | Другой PostgreSQL-контейнер и data directory, не основная БД приложения. |
| Т4. Подготовка и восстановление | Existing Goose migrations, fixtures, `controller.snapshot/restore` | Data/sequence/Goose fingerprint; отдельная проверка ownership/ACL. |
| Т5. Порядок | `controller.pipeline` | Последовательно unit → integration → E2E на одном stand. |
| Т6. Окружение этой ЛР | Compose runner связывается с `db`/`api` своего project | E2E вызывает реальный Go server, integration — его PostgreSQL. |
| Т7. Мобильное устройство | Неприменимо | Проект — web/API. |
| Т8. Fail-fast и отчёт | Pipeline markers, original results, workflow `always()` | После сбоя последующие этапы skipped; сбой инфраструктуры не превращается в успешный продуктовый тест. |
| Т9. Тренды | Allure history, branch artifact + job token | Стабильные идентификаторы; public history очищена. |
| Т10. Откат при падении | Host controller, `make test-restore RUN_ID=…` | Full exclusive восстанавливает snapshot, включая sequences; shared E2E удаляет только свои данные, не ручные изменения. |
| Т11. Вычитывание брокера | Неприменимо | Брокера/очередей в приложении нет. |
| Т12. Завершение сессий | Серверного session storage нет; cleanup проверяет HTTP 401 | Неприменима очистка хранилища сессий. JWT stateless: это не blacklist и не обещание вечного отзыва токена при повторном использовании ID. |
| Т13. GUI необязателен | REST E2E | Browser automation в покупку не добавлена. |
| Т14. Повторяемость | `make test-infra-check` | Три integration-прогона: одинаковые результаты и исходные fingerprints. |
| Т15. Параллельность | Два независимых STAND_ID в `acceptance.py` | Одновременные full runs, отдельные данные/сети/порты/RUN_ID; разные runner inputs имеют разные image tags. |
| Т16. Успешные тесты | `make test-all` | Фактические результаты ниже; не подменять GitHub запуск локальным. |
| Т17. Организация | Component files, `_Integration`, pytest fixture и видимые Act/Assert | Применена рекомендация многошагового сценария; Go-файлы организованы по компонентам, а не лабораторным. |

## Фактические проверки исправлений 2026-10-15

- `revision-oct15-e2e-smoke`: requests purchase passed; API fixture cleanup завершён.
- `revision-oct15-curl-smoke`: curl purchase passed, PCAP проверен tshark.
- `revision-oct15-full-3`: unit/integration/E2E passed; snapshot restored;
  Allure full/public: 743 passed, 1 expected infrastructure skip. Локальный снимок
  не содержит `.git`, поэтому bootstrap self-test skipped; PostgreSQL skips нет.
- 34 Python infrastructure unit tests прошли в окружении закреплённых зависимостей.
- Все 708 прежних history IDs присутствуют среди текущих результатов, а не только
  в скопированной истории. Добавлены новые случаи и результаты инфраструктуры.
- `actionlint@v1.7.7` и `git diff --check` прошли; независимое ревью не выявило
  Critical/Important.
- `make test-infra-check`, `check-70f92e62`: 17 ожидаемых исходов, exit 0;
  три integration повтора, два одновременных full, partial fixture/foreign-link guard,
  kill integration/E2E runner, kill exclusive/shared controller, explicit recovery,
  fail-fast и успешный следующий прогон; PCAP проверен. Recovery завершён.
- Облачная проверка исправлений: [Actions 38775314127](https://github.com/beastixq/marketplace/actions/runs/38775314127),
  commit `d28f259`: test job success, 744 passed, 0 failed/broken/skipped,
  unit/integration/E2E passed, recovery restored=true, полное покрытие/отчёты ЛР1
  сохранены. Full/public counts совпадают, old history сохранена.
- Очищенный cloud report открыт локально в настоящем браузере: Unit 630,
  Integration 76, Infrastructure 37, E2E 1, правильный заголовок и тренды.
  Первый `publish-report` был отклонён branch policy environment: TAD-Lab2 не была
  разрешена. Это отдельно от результатов тестов; доступность сайта подтверждает
  только успешная deployment job и HTTP/browser проверка URL.

## Ограничения

Full/integration требуют окна без ручных изменений; сервер на время snapshot/
restore останавливается. Shared E2E сохраняет работающий API и ручные записи, но
синтетические товары могут временно появляться в каталоге. При чужих связях cleanup
останавливается, а не удаляет их. После SIGKILL controller нужен explicit restore;
при выключенной машине немедленное восстановление не гарантируется.

Go JUnit даёт сценарии/статусы, но не внутренние бизнес-шаги. Детальные шаги и HTTP
вложения есть у pytest; публичная версия намеренно не содержит диагностических
вложений/трейсов. Известные BUG-001/BUG-004, cache TTL и JWT limitations не исправлялись
в бизнес-коде; учтены явно, см. [known issues](known-issues.md).
