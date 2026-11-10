# Backend Задачи — GO-IT

**Дата:** 10 ноября 2026
**Источник:** Технический аудит AUDIT-REPORT.md
**Исполнитель:** Backend Developer

---

## Сводная таблица

| Приоритет | Кол-во | Оценка | Статус |
|-----------|--------|--------|--------|
| P0 | 5 | ~16 часов | ⏳ Не начато |
| P1 | 8 | ~21 час | ⏳ Не начато |
| P2 | 5 | ~8 часов | ⏳ Не начато |
| **Итого** | **18** | **~45 часов** | |

---

## P0 — Критические

### AUD-001: Вызов session.Stop() при graceful shutdown

- **Priority:** P0
- **Файлы:** `cmd/web/main.go`, `internal/session/session.go`
- **Описание:** `cleanupLoop()` — бесконечный goroutine, `Stop()` не вызывается
- **Шаги:**
  1. В `main.go` после `pg.Close()` добавить: `srv.Sessions.Stop()`
  2. Проверить что `Stop()` в `session.go` корректно завершает цикл
- **Acceptance:** `go vet ./...` чистый; `kill -TERM` → goroutine < 1с

### AUD-002: Logout не удаляет сессию из server store

- **Priority:** P0
- **Файлы:** `internal/session/session.go`, `internal/http/handlers/handlers.go`
- **Описание:** `doLogout` сбрасывает cookie, но сессия остаётся в `Manager.store`
- **Шаги:**
  1. В `session.go` добавить метод: `func (m *Manager) Delete(id string)`
  2. В `handlers.go:doLogout` — вызвать `s.Sessions.Delete(id)` перед сбросом cookie
- **Acceptance:** После logout `Load` возвращает новую пустую сессию

### AUD-003: CaseAnswer — ошибка варианта рендерит напрямую (без PRG)

- **Priority:** P0
- **Файлы:** `internal/http/handlers/handlers.go`, `case_question.html`
- **Описание:** `ErrUnknownOption` → 422 HTML рендер → F5 = повторный POST
- **Шаги:**
  1. Заменить рендер 422 на: `sess.Attempt = state; SaveWithExpiry; Redirect(/cases/{slug}, 303)`
  2. В `case_question.html` добавить: `{{if .FlashError}}<div role="alert">{{.FlashError}}</div>{{end}}`
- **Acceptance:** F5 после ошибки → GET, не POST

### AUD-004: GET accept/decline → POST

- **Priority:** P0
- **Файлы:** `router.go`, `handlers.go`, `lk_mentor.html`, `lk_partner.html`
- **Описание:** Мутации через GET — CSRF-уязвимость через `<img>`
- **Шаги:**
  1. `router.go`: `r.Get(...)` → `r.Post(...)` для 4 маршрутов (accept/decline × mentor/partner)
  2. `handlers.go`: добавить `if r.Method != http.MethodPost { redirect }` и `CheckCSRF` в `lkMentorAction` и `lkPartnerAction`
  3. Шаблоны: `<a href=...>` → `<form method="post"><input type="hidden" name="csrf_token" value="{{.CSRF}}"><button>Принять</button></form>`
- **Acceptance:** GET → 303 redirect; POST без CSRF → 403

### AUD-005: `fail()` — Load без Save

- **Priority:** P0
- **Файлы:** `internal/http/handlers/handlers.go`
- **Описание:** `fail()` вызывает `Load` но не `Save`, CSRF не в cookie
- **Шаги:**
  1. В `fail()`: после `id, sess := s.Sessions.Load(r)` добавить `s.Sessions.Save(w, id, sess)`
- **Acceptance:** CSRF-токен попадает в cookie после ошибки

---

## P1 — Серьёзные

### AUD-006: Login — ошибки email/phone не видны

- **Priority:** P1
- **Файлы:** `templates/pages/login.html`, `handlers.go`
- **Описание:** `Data{"EmailError": ...}` есть, но в template нет `{{.EmailError}}`
- **Шаги:**
  1. В `login.html` после строк 39 добавить блок `{{if .EmailError}}...{{end}}` и `{{if .PhoneError}}...{{end}}`
  2. В `handlers.go` добавить `AlertError` во все ветки email/phone
- **Acceptance:** Ошибки email/phone показываются на странице

### AUD-007: CaseStart — бесполезный Save

- **Priority:** P1
- **Файлы:** `handlers.go`
- **Описание:** Ветка "уже идёт кейс" вызывает `Save` без изменений
- **Шаги:** Убрать `s.Sessions.Save(w, id, sess)` из ветки `Current`
- **Acceptance:** Кейс работает, F5 не сбрасывает прогресс

### AUD-008: Mentors filter перезаписывается

- **Priority:** P1
- **Файлы:** `handlers.go`
- **Описание:** `filter = m.Specialization` внутри цикла
- **Шаги:** Удалить строку 262 `filter = m.Specialization`
- **Acceptance:** Фильтрация по `?dir=Backend` работает корректно

### AUD-009: Seed ON CONFLICT DO UPDATE без WHERE

- **Priority:** P1
- **Файлы:** `internal/repository/postgres/store.go`
- **Описание:** Seed перезаписывает все поля при каждом запуске
- **Шаги:** Добавить `WHERE column IS DISTINCT FROM EXCLUDED.column` для всех полей в seed-запросах
- **Acceptance:** Повторный запуск без изменений данных не делает UPDATE

### AUD-010: Миграции не в транзакции

- **Priority:** P1
- **Файлы:** `internal/repository/postgres/store.go`
- **Описание:** SQL + INSERT в `schema_migrations` — нет атомарности
- **Шаги:** Обернуть `pool.Exec(...)` и `INSERT schema_migrations` в `pool.BeginTx()`
- **Acceptance:** При ошибке SQL — rollback; при ошибке INSERT — миграция не зафиксирована

### AUD-011: CSRF — привязка к fingerprint

- **Priority:** P1
- **Файлы:** `internal/session/session.go`
- **Описание:** CSRF не привязан к IP/User-Agent
- **Шаги:** Добавить поле `Fingerprint string` в `Data`, хешировать `SHA256(IP|UA)`, проверять в `CheckCSRF`
- **Acceptance:** Смена IP = CSRF валидация падает

### AUD-012: `safe` функция — XSS-риск

- **Priority:** P1
- **Файлы:** `internal/http/render/render.go`
- **Описание:** `template.HTML(s)` без валидации
- **Шаги:** Добавить комментарий-предупреждение об XSS
- **Acceptance:** Документировано ограничение; никто не передаёт raw user input

### AUD-013: Нет защиты от дублирования заявок

- **Priority:** P1
- **Файлы:** `application_service.go`, `handlers.go`, models, stores
- **Описание:** Пользователь может отправить заявку 5 раз
- **Шаги:**
  1. Добавить `ExistsDuplicate(ctx, kind, targetID, contact) bool` в интерфейс
  2. Реализовать в memory и postgres
  3. Проверка в `ApplicationService.Submit`
  4. Сообщение в formState при дубликате
- **Acceptance:** Повторная заявка → понятная ошибка

---

## P2 — Мелкие

### AUD-014: `page()` бесполезная перезапись cookie

- **Файлы:** `handlers.go`
- **Описание:** Каждый GET → Load + Save, даже если сессия не менялась
- **Шаги:** Создать `pageNoSave()` для публичных страниц
- **Acceptance:** Первый визит ставит cookie, повторные — нет

### AUD-015: Cleanup нет таймаута на мьютекс

- **Файлы:** `internal/session/session.go`
- **Описание:** `Cleanup()` может заблокироваться навсегда
- **Шаги:** Добавить `select { <-done, case <-time.After(5s) }`
- **Acceptance:** Timeout 5с при блокировке

### AUD-016: Migrate каждый Connect

- **Файлы:** `postgres/store.go`
- **Описание:** `migrate()` вызывается при каждом `Connect()`, health-check каждые 10с
- **Шаги:** Кэшировать результат проверки `schema_migrations`
- **Acceptance:** Повторный Connect быстрее (меньше запросов)

### AUD-017: Seed запускается всегда

- **Файлы:** `memory/store.go`
- **Описание:** `seed()` вызывается при каждом `New()`
- **Шаги:** Добавить флаг `s.seeded`
- **Acceptance:** Seed выполняется один раз

### AUD-018: Application без поля Role

- **Файлы:** `models.go`, `migrations/002_application_role.sql`
- **Описание:** SPEC говорит о `role` в applications, но код без него
- **Шаги:** Добавить поле `Role string`, миграцию, валидацию
- **Acceptance:** `domain.Application` имеет `Role`; валидация 2–50 символов

---

## Порядок выполнения

```
Phase 1 (KILLER): AUD-001 → AUD-005 (P0, все параллельно)
  ↓
Phase 2 (P1):     AUD-006, AUD-007, AUD-008 (независимые, параллельно)
  ↓
Phase 2b:         AUD-009, AUD-010 (зависят от понимания PG store)
  ↓
Phase 3 (P1):     AUD-011, AUD-012, AUD-013 (независимые)
  ↓
Phase 4 (P2):     AUD-014 — AUD-018 (все независимы)
```
