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

// Регрессия аудита 2026-11-11: спецификация объявила кабинеты готовыми, хотя ЛК ментора,
// ЛК партнёра, «Менторы» и «Настройки» ученика отвечали 500. Эти тесты открывают КАЖДУЮ
// страницу кабинета под своей ролью и проверяют ответ целиком.

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

// loginAs входит как роль через обычную форму и возвращает клиента с cookie.
func loginAs(t *testing.T, ts *httptest.Server, role string) *http.Client {
	t.Helper()
	c := jarClient(t)
	res, err := c.Get(ts.URL + "/auth")
	if err != nil {
		t.Fatal(err)
	}
	csrf := extractCSRF(t, readBody(t, res))
	res, err = postForm(c, ts.URL+"/auth", url.Values{"csrf_token": {csrf}, "role": {role}})
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	return c
}

func csrfFrom(t *testing.T, c *http.Client, ts *httptest.Server, path string) string {
	t.Helper()
	res, err := c.Get(ts.URL + path)
	if err != nil {
		t.Fatal(err)
	}
	return extractCSRF(t, readBody(t, res))
}

func TestEveryLKPageRendersForItsRole(t *testing.T) {
	ts := authTestServer(t)
	for role, pages := range lkPages {
		c := loginAs(t, ts, role)
		for _, p := range pages {
			res, err := c.Get(ts.URL + p)
			if err != nil {
				t.Fatal(err)
			}
			body := readBody(t, res)
			if res.StatusCode != http.StatusOK {
				t.Errorf("%s %s: статус %d", role, p, res.StatusCode)
				continue
			}
			if strings.Contains(body, "Не удалось") || strings.Contains(body, "ZgotmplZ") || strings.Contains(body, "<no value>") {
				t.Errorf("%s %s: в ответе следы ошибки шаблона", role, p)
			}
			if !strings.Contains(body, `<body class="lk-page">`) {
				t.Errorf("%s %s: у страницы кабинета нет класса lk-page (шапка сайта не скрыта)", role, p)
			}
			if strings.Count(body, `class="lk-sidebar"`) != 1 {
				t.Errorf("%s %s: сайдбар должен быть один", role, p)
			}
			if strings.Count(body, `aria-current="page"`) < 1 || !strings.Contains(body, `href="`+p+`" aria-current="page"`) {
				t.Errorf("%s %s: пункт меню не подсвечен", role, p)
			}
			if !strings.Contains(body, `action="/lk/`+role+`/logout"`) || !strings.Contains(body, "Выйти") {
				t.Errorf("%s %s: нет кнопки «Выйти»", role, p)
			}
			if strings.Contains(body, "数据") || regexp.MustCompile(`[\p{Han}]`).MatchString(body) {
				t.Errorf("%s %s: в тексте посторонние иероглифы", role, p)
			}
		}
	}
}

func TestLKRoleIsolation(t *testing.T) {
	ts := authTestServer(t)
	home := map[string]string{"student": "/lk/student", "mentor": "/lk/mentor", "partner": "/lk/partner"}
	for role := range lkPages {
		c := noFollow(loginAs(t, ts, role))
		for other, pages := range lkPages {
			if other == role {
				continue
			}
			for _, p := range pages {
				res, err := c.Get(ts.URL + p)
				if err != nil {
					t.Fatal(err)
				}
				res.Body.Close()
				if res.StatusCode != http.StatusSeeOther || res.Header.Get("Location") != home[role] {
					t.Errorf("%s открыл %s: статус %d, Location=%q (ожидали редирект в %s)", role, p, res.StatusCode, res.Header.Get("Location"), home[role])
				}
			}
		}
	}
}

func TestProviderLoginIsPostWithCSRFAndWhitelist(t *testing.T) {
	ts := authTestServer(t)

	// GET больше не входит
	c := noFollow(jarClient(t))
	res, _ := c.Get(ts.URL + "/auth/vk?role=mentor")
	res.Body.Close()
	if res.Header.Get("Location") != "/auth" {
		t.Fatalf("GET /auth/vk должен вести на форму, а не входить: %q", res.Header.Get("Location"))
	}
	res, _ = c.Get(ts.URL + "/lk/mentor")
	res.Body.Close()
	if res.Header.Get("Location") != "/auth" {
		t.Fatal("после GET /auth/vk пользователь не должен быть вошедшим")
	}

	// POST без токена
	res, _ = postForm(c, ts.URL+"/auth/vk", url.Values{"role": {"mentor"}})
	res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("POST без CSRF: статус %d, ожидали 403", res.StatusCode)
	}

	csrf := csrfFrom(t, c, ts, "/auth")
	// неизвестный провайдер
	res, _ = postForm(c, ts.URL+"/auth/evil", url.Values{"csrf_token": {csrf}})
	res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("неизвестный провайдер: статус %d, ожидали 404", res.StatusCode)
	}
	// произвольная роль превращается в «ученик»
	res, _ = postForm(c, ts.URL+"/auth/yandex", url.Values{"csrf_token": {csrf}, "role": {"admin"}})
	res.Body.Close()
	if res.Header.Get("Location") != "/lk/student" {
		t.Fatalf("role=admin: Location=%q, ожидали /lk/student", res.Header.Get("Location"))
	}

	// нормальный вход ментора через VK
	c2 := noFollow(jarClient(t))
	csrf = csrfFrom(t, c2, ts, "/auth")
	res, _ = postForm(c2, ts.URL+"/auth/vk", url.Values{"csrf_token": {csrf}, "role": {"mentor"}})
	res.Body.Close()
	if res.Header.Get("Location") != "/lk/mentor" {
		t.Fatalf("VK + mentor: Location=%q", res.Header.Get("Location"))
	}
}

func TestQuickLoginButtons(t *testing.T) {
	ts := authTestServer(t)
	for quick, want := range map[string]string{"mentor": "/lk/mentor", "partner": "/lk/partner"} {
		c := noFollow(jarClient(t))
		csrf := csrfFrom(t, c, ts, "/auth")
		res, _ := postForm(c, ts.URL+"/auth", url.Values{"csrf_token": {csrf}, "role": {"student"}, "quick": {quick}})
		res.Body.Close()
		if res.Header.Get("Location") != want {
			t.Errorf("quick=%s: Location=%q, ожидали %s", quick, res.Header.Get("Location"), want)
		}
	}
}

func TestLoginPageHasNoInlineScriptAndRealRadios(t *testing.T) {
	ts := authTestServer(t)
	res, _ := http.Get(ts.URL + "/auth")
	body := readBody(t, res)
	if strings.Contains(body, "<script>") || strings.Contains(body, "onclick=") {
		t.Error("на странице входа не должно быть inline-скриптов и onclick")
	}
	if strings.Count(body, `type="radio" name="role"`) != 3 || strings.Contains(body, `role="radio"`) {
		t.Error("роли должны быть тремя настоящими радиокнопками")
	}
	if regexp.MustCompile(`name="(email|password|phone)"[^>]*value="[^"]+"`).MatchString(body) {
		t.Error("поля входа должны быть пустыми")
	}
}

func TestSessionIsRotatedOnLoginAndForeignIDsAreRejected(t *testing.T) {
	ts := authTestServer(t)

	// чужой идентификатор в cookie не принимается
	req, _ := http.NewRequest("GET", ts.URL+"/auth", nil)
	req.AddCookie(&http.Cookie{Name: "goit_session", Value: "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"})
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	set := res.Header.Get("Set-Cookie")
	if strings.Contains(set, "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA") {
		t.Fatal("сервер принял идентификатор сессии, выбранный клиентом (фиксация сессии)")
	}

	// при входе идентификатор меняется, старый больше не даёт доступа
	c := noFollow(jarClient(t))
	u, _ := url.Parse(ts.URL)
	csrf := csrfFrom(t, c, ts, "/auth")
	before := c.Jar.Cookies(u)[0].Value
	res, _ = postForm(c, ts.URL+"/auth", url.Values{"csrf_token": {csrf}, "role": {"student"}})
	res.Body.Close()
	after := c.Jar.Cookies(u)[0].Value
	if before == after {
		t.Fatal("идентификатор сессии не изменился при входе")
	}
	old := &http.Client{CheckRedirect: noFollow(c).CheckRedirect}
	req, _ = http.NewRequest("GET", ts.URL+"/lk/student", nil)
	req.AddCookie(&http.Cookie{Name: "goit_session", Value: before})
	res, _ = old.Do(req)
	res.Body.Close()
	if res.Header.Get("Location") != "/auth" {
		t.Fatal("старый идентификатор после входа не должен открывать кабинет")
	}
}

func TestTamperedSessionCookieIsRejected(t *testing.T) {
	ts := authTestServer(t)
	c := loginAs(t, ts, "student")
	u, _ := url.Parse(ts.URL)
	val := c.Jar.Cookies(u)[0].Value
	forged := val[:len(val)-2] + "xx"
	req, _ := http.NewRequest("GET", ts.URL+"/lk/student", nil)
	req.AddCookie(&http.Cookie{Name: "goit_session", Value: forged})
	res, _ := (&http.Client{CheckRedirect: noFollow(c).CheckRedirect}).Do(req)
	res.Body.Close()
	if res.Header.Get("Location") != "/auth" {
		t.Fatal("cookie с испорченной подписью не должна открывать кабинет")
	}
}

// --- Менеджер сессий ---

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
	m.Save(w, "expired-session-id-123456", session.Data{CSRF: "c", LoggedIn: true, Role: "student", ExpiresAt: time.Now().Add(-time.Hour)})
	ck := cookieFrom(t, w)

	// даже до фоновой очистки истёкшая сессия не принимается
	if _, d := m.Load(requestWith(ck)); d.LoggedIn {
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

func TestStudentSettingsFlow(t *testing.T) {
	ts := authTestServer(t)
	c := loginAs(t, ts, "student")
	csrf := csrfFrom(t, c, ts, "/lk/student/settings")

	// ошибки профиля
	res, _ := postForm(c, ts.URL+"/lk/student/settings", url.Values{"csrf_token": {csrf}, "action": {"profile"}, "name": {"А"}, "email": {"не-почта"}, "phone": {"12"}})
	body := readBody(t, res)
	if res.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("невалидный профиль: статус %d", res.StatusCode)
	}
	for _, want := range []string{"Введите ФИО", "Проверьте email", "Проверьте телефон", `aria-invalid="true"`, `value="не-почта"`} {
		if !strings.Contains(body, want) {
			t.Errorf("в ответе нет %q", want)
		}
	}

	// успешное сохранение и отображение в кабинете
	res, _ = postForm(c, ts.URL+"/lk/student/settings", url.Values{"csrf_token": {csrf}, "action": {"profile"}, "name": {"Мария Петрова"}, "email": {"maria@example.com"}, "phone": {"+7 900 123-45-67"}})
	body = readBody(t, res)
	if res.StatusCode != http.StatusOK || !strings.Contains(body, "Профиль сохранён.") || !strings.Contains(body, "Мария Петрова") || !strings.Contains(body, "maria@example.com") {
		t.Fatalf("профиль не сохранился или не показан (статус %d)", res.StatusCode)
	}
	res, _ = c.Get(ts.URL + "/lk/student")
	if body = readBody(t, res); !strings.Contains(body, "Мария Петрова") || !strings.Contains(body, ">МП<") {
		t.Error("имя и инициалы из настроек не попали в обзор")
	}

	// пароль: слабый и несовпадающий
	res, _ = postForm(c, ts.URL+"/lk/student/settings", url.Values{"csrf_token": {csrf}, "action": {"password"}, "password_new": {"abc"}, "password_confirm": {"xyz"}})
	body = readBody(t, res)
	if res.StatusCode != http.StatusUnprocessableEntity || !strings.Contains(body, "Новый пароль:") || !strings.Contains(body, "Пароли не совпадают.") {
		t.Fatalf("ошибки пароля не показаны (статус %d)", res.StatusCode)
	}
	// пароль: верный
	res, _ = postForm(c, ts.URL+"/lk/student/settings", url.Values{"csrf_token": {csrf}, "action": {"password"}, "password_new": {"newpass123"}, "password_confirm": {"newpass123"}})
	if body = readBody(t, res); res.StatusCode != http.StatusOK || !strings.Contains(body, "Пароль изменён.") {
		t.Fatalf("смена пароля не подтверждена (статус %d)", res.StatusCode)
	}

	// без токена
	res, _ = postForm(c, ts.URL+"/lk/student/settings", url.Values{"action": {"profile"}, "name": {"Имя"}})
	res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("без CSRF: статус %d, ожидали 403", res.StatusCode)
	}
	// ментор не может менять настройки ученика
	m := noFollow(loginAs(t, ts, "mentor"))
	res, _ = postForm(m, ts.URL+"/lk/student/settings", url.Values{"csrf_token": {csrf}, "action": {"profile"}, "name": {"Имя"}})
	res.Body.Close()
	if res.StatusCode != http.StatusSeeOther || res.Header.Get("Location") != "/lk/mentor" {
		t.Fatalf("ментор на настройках ученика: статус %d, Location=%q", res.StatusCode, res.Header.Get("Location"))
	}
}

// --- Заявки в ЛК ментора и партнёра ---

func TestMentorDecisionsAreRememberedAndSafe(t *testing.T) {
	ts := authTestServer(t)
	c := loginAs(t, ts, "mentor")
	csrf := csrfFrom(t, c, ts, "/lk/mentor/applications")

	res, _ := c.Get(ts.URL + "/lk/mentor/applications")
	before := readBody(t, res)
	if !strings.Contains(before, `/lk/mentor/applications/1/accept`) || !strings.Contains(before, `/lk/mentor/applications/2/decline`) {
		t.Fatal("до решения должны быть кнопки по обеим заявкам")
	}

	res, _ = postForm(c, ts.URL+"/lk/mentor/applications/1/accept", url.Values{"csrf_token": {csrf}})
	after := readBody(t, res)
	if !strings.Contains(after, "Заявка принята.") || !strings.Contains(after, "Принята") || strings.Contains(after, `/lk/mentor/applications/1/accept`) {
		t.Fatal("после принятия заявка должна получить статус и потерять кнопки")
	}
	if !strings.Contains(after, `/lk/mentor/applications/2/accept`) {
		t.Fatal("вторая заявка не должна измениться")
	}

	res, _ = postForm(c, ts.URL+"/lk/mentor/applications/2/decline", url.Values{"csrf_token": {csrf}})
	if body := readBody(t, res); !strings.Contains(body, "Заявка отклонена.") || !strings.Contains(body, "Отклонена") {
		t.Fatal("отклонение не отображено")
	}

	// несуществующая заявка, чужая роль, нет токена
	res, _ = postForm(c, ts.URL+"/lk/mentor/applications/99/accept", url.Values{"csrf_token": {csrf}})
	res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("заявка 99: статус %d, ожидали 404", res.StatusCode)
	}
	res, _ = postForm(c, ts.URL+"/lk/mentor/applications/1/accept", url.Values{})
	res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Errorf("без CSRF: статус %d, ожидали 403", res.StatusCode)
	}
	p := noFollow(loginAs(t, ts, "partner"))
	res, _ = postForm(p, ts.URL+"/lk/mentor/applications/1/accept", url.Values{"csrf_token": {csrf}})
	res.Body.Close()
	if res.StatusCode != http.StatusSeeOther || res.Header.Get("Location") != "/lk/partner" {
		t.Errorf("партнёр принял заявку ментора: статус %d, Location=%q", res.StatusCode, res.Header.Get("Location"))
	}
}

func TestFlashTextComesOnlyFromKnownCodes(t *testing.T) {
	ts := authTestServer(t)
	c := loginAs(t, ts, "mentor")
	res, _ := c.Get(ts.URL + "/lk/mentor/applications?done=%3Cscript%3Ealert(1)%3C/script%3E&msg=Вы+выиграли")
	body := readBody(t, res)
	if strings.Contains(body, "<script>alert") || strings.Contains(body, "Вы выиграли") {
		t.Fatal("произвольный текст из адреса попал на страницу")
	}
}

func TestPartnerDecisionsWork(t *testing.T) {
	ts := authTestServer(t)
	c := loginAs(t, ts, "partner")
	csrf := csrfFrom(t, c, ts, "/lk/partner/applications")
	res, _ := postForm(c, ts.URL+"/lk/partner/applications/3/accept", url.Values{"csrf_token": {csrf}})
	body := readBody(t, res)
	if !strings.Contains(body, "Отклик принят.") || strings.Contains(body, `/lk/partner/applications/3/accept`) {
		t.Fatal("отклик партнёра не отмечен принятым")
	}
}

// --- Обзор ученика ---

func TestStudentOverviewIsConsistent(t *testing.T) {
	ts := authTestServer(t)
	c := loginAs(t, ts, "student")
	res, _ := c.Get(ts.URL + "/lk/student")
	body := readBody(t, res)

	// радар построен из данных: 6 вершин, не одинаковый шестиугольник
	m := regexp.MustCompile(`class="radar-data" points="([^"]+)"`).FindStringSubmatch(body)
	if m == nil || len(strings.Fields(m[1])) != 6 {
		t.Fatalf("радар навыков не построен: %v", m)
	}
	if !strings.Contains(body, "Frontend 85%") || !strings.Contains(body, "Backend 77%") {
		t.Error("описание радара должно перечислять значения навыков")
	}
	// демо-результаты согласованы с реальным числом вопросов
	if strings.Contains(body, "3 из 4") || strings.Contains(body, "5 из 5") {
		t.Error("в демо-данных остались устаревшие числа вопросов")
	}
	if !strings.Contains(body, "17 из 20") || !strings.Contains(body, "14 из 18") {
		t.Error("демо-результаты должны быть 17 из 20 и 14 из 18")
	}
	// рекомендация не предлагает «продолжить» уже пройденный кейс
	if strings.Contains(body, "Рекомендуем продолжить кейс «Первый день Backend") {
		t.Error("рекомендуется кейс, который уже показан как пройденный")
	}
	// ссылки на менторов ведут на существующих менторов
	for _, l := range regexp.MustCompile(`href="(/mentors/[a-z-]+)"`).FindAllStringSubmatch(body, -1) {
		r, _ := http.Get(ts.URL + l[1])
		r.Body.Close()
		if r.StatusCode != http.StatusOK {
			t.Errorf("ссылка %s ведёт на %d", l[1], r.StatusCode)
		}
	}
}

func TestRealCaseResultShowsInStudentLK(t *testing.T) {
	ts := authTestServer(t)
	c := jarClient(t)
	_, body := get(t, c, ts.URL+"/cases/product-manager")
	for i := 0; i < 30 && !strings.Contains(body, "Разбор ответов"); i++ {
		csrf := find(t, body, `name="csrf_token" value="([^"]+)"`)
		q := find(t, body, `name="question" value="([^"]+)"`)
		opt := find(t, body, `name="option" value="([^"]+)"`)
		res, err := c.PostForm(ts.URL+"/cases/product-manager/answer", url.Values{"csrf_token": {csrf}, "question": {q}, "option": {opt}})
		if err != nil {
			t.Fatal(err)
		}
		body = readBody(t, res)
	}
	if !strings.Contains(body, "Разбор ответов") {
		t.Fatal("кейс Product Manager не дошёл до результата")
	}

	// вход в кабинет не теряет результат
	csrf := csrfFrom(t, c, ts, "/auth")
	res, _ := postForm(c, ts.URL+"/auth", url.Values{"csrf_token": {csrf}, "role": {"student"}})
	lk := readBody(t, res)
	if !strings.Contains(lk, "Первый день Product Manager") {
		t.Fatal("реально пройденный кейс не показан в кабинете ученика")
	}
}

// --- Защита от регрессии визуальных багов ---

func TestLKIconsAreOutlineAndDoorIsUsed(t *testing.T) {
	css, err := readFile(t, "static/css/app.css")
	if err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`\.lk-nav-icon\s*\{[^}]*fill:\s*none[^}]*stroke:\s*currentColor`).MatchString(css) {
		t.Error("иконки ЛК должны быть контурными: нужен CSS fill:none + stroke:currentColor")
	}
	ts := authTestServer(t)
	c := loginAs(t, ts, "student")
	res, _ := c.Get(ts.URL + "/lk/student")
	body := readBody(t, res)
	if !strings.Contains(body, `d="M6 21V4a1 1 0 0 1 1-1h10a1 1 0 0 1 1 1v17"`) {
		t.Error("кнопка «Выйти» должна использовать иконку двери")
	}
}

func readFile(t *testing.T, rel string) (string, error) {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(moduleRoot(t), rel))
	return string(b), err
}

// Карты внутри сессии не должны делиться между копиями: иначе два одновременных запроса
// одной сессии падают с «concurrent map writes» (процесс завершается аварийно).
func TestSessionDataIsIsolatedBetweenLoads(t *testing.T) {
	m := session.New("secret-for-test-4")
	defer m.Stop()

	w := httptest.NewRecorder()
	d := session.Data{CSRF: "c"}
	d.Attempt.Answers = map[string]string{"q1": "a"}
	d.Handled = map[string]string{"mentor:1": "accepted"}
	m.Save(w, "isolation-session-id-12", d)
	ck := cookieFrom(t, w)

	d.Attempt.Answers["q2"] = "changed-after-save" // изменение исходной копии после Save
	_, first := m.Load(requestWith(ck))
	if _, leaked := first.Attempt.Answers["q2"]; leaked {
		t.Fatal("Save сохранил общую карту: изменения после Save попали в хранилище")
	}
	first.Attempt.Answers["q3"] = "x"
	first.Handled["partner:1"] = "declined"
	_, second := m.Load(requestWith(ck))
	if _, leaked := second.Attempt.Answers["q3"]; leaked {
		t.Fatal("Load отдал общую карту Answers: запись в одной копии видна в другой")
	}
	if _, leaked := second.Handled["partner:1"]; leaked {
		t.Fatal("Load отдал общую карту Handled")
	}
}

// Двойной клик или две вкладки шлют ответы одной сессии одновременно. Раньше это была гонка
// данных (карта Answers общая для всех копий сессии). Тест вероятностный и без -race может не
// падать на старом коде; детерминированную проверку даёт TestSessionDataIsIsolatedBetweenLoads.
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

// Спецификация не должна расходиться с кодом: каждый маршрут из router.go перечислен
// в SPECIFICATION.md §8, и заявленное общее число совпадает с фактическим.
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
	count := 0
	err = chi.Walk(routes, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		if strings.HasPrefix(route, "/static/") {
			return nil
		}
		count++
		if !strings.Contains(spec, "| "+method+" | `"+route+"` |") {
			t.Errorf("маршрут %s %s есть в router.go, но не описан в SPECIFICATION.md §8", method, route)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	want := fmt.Sprintf("Всего %d (включая обработчик 404)", count+1+1) // +статика +404
	if !strings.Contains(spec, want) {
		t.Errorf("в SPECIFICATION.md §8 нет строки %q", want)
	}
}
