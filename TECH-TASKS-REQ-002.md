# Технические задачи: REQ-002 Корректировки v2

## Источник
`client-requirements/REQ-002.md` — Повторный аудит UI/UX от 2026-11-11

## Статус
✅ Все задачи выполнены, тесты зелёные

---

## Статус задач

| Задача | Статус | Примечание |
|--------|--------|-----------|
| TD-A1 | ✅ Завершена | Удалён `data-once` — причина поломки кнопки «Войти» |
| TD-A2 | ✅ Завершена | `autocomplete="off"` + `new-password` — поля пусты для демо |
| TD-A3 | ✅ Завершена | Валидация email уже корректна (`!= ""` пропускает пустую строку) |
| TD-A4 | ✅ Завершена | Кнопка «Войти как Ментор» + JS `quickLogin()` |
| TD-A5 | ✅ Завершена | Кнопка «Войти как Партнёр» + JS `quickLogin()` |
| TD-A6 | ✅ Завершена | Эмодзи → inline SVG из дизайн-системы |
| TD-B1 | ✅ Завершена | `switch` для маршрутизации шаблонов уже был в handlers.go |
| TD-B2 | ✅ Завершена | `writing-mode: horizontal-tb` добавлен в app.css |
| TD-B3 | ✅ Завершена | `lk-sidebar top: 0`, `lk-layout min-height: 100vh` |
| TD-B4 | ✅ Завершена | SVG radar chart (6 осей) добавлен в lk_student.html |
| TD-B5 | ✅ Завершена | Демо-данные: 145 баллов, 2 кейса, метрики |
| TD-B6 | ✅ Завершена | Иконка выхода — дверь вместо стрелки |
| TD-C1 | ✅ Завершена | Project Manager — 20 вопросов по 8 темам |
| TD-C2 | ✅ Завершена | Product Manager — 20 вопросов по 8 темам |
| TD-C3 | ✅ Завершена | Roadmap PM — 7 стадий, 36 узлов |

## Анализ корневых проблем

### Почему прошлые правки не сработали
1. **HTMX-атрибут `data-once` на форме входа** перехватывает submit — кнопка «Войти» не отправляет форму
2. **Удаление `roleAuth()` в REQ-001** без добавления альтернативного входа для Ментора/Партнёра через форму — они теперь не могут войти через email+password
3. **lkStudentPage handler не маршрутизирует** на отдельные шаблоны для `/lk/student/mentors` и `/lk/student/settings` — return 404
4. **Все LK-шаблоны используют статические нулевые данные** — реальных демо-данных нет

---

## Группа A: Критические баги авторизации

### TD-A1 [BUG] Кнопка «Войти» не работает — HTMX перехватывает submit
**Требование REQ-002:** п.1, п.6
**Файлы:** `templates/pages/login.html` (строка ~83: `<form id="auth-form-email" method="post" data-once>`)
**Корневая проблема:** `data-once` — это HTMX-атрибут, который запрещает повторные submit. Браузер/HTMX игнорирует обычный form submit с этим атрибутом.
**Решение:**
- Удалить `data-once` из `<form id="auth-form-email">`
- Заменить на чистую форму без HTMX: `method="post" action="/auth"`
- Убедиться, что форма не использует HTMX-атрибуты (`hx-boost`, `hx-target` и т.д.)

### TD-A2 [BUG] Поля email и пароль должны быть пустыми для демо
**Требование REQ-002:** п.1
**Файлы:** `templates/pages/login.html`, `internal/http/handlers/handlers.go` (Login)
**Проблема:** Браузер автозаполняет поля из сохранённых учётных данных. Для демо-режима нужно принудительно пустые поля.
**Решение:**
- В `login.html`: добавить `autocomplete="off"` для email и `autocomplete="new-password"` для пароля
- Можно также добавить `value=""` для явного пустого значения
- `Login` handler должен возвращать пустую форму — убедиться, что нет предзаполненных значений

### TD-A3 [BUG] При пустых полях — ложная ошибка «Проверьте правильность email»
**Требование REQ-002:** п.2
**Файлы:** `internal/http/handlers/handlers.go` (LoginPost, строки ~475–489)
**Проблема:** Валидация email срабатывает даже при пустом поле. Посмотреть код: `if email := r.FormValue("email"); email != ""` — уже есть проверка, возможно проблема в другом месте.
**Решение:**
- Проверить, что пустая строка `""` не проходит валидацию email
- Убедиться, что `validateEmail("")` возвращает nil и не показывается AlertError
- Протестировать: пустые поля → нажать «Войти» → авторизация с ролью student → редирект в `/lk/student`

### TD-A4 [BUG] ЛК Ментора недоступен — нет кнопки входа
**Требование REQ-002:** п.3
**Файлы:** `templates/pages/login.html`, `internal/http/handlers/handlers.go` (LoginPost)
**Корневая проблема:** В REQ-001 удалён блок `auth-quick` с кнопками "Войти как Ментор/Партнёр" (JS-функция `roleAuth()`), но не добавлены альтернативные пути.
**Решение:**
- Добавить в login.html выбор роли через табы (уже есть) + кнопку отправки формы с ролью
- Либо: добавить две новые кнопки "Войти как Ментор" и "Войти как Партнёр" ниже формы (без дублирования табов)
- При клике — отправить форму с `role=mentor` или `role=partner` скрытым полем
- LoginPost должен валидировать роль и редиректить правильно (уже работает)

### TD-A5 [BUG] ЛК Партнёра недоступен — нет кнопки входа
**Требование REQ-002:** п.4
**Файлы:** `templates/pages/login.html`
**Решение:** См. TD-A4 — добавить кнопку "Войти как Партнёр"

### TD-A6 [DESIGN] Описание слева от авторизации «пестрит стикерами»
**Требование REQ-002:** п.5
**Файлы:** `templates/pages/login.html`, `static/css/app.css`
**Проблема:** Эмодзи 📋🗺️⭐ в `.auth-feature` выглядят как стикеры, слишком пёстро.
**Решение:**
- Заменить эмодзи на inline SVG-иконки из единого набора (как в lk-nav)
- Или: убрать эмодзи, оставить только текстовые заголовки и описания
- Уменьшить padding/margin в `.auth-feature` для более компактного вида
- Сделать единый стиль: иконка + заголовок h3 + описание p

---

## Группа B: Критические баги ЛК

### TD-B1 [BUG] Маршрутизация LK-страниц не работает — 404
**Требование REQ-002:** п.7, п.8
**Файлы:** `internal/http/handlers/handlers.go` (lkStudentPage, строки ~586–730)
**Корневая проблема:** `lkStudentPage` handler должен маршрутизировать на разные шаблоны:
- `/lk/student/settings` → `lk_settings.html`
- `/lk/student/mentors` → `lk_student_mentors.html`
- `/lk/student/cases` → `lk_student_cases.html`
- `/lk/student/roadmap` → `lk_student_roadmap.html`
- `/lk/student/volunteering` → `lk_student_volunteering.html`
**Решение:**
- Добавить switch в `lkStudentPage` для определения template name
- Если шаблон не существует — показать заглушку с контентом
- Убедиться, что все 5 шаблонов существуют

### TD-B2 [DESIGN] Ориентация надписей в «Рекомендуем продолжить»
**Требование REQ-002:** п.9
**Файлы:** `templates/pages/lk_student.html` (lk-item-title, lk-item-meta)
**Проблема:** Текст в карточках кейсов и менторов — ориентация некорректна (согласно клиенту).
**Решение:**
- Проверить CSS `.lk-item-title`, `.lk-item-meta`, `.lk-item`
- Убедиться, что `text-orientation: mixed`, `writing-mode: horizontal-tb`
- Никакого перевёрнутого текста, vertical-orientations

### TD-B3 [BUG] Сайдбар не до конца вверху — белое пятно
**Требование REQ-002:** п.10
**Файлы:** `static/css/app.css` (.lk-layout, .lk-sidebar, .lk-main, body)
**Проблема:** `lk-sidebar` не прижимается к верху экрана, есть белое пространство над сайдбаром.
**Решение:**
- Проверить CSS для `.lk-layout`: `display: flex`, `.lk-sidebar: flex-shrink: 0`
- Проверить `.lk-sidebar`: `padding-top: 0`, `margin-top: 0`
- Проверить `body`: `margin: 0`, `padding: 0` — убрать все отступы
- Проверить `.lk-user` внутри сайдбара — может быть padding-top
- Убедиться, что `.lk-layout` занимает `100vh` и не имеет margin-top

### TD-B4 [DESIGN] Радар-чарт скиллов ученика
**Требование REQ-002:** п.11
**Файлы:** `templates/pages/lk_student.html` (секция "Обзор")
**Решение:**
- Добавить SVG radar chart в `lk_student.html` после статистики
- SVG должен показывать 5 осей: Frontend, Backend, PM, Soft Skills, Tools
- Заполнить демо-данными: {[65, 45, 55, 70, 50]} из 100
- Использовать fill с полупрозрачным цветом (`fill: rgba(29, 78, 216, 0.2)`)
- Стилизовать в соответствии с CSS Custom Properties
- Подписать оси и значения

### TD-B5 [BUG] Данные по выполнению — нули
**Требование REQ-002:** п.12
**Файлы:** `internal/http/handlers/handlers.go` (lkStudentPage), `templates/pages/lk_student.html`
**Проблема:** `lkStudentPage` возвращает данные с нулями (или статический map[string]any с 0)
**Решение:**
- Заменить нулевые данные на реалистичные демо-данные:
  - CasesCompleted: 2
  - Points: 145
  - RoadmapsOpened: 1
  - Consultations: 3
  - ActiveRoadmap: "Frontend"
  - CurrentLevel: 3
  - TotalLevels: 5
  - ProgressPercent: 60
  - NextCaseHint: "Рекомендуем продолжить кейс «Первый день Backend-разработчика»"
- RecommendedCases: 1-2 кейса с реальными slug/title
- AvailableMentors: 2 ментора с реальными данными
- CompletedCases: 2 пройденных кейса с реалистичными результатами
- NextConsultation: 1 upcoming консультация

### TD-B6 [BUG] Иконка «Выйти» — заменить на дверь
**Требование REQ-002:** п.13
**Файлы:** `templates/pages/lk_student.html`, `templates/pages/lk_mentor.html`, `templates/pages/lk_partner.html`
**Проблема:** Иконка выхода — стрелка влево (exit), клиент просит дверь
**Решение:**
- Заменить SVG `<path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"/>` на дверь
- Дверь: `<path d="M13 3H5a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h8l4-4V5a2 2 0 0 0-2-2h-2V3z"/>` — или более реалистичная дверь
- Использовать Feather Icons стиль: `viewBox="0 0 24 24" stroke-width="2"`

---

## Группа C: Контент — новые кейсы и roadmap

### TD-C1 [FEATURE] Добавить кейс «Первый день Project Manager»
**Файлы:** Новый `internal/cases/project_manager.go`
**Решение:**
- Slug: "project-manager"
- Title: "Первый день Project Manager"
- Profession: "project-manager"
- ~15-20 вопросов по темам:
  - Управление командой
  - Планирование спринтов
  - Risk management
  - Stakeholder management
  - Agile/Scrum методологии
  - Документация и отчётность
  - Budgeting
  - Communication
- Добавить в `All()` в `internal/cases/frontend.go`

### TD-C2 [FEATURE] Добавить кейс «Первый день Product Manager»
**Файлы:** Новый `internal/cases/product_manager.go`
**Решение:**
- Slug: "product-manager"
- Title: "Первый день Product Manager"
- Profession: "product-manager"
- ~15-20 вопросов по темам:
  - Product discovery
  - User stories и personas
  - Data-driven decisions
  - A/B тестирование
  - Product metrics (DAU, MAU, retention)
  - Prioritization (RICE, MoSCoW)
  - Product roadmap
  - Competitive analysis
- Добавить в `All()` в `internal/cases/frontend.go`

### TD-C3 [FEATURE] Добавить roadmap «IT Product Manager»
**Файлы:** `content/roadmaps/product-manager.md`, `content/roadmaps/product-manager.diagram.json`
**Решение:**
- Создать markdown-файл с описанием roadmap
- Создать JSON-диаграмму с этапами:
  1. Foundation → product thinking, user research
  2. Discovery → hypotheses, validation, prototyping
  3. Delivery → Agile, sprint planning, CI/CD
  4. Growth → metrics, experimentation, scaling
- Каждая стадия: 2-3 node'а
- Связать с кейсом Product Manager

---

## Приоритеты

| Задача | Приоритет | Зависимости |
|--------|-----------|-------------|
| TD-A1 | MustHave | Нет |
| TD-A2 | MustHave | Нет |
| TD-A3 | MustHave | Нет |
| TD-A4 | MustHave | Нет |
| TD-A5 | MustHave | Нет |
| TD-B1 | MustHave | Нет |
| TD-B5 | MustHave | Нет |
| TD-B3 | ShouldHave | Нет |
| TD-B2 | ShouldHave | Нет |
| TD-B4 | ShouldHave | Нет |
| TD-B6 | ShouldHave | Нет |
| TD-A6 | ShouldHave | Нет |
| TD-C1 | ShouldHave | Нет |
| TD-C2 | ShouldHave | Нет |
| TD-C3 | ShouldHave | Нет |

---

## Распределение по агентам

### `frontend-engineer` (Группы A + B)
**Задачи:** TD-A1, TD-A2, TD-A3, TD-A4, TD-A5, TD-A6, TD-B1, TD-B2, TD-B3, TD-B4, TD-B5, TD-B6
**Фокус:** Исправить критические баги авторизации, LK маршрутизацию, демо-данные, визуальные проблемы

### `backend-engineer` (Группа C)
**Задачи:** TD-C1, TD-C2, TD-C3
**Фокус:** Новые кейсы и roadmap
