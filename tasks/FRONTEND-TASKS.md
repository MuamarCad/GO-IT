# Frontend Задачи — GO-IT

**Дата:** 10 ноября 2026
**Источник:** Технический аудит AUDIT-REPORT.md
**Исполнитель:** Frontend Developer

---

## Сводная таблица

| Приоритет | Кол-во | Оценка | Статус |
|-----------|--------|--------|--------|
| P0 | 4 | ~6 часов | ⏳ Не начато |
| P1 | 11 | ~9 часов | ⏳ Не начато |
| P2 | 5 | ~2 часа | ⏳ Не начато |
| **Итого** | **20** | **~17 часов** | |

---

## P0 — Критические

### T-01: Перевести `.preview` на CSS Custom Properties

- **Priority:** P0
- **Файлы:** `static/css/app.css`, `templates/pages/case_question.html`
- **Описание:** `.preview` использует захардкоженные цвета (`#ffffff`, `#1d4ed8`, `#4b5563`), не работает в тёмной теме
- **Что сделать:**
  1. Заменить `background: #ffffff` → `background: var(--surface)`
  2. Заменить `color: #101418` → `color: var(--text)`
  3. `.preview .tag` → `background: var(--accent-bg); color: var(--primary)`
  4. `.preview .meta` → `color: var(--muted)`
  5. `.preview .progress` → `background: var(--accent-bg); border-color: var(--border-strong)`
- **Acceptance:** Тёмная тема работает на `.preview`; contrast ≥ 4.5:1

### T-02: Убрать инлайн-стили из шаблонов

- **Priority:** P0
- **Файлы:** `lk_student.html`, `lk_mentor.html`, `lk_partner.html`, `login.html`, `case_result.html`, `app.css`
- **Описание:** 10+ инлайн-стилей — `margin-top`, `flex:none`, `color: var(--muted)` и т.д.
- **Что сделать:** Создать CSS-классы (см. полную таблицу в AUDIT-REPORT.md §3) и заменить в шаблонах
- **Acceptance:** `grep -rn "style=" templates/` — только допустимые (width для прогресс-баров)

### T-03: Исправить unclosed `<div>` в lk_mentor.html

- **Priority:** P0
- **Файлы:** `templates/pages/lk_mentor.html`
- **Описание:** `<div class="lk-badge lk-badge--ok">{{.Progress}}%</p>` — `</p>` вместо `</div>`
- **Что сделать:** Заменить `</p>` на `</div>`
- **Acceptance:** HTML-валидатор чистый, бейдж отображается корректно

### T-04: Личные кабинеты — данные из handler

- **Priority:** P0
- **Файлы:** `lk_student.html`, `lk_mentor.html`, `lk_partner.html`
- **Описание:** Все числа захардкожены: `3` кейса, `12` баллов, `40%` прогресс
- **Что сделать:** Заменить на `{{.Data.X}}` для всех динамических значений
- **Acceptance:** Шаблон рендерится без ошибок при пустых данных; все числа — Go-переменные

---

## P1 — Серьёзные

### T-05: `.tag--warm` для высокой контрастности

- **Priority:** P1
- **Файлы:** `static/css/app.css`
- **Что сделать:** Добавить `[data-contrast="high"] .tag--warm { background: var(--accent-bg); border: 1px solid var(--border); }`
- **Acceptance:** Контраст ≥ 4.5:1 в high-contrast

### T-06: `.btn--vk` и `.btn--ya` адаптация

- **Priority:** P1
- **Файлы:** `static/css/app.css`
- **Что сделать:** Добавить `[data-theme="dark"]` и `[data-contrast="high"]` стили для обеих кнопок
- **Acceptance:** Кнопки читаемы во всех 3 темах

### T-07: HTMX скачок при `show:window:top`

- **Priority:** P1
- **Файлы:** `templates/pages/case_question.html`, `static/js/theme.js`
- **Что сделать:** Заменить на `hx-swap="outerHTML swap:0.2s"` + плавный скролл к `#case-screen`
- **Acceptance:** Нет резкого скачка; страница плавно прокручивается

### T-08: Breadcrumbs → `<ol>`

- **Priority:** P1
- **Файлы:** 3 шаблона (case_question, case_result, roadmap), `app.css`
- **Что сделать:** Обернуть в `<ol>`, добавить CSS-разделитель `::before`
- **Acceptance:** Семантически верный `<ol>`, визуал без изменений

### T-09: Emoji → SVG в lk-nav

- **Priority:** P1
- **Файлы:** 3 LK-шаблона, `app.css`
- **Что сделать:** Заменить 13 emoji на inline SVG с `currentColor`
- **Acceptance:** Иконки одинаковые на Windows/macOS/Android

### T-10: `.lk-badge--ok` high-contrast

- **Priority:** P1
- **Файлы:** `static/css/app.css`
- **Что сделать:** Добавить `[data-contrast="high"]` с `border: 2px solid var(--border)`
- **Acceptance:** Бейдж виден чётко в high-contrast

### T-11: Таблица в lk_partner — горизонтальная прокрутка

- **Priority:** P1
- **Файлы:** `lk_partner.html`, `app.css`
- **Что сделать:** Обернуть в `<div class="table-wrapper">` с `overflow-x: auto`
- **Acceptance:** На 320px таблица не выходит за экран

### T-12: `.btn--sm` → 44px

- **Priority:** P1
- **Файлы:** `static/css/app.css`
- **Что сделать:** `min-height: 38px` → `min-height: 44px`
- **Acceptance:** Все кнопки LK ≥ 44px (WCAG 2.5.5)

### T-13: Прогресс-бар — aria-valuetext

- **Priority:** P1
- **Файлы:** case_question.html, case_result.html, lk_student.html
- **Что сделать:** Добавить `aria-valuetext="{{X}} процентов"`
- **Acceptance:** NVDA/VoiceOver озвучивает процент

### T-14: `:has()` fallback для Safari

- **Priority:** P1
- **Файлы:** `app.css`, `theme.js`
- **Что сделать:** JS-класс `.option--checked` для старых Safari
- **Acceptance:** Одинаковая подсветка на всех браузерах

### T-15: `dialog.rm-panel` — focus trap

- **Priority:** P1
- **Файлы:** `theme.js`
- **Что сделать:** Focus trap + autoFocus на заголовок
- **Acceptance:** Tab зациклен, фокус на контенте при открытии

---

## P2 — Мелкие

### T-16: `body` → `min-height: 100vh`
- **Файлы:** `app.css`
- **Acceptance:** Футер прижат на коротких страницах

### T-17: Footer nav → `<ul>`
- **Файлы:** `footer.html`, `app.css`
- **Acceptance:** Семантический `<nav><ul><li>`

### T-18: Unicode-символы ✓/✗ с aria
- **Файлы:** `case_result.html`
- **Acceptance:** `aria-hidden="true"` на символах

### T-19: `.lk-schedule-time` адаптация 320px
- **Файлы:** `app.css`
- **Acceptance:** Без горизонтального скролла

### T-20: `.lk-item:hover` — завершить микроинтеракт
- **Файлы:** `app.css`
- **Acceptance:** `box-shadow` + `translateY(-1px)` при hover

---

## Порядок выполнения

```
1. T-03 (15мин) → T-12 (15мин) → T-10 (15мин) → T-18 (15мин)
   → T-16 (15мин) → T-17 (30мин) → T-19 (15мин) → T-20 (30мин)
   → T-05 (30мин) → T-06 (30мин) → T-13 (30мин)
   → T-01 (1ч) → T-08 (1ч) → T-11 (1ч) → T-14 (1ч) → T-15 (1ч)
   → T-02 (2ч) → T-07 (1ч) → T-04 (2ч) → T-09 (3ч)
```
