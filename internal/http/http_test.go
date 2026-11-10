package web

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
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

func testServer(t *testing.T) *httptest.Server {
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
	h := &handlers.Server{
		Render:   r,
		Sessions: session.New("test-secret"),
		CaseSvc:  service.NewCaseService(cases.All()),
		Apps:     service.NewApplicationService(store, store),
		Catalog:  store,
		Content:  content.New("content"),
	}
	return httptest.NewServer(NewRouter(slog.Default(), h, "static"))
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func TestHomeAnd404(t *testing.T) {
	ts := testServer(t)
	defer ts.Close()

	res, err := http.Get(ts.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode != 200 || !strings.Contains(string(body), "Найди свой") {
		t.Fatalf("home %d %s", res.StatusCode, body)
	}

	res404, err := http.Get(ts.URL + "/no-such-page")
	if err != nil {
		t.Fatal(err)
	}
	defer res404.Body.Close()
	if res404.StatusCode != 404 {
		t.Fatalf("status %d", res404.StatusCode)
	}
}

func TestCaseFlowRedirect(t *testing.T) {
	ts := testServer(t)
	defer ts.Close()
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}

	res, err := client.Get(ts.URL + "/cases/frontend")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	csrf := find(t, string(body), `name="csrf_token" value="([^"]+)"`)
	q := find(t, string(body), `name="question" value="([^"]+)"`)

	form := url.Values{
		"csrf_token": {csrf},
		"question":   {q},
		"option":     {"q1_b"},
	}
	res, err = client.PostForm(ts.URL+"/cases/frontend/answer", form)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.Request.URL.Path != "/cases/frontend" {
		t.Fatalf("path %s", res.Request.URL.Path)
	}
}

func TestMentorApplicationRedirect(t *testing.T) {
	ts := testServer(t)
	defer ts.Close()
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}

	res, err := client.Get(ts.URL + "/mentors/ivan-frontend")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	csrf := find(t, string(body), `name="csrf_token" value="([^"]+)"`)

	form := url.Values{
		"csrf_token": {csrf},
		"name":       {"Анна Тестова"},
		"contact":    {"anna@example.com"},
		"message":    {"Нужна помощь"},
	}
	res, err = client.PostForm(ts.URL+"/mentors/ivan-frontend/applications", form)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.Request.URL.Query().Get("sent") != "1" {
		t.Fatalf("expected sent=1, got %s", res.Request.URL.String())
	}
}

func TestInvalidFormStaysSafe(t *testing.T) {
	ts := testServer(t)
	defer ts.Close()
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}
	res, err := client.Get(ts.URL + "/mentors/ivan-frontend")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	csrf := find(t, string(body), `name="csrf_token" value="([^"]+)"`)
	form := url.Values{"csrf_token": {csrf}, "name": {"A"}, "contact": {"x"}}
	res, err = client.PostForm(ts.URL+"/mentors/ivan-frontend/applications", form)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if strings.Contains(string(mustRead(res)), "SQL") {
		t.Fatal("leaked sql")
	}
}

func find(t *testing.T, body, pattern string) string {
	t.Helper()
	re := regexp.MustCompile(pattern)
	m := re.FindStringSubmatch(body)
	if len(m) < 2 {
		t.Fatalf("pattern %s not in %s", pattern, body[:min(400, len(body))])
	}
	return m[1]
}

func mustRead(res *http.Response) []byte {
	b, _ := io.ReadAll(res.Body)
	return b
}

func get(t *testing.T, client *http.Client, u string) (int, string) {
	t.Helper()
	res, err := client.Get(u)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	return res.StatusCode, string(mustRead(res))
}

// Все страницы отдаются, собираются из base.html и не содержат следов шаблонных ошибок.
func TestAllPagesRender(t *testing.T) {
	ts := testServer(t)
	defer ts.Close()
	client := &http.Client{Jar: func() *cookiejar.Jar { j, _ := cookiejar.New(nil); return j }()}

	pages := []string{
		"/", "/about", "/directions", "/directions/frontend", "/directions/backend", "/directions/project-manager",
		"/cases", "/cases/frontend", "/mentors", "/mentors?dir=Backend", "/mentors?dir=Nope", "/mentors/ivan-frontend",
		"/volunteering", "/volunteering/nko-landing", "/roadmaps", "/roadmaps/frontend", "/roadmaps/backend",
		"/roadmaps/project-manager", "/blog", "/blog/first-steps-in-it", "/blog/how-to-build-a-pet-project",
	}
	for _, p := range pages {
		status, body := get(t, client, ts.URL+p)
		if status != http.StatusOK {
			t.Errorf("%s: статус %d", p, status)
			continue
		}
		for _, bad := range []string{"ZgotmplZ", "<no value>", "%!"} {
			if strings.Contains(body, bad) {
				t.Errorf("%s: в HTML найдено %q", p, bad)
			}
		}
		for _, want := range []string{`<main id="main">`, `class="skip-link"`, `/static/js/theme-init.js`, `aria-label="Основная навигация"`} {
			if !strings.Contains(body, want) {
				t.Errorf("%s: нет %q", p, want)
			}
		}
		if strings.Contains(body, "unpkg.com") {
			t.Errorf("%s: внешний CDN не нужен, HTMX лежит в static", p)
		}
	}

	status, body := get(t, client, ts.URL+"/no-such-page")
	if status != http.StatusNotFound || !strings.Contains(body, "Страница не найдена") || !strings.Contains(body, `<main id="main">`) {
		t.Errorf("404 должна быть оформленной страницей, статус %d", status)
	}
}

func TestDirectionWithoutCaseHidesCTA(t *testing.T) {
	ts := testServer(t)
	defer ts.Close()
	client := &http.Client{}

	_, pm := get(t, client, ts.URL+"/directions/project-manager")
	if strings.Contains(pm, `href="/cases/`) || !strings.Contains(pm, "в разработке") {
		t.Fatal("у PM нет кейса: кнопка «Пройти кейс» не должна вести на чужой кейс")
	}
	_, front := get(t, client, ts.URL+"/directions/frontend")
	if !strings.Contains(front, `href="/cases/frontend"`) {
		t.Fatal("у Frontend кнопка кейса должна быть")
	}
	_, back := get(t, client, ts.URL+"/directions/backend")
	if !strings.Contains(back, `href="/cases/backend"`) {
		t.Fatal("у Backend кнопка кейса должна быть")
	}
}

func TestMentorsFilter(t *testing.T) {
	ts := testServer(t)
	defer ts.Close()
	client := &http.Client{}

	_, body := get(t, client, ts.URL+"/mentors?dir=Backend")
	if !strings.Contains(body, "Дарья") || strings.Contains(body, "Иван С.") {
		t.Fatal("фильтр Backend должен оставить только Backend-менторов")
	}
	_, empty := get(t, client, ts.URL+"/mentors?dir=Nope")
	if !strings.Contains(empty, "пока нет") {
		t.Fatal("для пустого фильтра нужно пустое состояние")
	}
}

func TestInvalidFormShowsFieldErrors(t *testing.T) {
	ts := testServer(t)
	defer ts.Close()
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}

	_, body := get(t, client, ts.URL+"/volunteering/nko-landing")
	csrf := find(t, body, `name="csrf_token" value="([^"]+)"`)
	res, err := client.PostForm(ts.URL+"/volunteering/nko-landing/applications", url.Values{
		"csrf_token": {csrf}, "name": {"A"}, "contact": {""}, "message": {"привет"},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	page := string(mustRead(res))
	if res.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("ожидали 422, получили %d", res.StatusCode)
	}
	for _, want := range []string{"Введите имя", "Укажите контакт", `aria-invalid="true"`, `value="A"`, "привет"} {
		if !strings.Contains(page, want) {
			t.Errorf("в ответе нет %q", want)
		}
	}
}

func TestForeignCSRFShowsErrorPage(t *testing.T) {
	ts := testServer(t)
	defer ts.Close()
	res, err := http.PostForm(ts.URL+"/mentors/ivan-frontend/applications", url.Values{"csrf_token": {"bad"}, "name": {"Анна"}, "contact": {"a@b.c"}})
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	page := string(mustRead(res))
	if res.StatusCode != http.StatusForbidden || !strings.Contains(page, "Сессия устарела") || !strings.Contains(page, `<main id="main">`) {
		t.Fatalf("403 должна быть оформленной страницей, статус %d", res.StatusCode)
	}
}

func TestRoadmapLinksToCaseOnlyWhenItExists(t *testing.T) {
	ts := testServer(t)
	defer ts.Close()
	client := &http.Client{}

	_, front := get(t, client, ts.URL+"/roadmaps/frontend")
	if !strings.Contains(front, `data-roadmap="frontend"`) || !strings.Contains(front, `href="/cases/frontend"`) {
		t.Fatal("roadmap Frontend: нужен data-roadmap и ссылка на кейс")
	}
	_, back := get(t, client, ts.URL+"/roadmaps/backend")
	if !strings.Contains(back, `data-roadmap="backend"`) || !strings.Contains(back, `href="/cases/backend"`) {
		t.Fatal("roadmap Backend: нужен data-roadmap и ссылка на кейс")
	}
}

func TestCaseFullFlowShowsResultWithTopics(t *testing.T) {
	ts := testServer(t)
	defer ts.Close()
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}

	correct := map[string]string{
		"q1": "q1_b", "q2": "q2_b", "q3": "q3_a", "q4": "q4_a",
		"q5": "q5_a", "q6": "q6_b", "q7": "q7_a", "q8": "q8_a",
		"q9": "q9_a", "q10": "q10_a", "q11": "q11_a", "q12": "q12_b",
		"q13": "q13_a", "q14": "q14_a", "q15": "q15_a", "q16": "q16_a",
		"q17": "q17_a", "q18": "q18_a", "q19": "q19_a", "q20": "q20_a",
	}
	_, body := get(t, client, ts.URL+"/cases/frontend")
	for step := 0; step < 20; step++ {
		if !strings.Contains(body, `role="progressbar"`) || !strings.Contains(body, `<span class="letter"`) {
			t.Fatalf("шаг %d: нет прогресса или букв вариантов", step+1)
		}
		csrf := find(t, body, `name="csrf_token" value="([^"]+)"`)
		q := find(t, body, `name="question" value="([^"]+)"`)
		res, err := client.PostForm(ts.URL+"/cases/frontend/answer", url.Values{"csrf_token": {csrf}, "question": {q}, "option": {correct[q]}})
		if err != nil {
			t.Fatal(err)
		}
		body = string(mustRead(res))
		res.Body.Close()
	}
	for _, want := range []string{"20 из 20", "По темам", "Состояния форм", "Пройти заново", `href="/roadmaps/frontend"`} {
		if !strings.Contains(body, want) {
			t.Errorf("на странице результата нет %q", want)
		}
	}
}

func TestRoadmapDiagramRendered(t *testing.T) {
	ts := testServer(t)
	defer ts.Close()
	client := &http.Client{}

	for _, slug := range []string{"frontend", "backend", "project-manager"} {
		_, body := get(t, client, ts.URL+"/roadmaps/"+slug)
		if n := strings.Count(body, `class="rm-stage"`); n < 6 {
			t.Errorf("%s: этапов схемы %d, ожидали не меньше 6", slug, n)
		}
		for _, want := range []string{`class="rm-flow"`, "rm-node--core", "Выберите один", `href="/volunteering"`, `href="/mentors"`, "Основа: "} {
			if !strings.Contains(body, want) {
				t.Errorf("%s: в схеме нет %q", slug, want)
			}
		}
		// схема стоит между введением и разделами, а чек-лист и оглавление остаются в основной части
		if strings.Index(body, `class="rm-flow"`) < strings.Index(body, "<h1") || strings.Index(body, `class="rm-flow"`) > strings.Index(body, `data-roadmap="`+slug+`"`) {
			t.Errorf("%s: схема стоит не на своём месте", slug)
		}
		if strings.Count(body, "<h1") != 1 {
			t.Errorf("%s: на странице должен быть ровно один h1", slug)
		}
	}
}

func TestRoadmapTopicPanel(t *testing.T) {
	ts := testServer(t)
	defer ts.Close()
	_, body := get(t, &http.Client{}, ts.URL+"/roadmaps/frontend")

	for _, want := range []string{
		`<dialog class="rm-panel" id="rm-panel"`,    // панель
		`data-topic="t-html-semantic"`,              // узел открывает панель
		`href="#t-html-semantic"`,                   // запасной якорь без JS
		`id="t-html-semantic"`,                      // справочник тем
		`target="_blank" rel="noopener noreferrer"`, // внешние ссылки безопасны
		"developer.mozilla.org",                     // домен показывается
		"(откроется в новой вкладке)",               // подсказка для скринридера
		`data-panel-close`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("нет %q", want)
		}
	}
	// Узел-ссылка (волонтёрство) остаётся обычной ссылкой, а не открывает панель.
	if strings.Contains(body, `data-topic="t-volunteering"`) || !strings.Contains(body, `href="/volunteering"`) {
		t.Error("узел волонтёрства должен быть обычной ссылкой")
	}
	if strings.Contains(body, "javascript:") {
		t.Error("javascript: в ссылках недопустим")
	}
	// Для каждого узла с панелью есть и якорь, и статья справочника.
	anchors := strings.Count(body, `data-topic="t-`)
	articles := strings.Count(body, `class="rm-topic"`)
	if anchors == 0 || anchors != articles {
		t.Errorf("узлов с панелью %d, статей справочника %d", anchors, articles)
	}
}
