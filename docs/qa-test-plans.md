# QA Test Plans — GO-IT Platform

**Дата создания:** 10 ноября 2026
**Версия:** 1.0
**Статус:** Черновик для исполнения
**Среда тестирования:** `docker compose up` → `http://localhost:8080`

---

## Часть 1. Детальные тест-планы по результатам аудита

---

### UI-001. Тёмная тема — сломана компонент `.preview`

| Параметр | Значение |
|----------|----------|
| **ID** | UI-001 |
| **Название** | Тёмная тема не применяется к блоку `.preview` на главной странице |
| **Тип** | UI / CSS |
| **Приоритет** | 🔴 High |
| **Область** | `static/css/app.css` (строки ~574-583), `templates/pages/home.html` |

**Описание проблемы:** Класс `.preview` имеет жёстко заданные светлые цвета (`background: #ffffff; color: #101418;`). Для `[data-theme="dark"]` правило отсутствует — блок остаётся белым на тёмном фоне.

**Шаги воспроизведения:**

| Шаг | Действие |
|-----|----------|
| 1 | Запустить приложение: `docker compose up --build` |
| 2 | Открыть `http://localhost:8080` в Chrome |
| 3 | Нажать кнопку ☀️ (переключить тему) → включится тёмная тема |
| 4 | Прокрутить до блока «Найди свой путь в IT» |
| 5 | Визуально оценить блок `.preview` справа от заголовка |

**Ожидаемый результат:** Блок `.preview` адаптируется к тёмной теме: фон тёмный, текст светлый, бордеры корректны.

**Фактический результат (ожидаемый баг):** Блок остаётся белым (`#ffffff`) на тёмном фоне (`#070b18`). Контрастность нарушена.

**Детали для разработчика:** Добавить в CSS:
```css
[data-theme="dark"] .preview {
  background: var(--surface);
  color: var(--text);
  border-color: var(--border);
}
[data-theme="dark"] .preview .meta,
[data-theme="dark"] .preview .hint {
  color: var(--muted);
}
[data-theme="dark"] .preview .tag {
  background: var(--accent-bg);
  color: var(--primary);
}
[data-theme="dark"] .preview .progress {
  background: var(--accent-bg);
  border-color: var(--border-strong);
}
[data-theme="dark"] .preview .progress > span {
  background: var(--primary);
}
[data-theme="dark"] .preview-opts li {
  border-color: var(--border-strong);
}
[data-theme="dark"] .preview-opts li.on {
  border-color: var(--primary);
  background: var(--accent-bg);
}
```

**Критерии приёмки:**
- [ ] `.preview` в тёмной теме: фон = `var(--surface)`, текст = `var(--text)`
- [ ] Вложенный `.tag` корректен в тёмной теме
- [ ] `.preview-opts li` бордеры видимы
- [ ] `.preview-opts li.on` выделен
- [ ] `.progress` внутри `.preview` корректен

**Инструменты:** Chrome DevTools → Toggle Device Toolbar → Emulate CSS prefers-color-scheme / Manual theme toggle.

---

### UI-002. Личные кабинеты — статичные данные

| Параметр | Значение |
|----------|----------|
| **ID** | UI-002 |
| **Название** | Данные в ЛК (ученик, ментор, партнёр) не зависят от БД — всегда одни и те же |
| **Тип** | UI / Backend |
| **Приоритет** | 🟡 Medium |
| **Область** | `internal/http/handlers/handlers.go` (lkStudentPage, lkMentorPage, lkPartnerPage) |

**Шаги воспроизведения:**

| Шаг | Действие |
|-----|----------|
| 1 | Войти в ЛК ученика: POST `/auth` с ролью `student` |
| 2 | Перейти на `/lk/student` |
| 3 | Записать значения: «Кейса пройдено» = 3, «Баллов набрано» = 12 |
| 4 | В PostgreSQL выполнить: `DELETE FROM applications;` (если seed создал записи) |
| 5 | Перезагрузить страницу `/lk/student` |
| 6 | Сравнить значения |

**Ожидаемый результат:** Данные в ЛК соответствуют текущему состоянию БД (сессию, кейсы, заявки).

**Фактический результат (ожидаемый баг):** Данные статичны — всегда «3 кейса», «12 баллов», «5 учеников», «18 консультаций».

**Критерии приёмки:**
- [ ] ЛК ученика: количество кейсов = реальное количество завершённых attempt в сессии/БД
- [ ] ЛК ментора: NewApplications = реальное количество pending заявок
- [ ] ЛК партнёра: ActiveTasks = реальное количество published задач
- [ ] При перезагрузке страницы данные не меняются (консистентность)

---

### UI-003. Breadcrumbs — невалидный HTML

| параметр | Значение |
|----------|----------|
| **ID** | UI-003 |
| **Название** | Навигационная крошка не является семантическим списком |
| **Тип** | UI / HTML |
| **Приоритет** | 🟡 Medium |
| **Область** | `templates/pages/case_question.html`, `templates/pages/case_result.html` |

**Текущий HTML:**
```html
<nav class="breadcrumbs" aria-label="Хлебные крошки">
  <a href="/cases">Кейсы</a>
  <span aria-hidden="true">/</span>
  <span aria-current="page">Frontend</span>
</nav>
```

**Шаги воспроизведения:**

| Шаг | Действие |
|-----|----------|
| 1 | Открыть `/cases/frontend` (начало кейса) |
| 2 | Проверить HTML через DevTools → Elements |
| 3 | Проверить через W3C Nu Validator или Lighthouse |

**Ожидаемый результат:** Breadcrumbs — семантический `<ol>` или `<ul>` с `<li>`, соответствующий schema.org/BreadcrumbList.

**Фактический результат:** Просто `<nav>` с инлайн `<a>` и `<span>` — невалидная структура.

**Рекомендуемый HTML:**
```html
<nav aria-label="Хлебные крошки">
  <ol itemscope itemtype="https://schema.org/BreadcrumbList">
    <li itemprop="itemListElement" itemscope itemtype="https://schema.org/BreadcrumbListItem">
      <a itemprop="item" href="/cases"><span itemprop="name">Кейсы</span></a>
    </li>
    <li itemprop="itemListElement" itemscope itemtype="https://schema.org/BreadcrumbListItem">
      <span itemprop="name" aria-current="page">Frontend</span>
    </li>
  </ol>
</nav>
```

**Критерии приёмки:**
- [ ] Breadcrumbs используют `<ol>` или `<ul>`
- [ ] Каждый элемент — `<li>`
- [ ] Активный элемент имеет `aria-current="page"`
- [ ] W3C Validator не ругается
- [ ] Lighthouse AXE: "nav landmark should contain an ol or ul" — Pass

---

### UI-004. Эмодзи-иконки в lk-nav — разные на разных ОС

| параметр | значение |
|----------|----------|
| **ID** | UI-004 |
| **Название** | Unicode-эмодзи рендерятся по-разному на macOS, Windows, Linux, мобильных |
| **Тип** | UI / Cross-browser |
| **Приоритет** | 🟡 Medium |
| **Область** | `templates/pages/lk_student.html`, `lk_mentor.html`, `lk_partner.html` |

**Шаги воспроизведения:**

| Шаг | Действие |
|-----|----------|
| 1 | Войти в ЛК ментора |
| 2 | Посмотреть иконки в навигации на macOS (Safari) |
| 3 | Посмотреть на Windows (Chrome) |
| 4 | Посмотреть на Android (Chrome) |
| 5 | Сравнить внешний вид 📩, 👥, ✏️, 📅, 💚, 🗺️ |

**Ожидаемый результат:** Иконки выглядят одинаково или используются SVG-иконки вместо Unicode.

**Фактический результат:** Эмодзи 📩 (envelope) = синий на Apple, серый квадрат на Android, другой на Windows.

**Рекомендация:** Заменить Unicode-эмодзи на SVG-иконки (как в `header.html` — там SVG для icon-btn).

**Критерии приёмки:**
- [ ] Все иконки в lk-nav — SVG или CSS-based (не Unicode)
- [ ] Визуально одинаковы в Chrome/macOS, Chrome/Windows, Firefox/Linux, Safari/iOS, Chrome/Android
- [ ] Сохранён `aria-hidden="true"` на иконках

---

### UI-005. HTMX скачок при переключении шагов кейса

| параметр | значение |
|----------|----------|
| **ID** | UI-005 |
| **Название** | Резкий скачок viewport при HTMX-замене контента |
| **Тип** | UI / JS |
| **Приоритет** | 🔴 High |
| **Область** | `templates/pages/case_question.html` (HTMX attributes) |

**Текущий HTMX:**
```html
hx-swap="outerHTML show:window:top"
```

**Шаги воспроизведения:**

| Шаг | Действие |
|-----|----------|
| 1 | Открыть `/cases/frontend` |
| 2 | Выбрать любой вариант ответа |
| 3 | Нажать «Ответить» |
| 4 | Наблюдать за поведением скролла при переходе на следующий шаг |

**Ожидаемый результат:** Плавный скролл к заголовку нового шага, без рывка.

**Фактический результат:** Резкий скачок viewport в самое начало страницы (`show:window:top` — это «показать верх окна» при свопе).

**Критерии приёмки:**
- [ ] При переходе на следующий шаг контент появляется без резкого скролл-скачка
- [ ] Элемент с `data-focus` получает фокус после свапа
- [ ] `show:window:top` заменён на `show:#main:top` или `settle`
- [ ] На мобильных устройствах — нет потери позиции чтения

---

### UI-006. Инлайн-стили нарушают CSS-консистентность

| параметр | значение |
|----------|----------|
| **ID** | UI-006 |
| **Название** | Инлайн-стили в шаблонах ЛК обходят CSS Custom Properties |
| **Тип** | UI / CSS |
| **Приоритет** | 🟡 Medium |
| **Область** | Все LK-шаблоны |

**Примеры инлайн-стилей из кода:**

| Шаблон | Инлайн-стиль | Проблема |
|--------|-------------|----------|
| `lk_student.html` | `style="margin-bottom: 8px;"` | Жёсткий margin, не тематизируется |
| `lk_student.html` | `style="color: var(--muted);" ` | Работает, но лучше в CSS-классе |
| `lk_student.html` | `style="font-weight: 600; color: var(--primary);" ` | То же самое |
| `lk_mentor.html` | `style="margin-left:auto;"` | Жёсткое значение |
| `lk_mentor.html` | `style="display:flex; gap:8px;"` | Повторяющийся паттерн, нет CSS-класса |
| `lk_partner.html` | `style="width: 100%; border-collapse: collapse;"` | Inline-table лучше или CSS-класс |

**Шаги воспроизведения:**

| Шаг | Действие |
|-----|----------|
| 1 | Открыть DevTools → Elements |
| 2 | Найти все `style="..."` атрибуты в LK-страницах |
| 3 | Проверить, есть ли CSS-класс для каждого паттерна |

**Ожидаемый результат:** Инлайн-стили минимальны (только динамические значения: `width: X%` для progress).

**Критерии приёмки:**
- [ ] Нет инлайн-стилей для статичных значений (margin, padding, color, font-weight)
- [ ] Динамические значения (`width: {{percent}}%`) допустимы
- [ ] Все повторяющиеся паттерны вынесены в CSS-классы

---

### UI-007. Кнопки btn--vk/btn--ya не читаются в тёмной теме

| параметр | значение |
|----------|----------|
| **ID** | UI-007 |
| **Название** | Кнопки входа через VK/Yandex имеют жёсткие цвета, нарушающие тёмную тему |
| **Тип** | UI / CSS |
| **Приоритет** | 🔴 High |
| **Область** | `static/css/app.css` (строки ~439-452), `templates/pages/login.html` |

**Текущий CSS:**
```css
.btn--vk { background: #0077ff; color: #fff; border-color: #0077ff; }
.btn--vk:hover { background: #0066dd; }
.btn--ya { background: #fc3f1d; color: #fff; border-color: #fc3f1d; }
.btn--ya:hover { background: #e8350f; }
```

**Шаги воспроизведения:**

| Шаг | Действие |
|-----|----------|
| 1 | Открыть `/auth` |
| 2 | Включить тёмную тему |
| 3 | Проверить кнопки «Войти через VK» и «Войти через Яндекс» |

**Ожидаемый результат:** Бренд-цвета VK (#0077ff) и Яндекс (#fc3f1d) сохранены (это брендинг), но текст остаётся читаемым.

**Фактический результат:** Бренд-цвета сохранены — это ОК для брендинга. ПРОВЕРИТЬ: нет ли пересвета текста на этих фонах.

**Критерии приёмки:**
- [ ] Кнопка VK: `#0077ff` фон, белый текст → контраст ≥ 4.5:1 ✓ (3.89:1 — borderline)
- [ ] Кнопка Яндекс: `#fc3f1d` фон, белый текст → контраст ≥ 4.5:1 ✓ (4.16:1 — borderline)
- [ ] В тёмной теме кнопки сохраняют бренд-цвета
- [ ] На `data-contrast="high"` — кнопки имеют обводку/альтернативу

---

### UI-008. .btn--sm слишком мал для тач-таргетов

| параметр | значение |
|----------|----------|
| **ID** | UI-008 |
| **Название** | Мини-кнопки (38px min-height) ниже WCAG 2.5.5 тач-таргета (44px) |
| **Тип** | UI / Accessibility |
| **Приоритет** | 🟡 Medium |
| **Область** | `static/css/app.css` (~line 345), все LK-шаблоны |

**Текущий CSS:**
```css
.btn--sm { min-height: 38px; padding: 0 18px; font-size: 0.92rem; border-radius: 8px; }
```

**Шаги воспроизведения:**

| Шаг | Действие |
|-----|----------|
| 1 | Открыть страницу `/lk/mentor` |
| 2 | Кнопки «Принять» / «Отклонить» в заявках имеют класс `btn--sm` |
| 3 | Открыть DevTools → Compute Element Size |
| 4 | Проверить height каждой кнопки |

**Ожидаемый результат:** Все интерактивные элементы ≥ 44×44px (WCAG 2.5.5 Target Size).

**Фактический результат:** `btn--sm` = 38px min-height — на 6px ниже минимума.

**Критерии приёмки:**
- [ ] `btn--sm` min-height ≥ 44px (или target area ≥ 44px including padding)
- [ ] Кнопки «Принять/Отклонить» в ЛК ментора и партнёра ≥ 44px
- [ ] Кнопки «Продолжить/Записаться» в рекомендациях ≥ 44px

---

### UI-009. Таблица в lk_partner без горизонтальной прокрутки

| параметр | значение |
|----------|----------|
| **ID** | UI-009 |
| **Название** | Таблица «Сводка по волонтёрству» выходит за границы на узких экранах |
| **Тип** | UI / Responsive |
| **Приоритет** | 🟢 Low |
| **Область** | `templates/pages/lk_partner.html` |

**Шаги воспроизведения:**

| Шаг | Действие |
|-----|----------|
| 1 | Открыть `/lk/partner` |
| 2 | Уменьшить ширину окна до 400px (или открыть на мобильном) |
| 3 | Проверить таблицу «Сводка по волонтёрству» |

**Ожидаемый результат:** Таблица имеет горизонтальную прокрутку или адаптируется.

**Фактический результат:** Таблица выходит за границы контейнера, контент обрезается.

**Критерии приёмки:**
- [ ] Таблица обернута в контейнер с `overflow-x: auto`
- [ ] На мобильных — таблица скроллится горизонтально без обрезки
- [ ] Нет горизонтального скролла на `body`

---

### BE-001. CSRF — проверка на GET-запросы не работает

| параметр | значение |
|----------|----------|
| **ID** | BE-001 |
| **Название** | CheckCSRF пропускает все не-POST/PUT/PATCH/DELETE запросы, включая GET-аутентификацию |
| **Тип** | Backend / Security |
| **Приоритет** | 🔴 Critical |
| **Область** | `internal/session/session.go` (строки ~100-106) |

**Текущий код:**
```go
func (m *Manager) CheckCSRF(r *http.Request, data Data) bool {
    if r.Method != http.MethodPost && r.Method != http.MethodPut &&
       r.Method != http.MethodPatch && r.Method != http.MethodDelete {
        return true  // ← пропускает GET, HEAD, OPTIONS
    }
    token := r.FormValue(csrfName)
    return token != "" && hmac.Equal([]byte(token), []byte(data.CSRF))
}
```

**Шаги воспроизведения:**

| Шаг | Действие |
|-----|----------|
| 1 | Авторизоваться как ментор |
| 2 | Скопировать URL принятия заявки: `/lk/mentor/applications/1/accept` |
| 3 | Открыть этот URL напрямую в новой вкладке (GET) |
| 4 | Проверить, принята ли заявка |

**Ожидаемый результат:** GET-запросы к чувствительным endpoint'ам требуют CSRF-токен или перенаправляют на `/auth`.

**Фактический результат:** Accept/Decline работают через GET — любой URL можно открыть напрямую, CSRF не проверяется.

**Критерии приёмки:**
- [ ] Accept/Decline endpoints используют POST или PUT
- [ ] CSRF-токен проверяется на Accept/Decline
- [ ] GET-запрос к Accept/Decline → 405 Method Not Allowed или редирект

---

### BE-002. Logout не удаляет сессию из server store

| параметр | значение |
|----------|----------|
| **ID** | BE-002 |
| **Название** | doLogout сбрасывает cookie, но не удаляет сессию из `Manager.store` |
| **Тип** | Backend / Security |
| **Приоритет** | 🟡 Medium |
| **Область** | `internal/http/handlers/handlers.go` (doLogout), `internal/session/session.go` |

**Текущий код doLogout:**
```go
func (s *Server) doLogout(w http.ResponseWriter, r *http.Request, redirectOnFail string) {
    if r.Method != http.MethodPost {
        http.Redirect(w, r, "/auth", http.StatusSeeOther)
        return
    }
    http.SetCookie(w, &http.Cookie{
        Name: "goit_session", Value: "", Path: "/", HttpOnly: true, MaxAge: -1,
    })
    http.Redirect(w, r, "/", http.StatusSeeOther)
}
```

**Шаги воспроизведения:**

| Шаг | Действие |
|-----|----------|
| 1 | Войти в систему → сессия создана в `Manager.store` |
| 2 | Выйти через Logout |
| 3 | Cookie сброшен, но сессия в `Manager.store` остаётся |
| 4 | Повторный Login → создаётся НОВАЯ сессия |
| 5 | Через CleanupInterval (1 час) старая сессия удаляется |

**Ожидаемый результат:** При logout сессия удаляется из `Manager.store` немедленно.

**Фактический результат:** Сессия остаётся в памяти до автоматической очистки через 1 час.

**Критерии приёмки:**
- [ ] `doLogout` вызывает `s.Sessions.Delete(id)` или аналог
- [ ] После logout и reload — старая сессия удалена из store
- [ ] Память не течёт (проверить через pprof при множественных logout)

---

### BE-003. CaseAnswer — нет PRG (повторная отправка при F5)

| параметр | значение |
|----------|----------|
| **ID** | BE-003 |
| **Название** | CaseAnswer при ошибке валидации рендерит страницу напрямую (не PRG) |
| **Тип** | Backend / UX |
| **Приоритет** | 🟡 Medium |
| **Область** | `internal/http/handlers/handlers.go` (CaseAnswer, ~line 163) |

**Текущий код (ошибка валидации опции):**
```go
if errors.Is(err, service.ErrUnknownOption) && outcome.Next != nil {
    s.Render.HTML(w, "case_question.html", ...)  // ← прямая рендер-ответь, не редирект!
    return
}
```

**Шаги воспроизведения:**

| Шаг | Действие |
|-----|----------|
| 1 | Пройти кейс до последнего шага |
| 2 | Ответить на вопрос |
| 3 | Если сервер вернёт ErrUnknownOption — страница ошибки отрендерится напрямую |
| 4 | Нажать F5 (refresh) на странице ошибки |

**Ожидаемый результат:** После любого POST → redirect (PRG pattern). F5刷新 не вызывает повторную отправку.

**Фактический результат:** При ошибке валидации страница рендерится напрямую → F5 вызывает повторный POST.

**Критерии приёмки:**
- [ ] Все успешные ответы ведут к `http.Redirect(..., http.StatusSeeOther)`
- [ ] Ошибки валидации через flash-сообщение + redirect
- [ ] F5 на странице результата кейса не вызывает повторную отправку

---

### BE-004. Accept/Decline через GET — CSRF уязвимость

| параметр | значение |
|----------|----------|
| **ID** | BE-004 |
| **Название** | Accept и Decline — GET-запросы, подвержены CSRF через `<img src=...>` или `<a href=...>` |
| **Тип** | Backend / Security |
| **Приоритет** | 🔴 Critical |
| **Область** | `internal/http/handlers/handlers.go` (lkMentorAction, lkPartnerAction) |

**Текущий код:**
```go
func (s *Server) LKMentorAccept(w http.ResponseWriter, r *http.Request) {
    s.lkMentorAction(w, r, "accepted")
}
func (s *Server) lkMentorAction(w http.ResponseWriter, r *http.Request, action string) {
    // ...проверка LoggedIn...
    http.Redirect(w, r, "/lk/mentor/applications?msg="+msg, http.StatusSeeOther)
}
```

**Шаги воспроизведения:**

| Шаг | Действие |
|-----|----------|
| 1 | Войти как ментор |
| 2 | Создать HTML с `<a href="/lk/mentor/applications/1/accept">Принять</a>` |
| 3 | Отправить эту страницу ментору (имитация phishing) |
| 4 | Ментор открывает страницу → заявка принимается автоматически (GET) |

**Ожидаемый результат:** Accept/Decline — только POST с CSRF-токеном.

**Фактический результат:** Любой GET-запрос принимает/отклоняет заявку. CSRF = ноль.

**Критерии приёмки:**
- [ ] `/lk/mentor/applications/{id}/accept` → только POST
- [ ] `/lk/mentor/applications/{id}/decline` → только POST
- [ ] `/lk/partner/applications/{id}/accept` → только POST
- [ ] `/lk/partner/applications/{id}/decline` → только POST
- [ ] CSRF-токен проверяется внутри `lkMentorAction` / `lkPartnerAction`
- [ ] GET-запрос → 405 Method Not Allowed

---

### BE-005. Seed перезаписывает данные менторов

| параметр | значение |
|----------|----------|
| **ID** | BE-005 |
| **Название** | ON CONFLICT DO UPDATE в seed перезаписывает изменения менторов при каждом старте |
| **Тип** | Backend / Data integrity |
| **Приоритет** | 🟡 Medium |
| **Область** | `internal/repository/postgres/store.go` (seed logic) |

**Шаги воспроизведения:**

| Шаг | Действие |
|-----|----------|
| 1 | Запустить приложение с PostgreSQL |
| 2 | Войти как ментор, изменить данные через API/DB (если есть) или создать заявку |
| 3 | Перезапустить приложение: `docker compose restart` |
| 4 | Проверить, сохранились ли изменения ментора/заявки |

**Ожидаемый результат:** Seed выполняется один раз при первой миграции, не перезаписывает данные.

**Фактический результат:** `ON CONFLICT DO UPDATE` на каждом старте — seed перезаписывает все поля менторов.

**Критерии приёмки:**
- [ ] Seed использует `ON CONFLICT DO NOTHING` или флажок `seeded`
- [ ] После seed'а ручные изменения менторов сохраняются
- [ ] Заявки волонтёров не перезаписываются при рестарте

---

### AX-001. Breadcrumbs не <ol>

| параметр | значение |
|----------|----------|
| **ID** | AX-001 |
| **Название** | Breadcrumbs не являются семантическим `<ol>` — скринридеры не распознают порядок |
| **Тип** | Accessibility |
| **Приоритет** | 🟡 Medium |
| **Область** | `case_question.html`, `case_result.html` |

**Шаги воспроизведения:**

| Шаг | Действие |
|-----|----------|
| 1 | Включить VoiceOver (macOS: Cmd+F5) или NVDA (Windows) |
| 2 | Открыть `/cases/frontend` |
| 3 | Навигировать к breadcrumbs |
| 4 | Послушать, как скринридер озвучивает навигацию |

**Ожидаемый результат:** Скринридер озвучивает: «Навигационная цепочка: список из 2 элементов. Ссылка: Кейсы. Текущая страница: Frontend.»

**Фактический результат:** Скринридер озвучивает элементы по отдельности без контекста «навигационный список».

**Критерии приёмки:**
- [ ] Breadcrumbs — `<nav><ol>` (как в UI-003)
- [ ] VoiceOver/NVDA озвучивают контекст списка
- [ ] Lighthouse AXE: "Breadcrumb landmark should contain an ol or ul" — Pass
- [ ] Schema.org/BreadcrumbList микроразметка

---

### AX-002. Прогресс-бар без aria-valuetext

| параметр | значение |
|----------|----------|
| **ID** | AX-002 |
| **Название** | Прогресс-бар на странице кейса имеет aria-valuenow, но нет aria-valuetext |
| **Тип** | Accessibility |
| **Приоритет** | 🟡 Medium |
| **Область** | `templates/pages/case_question.html` |

**Текущий HTML:**
```html
<div class="progress" role="progressbar"
     aria-label="Прогресс кейса"
     aria-valuemin="0" aria-valuemax="{{.Data.Total}}"
     aria-valuenow="{{sub .Data.Step 1}}"
     aria-valuetext="Шаг {{.Data.Step}} из {{.Data.Total}}">
```

**Замечание:** `aria-valuetext` **УЖЕ ЕСТЬ** в коде. Проверить на странице результата.

**На странице результата (`case_result.html`):**
```html
<div class="progress" role="img" aria-label="{{.Data.Score}} из {{.Data.MaxScore}}">
```
— Здесь `role="img"`, а не `role="progressbar"`. Это корректно.

**Шаги воспроизведения:**

| Шаг | Действие |
|-----|----------|
| 1 | Открыть `/cases/frontend`, шаг 2 |
| 2 | Проверить прогресс-бар через DevTools Accessibility |
| 3 | Проверить aria-valuetext |

**Ожидаемый результат:** aria-valuetext = «Шаг 2 из 4» (текстовое описание числа).

**Фактический результат:** aria-valuetext присутствует — проверено. ПРОВЕРИТЬ другие прогресс-бары в ЛК.

**Критерии приёмки:**
- [ ] Прогресс-бар на case_question.html: aria-valuetext = текстовое описание
- [ ] Прогресс-бар в lk_student.html: имеет aria-valuetext «40%» или аналог
- [ ] Все `role="progressbar"` имеют aria-valuetext

---

### AX-003. Unicode-символы ✓/✗ без aria-label

| параметр | значение |
|----------|----------|
| **ID** | AX-003 |
| **Название** | Символы ✓ и ✗ не имеют aria-label — скринридеры озвучивают как символы |
| **Тип** | Accessibility |
| **Приоритет** | 🟡 Medium |
| **Область** | `templates/pages/case_result.html` |

**Текущий HTML:**
```html
{{if .IsCorrect}}
  <p class="status ok">✓ Подходящий выбор</p>
{{else}}
  <p class="status miss">✗ Более точный вариант: {{.Correct.Text}}</p>
{{end}}
```

**Шаги воспроизведения:**

| Шаг | Действие |
|-----|----------|
| 1 | Пройти кейс до результата |
| 2 | Включить VoiceOver/NVDA |
| 3 | Перейти к строкам статуса |

**Ожидаемый результат:** Скринридер озвучивает: «Подходящий выбор» или «Более точный вариант: ...»

**Фактический результат:** Скринридер может произнести: «Check mark. Подходящий выбор» или «X mark. Более точный вариант...»

**Критерии приёмки:**
- [ ] `role="img"` + `aria-label="Правильно"` на ✓
- [ ] `role="img"` + `aria-label="Неправильно"` на ✗
- [ ] Либо заменить на `<span aria-hidden="true">✓</span>`
- [ ] Lighthouse AXE: "Elements must have sufficient color contrast" (проверить цвета #0f7b3e / #b42318)

---

### AX-004. dialog.rm-panel без focus trap

| параметр | значение |
|----------|----------|
| **ID** | AX-004 |
| **Название** | `<dialog.rm-panel>` не имеет focus trap — Tab-навигация уходит за пределы диалога |
| **Тип** | Accessibility |
| **Приоритет** | 🔴 High |
| **Область** | `templates/partials/roadmap_diagram.html`, JS-файлы темы |

**Шаги воспроизведения:**

| Шаг | Действие |
|-----|----------|
| 1 | Открыть `/roadmaps/frontend` |
| 2 | Нажать на тему в схеме (откроется dialog.rm-panel) |
| 3 | Нажать Tab repeatedly |
| 4 | Проследить, уходит ли фокус за пределы диалога |

**Ожидаемый результат:** Фокус захвачен внутри диалога. Tab зациклен между элементами внутри panel. Escape закрывает panel.

**Фактический результат:** Фокус уходит на элементы страницы за dialog (кнопки, ссылки).

**Критерии приёмки:**
- [ ] Focus trap: Tab/Shift+Tab циклируют внутри dialog
- [ ] Закрытие по Escape
- [ ] При открытии dialog — фокус на заголовок `.rm-panel-title`
- [ ] При закрытии — фокус возвращается на триггер (rm-node-body)
- [ ] `inert` на body или `aria-hidden="true"` на page content
- [ ] Логика focus trap — в JS-файле (theme.js или отдельный файл)

---

### AX-005. Skip-link не тестируется

| параметр | значение |
|----------|----------|
| **ID** | AX-005 |
| **Название** | Skip-link существует в CSS, но не найден в HTML |
| **Тип** | Accessibility |
| **Приоритет** | 🟢 Low |
| **Область** | `static/css/app.css` (~line 73-76), `templates/layouts/base.html` |

**Шаги воспроизведения:**

| Шаг | Действие |
|-----|----------|
| 1 | Открыть любой页面 |
| 2 | Нажать Tab сразу после загрузки |
| 3 | Проверить, появляется ли ссылка «Skip to content» |

**Ожидаемый результат:** Первый Tab → skip-link visible → Enter → прыжок к `<main>` или первому `<h1>`.

**Критерии приёмки:**
- [ ] Skip-link присутствует в HTML: `<a class="skip-link" href="#main">Перейти к содержимому</a>`
- [ ] Skip-link visible при фокусе
- [ ] Переход к содержательному контенту (не к шапке)

---

### AX-006. Контрастность цветов в тёмной теме

| параметр | значение |
|----------|----------|
| **ID** | AX-006 |
| **Название** | Проверить контрастность всех цветов тёмной темы |
| **Тип** | Accessibility |
| **Приоритет** | 🟡 Medium |
| **Область** | `static/css/app.css` `[data-theme="dark"]` блок |

**Шаги воспроизведения:**

| Шаг | Действие |
|-----|----------|
| 1 | Включить тёмную тему |
| 2 | Проверить Lighthouse AXE контраст |
| 3 | Проверить цвета: `--muted: #b8c4db` на фоне `--bg: #070b18` |

**Расчёт контраста:**
- `#b8c4db` на `#070b18` ≈ 6.3:1 — Pass (≥ 4.5:1) ✓
- `#7f90b3` на `#070b18` ≈ 4.6:1 — Pass (≥ 3:1 для UI components) ✓
- `#2b3a5e` на `#070b18` ≈ 7.0:1 — Pass ✓

**Критерии приёмки:**
- [ ] Lighthouse AXE: "Elements must have sufficient color contrast" — нет Critical/Warnings
- [ ] Все body-text ≥ 4.5:1
- [ ] Все UI elements (бордеры, иконки) ≥ 3:1
- [ ] Высокая контрастность работает для всех цветов

---

### AX-007. focus-visible на button.icon-btn

| параметр | значение |
|----------|----------|
| **ID** | AX-007 |
| **Название** | Кнопки переключения темы — visible focus ring с `outline-offset: 3px` |
| **Тип** | Accessibility |
| **Приоритет** | 🟢 Low |
| **Область** | `static/css/app.css` |

**Проверка:** `.icon-btn` наследует `:focus-visible` из базовых стилей. Проверить визуальное отображение.

**Критерии приёмки:**
- [ ] `:focus-visible` на icon-btn — видимая обводка
- [ ] Не перекрывает SVG-иконку
- [ ] Нормально отображается во всех 3 темах

---

### BE-006. Auth — нет защиты от brute-force

| параметр | значение |
|----------|----------|
| **ID** | BE-006 |
| **Название** | LoginPost — нет rate limiting на попытки входа |
| **Тип** | Backend / Security |
| **Приоритет** | 🟡 Medium |
| **Область** | `internal/http/handlers/handlers.go` (LoginPost) |

**Шаги воспроизведения:**

| Шаг | Действие |
|-----|----------|
| 1 | Отправить 100+ POST-запросов на `/auth` с неправильными данными |
| 2 | Проверить, блокируется ли IP / сессия |

**Ожидаемый результат:** Rate limiting на login endpoint'е.

**Критерии приёмки:**
- [ ] После N неудачных попыток — задержка или блокировка
- [ ] Заголовок `Retry-After` при rate limit

---

### AX-008. Ссылки в новой вкладке без предупреждения

| параметр | значение |
|----------|----------|
| **ID** | AX-008 |
| **Название** | Внешние ссылки (`target="_blank"`) — есть предупреждение, проверить корректность |
| **Тип** | Accessibility |
| **Приоритет** | 🟢 Low |
| **Область** | `templates/partials/roadmap_diagram.html` |

**Текущий код:**
```html
<a href="{{.URL}}" target="_blank" rel="noopener noreferrer">
  <span class="rm-link-title">{{.Title}}</span>
  <small>{{host .URL}}</small>
  <span class="visually-hidden"> (откроется в новой вкладке)</span>
</a>
```

**Критерии приёмки:**
- [ ] `rel="noopener noreferrer"` присутствует
- [ ] `visually-hidden` текст = «(откроется в новой вкладке)»
- [ ] Скринридер озвучивает предупреждение
- [ ] Lighthouse AXE: "Links do not open new window without warning" — Pass

---

## Часть 2. Регрессионный чек-лист (50 проверок)

### Общие (10)

| № | Проверка | Область | Тип |
|---|----------|---------|-----|
| GEN-01 | `/health/live` возвращает 200 «ok» | Health | Backend |
| GEN-02 | `/health/ready` возвращает 200 (PostgreSQL запущен) | Health | Backend |
| GEN-03 | `/health/ready` возвращает 503 (PostgreSQL остановлен) | Health | Backend |
| GEN-04 | `/static/*` — CSS/JS/шрифты загружаются | Static | UI |
| GEN-05 | `/` — главная страница рендерится, 200 OK | Home | UI |
| GEN-06 | `/about` — страница рендерится | About | UI |
| GEN-07 | `/nonexistent` — 404 page (не blank error) | Fallback | UI |
| GEN-08 | `HEAD` запросы возвращают 200 без body | Chi middleware | Backend |
| GEN-09 | CORS — нет лишних заголовков (не web API) | Security | Backend |
| GEN-10 | Content-Security-Policy — нет заголовка (не требуется для SSR, но проверить что нет конфликтов) | Security | Backend |

### Навигация (5)

| № | Проверка | Область | Тип |
|---|----------|---------|-----|
| NAV-01 | Навигация — все ссылки работают, не 404 | Header | UI |
| NAV-02 | `aria-current="page"` на активной странице | Header | AX |
| NAV-03 | Бургер-меню на мобильных (JS включён) | Header | UI |
| NAV-04 | Skip-link работает | Layout | AX |
| NAV-05 | Breadcrumbs на case_question и case_result (см. UI-003, AX-001) | Case | AX |

### Темы (5)

| № | Проверка | Область | Тип |
|---|----------|---------|-----|
| TM-01 | Светлая тема — все элементы читаемы | CSS | UI |
| TM-02 | Тёмная тема — все элементы читаемы | CSS | UI |
| TM-03 | Высокая контрастность — все элементы различимы | CSS | AX |
| TM-04 | Переключение темы через кнопку — JS работает | JS | UI |
| TM-05 | Системная тема (`prefers-color-scheme`) применяется при первом визите | CSS | AX |

### Кейсы (7)

| № | Проверка | Область | Тип |
|---|----------|---------|-----|
| CS-01 | `/cases` — каталог рендерится | Case list | UI |
| CS-02 | `/cases/frontend` — начало кейса, форма рендерится | Case start | UI |
| CS-03 | Ответ на вопрос → переход на следующий шаг | Case answer | Backend |
| CS-04 | Неправильный ответ → переход на следующий шаг | Case answer | Backend |
| CS-05 | Последний вопрос → redirect на /result | Case flow | Backend |
| CS-06 | `/cases/frontend/result` — страница результата рендерится | Case result | UI |
| CS-07 | F5 на result — не вызывает повторный POST | Case PRG | Backend |

### Заявки (5)

| № | Проверка | Область | Тип |
|---|----------|---------|-----|
| AP-01 | Страница ментора с формой заявки рендерится | Mentor | UI |
| AP-02 | Заявка на ментора — валидация empty name/contact | Application | Backend |
| AP-03 | Заявка на волонтёрство — валидация | Application | Backend |
| AP-04 | Успешная заявка → редирект с `?sent=1` | Application | Backend |
| AP-05 | CSRF-токен в форме, сервер проверяет | CSRF | Security |

### Личные кабинеты (10)

| № | Проверка | Область | Тип |
|---|----------|---------|-----|
| LK-01 | `/auth` — страница входа рендерится | Login | UI |
| LK-02 | Login POST → редирект на `/lk/student` | Login | Backend |
| LK-03 | `/lk/student` — без авторизации → редирект на `/auth` | Auth check | Backend |
| LK-04 | `/lk/mentor` — без авторизации → редирект на `/auth` | Auth check | Backend |
| LK-05 | `/lk/partner` — без авторизации → редирект на `/auth` | Auth check | Backend |
| LK-06 | ЛК ученика — все подстраницы рендерятся (cases, roadmap, mentors, volunteering, settings) | Student LK | UI |
| LK-07 | ЛК ментора — все подстраницы рендерятся (applications, schedule, students, profile) | Mentor LK | UI |
| LK-08 | ЛК партнёра — все подстраницы рендерятся (tasks, applications, analytics, settings) | Partner LK | UI |
| LK-09 | Logout — сброс сессии, редирект на `/` | Logout | Backend |
| LK-10 | Логин с некорректным email → ошибка валидации | Login | Backend |

### Roadmaps (4)

| № | Проверка | Область | Тип |
|---|----------|---------|-----|
| RD-01 | `/roadmaps` — список рендерится | Roadmap list | UI |
| RD-02 | `/roadmaps/frontend` — схема рендерится | Roadmap | UI |
| RD-03 | `dialog.rm-panel` — открывается по клику на тему | Roadmap panel | UI/AX |
| RD-04 | Focus trap в dialog.rm-panel (см. AX-004) | Roadmap panel | AX |

### Блог (2)

| № | Проверка | Область | Тип |
|---|----------|---------|-----|
| BL-01 | `/blog` — список статей | Blog list | UI |
| BL-02 | `/blog/{slug}` — статья рендерится, Markdown → HTML | Blog article | UI |

### Направления (2)

| № | Проверка | Область | Тип |
|---|----------|---------|-----|
| DR-01 | `/directions` — каталог направлений | Direction list | UI |
| DR-02 | `/directions/{slug}` — детальная страница направления | Direction detail | UI |

### Мобильные/Кросс-браузер (2)

| № | Проверка | Область | Тип |
|---|----------|---------|-----|
| MB-01 | Адаптив на 375px (iPhone SE) — контент не обрезается | Responsive | UI |
| MB-02 | Адаптив на 768px (iPad) — layout корректен | Responsive | UI |

---

## Часть 3. Матрица покрытия: Задача × Тест

```
┌──────────────────────────────────────────────────────────────────────────────┐
│  ЛЕГЕНДА                                                                    │
│  T = Unit-тест (Go _test.go)                                                │
│  I = Integration-тест (testServer)                                          │
│  U = UI-ручное тестирование (DevTools, browser)                             │
│  A = Accessibility-тест (VoiceOver, NVDA, Lighthouse)                       │
│  S = Security-тест (curl, browser)                                          │
└──────────────────────────────────────────────────────────────────────────────┘

Задача        │ UI-001│ UI-002│ UI-003│ UI-004│ UI-005│ UI-006│ UI-007│ UI-008│ UI-009│ BE-001│ BE-002│ BE-003│ BE-004│ BE-005│ AX-001│ AX-002│ AX-003│ AX-004│ AX-005│ AX-006│ AX-007│ AX-008│ BE-006
──────────────┼───────┼───────┼───────┼───────┼───────┼───────┼───────┼───────┼───────┼───────┼───────┼───────┼───────┼───────┼───────┼───────┼───────┼───────┼───────┼───────┼───────┼───────┼───────
CSRF check    │       │       │       │       │       │       │       │       │       │   T   │   S   │   I   │   S   │       │       │       │       │       │       │       │       │       │
Logout cleanup│       │       │       │       │       │       │       │       │       │       │   T   │   I   │   S   │       │       │       │       │       │       │       │       │       │
CaseAnswer PRG│       │       │       │       │       │       │       │       │       │       │       │   I   │   I   │       │       │       │       │       │       │       │       │       │
Accept GET    │       │       │       │       │       │       │       │       │       │   S   │       │   I   │   S   │       │       │       │       │       │       │       │       │       │
Seed UPDATE   │       │       │       │       │       │       │       │       │       │       │       │       │       │   I   │       │       │       │       │       │       │       │       │
Preview CSS   │   U   │       │       │       │       │       │       │       │       │       │       │       │       │       │       │       │       │       │       │       │       │       │
LK static     │       │   U   │       │       │       │       │       │       │       │       │       │       │       │       │       │       │       │       │       │       │       │       │
Breadcrumbs   │       │       │   U   │       │       │       │       │       │       │       │       │       │       │       │   A   │       │       │       │   A   │       │       │       │
Emoji icons   │       │       │       │   U   │       │       │       │       │       │       │       │       │       │       │       │       │       │       │       │       │       │       │
HTMX jump     │       │       │       │       │   U   │       │       │       │       │       │       │       │       │       │       │       │       │       │       │       │       │       │
Inline styles │       │       │       │       │       │   U   │       │       │       │       │       │       │       │       │       │       │       │       │       │       │       │       │
VK/YA buttons │       │       │       │       │       │       │   U   │       │       │       │       │       │       │       │       │       │       │       │       │       │       │       │
btn--sm       │       │       │       │       │       │       │       │   U   │       │       │       │       │       │       │       │       │       │       │       │       │       │       │
Partner table │       │       │       │       │       │       │       │       │   U   │       │       │       │       │       │       │       │       │       │       │       │       │       │
AX: Breadcrumbs│       │       │       │       │       │       │       │       │       │       │       │       │       │       │   A   │       │       │       │       │       │       │       │
AX: ARIA PRG  │       │       │       │       │       │       │       │       │       │       │       │       │       │       │       │   U   │       │       │       │       │       │       │
AX: ✓/✗       │       │       │       │       │       │       │       │       │       │       │       │       │       │       │       │       │   A   │       │       │       │       │       │
AX: Focus Trap│       │       │       │       │       │       │       │       │       │       │       │       │       │       │       │       │       │   U   │       │       │       │       │
AX: Skip Link │       │       │       │       │       │       │       │       │       │       │       │       │       │       │       │       │       │       │   U   │       │       │       │
AX: Contrast  │       │       │       │       │       │       │       │       │       │       │       │       │       │       │       │   U   │       │       │       │   A   │       │       │
AX: Focus Vis │       │       │       │       │       │       │       │       │       │       │       │       │       │       │       │       │       │       │       │       │   U   │       │
AX: Target    │       │       │       │       │       │       │       │       │       │       │       │       │       │       │       │       │       │       │       │       │   U   │       │
AX: Links     │       │       │       │       │       │       │       │       │       │       │       │       │       │       │       │       │       │       │       │       │       │   A   │
Brute Force   │       │       │       │       │       │       │       │       │       │       │       │       │       │       │       │       │       │       │       │       │       │   S   │

┌──────────────────────────────────────────────────────────────────────────────┐
│  ЛК-Чеклист покрытие:                                                        │
│                                                                              │
│  GEN-01 → [none] (health check)                                             │
│  GEN-02 → [none] (health check)                                             │
│  GEN-03 → [none] (health check)                                             │
│  GEN-04 → [none] (static file serving)                                      │
│  GEN-05 → [none]                                                            │
│  GEN-06 → [none]                                                            │
│  GEN-07 → [none]                                                            │
│  GEN-08 → [none]                                                            │
│  GEN-09 → [none]                                                            │
│  GEN-10 → [none]                                                            │
│  NAV-01 → [none]                                                            │
│  NAV-02 → AX-001 (partial overlap)                                          │
│  NAV-03 → [none]                                                            │
│  NAV-04 → AX-005                                                            │
│  NAV-05 → UI-003, AX-001                                                    │
│  TM-01 → [none]                                                             │
│  TM-02 → UI-001, UI-007, AX-006                                            │
│  TM-03 → AX-006                                                             │
│  TM-04 → [none]                                                             │
│  TM-05 → [none]                                                             │
│  CS-01 → [none]                                                             │
│  CS-02 → [none]                                                             │
│  CS-03 → BE-003                                                             │
│  CS-04 → [none]                                                             │
│  CS-05 → BE-003                                                             │
│  CS-06 → [none]                                                             │
│  CS-07 → BE-003                                                             │
│  AP-01 → [none]                                                             │
│  AP-02 → [none]                                                             │
│  AP-03 → [none]                                                             │
│  AP-04 → [none]                                                             │
│  AP-05 → BE-001                                                             │
│  LK-01 → [none]                                                             │
│  LK-02 → [none]                                                             │
│  LK-03 → BE-001                                                             │
│  LK-04 → BE-001                                                             │
│  LK-05 → BE-001                                                             │
│  LK-06 → UI-002                                                             │
│  LK-07 → UI-002                                                             │
│  LK-08 → UI-002, UI-009                                                     │
│  LK-09 → BE-002                                                             │
│  LK-10 → [none]                                                             │
│  RD-01 → [none]                                                             │
│  RD-02 → [none]                                                             │
│  RD-03 → AX-004                                                             │
│  RD-04 → AX-004                                                             │
│  BL-01 → [none]                                                             │
│  BL-02 → [none]                                                             │
│  DR-01 → [none]                                                             │
│  DR-02 → [none]                                                             │
│  MB-01 → UI-009                                                             │
│  MB-02 → [none]                                                             │
└──────────────────────────────────────────────────────────────────────────────┘
```

---

## Сводная таблица приоритетов

| Приоритет | Кол-во | ID'ы |
|-----------|--------|------|
| 🔴 Critical/High | 6 | UI-001, UI-005, UI-007, BE-001, BE-004, AX-004 |
| 🟡 Medium | 12 | UI-002, UI-003, UI-004, UI-006, UI-008, BE-002, BE-003, BE-005, AX-001, AX-002, AX-003, AX-006, BE-006 |
| 🟢 Low | 4 | UI-009, AX-005, AX-007, AX-008 |

## Рекомендуемый порядок исправления

1. **🔴 Security-критичные:** BE-001 (CSRF), BE-004 (Accept/Decline GET)
2. **🔴 UX-критичные:** UI-001 (тёмная тема preview), UI-005 (HTMX скачок)
3. **🟡 Medium:** UI-007 (VK/YA), AX-004 (focus trap), BE-002 (logout), BE-003 (PRG), UI-003 (breadcrumbs)
4. **🟡 Medium:** UI-008 (btn--sm), BE-005 (seed), UI-006 (inline), AX-003 (✓/✗)
5. **🟡 Medium:** UI-002 (LK static), UI-004 (emoji), AX-002 (aria-valuetext), AX-006 (contrast), BE-006 (brute-force)
6. **🟢 Low:** UI-009 (table), AX-005 (skip-link), AX-007 (focus-visible), AX-008 (target blank)

---

## Appendix: Команды для запуска тестов

```bash
# Запуск приложения
docker compose up --build

# Go-тесты
make test

# Проверка внешних ссылок
make check-links

# Форматирование
make fmt

# Docker Compose (если нужно пересобрать)
make compose

# Ручное тестирование CSRF
curl -v http://localhost:8080/lk/mentor/applications/1/accept
# Ожидаемо: redirect или 405 (после fix)

# Ручное тестирование logout
curl -v -X POST http://localhost:8080/lk/student/logout -c cookies.txt
curl -v -c cookies.txt http://localhost:8080/lk/student
# Ожидаемо: redirect на /auth

# Лайтхаус (Lighthouse CI)
npx lighthouse http://localhost:8080 --view --output=json --output-path=lh-report.json
```
