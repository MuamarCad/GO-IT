# Технический аудит GO-IT

**Дата:** 10 ноября 2026
**Версия:** 1.0
**Автор:** Tech Lead (AI-audit)
**Статус:** 🔴 Требуется исправление

> Связанные документы: [SPECIFICATION.md](SPECIFICATION.md) · [PLAN.md](PLAN.md) · [QA-тест-планы](docs/qa-test-plans.md) · [Design Review](DESIGN-REVIEW.md)

---

## 1. Executive Summary

| Метрика | Значение |
|---------|----------|
| **Всего страниц** | 21 |
| **Всего маршрутов** | 45 |
| **Всего тестов** | 31 (5 файлов) |
| **Покрытие тестами** | ~50% |
| **Найденных UI-ошибок** | **32** |
| **Найденных backend-проблем** | **18** |
| **Найдено уязвимостей безопасности** | **8** |
| **Общее состояние** | 🟡 MVP работает, но требует доработки перед защитой |

### Критические выводы

1. **4 P0 backend-уязвимости** — CSRF-проблемы, утечка сессий, PRG-нарушение
2. **3 P0 frontend-ошибки** — сломанная тёмная тема, статичные LK, unclosed HTML-теги
3. **32 UI-проблемы** — инлайн-стили, несогласованность тем, emoji-иконки
4. **Документация рассинхронизирована** — seed-данные, статусы требований не соответствуют коду

---

## 2. Backend Audit (18 проблем)

### P0 — Критические (блокируют защиту)

| ID | Проблема | Файлы | Влияние |
|----|----------|-------|---------|
| BE-001 | `cleanupLoop` не останавливается | `session.go`, `main.go` | Утечка goroutine при restart |
| BE-002 | Logout не удаляет сессию из store | `session.go`, `handlers.go` | Утечка памяти, session replay |
| BE-003 | CaseAnswer без PRG | `handlers.go:140-149` | F5 = повторный POST |
| BE-004 | GET accept/decline — CSRF-уязвимость | `router.go`, `handlers.go` | `<img src=accept>` атака |
| BE-005 | `fail()` — Load без Save | `handlers.go:546` | CSRF токен не в cookie |

### P1 — Серьёзные

| ID | Проблема | Файлы | Влияние |
|----|----------|-------|---------|
| BE-006 | Ошибки email/phone не видны в UI | `handlers.go`, `login.html` | Пользователь не знает об ошибке |
| BE-007 | CaseStart — бесполезный Save | `handlers.go:105` | Лишняя перезапись cookie |
| BE-008 | Filter перезаписывается в цикле | `handlers.go:262` | Баг-прими |
| BE-009 | Seed перезаписывает данные | `postgres/store.go` | Лишние UPDATE |
| BE-010 | Миграции не в транзакции | `postgres/store.go` | Race condition |
| BE-011 | CSRF не привязан к fingerprint | `session.go` | MITM-риск |
| BE-012 | `safe` функция — XSS-риск | `render.go` | Потенциальный XSS |
| BE-013 | Нет защиты от дублирования заявок | `application_service.go` | Дубликаты заявок |

### P2 — Мелкие

| ID | Проблема | Влияние |
|----|----------|---------|
| BE-014 | `page()` бесполезная перезапись cookie | Трафик |
| BE-015 | Cleanup нет таймаута на мьютекс | Rare deadlock |
| BE-016 | Migrate каждый Connect | Лишняя нагрузка |
| BE-017 | Seed запускается всегда | In-memory |
| BE-018 | Application без поля Role | Расхождение со SPEC |

---

## 3. Frontend/UI Audit (32 проблемы)

### P0 — Критические

| ID | Проблема | Файлы | Влияние |
|----|----------|-------|---------|
| UI-01 | `.preview` сломан в тёмной теме | `app.css`, `case_question.html` | Белый текст на белом фоне |
| UI-02 | LK полностью статичные | `lk_*.html` | Нет реальных данных |
| UI-03 | unclosed `<div>` в lk_mentor.html | `lk_mentor.html` | Невалидный HTML |
| UI-04 | 10+ инлайн-стилей | Все LK-шаблоны | Нарушение DRY, сложный рефакторинг |

### P1 — Серьёзные

| ID | Проблема | Файлы | Влияние |
|----|----------|-------|---------|
| UI-05 | `.tag--warm` не для high-contrast | `app.css` | Нечитаемо |
| UI-06 | `.btn--vk/btn--ya` без адаптации | `app.css` | Брендовые кнопки не читаются |
| UI-07 | HTMX скачок вверх | `case_question.html` | UX — потеря контекста |
| UI-08 | Breadcrumbs не `<ol>` | 3 шаблона | Семантика, accessibility |
| UI-09 | Emoji-иконки в lk-nav | 3 LK-шаблона | Разный рендеринг на ОС |
| UI-10 | `lk-badge--ok` без high-contrast | `app.css` | Бейдж не виден |
| UI-11 | Таблица без скролла | `lk_partner.html` | Выход за экран |
| UI-12 | `btn--sm` = 38px | `app.css` | Меньше WCAG 44px тач-таргета |
| UI-13 | Прогресс-бар без aria-valuetext | 3 шаблона | Accessibility |
| UI-14 | `:has()` fallback для Safari | `app.css`, JS | Старый Safari |
| UI-15 | `dialog.rm-panel` без focus trap | `theme.js` | Accessibility |

### P2 — Мелкие

| ID | Проблема | Влияние |
|----|----------|---------|
| UI-16 | `body` без `min-height: 100vh` | Футер не прижат |
| UI-17 | Footer nav без `<ul>` | Семантика |
| UI-18 | Unicode-символы ✓/✗ без aria | Accessibility |
| UI-19 | `.lk-schedule-time` на 320px | Layout |
| UI-20 | `.lk-item:hover` — незавершённый hover | UX |
| UI-21 — UI-32 | Прочие (см. [DESIGN-REVIEW.md](DESIGN-REVIEW.md)) | Различные |

---

## 4. Security Audit (8 уязвимостей)

| ID | Уязвимость | CVSS | Описание |
|----|-----------|------|---------|
| SEC-01 | GET accept/decline | 7.5 | CSRF через `<img>` |
| SEC-02 | Session replay | 6.5 | Logout не удаляет сессию |
| SEC-03 | CSRF token mismatch | 6.0 | `fail()` не устанавливает cookie |
| SEC-04 | POST repeat on F5 | 5.5 | CaseAnswer без PRG |
| SEC-05 | Goroutine leak | 4.0 | `cleanupLoop` не останавливается |
| SEC-06 | Seed data overwrite | 3.5 | `ON CONFLICT DO UPDATE` без WHERE |
| SEC-07 | Migration race | 3.0 | Без транзакции |
| SEC-08 | `safe` — potential XSS | 8.5 | Без валидации bluemonday |

---

## 5. Design Audit

### Оценка экранов

| Экран | Оценка | Главная проблема |
|-------|--------|-----------------|
| Главная | ⭐⭐⭐⭐☆ | Хорошая структура, хороший CTA |
| Кейс вопрос | ⭐⭐⭐⭐☆ | Отличный UX, но скачок HTMX |
| Кейс результат | ⭐⭐⭐☆☆ | Дисклеймер в `hint` (слабый) |
| Логин | ⭐☆☆☆☆ | 7 кнопок, перегружен |
| 404 | ⭐⭐⭐⭐☆ | Хороший |
| ЛК Студент | ⭐⭐☆☆☆ | Статичные данные, эмодзи |
| ЛК Ментор | ⭐⭐☆☆☆ | Статичные данные, emoji |
| ЛК Партнёр | ⭐⭐☆☆☆ | Статичные данные, emoji |
| Направления | ⭐⭐⭐☆☆ | Хорошие карточки |
| Roadmap | ⭐⭐⭐☆☆ | Хороший, но focus trap |

### Product Design рекомендации (ТОП-5)

1. **Заменить emoji → SVG** в LK-навигации (13 иконок)
2. **Создать систему spacing tokens** (`--space-1`...`--space-9`)
3. **Упростить страницу логина** (7 → 3 блока)
4. **Добавить `.btn--danger`** для кнопок «Отклонить»
5. **Стандартизировать заголовки** — убрать inline `font-size`

---

## 6. Итоговые задачи для команды

### Frontend — 20 задач (T-01…T-20)

**См. файл:** `tasks/FRONTEND-TASKS.md`

| Приоритет | Кол-во | Оценка |
|-----------|--------|--------|
| P0 | 4 | ~6 часов |
| P1 | 11 | ~9 часов |
| P2 | 5 | ~2 часа |
| **Итого** | **20** | **~17 часов** |

### Backend — 18 задач (AUD-001…AUD-018)

**См. файл:** `tasks/BACKEND-TASKS.md`

| Приоритет | Кол-во | Оценка |
|-----------|--------|--------|
| P0 | 5 | ~16 часов |
| P1 | 8 | ~21 час |
| P2 | 5 | ~8 часов |
| **Итого** | **18** | **~45 часов** |

### QA — 18 тест-планов + 50 проверок

**См. файл:** `docs/qa-test-plans.md`

| Тип | Кол-во |
|-----|--------|
| Детальные тест-планы | 18 |
| Регрессионных проверок | 50 |
| Браузеры для тестирования | 5 (Chrome, Firefox, Safari, Edge, Mobile) |
| AX-инструменты | 3 (VoiceOver, NVDA, Lighthouse) |

---

## 7. Порядок исправления

### Фаза 1: Security Blockers (1-2 дня)

```
1. SEC-01 → BE-004: GET accept/decline → POST
2. SEC-02 → BE-002: Logout удаляет сессию
3. SEC-04 → BE-005: fail() — Load + Save
4. SEC-03 → BE-003: CaseAnswer PRG
5. SEC-05 → BE-001: session.Stop()
```

### Фаза 2: Critical UI (1-2 дня)

```
1. UI-01 → T-01: .preview на CSS Custom Properties
2. UI-04 → T-02: Убрать инлайн-стили
3. UI-03 → T-03: Исправить unclosed div
4. UI-02 → T-04: LK из данных handler
```

### Фаза 3: UI Polish (2-3 дня)

```
T-05…T-20 (см. FRONTEND-TASKS.md)
```

### Фаза 4: Backend Polish (3-4 дня)

```
AUD-006…AUD-018 (см. BACKEND-TASKS.md)
```

### Фаза 5: Testing (2 дня)

```
Все тест-планы из qa-test-plans.md
```

---

## 8. Метрики успеха аудита

| Метрика | Было | Цель |
|---------|------|------|
| UI-ошибок | 32 | 0 |
| Backend-проблем | 18 | 0 |
| Уязвимостей | 8 | 0 |
| Инлайн-стилей | 10+ | 0 |
| Эмодзи в навигации | 13 | 0 (SVG) |
| Рассинхронизации док | 5 | 0 |

---

## 9. Документация аудита

| Файл | Описание |
|------|---------|
| `AUDIT-REPORT.md` (этот файл) | Сводный отчёт Tech Lead |
| `tasks/FRONTEND-TASKS.md` | 20 задач для frontend |
| `tasks/BACKEND-TASKS.md` | 18 задач для backend |
| `docs/qa-test-plans.md` | 18 тест-планов + 50 проверок |
| `DESIGN-REVIEW.md` | Product design рекомендации |
| `SPECIFICATION.md` (v1.3) | Обновлённая спецификация |
| `PLAN.md` (v1.3) | Обновлённый план развития |

---

## 10. Приложение A: Краткая статистика проекта

| Метрика | Значение |
|---------|----------|
| Go-файлов | ~25 |
| Шаблонных файлов | 21 page + 5 partial + 1 layout = 27 |
| CSS-файлов | 1 |
| JS-файлов | 3 |
| Маршрутов | 45 |
| Тестов | 31 |
| Доменных типов | 21 |
| SQL-миграций | 1 |
| Markdown-файлов | 5 |
| Diagram JSON | 3 |

---

**Конец отчёта.**
