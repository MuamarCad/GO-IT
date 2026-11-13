package web

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"go-it/internal/session"
)

// Регрессия аудита 2026-11-11: кабинеты ментора, партнёра, «Менторы» и «Настройки» ученика
// отвечали 500 при «зелёных» тестах. Эти тесты открывают КАЖДУЮ страницу кабинета.
// Кабинеты ничего не помнят: открываются по адресу без входа, без cookie и без CSRF.

var lkPages = map[string][]string{
	"student": {"/lk/student", "/lk/student/cases", "/lk/student/roadmap", "/lk/student/mentors", "/lk/student/volunteering", "/lk/student/settings"},
	"mentor":  {"/lk/mentor", "/lk/mentor/applications", "/lk/mentor/schedule", "/lk/mentor/students", "/lk/mentor/profile"},
	"partner": {"/lk/partner", "/lk/partner/tasks", "/lk/partner/applications", "/lk/partner/analytics", "/lk/partner/settings"},
}

func noFollow(c *http.Client) *http.Client {
	return &http.Client{Jar: c.Jar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
}

func readBody(t *testing.T, res *http.Response) string {
	t.Helper()
	defer res.Body.Close()
	b, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func cookieFrom(t *testing.T, w *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	cs := w.Result().Cookies()
	if len(cs) == 0 {
		t.Fatal("Save не выставил cookie")
	}
	return cs[0]
}

func requestWith(c *http.Cookie) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.AddCookie(c)
	return r
}

func TestSessionExpiredIsRemovedAndRejected(t *testing.T) {
	m := session.New("secret-for-test-1")
	defer m.Stop()
	w := httptest.NewRecorder()
	m.Save(w, "expired-session-id-123456", session.Data{CSRF: "c", ExpiresAt: time.Now().Add(-time.Hour)})
	ck := cookieFrom(t, w)

	// даже до фоновой очистки истёкшая сессия не принимается
	if _, d := m.Load(requestWith(ck)); d.CSRF == "c" {
		t.Fatal("истёкшая сессия принята до Cleanup")
	}
}

func TestAnonymousSessionsGetExpiryAndAreCleaned(t *testing.T) {
	m := session.New("secret-for-test-2")
	defer m.Stop()

	w := httptest.NewRecorder()
	m.Save(w, "fresh-anon-session-12345", session.Data{CSRF: "fresh"})
	ck := cookieFrom(t, w)
	m.Cleanup()
	if _, d := m.Load(requestWith(ck)); d.CSRF != "fresh" {
		t.Fatal("свежая анонимная сессия удалена")
	}

	// анонимная сессия с истёкшим сроком очищается (раньше такие накапливались навсегда)
	w = httptest.NewRecorder()
	m.Save(w, "stale-anon-session-12345", session.Data{CSRF: "stale", ExpiresAt: time.Now().Add(-time.Minute)})
	stale := cookieFrom(t, w)
	m.Cleanup()
	if _, d := m.Load(requestWith(stale)); d.CSRF == "stale" {
		t.Fatal("просроченная анонимная сессия не очищена")
	}
}

func TestLoadWithoutSaveLeavesNoSession(t *testing.T) {
	m := session.New("secret-for-test-3")
	defer m.Stop()
	id1, _ := m.Load(httptest.NewRequest(http.MethodGet, "/", nil))
	id2, _ := m.Load(httptest.NewRequest(http.MethodGet, "/", nil))
	if id1 == id2 {
		t.Fatal("каждый запрос без cookie должен получать новый идентификатор")
	}
}

// --- Настройки ученика ---

func TestSessionDataIsIsolatedBetweenLoads(t *testing.T) {
	m := session.New("secret-for-test-4")
	defer m.Stop()

	w := httptest.NewRecorder()
	d := session.Data{CSRF: "c"}
	d.Attempt.Answers = map[string]string{"q1": "a"}
	m.Save(w, "isolation-session-id-12", d)
	ck := cookieFrom(t, w)

	d.Attempt.Answers["q2"] = "changed-after-save" // изменение исходной копии после Save
	_, first := m.Load(requestWith(ck))
	if _, leaked := first.Attempt.Answers["q2"]; leaked {
		t.Fatal("Save сохранил общую карту: изменения после Save попали в хранилище")
	}
	first.Attempt.Answers["q3"] = "x"
	_, second := m.Load(requestWith(ck))
	if _, leaked := second.Attempt.Answers["q3"]; leaked {
		t.Fatal("Load отдал общую карту Answers: запись в одной копии видна в другой")
	}
}

func TestConcurrentAnswersInOneSessionDoNotCrash(t *testing.T) {
	ts := authTestServer(t)
	c := jarClient(t)
	c.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	_, body := get(t, c, ts.URL+"/cases/frontend")
	csrf := find(t, body, `name="csrf_token" value="([^"]+)"`)
	q := find(t, body, `name="question" value="([^"]+)"`)
	opt := find(t, body, `name="option" value="([^"]+)"`)

	done := make(chan struct{})
	for i := 0; i < 64; i++ {
		go func() {
			defer func() { done <- struct{}{} }()
			for j := 0; j < 20; j++ {
				res, err := c.PostForm(ts.URL+"/cases/frontend/answer", url.Values{"csrf_token": {csrf}, "question": {q}, "option": {opt}})
				if err == nil {
					res.Body.Close()
				}
			}
		}()
	}
	for i := 0; i < 64; i++ {
		<-done
	}
}

func TestSpecificationListsEveryRoute(t *testing.T) {
	spec, err := readFile(t, "SPECIFICATION.md")
	if err != nil {
		t.Fatal(err)
	}
	h := newTestRouter(t)
	routes, ok := h.(chi.Routes)
	if !ok {
		t.Fatal("роутер не реализует chi.Routes")
	}
	inCode := map[string]bool{}
	count := 0
	err = chi.Walk(routes, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		if strings.HasPrefix(route, "/static/") {
			return nil
		}
		count++
		inCode[method+" "+route] = true
		if !strings.Contains(spec, "| "+method+" | `"+route+"` |") {
			t.Errorf("маршрут %s %s есть в router.go, но не описан в SPECIFICATION.md §8", method, route)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	// и наоборот: в документе нет маршрутов, которых уже нет в коде
	for _, m := range regexp.MustCompile("(?m)^\\| (GET|POST) \\| `([^`]+)` \\|").FindAllStringSubmatch(spec, -1) {
		if strings.HasPrefix(m[2], "/static/") {
			continue
		}
		if !inCode[m[1]+" "+m[2]] {
			t.Errorf("в SPECIFICATION.md описан маршрут %s %s, которого нет в router.go", m[1], m[2])
		}
	}
	want := fmt.Sprintf("Всего %d (включая обработчик 404)", count+1+1) // +статика +404
	if !strings.Contains(spec, want) {
		t.Errorf("в SPECIFICATION.md §8 нет строки %q", want)
	}
}

func TestNotFoundDoesNotCreateSession(t *testing.T) {
	ts := authTestServer(t)
	for _, p := range []string{"/favicon.ico", "/apple-touch-icon.png", "/no-such-page"} {
		res, err := http.Get(ts.URL + p)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode != http.StatusNotFound || res.Header.Get("Set-Cookie") != "" {
			t.Errorf("%s: статус %d, Set-Cookie=%q (404 не должна трогать сессию)", p, res.StatusCode, res.Header.Get("Set-Cookie"))
		}
	}
}

func TestDiagnoseExplainsCSRFFailure(t *testing.T) {
	m := session.New("secret-for-test-5")
	defer m.Stop()
	if got := m.Diagnose(httptest.NewRequest(http.MethodPost, "/", nil)); !strings.Contains(got, "не получена") {
		t.Errorf("без cookie: %q", got)
	}
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	r.AddCookie(&http.Cookie{Name: "goit_session", Value: "чужая.подпись"})
	if got := m.Diagnose(r); !strings.Contains(got, "подпись") {
		t.Errorf("чужая подпись: %q", got)
	}
	w := httptest.NewRecorder()
	m.Save(w, "diagnose-session-id-1234", session.Data{CSRF: "c"})
	if got := m.Diagnose(requestWith(cookieFrom(t, w))); !strings.Contains(got, "токен формы") {
		t.Errorf("известная сессия: %q", got)
	}
	m2 := session.New("secret-for-test-5") // тот же секрет, пустая память — как после перезапуска
	defer m2.Stop()
	if got := m2.Diagnose(requestWith(cookieFrom(t, w))); !strings.Contains(got, "перезапускался") {
		t.Errorf("после перезапуска: %q", got)
	}
}

func readFile(t *testing.T, rel string) (string, error) {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(moduleRoot(t), rel))
	return string(b), err
}

func TestLKIconsAreOutlineAndDoorIsUsed(t *testing.T) {
	css, err := readFile(t, "static/css/app.css")
	if err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`\.lk-nav-icon\s*\{[^}]*fill:\s*none[^}]*stroke:\s*currentColor`).MatchString(css) {
		t.Error("иконки ЛК должны быть контурными: нужен CSS fill:none + stroke:currentColor")
	}
	ts := authTestServer(t)
	_, body, _ := getLK(t, ts, "/lk/student")
	if !strings.Contains(body, `d="M6 21V4a1 1 0 0 1 1-1h10a1 1 0 0 1 1 1v17"`) {
		t.Error("кнопка «Выйти» должна использовать иконку двери")
	}
}

func getLK(t *testing.T, ts *httptest.Server, path string) (int, string, http.Header) {
	t.Helper()
	res, err := http.Get(ts.URL + path)
	if err != nil {
		t.Fatal(err)
	}
	return res.StatusCode, readBody(t, res), res.Header
}

func TestEveryLKPageOpensDirectlyWithoutLogin(t *testing.T) {
	ts := authTestServer(t)
	for role, pages := range lkPages {
		for _, p := range pages {
			status, body, hdr := getLK(t, ts, p)
			if status != http.StatusOK {
				t.Errorf("%s: статус %d", p, status)
				continue
			}
			if hdr.Get("Set-Cookie") != "" {
				t.Errorf("%s: кабинет не должен выдавать cookie", p)
			}
			for _, bad := range []string{"Не удалось", "ZgotmplZ", "<no value>", "csrf_token"} {
				if strings.Contains(body, bad) {
					t.Errorf("%s: в ответе найдено %q", p, bad)
				}
			}
			if !strings.Contains(body, `<body class="lk-page">`) {
				t.Errorf("%s: шапка сайта не скрыта (нет lk-page)", p)
			}
			if strings.Count(body, `class="lk-sidebar"`) != 1 {
				t.Errorf("%s: сайдбар должен быть один", p)
			}
			if !strings.Contains(body, `href="`+p+`" aria-current="page"`) {
				t.Errorf("%s: пункт меню не подсвечен", p)
			}
			if !strings.Contains(body, `<a class="lk-logout-btn" href="/">`) {
				t.Errorf("%s: «Выйти» должна быть ссылкой на главную", p)
			}
			if regexp.MustCompile(`[\p{Han}]`).MatchString(body) {
				t.Errorf("%s: в тексте посторонние иероглифы", p)
			}
			_ = role
		}
	}
}

func TestLoginThenLKEndToEnd(t *testing.T) {
	ts := authTestServer(t)
	c := &http.Client{} // следует редиректам, без cookie jar
	for role, home := range lkHome {
		res, err := c.PostForm(ts.URL+"/auth", url.Values{"role": {role}})
		if err != nil {
			t.Fatal(err)
		}
		body := readBody(t, res)
		if res.StatusCode != http.StatusOK || res.Request.URL.Path != home || !strings.Contains(body, `class="lk-sidebar"`) {
			t.Errorf("%s: после входа ожидали страницу %s, получили %d %s", role, home, res.StatusCode, res.Request.URL.Path)
		}
	}
}

// ---------- Решения по заявкам живут в адресе ----------

func TestMentorDecisionsLiveInTheURL(t *testing.T) {
	ts := authTestServer(t)
	_, body, _ := getLK(t, ts, "/lk/mentor/applications")
	if strings.Count(body, "Принять</a>") != 2 || strings.Contains(body, "Принята") {
		t.Fatal("до решения должны быть две заявки с кнопками и без статусов")
	}
	acceptFirst := regexp.MustCompile(`href="(/lk/mentor/applications\?h=1%3Aaccepted)"`).FindStringSubmatch(body)
	if acceptFirst == nil {
		t.Fatal("нет ссылки «Принять» для заявки 1")
	}

	_, body, _ = getLK(t, ts, acceptFirst[1])
	if !strings.Contains(body, "Принята") || strings.Count(body, "Принять</a>") != 1 {
		t.Fatal("заявка 1 должна получить статус и потерять кнопки, заявка 2 остаётся")
	}
	// ссылка на вторую заявку сохраняет решение по первой
	declineSecond := regexp.MustCompile(`href="(/lk/mentor/applications\?h=1%3Aaccepted%2C2%3Adeclined)"`).FindStringSubmatch(body)
	if declineSecond == nil {
		t.Fatal("ссылка «Отклонить» для заявки 2 должна нести и решение по заявке 1")
	}
	_, body, _ = getLK(t, ts, declineSecond[1])
	if !strings.Contains(body, "Принята") || !strings.Contains(body, "Отклонена") || strings.Contains(body, "Принять</a>") {
		t.Fatal("обе заявки должны получить статусы")
	}
	// счётчик новых заявок на обзоре
	_, over, _ := getLK(t, ts, "/lk/mentor?h=1%3Aaccepted")
	if !regexp.MustCompile(`<span class="n">1</span><small>Новых заявок`).MatchString(over) {
		t.Error("после одного решения осталась одна новая заявка")
	}
}

func TestDecisionsParameterIsSafe(t *testing.T) {
	ts := authTestServer(t)
	for _, h := range []string{"99:accepted", "1:hack", "<script>alert(1)</script>", "1:accepted,,,:::", strings.Repeat("1:accepted,", 500)} {
		status, body, _ := getLK(t, ts, "/lk/mentor/applications?h="+url.QueryEscape(h))
		if status != http.StatusOK {
			t.Errorf("h=%q: статус %d", h, status)
		}
		if strings.Contains(body, "<script>alert") || strings.Contains(body, "hack") {
			t.Errorf("h=%q: значение попало на страницу", h)
		}
	}
	_, body, _ := getLK(t, ts, "/lk/mentor/applications?h=99%3Aaccepted%2C1%3Ahack")
	if strings.Contains(body, "Принята") || strings.Contains(body, "Отклонена") {
		t.Error("неизвестные заявки и действия должны игнорироваться")
	}
}

func TestPartnerDecisionsLiveInTheURL(t *testing.T) {
	ts := authTestServer(t)
	_, body, _ := getLK(t, ts, "/lk/partner/applications?h=3%3Aaccepted")
	if !strings.Contains(body, "Принят") || strings.Count(body, "Принять</a>") != 2 {
		t.Fatal("отклик 3 должен быть принят, два других остаются с кнопками")
	}
	_, over, _ := getLK(t, ts, "/lk/partner?h=3%3Aaccepted")
	if !strings.Contains(over, `<span class="n">10</span><small>Принято`) {
		t.Error("счётчик «Принято» должен вырасти до 10")
	}
}

// ---------- Настройки: проверка без сохранения ----------

func TestStudentSettingsValidationWithoutSession(t *testing.T) {
	ts := authTestServer(t)
	c := noFollow(&http.Client{})

	post := func(v url.Values) (*http.Response, string) {
		res, err := c.PostForm(ts.URL+"/lk/student/settings", v)
		if err != nil {
			t.Fatal(err)
		}
		return res, readBody(t, res)
	}

	res, body := post(url.Values{"action": {"profile"}, "name": {"А"}, "email": {"не-почта"}, "phone": {"12"}})
	if res.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("невалидный профиль: статус %d", res.StatusCode)
	}
	for _, want := range []string{"Введите ФИО", "Проверьте email", "Проверьте телефон", `aria-invalid="true"`, `value="не-почта"`} {
		if !strings.Contains(body, want) {
			t.Errorf("нет %q", want)
		}
	}
	if res.Header.Get("Set-Cookie") != "" {
		t.Error("настройки не должны выдавать cookie")
	}

	res, _ = post(url.Values{"action": {"profile"}, "name": {"Мария Петрова"}, "email": {"maria@example.com"}, "phone": {"+7 900 123-45-67"}})
	if res.StatusCode != http.StatusSeeOther || res.Header.Get("Location") != "/lk/student/settings?done=profile" {
		t.Fatalf("валидный профиль: статус %d, Location=%q", res.StatusCode, res.Header.Get("Location"))
	}
	_, flash, _ := getLK(t, ts, "/lk/student/settings?done=profile")
	if !strings.Contains(flash, "Профиль сохранён. В демо-режиме данные нигде не хранятся.") {
		t.Error("после сохранения должно быть сообщение с оговоркой про демо")
	}

	res, body = post(url.Values{"action": {"password"}, "password_new": {"abc"}, "password_confirm": {"xyz"}})
	if res.StatusCode != http.StatusUnprocessableEntity || !strings.Contains(body, "Новый пароль:") || !strings.Contains(body, "Пароли не совпадают.") {
		t.Fatalf("ошибки пароля не показаны (статус %d)", res.StatusCode)
	}
	res, _ = post(url.Values{"action": {"password"}, "password_new": {"newpass123"}, "password_confirm": {"newpass123"}})
	if res.StatusCode != http.StatusSeeOther || res.Header.Get("Location") != "/lk/student/settings?done=password" {
		t.Fatalf("верный пароль: статус %d, Location=%q", res.StatusCode, res.Header.Get("Location"))
	}
	res, _ = post(url.Values{"action": {"что-то"}})
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("неизвестное действие: статус %d, ожидали 400", res.StatusCode)
	}
}

func TestFlashTextComesOnlyFromKnownCodes(t *testing.T) {
	ts := authTestServer(t)
	_, body, _ := getLK(t, ts, "/lk/student/settings?done=%3Cscript%3Ealert(1)%3C/script%3E&msg=Вы+выиграли")
	if strings.Contains(body, "<script>alert") || strings.Contains(body, "Вы выиграли") {
		t.Fatal("произвольный текст из адреса попал на страницу")
	}
}

// ---------- Обзор ученика ----------

func TestStudentOverviewIsConsistent(t *testing.T) {
	ts := authTestServer(t)
	_, body, _ := getLK(t, ts, "/lk/student")

	m := regexp.MustCompile(`class="radar-data" points="([^"]+)"`).FindStringSubmatch(body)
	if m == nil || len(strings.Fields(m[1])) != 6 {
		t.Fatalf("радар навыков не построен: %v", m)
	}
	if !strings.Contains(body, "Frontend 85%") || !strings.Contains(body, "Backend 77%") {
		t.Error("описание радара должно перечислять значения навыков")
	}
	if strings.Contains(body, "3 из 4") || strings.Contains(body, "5 из 5") {
		t.Error("в демо-данных остались устаревшие числа вопросов")
	}
	if !strings.Contains(body, "17 из 20") || !strings.Contains(body, "14 из 18") {
		t.Error("демо-результаты должны быть 17 из 20 и 14 из 18")
	}
	if strings.Contains(body, "Рекомендуем продолжить кейс «Первый день Backend") {
		t.Error("рекомендуется кейс, который уже показан как пройденный")
	}
	for _, l := range regexp.MustCompile(`href="(/mentors/[a-z-]+)"`).FindAllStringSubmatch(body, -1) {
		r, _ := http.Get(ts.URL + l[1])
		r.Body.Close()
		if r.StatusCode != http.StatusOK {
			t.Errorf("ссылка %s ведёт на %d", l[1], r.StatusCode)
		}
	}
}

func TestPagesWithTokensAreNotCached(t *testing.T) {
	ts := authTestServer(t)
	res, _ := http.Get(ts.URL + "/cases/frontend")
	res.Body.Close()
	if res.Header.Get("Cache-Control") != "no-store" {
		t.Errorf("Cache-Control=%q, ожидали no-store", res.Header.Get("Cache-Control"))
	}
}

func TestTamperedSessionCookieStartsFreshSession(t *testing.T) {
	m := session.New("secret-for-test-6")
	defer m.Stop()
	w := httptest.NewRecorder()
	m.Save(w, "tamper-session-id-12345", session.Data{CSRF: "mine"})
	ck := cookieFrom(t, w)
	forged := *ck
	forged.Value = ck.Value[:len(ck.Value)-2] + "xx"
	if _, d := m.Load(requestWith(&forged)); d.CSRF == "mine" {
		t.Fatal("cookie с испорченной подписью приняла чужую сессию")
	}
	if _, d := m.Load(requestWith(ck)); d.CSRF != "mine" {
		t.Fatal("исходная cookie должна работать")
	}
}

func mustParse(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return u
}
