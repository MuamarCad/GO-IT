# GO-IT — GIGACODE Context

**Версия:** 1.0
**Дата:** 10 ноября 2026
**Статус:** Действующий

> Связанные документы: [Спецификация](SPECIFICATION.md) · [План развития](PLAN.md) · [DOC-STYLE](DOC-STYLE.md)

## Directory Overview

**GO-IT** is a student-built MVP of a social educational web platform for exploring IT professions. It helps beginners discover careers (Frontend, Backend, IT Project Manager) through interactive quizzes ("career cases"), structured roadmaps, mentor profiles, and volunteer opportunities. The project is built as a single-server Go application with server-side rendering, deployed via Docker Compose.

**Location:** `/home/lophandm/Рабочий стол/MIU_FSE/GO-IT/`

**Status:** Student MVP — no public deployment, runs locally only.

---

## Key Files

| File | Purpose |
|------|---------|
| `README.md` | User-facing documentation: features, Docker/without-Docker run, demo scenario, checklist |
| `SPECIFICATION.md` | Full project specification (61+ requirements, data models, user journeys, acceptance criteria) |
| `PLAN.md` | Development plan: goals, tasks, risks, metrics, pre-defense checklist |
| `go.mod` | Module `go-it`, Go 1.22, 4 direct dependencies (chi, pgx, bluemonday, goldmark) |
| `Makefile` | Commands: `run`, `test`, `fmt`, `compose`, `check-links` |
| `Dockerfile` | Multi-stage build: golang:1.22-alpine → alpine:3.20 (non-root user `app`) |
| `docker-compose.yml` | Two services: `web` (port 8080) + `postgres:16-alpine` (port 5432, healthcheck) |
| `.env.example` | Default ENV config: APP_ENV, HTTP_ADDR, DATABASE_URL, SESSION_SECRET, CSRF_SECRET |
| `.gitignore` | Excludes `.env`, `bin/`, `tmp/`, IDE files |
| `.dockerignore` | Excludes `.git`, `.env`, `draft`, `docs`, `*.pdf`, `*.pptx` |

---

## Project Architecture

### Layered Architecture (Hexagonal-style)

```
Browser → Chi Router → Handlers → Services → Repositories → PostgreSQL/In-Memory
                         ↓          ↓           ↓
                   Session Mgr   Domain    Content Loader
                   CSRF          Models    (Markdown → HTML)
```

### Package Structure (14 packages, 19 Go Files)

| Package | Files | Purpose |
|---------|-------|---------|
| `cmd/web` | 1 | Entry point, DI wiring, graceful shutdown |
| `internal/config` | 1 | ENV variable loading (`APP_ENV`, `DATABASE_URL`, `SESSION_SECRET`, `CSRF_SECRET`) |
| `internal/domain` | 1 | 21 pure domain types (`Direction`, `CareerCase`, `Mentor`, `Application`, `Roadmap`, `Diagram`, `Article`, etc.) |
| `internal/catalog` | 1 | Seed data: 3 directions, 4 mentors, 4 volunteer tasks |
| `internal/cases` | 1 | Career case data: 1 case (Frontend), 4 questions, 3 options each |
| `internal/session` | 1 | Cookie-based sessions, CSRF tokens, in-memory store |
| `internal/content` | 2 | Markdown → HTML via goldmark, XSS via bluemonday, diagram.json validation |
| `internal/http` | 2 | Chi router: 45 routes (incl. auth + lk), http_test.go |
| `internal/http/handlers` | 1 | 30+ HTTP handlers in single `Server` struct (~719 lines) |
| `internal/http/render` | 1 | `html/template` wrapper with custom functions |
| `internal/service` | 2 | `CaseService` (quiz logic), `ApplicationService` (form validation) |
| `internal/repository/memory` | 1 | In-memory store with `sync.Mutex` (development mode) |
| `internal/repository/postgres` | 1 | PostgreSQL store via `pgxpool`, embed migrations + seed |
| `internal/http/middleware` | 0 | Empty stub — placeholder for future middleware |

### HTTP Routes (45 total)

#### Health & Static (3)

| Method | Path | Handler | Purpose |
|--------|------|---------|---------|
| GET | `/health/live` | `Live` | Always returns 200 "ok" |
| GET | `/health/ready` | `Ready` | Returns 503 if DB is down |
| GET | `/static/*` | FileServer | Serves CSS, JS, fonts, images |

#### Main Pages (8)

| Method | Path | Handler | Purpose |
|--------|------|---------|---------|
| GET | `/` | `Home` | Main page with all sections |
| GET | `/about` | `About` | About page |
| GET | `/directions` | `Directions` | Directory of IT directions |
| GET | `/directions/{slug}` | `Direction` | Direction detail page |
| GET | `/cases` | `Cases` | Case catalog |
| GET | `/cases/{slug}` | `CaseStart` | Start or resume a case |
| GET | `/mentors` | `Mentors` | Mentor directory (filter by `?dir=Backend`) |
| GET | `/mentors/{slug}` | `Mentor` | Mentor profile + application form |

#### Career Cases (3)

| Method | Path | Handler | Purpose |
|--------|------|---------|---------|
| POST | `/cases/{slug}/answer` | `CaseAnswer` | Submit case answer (server validates) |
| GET | `/cases/{slug}/result` | `CaseResult` | Results with score, explanations, topic breakdown |

#### Volunteer (3)

| Method | Path | Handler | Purpose |
|--------|------|---------|---------|
| GET | `/volunteering` | `Volunteering` | Volunteer tasks directory |
| GET | `/volunteering/{slug}` | `VolunteerTask` | Volunteer task detail + application form |
| POST | `/volunteering/{slug}/applications` | `VolunteerApply` | Submit volunteer application |

#### Roadmaps (2)

| Method | Path | Handler | Purpose |
|--------|------|---------|---------|
| GET | `/roadmaps` | `Roadmaps` | Roadmap list |
| GET | `/roadmaps/{slug}` | `Roadmap` | Roadmap page with visual diagram |

#### Blog (2)

| Method | Path | Handler | Purpose |
|--------|------|---------|---------|
| GET | `/blog` | `Blog` | Blog article list |
| GET | `/blog/{slug}` | `Article` | Single article page |

#### Auth (3)

| Method | Path | Handler | Purpose |
|--------|------|---------|---------|
| GET | `/auth` | `AuthHandler` | OAuth authorization page |
| GET | `/auth/{provider}` | `AuthHandler` | OAuth via provider (GitHub, VK, etc.) |
| POST | `/auth` | `AuthCallback` | OAuth callback, creates session |

#### Student Dashboard (6)

| Method | Path | Handler | Purpose |
|--------|------|---------|---------|
| GET | `/lk/student` | `StudentDashboard` | Student overview |
| GET | `/lk/student/cases` | `StudentCases` | Student's case progress |
| GET | `/lk/student/roadmap` | `StudentRoadmap` | Student's learning paths |
| GET | `/lk/student/mentors` | `StudentMentors` | Student's mentor connections |
| GET | `/lk/student/volunteering` | `StudentVolunteering` | Student's volunteer activities |
| GET | `/lk/student/settings` | `StudentSettings` | Student account settings |

#### Mentor Dashboard (7)

| Method | Path | Handler | Purpose |
|--------|------|---------|---------|
| GET | `/lk/mentor` | `MentorDashboard` | Mentor overview |
| GET | `/lk/mentor/applications` | `MentorApplications` | Pending applications |
| GET | `/lk/mentor/schedule` | `MentorSchedule` | Mentor's schedule |
| GET | `/lk/mentor/students` | `MentorStudents` | Assigned students |
| GET | `/lk/mentor/profile` | `MentorProfile` | Mentor profile edit |
| GET | `/lk/mentor/applications/{id}/accept` | `MentorAccept` | Accept application |
| GET | `/lk/mentor/applications/{id}/decline` | `MentorDecline` | Decline application |

#### Partner Dashboard (7)

| Method | Path | Handler | Purpose |
|--------|------|---------|---------|
| GET | `/lk/partner` | `PartnerDashboard` | Partner overview |
| GET | `/lk/partner/tasks` | `PartnerTasks` | Partner's tasks |
| GET | `/lk/partner/applications` | `PartnerApplications` | Partner's applications |
| GET | `/lk/partner/analytics` | `PartnerAnalytics` | Partner analytics |
| GET | `/lk/partner/settings` | `PartnerSettings` | Partner account settings |
| GET | `/lk/partner/applications/{id}/accept` | `PartnerAccept` | Accept application |
| GET | `/lk/partner/applications/{id}/decline` | `PartnerDecline` | Decline application |

#### Fallback (1)

| Method | Path | Handler | Purpose |
|--------|------|---------|---------|
| — | *fallback* | `NotFound` | 404 styled error page |

---

## Building and Running

### Via Docker Compose (recommended)

```bash
cp .env.example .env
docker compose up --build
```

Opens at `http://localhost:8080`

### Without Docker

Requires Go 1.22+ and optionally PostgreSQL. Without `DATABASE_URL`, the app uses in-memory storage (good for CSS/HTML development).

```bash
make run          # Run app locally
make test         # Run all tests
make fmt          # Format Go code
make compose      # Docker Compose up
make check-links  # Check external links in roadmaps (requires internet)
```

### Pre-launch Checklist

- [ ] `docker compose up --build` starts site and DB
- [ ] `go test ./...` passes
- [ ] Case completes from start to result
- [ ] Mentor application saves (persists after container restart)
- [ ] `.env` is not in Git

---

## Dependencies

| Dependency | Version | Purpose |
|-----------|---------|---------|
| `github.com/go-chi/chi/v5` | v5.1.0 | HTTP router |
| `github.com/jackc/pgx/v5` | v5.7.1 | PostgreSQL driver with connection pool |
| `github.com/microcosm-cc/bluemonday` | v1.0.27 | XSS sanitization (UGP policy) |
| `github.com/yuin/goldmark` | v1.7.8 | Markdown parser (GFM extension) |

---

## Frontend Architecture

- **Templates:** `html/template` — no SPA, no JS framework
  - `templates/layouts/base.html` — base skeleton
  - `templates/pages/` — 21 page templates (each defines only `content` block)
    - 17 core: home, about, directions, direction, cases, case_question, case_result, mentors, mentor, volunteering, volunteer, roadmaps, roadmap, blog, article, not_found, error
    - 4 dashboard: login, lk_student, lk_mentor, lk_partner
  - `templates/partials/` — 5 reusable fragments (header, footer, logo, application_form, roadmap_diagram)
- **CSS:** `static/css/app.css` — CSS Custom Properties for 3 themes: light, dark, high contrast
- **JavaScript:** HTMX (local, no CDN) + `theme-init.js` (sync, in `<head>`) + `theme.js` (defer)
- **Fonts:** Local WOFF2 files (Unbounded + Onest), no CDN
- **Roadmap diagrams:** Rendered from JSON (`content/roadmaps/{slug}.diagram.json`), pure HTML, keyboard-accessible

---

## Data Storage

### PostgreSQL Schema

```sql
mentors (slug PK, display_name, specialization, level, format, topics, description)
volunteer_requests (slug PK, title, organization, category, skills, scale, description, expected_result)
applications (id BIGSERIAL PK, kind CHECK, target_id, name, contact, message, status CHECK, created_at)
schema_migrations (version PK)
```

### Content Storage (Markdown + JSON)

| Content | Source | Format |
|---------|--------|--------|
| Roadmaps | `content/roadmaps/*.md` | Markdown |
| Roadmap diagrams | `content/roadmaps/*.diagram.json` | JSON (stages → nodes) |
| Blog articles | `content/blog/*.md` | Markdown with `Дата:` metadata |
| Cases | `internal/cases/frontend.go` | Hardcoded Go |
| Directions | `internal/catalog/catalog.go` | Hardcoded Go |
| Seed mentors/tasks | `internal/catalog/catalog.go` | Hardcoded Go |

### Database Modes

| Mode | Condition | Behavior |
|------|-----------|----------|
| PostgreSQL | `DATABASE_URL` set | Full persistence, migrations applied, seed on startup |
| In-memory | `DATABASE_URL` empty | Data in `map[string]Data` with `sync.Mutex` (lost on restart) |

---

## Testing

**Test files:** 5 test files with 31 test functions

| File | Tests | Coverage |
|------|-------|----------|
| `internal/http/http_test.go` | 12 test functions + testServer helper | Home/404, case flow, form validation, CSRF, mentor/volunteer apply, all pages render, roadmap diagrams, topic panels |
| `internal/service/case_service_test.go` | 8 | Happy path, unknown option, skip prevention, no double score, missing state |
| `internal/service/application_service_test.go` | 4 | Valid application, unknown target, field validation |
| `internal/content/loader_test.go` | 5 | Diagram JSON validation (17 negative cases), real roadmaps, no-diagram case |
| `internal/content/links_test.go` | 1 | External link alive check (via `make check-links`) |

**Not yet tested:** `session.Manager`, `config` package, integration tests with postgres

---

## Security Features

| Feature | Implementation |
|---------|---------------|
| CSRF protection | Cookie-based session tokens, HMAC comparison on POST/PUT/PATCH/DELETE |
| XSS sanitization | bluemonday UGP policy on all Markdown output |
| SQL injection prevention | pgx parameterized queries (`$1`, `$2`) |
| Body size limit | 1 MB via `http.MaxBytesReader` |
| Error safety | No SQL leaks, no stack traces in 500 responses |
| Docker | Non-root user (`USER app`) |
| Secrets | All via ENV variables, `.env` in `.gitignore` |
| Timeouts | ReadHeaderTimeout 5s, ReadTimeout 15s, WriteTimeout 15s, IdleTimeout 60s |
| Graceful shutdown | SIGINT/SIGTERM handled, 8-second timeout |

---

## Development Conventions

- **Go formatting:** `gofmt -w .` (via `make fmt`)
- **No CI/CD:** No GitHub Actions or GitLab CI configured
- **No linter:** No golangci-lint or staticcheck in Makefile
- **Test co-location:** `_test.go` files next to source files (standard Go)
- **No code comments:** Source has minimal or no doc-comments on exported identifiers
- **Single handler file:** All 30+ handlers in `internal/http/handlers/handlers.go` (~719 lines)
- **Package aliasing:** `web` package is an alias for `internal/http` (imported as `web "go-it/internal/http"`)

---

## Known Gaps & Technical Debt

1. **Empty directories:** `internal/app/`, `internal/http/middleware/`, `migrations/`, `tests/` — stub directories not used (note: `migrations/` at root is a duplicate; real migrations are in `internal/repository/postgres/migrations/`)
2. **No session cleanup:** In-memory `session.Manager` never evicts old sessions — memory leak under load
3. **No request logging:** No middleware for logging HTTP method, path, status, duration
4. **Single handler file:** `handlers.go` (~719 lines) — no separation by domain
5. **Hardcoded directions:** `catalog.Directions()` called directly in handlers, bypassing `Catalog` interface
6. **Migrations embedded in binary:** SQL files are `//go:embed`ed — can't apply migrations independently
7. **Seed on every connect:** `ON CONFLICT DO UPDATE` runs on every app startup
8. **No doc-comments:** Package and type documentation is minimal
9. **No CI/CD:** No automated testing or linting pipeline
10. **No integration tests:** No test against real postgres (testcontainers or temp container)
11. **Session auth in-memory:** Sessions stored only in memory — lost on restart (acceptable for MVP)

---

## Draft Artifacts (Not Part of Production)

Located in `draft/`:

| File | Description |
|------|-----------|
| `Project-Plan_GO-IT.md` | Full MVP concept (668 lines): problem, scope, functional requirements, architecture, 8-week plan, risks |
| `Project-Plan_GO-IT.pdf` | PDF version of the plan |
| `Main.dc.html` | Landing page design mockup (source of CSS design tokens) |
| `GO-IT -- лендинг.pdf` | Landing page PDF |
| `Смартфон.pdf`, `Смартфон, тёмная тема.pdf` | Mobile view mockups |
| `go-it.pptx` | Presentation deck |

---

## API Documentation

**`docs/docs.json`** — OpenAPI 3.0.3 specification (1380 lines) defining a future REST API. Covers health, cases, mentors, volunteering, resources, and articles endpoints. **Not currently implemented** — the app only serves HTML.

---

## Quick Reference

### ENV Variables

| Variable | Default | Required in Production |
|----------|---------|----------------------|
| `APP_ENV` | `development` | No |
| `HTTP_ADDR` | `:8080` | No |
| `DATABASE_URL` | (empty → in-memory) | **Yes** |
| `SESSION_SECRET` | `dev-session-secret-change-me` | No (min 16 chars) |
| `CSRF_SECRET` | `dev-csrf-secret-change-me-32b!!` | No (min 16 chars) |

### Seed Data

| Type | Count | Source |
|------|-------|--------|
| Directions | 3 | Frontend, Backend, Project Manager |
| Career Cases | 1 | Frontend (4 questions) |
| Roadmaps | 3 | frontend.md, backend.md, project-manager.md |
| Blog Articles | 2 | first-steps-in-it.md, how-to-build-a-pet-project.md |
| Mentors | 4 | Daria P., Ivan S., Sofia K., Artem L. |
| Volunteer Tasks | 4 | NGO landing, school roadmap, library form, club a11y |

### Template Count

| Type | Count |
|------|-------|
| Layouts | 1 (base.html) |
| Pages | 21 (17 core: home, about, directions, direction, cases, case_question, case_result, mentors, mentor, volunteering, volunteer, roadmaps, roadmap, blog, article, not_found, error; 4 dashboard: login, lk_student, lk_mentor, lk_partner) |
| Partials | 5 (application_form, footer, header, logo, roadmap_diagram) |
| **Total** | **27** |

### Static Assets

| Type | Count |
|------|-------|
| CSS | 1 (app.css, 466 lines) |
| JS | 3 (htmx.min.js, theme-init.js, theme.js) |
| Fonts | 6 WOFF2 files + 2 license files |
| License (htmx) | 1 (LICENSE-htmx.txt) |
| Images | 1 (favicon.svg) |
