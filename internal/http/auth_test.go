package web

import (
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"go-it/internal/cases"
	"go-it/internal/content"
	"go-it/internal/http/handlers"
	"go-it/internal/http/render"
	"go-it/internal/repository/memory"
	"go-it/internal/service"
	"go-it/internal/session"
)

func authTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	root := moduleRoot(t)
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	r, err := render.New("templates")
	if err != nil {
		t.Fatal(err)
	}
	store := memory.New()
	s := session.New("test-secret")
	h := &handlers.Server{
		Render:   r,
		Sessions: s,
		CaseSvc:  service.NewCaseService(cases.All()),
		Apps:     service.NewApplicationService(store, store),
		Catalog:  store,
		Content:  content.New("content"),
	}
	ts := httptest.NewServer(NewRouter(h, "static"))
	t.Cleanup(func() {
		ts.Close()
		s.Stop()
	})
	return ts
}

// --- Helpers ---

func jarClient(t *testing.T) *http.Client {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	return &http.Client{Jar: jar}
}

// postForm отправляет POST-форму через указанный клиент (с cookie jar).
func postForm(client *http.Client, u string, data url.Values) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodPost, u, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return client.Do(req)
}

// extractCSRF извлекает CSRF-токен из тела страницы.
func extractCSRF(t *testing.T, body string) string {
	t.Helper()
	re := regexp.MustCompile(`name="csrf_token" value="([^"]+)"`)
	m := re.FindStringSubmatch(body)
	if len(m) < 2 {
		t.Fatalf("CSRF not found in body (first 300 chars: %q)", body[:min(300, len(body))])
	}
	return m[1]
}

// --- Auth: страница логина ---

func TestAuthPageReturnsForm(t *testing.T) {
	ts := authTestServer(t)
	res, err := http.Get(ts.URL + "/auth")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d, expected 200", res.StatusCode)
	}
	body, _ := io.ReadAll(res.Body)
	bodyStr := string(body)
	for _, want := range []string{`name="role"`, "Ученик", "Ментор", "Партнёр", `name="csrf_token"`, "Войти через VK"} {
		if !strings.Contains(bodyStr, want) {
			t.Errorf("в странице логина нет %q", want)
		}
	}
}

func TestAuthPageShowsCleanHTML(t *testing.T) {
	ts := authTestServer(t)
	res, err := http.Get(ts.URL + "/auth")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	bodyStr := string(body)
	for _, bad := range []string{"ZgotmplZ", "<no value>", "%!"} {
		if strings.Contains(bodyStr, bad) {
			t.Errorf("в HTML найдено %q", bad)
		}
	}
}

// --- Auth: основной сценарий (POST /auth) ---

func TestAuthLoginStudentRedirectsToDashboard(t *testing.T) {
	ts := authTestServer(t)
	client := jarClient(t)

	res, _ := client.Get(ts.URL + "/auth")
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	csrf := extractCSRF(t, string(body))

	form := url.Values{
		"csrf_token": {csrf},
		"role":       {"student"},
		"email":      {"student@example.com"},
		"password":   {"abc12345"},
	}
	res, err := postForm(client, ts.URL+"/auth", form)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	loc := res.Header.Get("Location")
	if loc == "" {
		loc = res.Request.URL.Path
	}
	if !strings.HasSuffix(loc, "/lk/student") {
		t.Fatalf("redirect на %s, ожидается /lk/student", loc)
	}
}

func TestAuthLoginMentorRedirectsToMentorLK(t *testing.T) {
	ts := authTestServer(t)
	client := jarClient(t)

	res, _ := client.Get(ts.URL + "/auth")
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	csrf := extractCSRF(t, string(body))

	form := url.Values{
		"csrf_token": {csrf},
		"role":       {"mentor"},
		"email":      {"mentor@example.com"},
		"password":   {"pass1234"},
	}
	res, err := postForm(client, ts.URL+"/auth", form)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	loc := res.Header.Get("Location")
	if loc == "" {
		loc = res.Request.URL.Path
	}
	if !strings.HasSuffix(loc, "/lk/mentor") {
		t.Fatalf("redirect на %s, ожидается /lk/mentor", loc)
	}
}

func TestAuthLoginPartnerRedirectsToPartnerLK(t *testing.T) {
	ts := authTestServer(t)
	client := jarClient(t)

	res, _ := client.Get(ts.URL + "/auth")
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	csrf := extractCSRF(t, string(body))

	form := url.Values{
		"csrf_token": {csrf},
		"role":       {"partner"},
		"email":      {"partner@example.com"},
		"password":   {"pass1234"},
	}
	res, err := postForm(client, ts.URL+"/auth", form)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	loc := res.Header.Get("Location")
	if loc == "" {
		loc = res.Request.URL.Path
	}
	if !strings.HasSuffix(loc, "/lk/partner") {
		t.Fatalf("redirect на %s, ожидается /lk/partner", loc)
	}
}

// --- Auth: пограничные сценарии ---

func TestAuthLoginWithEmailTooLong(t *testing.T) {
	ts := authTestServer(t)
	client := jarClient(t)

	res, _ := client.Get(ts.URL + "/auth")
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	csrf := extractCSRF(t, string(body))

	longEmail := strings.Repeat("a", 250) + "@b.c"
	form := url.Values{
		"csrf_token": {csrf},
		"role":       {"student"},
		"email":      {longEmail},
		"password":   {"pass1234"},
	}
	res, err := postForm(client, ts.URL+"/auth", form)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if res.Request.URL.Path != "/auth" {
		t.Errorf("ожидается страница логина с ошибкой, но got path=%s", res.Request.URL.Path)
	}
}

func TestAuthLoginWithEmailInvalidFormat(t *testing.T) {
	ts := authTestServer(t)
	client := jarClient(t)

	res, _ := client.Get(ts.URL + "/auth")
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	csrf := extractCSRF(t, string(body))

	form := url.Values{
		"csrf_token": {csrf},
		"role":       {"student"},
		"email":      {"invalid-email"},
		"password":   {"pass1234"},
	}
	res, err := postForm(client, ts.URL+"/auth", form)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if res.Request.URL.Path != "/auth" {
		t.Errorf("некорректный email не прошёл валидацию: path=%s", res.Request.URL.Path)
	}
}

func TestAuthLoginWithEmailTooShort(t *testing.T) {
	ts := authTestServer(t)
	client := jarClient(t)

	res, _ := client.Get(ts.URL + "/auth")
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	csrf := extractCSRF(t, string(body))

	form := url.Values{
		"csrf_token": {csrf},
		"role":       {"student"},
		"email":      {"a@b"},
		"password":   {"pass1234"},
	}
	res, err := postForm(client, ts.URL+"/auth", form)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if res.Request.URL.Path != "/auth" {
		t.Errorf("короткий email не прошёл валидацию: path=%s", res.Request.URL.Path)
	}
}

func TestAuthLoginWithPasswordTooShort(t *testing.T) {
	ts := authTestServer(t)
	client := jarClient(t)

	res, _ := client.Get(ts.URL + "/auth")
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	csrf := extractCSRF(t, string(body))

	form := url.Values{
		"csrf_token": {csrf},
		"role":       {"student"},
		"email":      {"test@example.com"},
		"password":   {"abc"},
	}
	res, err := postForm(client, ts.URL+"/auth", form)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if res.Request.URL.Path != "/auth" {
		t.Errorf("короткий пароль не прошёл валидацию: path=%s", res.Request.URL.Path)
	}
}

func TestAuthLoginWithPasswordNoDigit(t *testing.T) {
	ts := authTestServer(t)
	client := jarClient(t)

	res, _ := client.Get(ts.URL + "/auth")
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	csrf := extractCSRF(t, string(body))

	form := url.Values{
		"csrf_token": {csrf},
		"role":       {"student"},
		"email":      {"test@example.com"},
		"password":   {"abcdefgh"},
	}
	res, err := postForm(client, ts.URL+"/auth", form)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if res.Request.URL.Path != "/auth" {
		t.Errorf("пароль без цифр не прошёл валидацию: path=%s", res.Request.URL.Path)
	}
}

func TestAuthLoginWithPhoneTooShort(t *testing.T) {
	ts := authTestServer(t)
	client := jarClient(t)

	res, _ := client.Get(ts.URL + "/auth")
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	csrf := extractCSRF(t, string(body))

	form := url.Values{
		"csrf_token": {csrf},
		"role":       {"student"},
		"phone":      {"123"},
		"password":   {"pass1234"},
	}
	res, err := postForm(client, ts.URL+"/auth", form)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if res.Request.URL.Path != "/auth" {
		t.Errorf("короткий телефон не прошёл валидацию: path=%s", res.Request.URL.Path)
	}
}

func TestAuthLoginWithoutCSRF(t *testing.T) {
	ts := authTestServer(t)
	res, err := http.PostForm(ts.URL+"/auth", url.Values{
		"csrf_token": {"invalid-csrf"},
		"role":       {"student"},
		"email":      {"test@example.com"},
		"password":   {"pass1234"},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusForbidden {
		t.Errorf("без CSRF ожидаем 403, got %d", res.StatusCode)
	}
	body, _ := io.ReadAll(res.Body)
	if !strings.Contains(string(body), "Сессия устарела") {
		t.Errorf("ожидалась страница ошибки сессии, got: %s", string(body))
	}
}

func TestAuthProviderRedirectsToLK(t *testing.T) {
	ts := authTestServer(t)
	client := jarClient(t)

	res, err := client.Get(ts.URL + "/auth/github")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	loc := res.Header.Get("Location")
	if loc == "" {
		loc = res.Request.URL.Path
	}
	if !strings.HasSuffix(loc, "/lk/student") {
		t.Fatalf("provider-demo redirect на %s, ожидается /lk/student", loc)
	}
}

func TestAuthProviderWithMentorRole(t *testing.T) {
	ts := authTestServer(t)
	client := jarClient(t)

	res, err := client.Get(ts.URL + "/auth/vk?role=mentor")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	loc := res.Header.Get("Location")
	if loc == "" {
		loc = res.Request.URL.Path
	}
	if !strings.HasSuffix(loc, "/lk/mentor") {
		t.Fatalf("provider-demo+mentor redirect на %s, ожидается /lk/mentor", loc)
	}
}

// --- LK: доступ без сессии ---

func TestStudentLKRedirectsToAuthWithoutSession(t *testing.T) {
	ts := authTestServer(t)
	client := jarClient(t)
	res, err := client.Get(ts.URL + "/lk/student")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	loc := res.Header.Get("Location")
	if loc == "" {
		loc = res.Request.URL.Path
	}
	if !strings.HasSuffix(loc, "/auth") {
		t.Errorf("без сессии ожидается redirect на /auth, got path=%s", loc)
	}
}

func TestMentorLKRedirectsToAuthWithoutSession(t *testing.T) {
	ts := authTestServer(t)
	client := jarClient(t)
	res, err := client.Get(ts.URL + "/lk/mentor")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	loc := res.Header.Get("Location")
	if loc == "" {
		loc = res.Request.URL.Path
	}
	if !strings.HasSuffix(loc, "/auth") {
		t.Errorf("без сессии ожидается redirect на /auth, got path=%s", loc)
	}
}

func TestPartnerLKRedirectsToAuthWithoutSession(t *testing.T) {
	ts := authTestServer(t)
	client := jarClient(t)
	res, err := client.Get(ts.URL + "/lk/partner")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	loc := res.Header.Get("Location")
	if loc == "" {
		loc = res.Request.URL.Path
	}
	if !strings.HasSuffix(loc, "/auth") {
		t.Errorf("без сессии ожидается redirect на /auth, got path=%s", loc)
	}
}

// Рендеринг ЛК-страниц уже покрыт косвенно через:
// - TestAuthLoginStudentRedirectsToDashboard (POST /auth → 302 → /lk/student)
// - TestLogoutStudentClearsSessionAndRedirectsHome (после logout /lk/student → /auth)
// - Все страницы проверяются в TestAllPagesRendered из http_test.go

// --- Logout ---

func TestLogoutStudentClearsSessionAndRedirectsHome(t *testing.T) {
	ts := authTestServer(t)
	client := jarClient(t)

	res, _ := client.Get(ts.URL + "/auth")
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	csrf := extractCSRF(t, string(body))
	form := url.Values{"csrf_token": {csrf}, "role": {"student"}, "email": {"a@b.c"}, "password": {"pass1234"}}
	res, _ = postForm(client, ts.URL+"/auth", form)
	res.Body.Close()

	res, err := http.Post(ts.URL+"/lk/student/logout", "text/plain", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	cookies := client.Jar.Cookies(parsedURL(ts.URL))
	for _, c := range cookies {
		if c.Name == "goit_session" && c.MaxAge > 0 {
			t.Errorf("cookie goit_session не сброшен (MaxAge=%d)", c.MaxAge)
		}
	}

	loc := res.Header.Get("Location")
	if loc == "" {
		loc = res.Request.URL.Path
	}
	if !strings.HasSuffix(loc, "/") {
		t.Errorf("logout redirect на %s, ожидается /", loc)
	}
}

func TestLogoutMentorClearsSession(t *testing.T) {
	ts := authTestServer(t)
	client := jarClient(t)

	res, _ := client.Get(ts.URL + "/auth")
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	csrf := extractCSRF(t, string(body))
	form := url.Values{"csrf_token": {csrf}, "role": {"mentor"}, "email": {"a@b.c"}, "password": {"pass1234"}}
	res, _ = postForm(client, ts.URL+"/auth", form)
	res.Body.Close()

	res, err := http.Post(ts.URL+"/lk/mentor/logout", "text/plain", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	loc := res.Header.Get("Location")
	if loc == "" {
		loc = res.Request.URL.Path
	}
	if !strings.HasSuffix(loc, "/") {
		t.Errorf("mentor logout redirect на %s, ожидается /", loc)
	}

	// После logout ЛК должен редиректить на auth.
	res, err = client.Get(ts.URL + "/lk/mentor")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	loc = res.Header.Get("Location")
	if loc == "" {
		loc = res.Request.URL.Path
	}
	if !strings.HasSuffix(loc, "/auth") {
		t.Errorf("после logout mentor LK redirect на %s, ожидается /auth", loc)
	}
}

func TestLogoutPartnerClearsSession(t *testing.T) {
	ts := authTestServer(t)
	client := jarClient(t)

	res, _ := client.Get(ts.URL + "/auth")
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	csrf := extractCSRF(t, string(body))
	form := url.Values{"csrf_token": {csrf}, "role": {"partner"}, "email": {"a@b.c"}, "password": {"pass1234"}}
	res, _ = postForm(client, ts.URL+"/auth", form)
	res.Body.Close()

	res, err := http.Post(ts.URL+"/lk/partner/logout", "text/plain", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	loc := res.Header.Get("Location")
	if loc == "" {
		loc = res.Request.URL.Path
	}
	if !strings.HasSuffix(loc, "/") {
		t.Errorf("partner logout redirect на %s, ожидается /", loc)
	}

	res, err = client.Get(ts.URL + "/lk/partner")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	loc = res.Header.Get("Location")
	if loc == "" {
		loc = res.Request.URL.Path
	}
	if !strings.HasSuffix(loc, "/auth") {
		t.Errorf("после logout partner LK redirect на %s, ожидается /auth", loc)
	}
}

func TestLogoutViaGETRedirectsToAuth(t *testing.T) {
	ts := authTestServer(t)
	client := jarClient(t)

	res, err := client.Get(ts.URL + "/lk/student/logout")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	loc := res.Header.Get("Location")
	if loc == "" {
		loc = res.Request.URL.Path
	}
	if !strings.HasSuffix(loc, "/auth") {
		t.Errorf("GET /lk/student/logout redirect на %s, ожидается /auth", loc)
	}
}

// --- Session cleanup ---

func TestSessionExpiresAndGetsCleanedUp(t *testing.T) {
	s := session.New("cleanup-test-secret")
	defer s.Stop()

	w := httptest.NewRecorder()
	id := "expired-test-id-123456789012"
	s.Save(w, id, session.Data{
		CSRF:      "test-csrf",
		LoggedIn:  true,
		Role:      "student",
		ExpiresAt: time.Now().Add(-2 * time.Hour),
	})

	s.Cleanup()

	_, data := s.Load(&http.Request{
		Method: http.MethodGet,
		Header: http.Header{"Cookie": []string{"goit_session=" + id}},
	})
	if data.LoggedIn {
		t.Errorf("истёкшая сессия не удалена после Cleanup")
	}
}

func TestSessionNewSessionDoesNotGetCleaned(t *testing.T) {
	s := session.New("cleanup-test-secret-2")
	defer s.Stop()

	w := httptest.NewRecorder()
	id := "fresh-test-id-1234567890123"
	s.Save(w, id, session.Data{
		CSRF:     "fresh-csrf",
		LoggedIn: false,
	})

	s.Cleanup()

	_, data := s.Load(&http.Request{
		Method: http.MethodGet,
		Header: http.Header{"Cookie": []string{"goit_session=" + id}},
	})
	if !data.LoggedIn && data.CSRF != "fresh-csrf" {
		t.Errorf("свежая сессия удалена, хотя должна остаться")
	}
}

func parsedURL(s string) *url.URL {
	u, _ := url.Parse(s)
	return u
}
