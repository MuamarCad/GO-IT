package web

import (
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"

	"go-it/internal/cases"
	"go-it/internal/content"
	"go-it/internal/http/handlers"
	"go-it/internal/http/render"
	"go-it/internal/repository/memory"
	"go-it/internal/service"
	"go-it/internal/session"
)

func newTestRouter(t *testing.T) http.Handler {
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
	s := session.New("test-secret-0123456789")
	t.Cleanup(s.Stop)
	h := &handlers.Server{
		Render:   r,
		Sessions: s,
		CaseSvc:  service.NewCaseService(cases.All()),
		Apps:     service.NewApplicationService(store, store),
		Catalog:  store,
		Content:  content.New("content"),
	}
	return NewRouter(slog.Default(), h, "static")
}

func authTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	ts := httptest.NewServer(newTestRouter(t))
	t.Cleanup(ts.Close)
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

func postForm(client *http.Client, u string, data url.Values) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodPost, u, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return client.Do(req)
}

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

func parsedURL(s string) *url.URL {
	u, _ := url.Parse(s)
	return u
}

// ---------- Вход: только визуальная механика ----------

var lkHome = map[string]string{"student": "/lk/student", "mentor": "/lk/mentor", "partner": "/lk/partner"}

func TestLoginPageIsPlainAndStateless(t *testing.T) {
	ts := authTestServer(t)
	res, err := http.Get(ts.URL + "/auth")
	if err != nil {
		t.Fatal(err)
	}
	body := readBody(t, res)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("статус %d", res.StatusCode)
	}
	if res.Header.Get("Set-Cookie") != "" {
		t.Error("страница входа не должна выдавать cookie")
	}
	if strings.Contains(body, "csrf_token") {
		t.Error("на странице входа не должно быть CSRF-токена")
	}
	if strings.Count(body, `type="radio" name="role"`) != 3 || strings.Contains(body, `role="radio"`) {
		t.Error("роли должны быть тремя настоящими радиокнопками")
	}
	if strings.Contains(body, "<script>") || strings.Contains(body, "onclick=") {
		t.Error("на странице входа не должно быть inline-скриптов и onclick")
	}
	if regexp.MustCompile(`name="(email|password|phone)"[^>]*value="[^"]+"`).MatchString(body) {
		t.Error("поля входа должны быть пустыми")
	}
	for _, want := range []string{`formaction="/auth/vk"`, `formaction="/auth/yandex"`, `name="quick" value="mentor"`, `name="quick" value="partner"`} {
		if !strings.Contains(body, want) {
			t.Errorf("нет элемента %s", want)
		}
	}
}

// Любой ввод, в том числе пустой, ведёт в выбранный кабинет; ни одна из операций не выдаёт cookie.
func TestLoginAlwaysRedirectsToChosenRoleWithoutCookies(t *testing.T) {
	ts := authTestServer(t)
	c := noFollow(&http.Client{})
	cases := []struct {
		name, method, path string
		form               url.Values
		want               string
	}{
		{"пустая форма", "POST", "/auth", url.Values{}, "/lk/student"},
		{"ученик", "POST", "/auth", url.Values{"role": {"student"}}, "/lk/student"},
		{"ментор", "POST", "/auth", url.Values{"role": {"mentor"}}, "/lk/mentor"},
		{"партнёр", "POST", "/auth", url.Values{"role": {"partner"}}, "/lk/partner"},
		{"мусор в полях", "POST", "/auth", url.Values{"role": {"mentor"}, "email": {"не почта"}, "password": {"1"}, "phone": {"abc"}}, "/lk/mentor"},
		{"очень длинный email", "POST", "/auth", url.Values{"role": {"partner"}, "email": {strings.Repeat("a", 500) + "@x.ru"}}, "/lk/partner"},
		{"неизвестная роль", "POST", "/auth", url.Values{"role": {"admin"}}, "/lk/student"},
		{"быстрый вход ментора", "POST", "/auth", url.Values{"role": {"student"}, "quick": {"mentor"}}, "/lk/mentor"},
		{"быстрый вход партнёра", "POST", "/auth", url.Values{"quick": {"partner"}}, "/lk/partner"},
		{"быстрый вход с мусором", "POST", "/auth", url.Values{"quick": {"root"}, "role": {"partner"}}, "/lk/partner"},
		{"VK, POST", "POST", "/auth/vk", url.Values{"role": {"mentor"}}, "/lk/mentor"},
		{"Yandex, POST", "POST", "/auth/yandex", url.Values{"role": {"partner"}}, "/lk/partner"},
		{"VK, GET", "GET", "/auth/vk?role=mentor", nil, "/lk/mentor"},
		{"неизвестный провайдер", "POST", "/auth/anything", url.Values{"role": {"student"}}, "/lk/student"},
	}
	for _, tc := range cases {
		var res *http.Response
		var err error
		if tc.method == "GET" {
			res, err = c.Get(ts.URL + tc.path)
		} else {
			res, err = c.PostForm(ts.URL+tc.path, tc.form)
		}
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		res.Body.Close()
		if res.StatusCode != http.StatusSeeOther || res.Header.Get("Location") != tc.want {
			t.Errorf("%s: статус %d, Location=%q, ожидали 303 на %s", tc.name, res.StatusCode, res.Header.Get("Location"), tc.want)
		}
		if res.Header.Get("Set-Cookie") != "" {
			t.Errorf("%s: вход не должен выдавать cookie, получили %q", tc.name, res.Header.Get("Set-Cookie"))
		}
	}
}

func TestLoginRedirectTargetsAreFixedPaths(t *testing.T) {
	ts := authTestServer(t)
	c := noFollow(&http.Client{})
	for _, evil := range []string{"//evil.example", "https://evil.example", "/lk/../admin", "student\r\nLocation: https://evil.example"} {
		res, err := c.PostForm(ts.URL+"/auth", url.Values{"role": {evil}, "quick": {evil}})
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.Header.Get("Location") != "/lk/student" {
			t.Errorf("role=%q: Location=%q (адрес перехода должен строиться из фиксированных путей)", evil, res.Header.Get("Location"))
		}
	}
}
