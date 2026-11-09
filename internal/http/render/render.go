package render

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"
	"unicode"
)

// Renderer хранит по одному набору шаблонов на каждую страницу:
// layouts/*.html + partials/*.html + pages/<имя>.html.
// Страница определяет блок "content", общий каркас лежит в layouts/base.html.
type Renderer struct {
	pages map[string]*template.Template
}

func New(templatesDir string) (*Renderer, error) {
	shared, err := globAll(templatesDir, "layouts", "partials")
	if err != nil {
		return nil, err
	}
	pages, err := filepath.Glob(filepath.Join(templatesDir, "pages", "*.html"))
	if err != nil {
		return nil, err
	}
	if len(pages) == 0 {
		return nil, fmt.Errorf("render: в %s/pages нет шаблонов", templatesDir)
	}

	r := &Renderer{pages: make(map[string]*template.Template, len(pages))}
	for _, page := range pages {
		files := append(append([]string{}, shared...), page)
		t, err := template.New(filepath.Base(page)).Funcs(funcs()).ParseFiles(files...)
		if err != nil {
			return nil, fmt.Errorf("render: %s: %w", filepath.Base(page), err)
		}
		r.pages[filepath.Base(page)] = t
	}
	return r, nil
}

func globAll(root string, dirs ...string) ([]string, error) {
	var out []string
	for _, d := range dirs {
		m, err := filepath.Glob(filepath.Join(root, d, "*.html"))
		if err != nil {
			return nil, err
		}
		out = append(out, m...)
	}
	return out, nil
}

type Page struct {
	Title       string
	Description string
	Active      string
	CSRF        string
	Flash       string
	FlashError  string
	Data        any
}

// HTML сначала полностью рендерит страницу в буфер, чтобы при ошибке шаблона
// пользователь не получил половину страницы со статусом 200.
func (r *Renderer) HTML(w http.ResponseWriter, name string, page Page, status int) {
	var buf bytes.Buffer
	if err := r.Execute(&buf, name, page); err != nil {
		http.Error(w, "Не удалось отобразить страницу.", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(buf.Bytes())
}

func (r *Renderer) Execute(w io.Writer, name string, page Page) error {
	t, ok := r.pages[name]
	if !ok {
		return fmt.Errorf("render: неизвестный шаблон %q", name)
	}
	return t.ExecuteTemplate(w, "base", page)
}

func funcs() template.FuncMap {
	return template.FuncMap{
		// safe помечает HTML доверенным. Использовать только для текста, прошедшего bluemonday (см. content.Loader).
		"safe":     func(s string) template.HTML { return template.HTML(s) },
		"add":      func(a, b int) int { return a + b },
		"sub":      func(a, b int) int { return a - b },
		"pct":      percent,
		"letter":   letter,
		"initials": initials,
		"capfirst": capfirst,
		"ruDate":   ruDate,
		"dict":     dict,
		"kindName": kindName,
		"host":     host,
	}
}

func percent(part, total int) int {
	if total <= 0 {
		return 0
	}
	p := part * 100 / total
	if p < 0 {
		return 0
	}
	if p > 100 {
		return 100
	}
	return p
}

func letter(i int) string {
	if i < 0 || i >= 26 {
		return ""
	}
	return string(rune('A' + i))
}

// initials: «Дарья П.» -> «ДП».
func initials(name string) string {
	var out []rune
	for _, word := range strings.Fields(name) {
		for _, r := range word {
			if unicode.IsLetter(r) {
				out = append(out, unicode.ToUpper(r))
				break
			}
		}
		if len(out) == 2 {
			break
		}
	}
	return string(out)
}

func capfirst(s string) string {
	for i, r := range s {
		return string(unicode.ToUpper(r)) + s[i+len(string(r)):]
	}
	return s
}

var ruMonths = [...]string{"января", "февраля", "марта", "апреля", "мая", "июня", "июля", "августа", "сентября", "октября", "ноября", "декабря"}

// ruDate: «2026-03-01» -> «1 марта 2026». Если формат неизвестен, возвращает строку как есть.
func ruDate(s string) string {
	t, err := time.Parse("2006-01-02", strings.TrimSpace(s))
	if err != nil {
		return s
	}
	return fmt.Sprintf("%d %s %d", t.Day(), ruMonths[t.Month()-1], t.Year())
}

// dict собирает map для передачи нескольких значений в partial: {{template "x" dict "A" 1 "B" 2}}.
func dict(pairs ...any) (map[string]any, error) {
	if len(pairs)%2 != 0 {
		return nil, fmt.Errorf("dict: нечётное число аргументов")
	}
	m := make(map[string]any, len(pairs)/2)
	for i := 0; i < len(pairs); i += 2 {
		k, ok := pairs[i].(string)
		if !ok {
			return nil, fmt.Errorf("dict: ключ %v не строка", pairs[i])
		}
		m[k] = pairs[i+1]
	}
	return m, nil
}

// kindName — подпись вида узла схемы. Нужна и в легенде, и для скринридеров (вид не должен передаваться только цветом).
func kindName(kind string) string {
	switch kind {
	case "core":
		return "Основа"
	case "also":
		return "На выбор"
	case "optional":
		return "По желанию"
	}
	return ""
}

// host возвращает домен ссылки для подписи («developer.mozilla.org»); при ошибке разбора — пустую строку.
func host(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return strings.TrimPrefix(u.Hostname(), "www.")
}
