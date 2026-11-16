# Документация GO-IT

**Статус:** Действует
**Владелец:** Tech Lead
**Обновлено:** 2026-11-13

Карта документации проекта: что где лежит, кому что читать и как добавить новый документ. Правила оформления — в [стандарте документации](standards/documentation-standard.md).

## С чего начать

| Я хочу… | Читать |
|---------|--------|
| Понять, что за проект и запустить его | [README](../README.md), [руководство разработчика](engineering/development.md) |
| Узнать, что система должна уметь | [Спецификация](product/specification.md) |
| Понять, как всё устроено и почему | [Обзор архитектуры](architecture/overview.md), [ADR](architecture/adr/README.md) |
| Внести изменение | [CONTRIBUTING](../CONTRIBUTING.md), [руководство разработчика](engineering/development.md) |
| Добавить кейс, roadmap или статью | [Руководство по контенту](engineering/content-guide.md) |
| Проверить качество перед защитой | [Стратегия тестирования](engineering/testing.md), [дорожная карта](planning/roadmap.md) |
| Починить сломавшийся сервис | [Runbook](engineering/runbook.md) |
| Изменить интерфейс | [Дизайн-система](design/design-system.md) |
| Узнать, что просил заказчик | [Журнал требований](product/requirements-log.md) |
| Написать новый документ | [Стандарт](standards/documentation-standard.md), [шаблоны](templates/README.md) |

## Карта документов

| Документ | Тип | Владелец |
|----------|-----|----------|
| [Спецификация](product/specification.md) | Справочник | Системный аналитик |
| [Журнал требований](product/requirements-log.md) | Справочник | Системный аналитик |
| [Обзор архитектуры](architecture/overview.md) | Объяснение | Tech Lead |
| [Индекс ADR](architecture/adr/README.md) | Записи решений | Tech Lead |
| [Дизайн-система](design/design-system.md) | Объяснение и справочник | Продуктовый дизайнер |
| [Бренд-бук](design/brand-book.docx) | Справочник (файл Word) | Продуктовый дизайнер |
| [Руководство разработчика](engineering/development.md) | Инструкция | Tech Lead |
| [Стратегия тестирования](engineering/testing.md) | Инструкция и справочник | QA |
| [Руководство по контенту](engineering/content-guide.md) | Инструкция и справочник | Системный аналитик |
| [Runbook](engineering/runbook.md) | Эксплуатация | Backend |
| [Дорожная карта и технический долг](planning/roadmap.md) | План | Tech Lead |
| [Аудит, раунд 1 (2026-11-10)](quality/audits/2026-11-10-audit.md) | Запись качества | Tech Lead |
| [Аудит, раунд 2 (2026-11-11)](quality/audits/2026-11-11-audit.md) | Запись качества | Tech Lead |
| [Постмортем: страницы кабинетов отвечали 500](quality/postmortems/2026-11-11-lk-500.md) | Запись качества | Tech Lead |
| [Постмортем: вход завершается 403](quality/postmortems/2026-11-12-login-403.md) | Запись качества | Tech Lead |
| [RFC-0001: публичный REST API (отложен)](rfc/0001-rest-api/README.md) | Предложение | Системный аналитик |
| [Стандарт документации](standards/documentation-standard.md) | Норматив | Tech Lead |
| [Глоссарий](standards/glossary.md) | Справочник | Системный аналитик |
| [Каталог шаблонов](templates/README.md) | Шаблоны | Tech Lead |

Вне `docs/`: [README](../README.md), [CONTRIBUTING](../CONTRIBUTING.md), [SECURITY](../SECURITY.md), [CHANGELOG](../CHANGELOG.md), [GIGACODE](../GIGACODE.md) (контекст для AI-агента). Исходные макеты и презентации лежат в `draft/` и документацией не считаются.

## Как добавить документ

1. Выберите тип и скопируйте [шаблон](templates/README.md) в папку своего типа.
2. Заполните метаданные (`Статус`, `Владелец`, `Обновлено`) и разделы, удалите подсказки.
3. Добавьте строку в карту выше (ADR — в [индекс ADR](architecture/adr/README.md)).
4. Выполните `make docs-check`.

## Перемещённые и удалённые документы

| Был | Стало |
|-----|-------|
| `SPECIFICATION.md` | [product/specification.md](product/specification.md) |
| `PLAN.md` | [planning/roadmap.md](planning/roadmap.md) |
| `DOC-STYLE.md` | [standards/documentation-standard.md](standards/documentation-standard.md) |
| `AUDIT-REPORT.md` | [quality/audits/2026-11-10-audit.md](quality/audits/2026-11-10-audit.md) и [2026-11-11-audit.md](quality/audits/2026-11-11-audit.md) |
| `docs/docs.json` | [rfc/0001-rest-api/openapi.draft.json](rfc/0001-rest-api/openapi.draft.json) |
| `BRAND-BOOK.docx` | [design/brand-book.docx](design/brand-book.docx) |
| `DESIGN-REVIEW.md` | Удалён: заменён [дизайн-системой](design/design-system.md) |
| `docs/qa-test-plans.md` | Удалён: заменён [стратегией тестирования](engineering/testing.md) |
| `TECH-TASKS-REQ-001.md`, `TECH-TASKS-REQ-002.md` | Удалены: статусы перенесены в [журнал требований](product/requirements-log.md) |
| `tasks/` | Удалена: задания раунда 1 выполнены, история в Git |
