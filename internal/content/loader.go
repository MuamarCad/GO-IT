package content

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"

	"go-it/internal/domain"
)

type Loader struct {
	md     goldmark.Markdown
	policy *bluemonday.Policy
	root   string
}

func New(root string) *Loader {
	return &Loader{
		root:   root,
		policy: bluemonday.UGCPolicy(),
		md: goldmark.New(
			goldmark.WithExtensions(extension.GFM),
			goldmark.WithParserOptions(parser.WithAutoHeadingID()),
			goldmark.WithRendererOptions(html.WithUnsafe()),
		),
	}
}

func (l *Loader) Roadmaps() ([]domain.Roadmap, error) {
	items, err := loadMarkdown(l, filepath.Join(l.root, "roadmaps"), parseRoadmapMeta)
	if err != nil {
		return nil, err
	}
	for i := range items {
		d, err := l.loadDiagram(items[i].Slug)
		if err != nil {
			return nil, fmt.Errorf("roadmap %s: схема: %w", items[i].Slug, err)
		}
		items[i].Diagram = d
	}
	return items, nil
}

const (
	maxSummaryRunes = 400
	maxNodeLinks    = 5
)

var nodeID = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

type diagramFile struct {
	Stages []struct {
		Title  string `json:"title"`
		Note   string `json:"note"`
		Choose bool   `json:"choose"`
		Nodes  []struct {
			ID      string `json:"id"`
			Title   string `json:"title"`
			Kind    string `json:"kind"`
			Href    string `json:"href"`
			Summary string `json:"summary"`
			Links   []struct {
				Title string `json:"title"`
				URL   string `json:"url"`
			} `json:"links"`
		} `json:"nodes"`
	} `json:"stages"`
}

// loadDiagram читает roadmaps/<slug>.diagram.json. Нет файла — нет схемы (не ошибка).
// Формат строгий: лишние поля, пустые названия и неизвестные виды узлов считаются ошибкой.
func (l *Loader) loadDiagram(slug string) (*domain.Diagram, error) {
	raw, err := os.ReadFile(filepath.Join(l.root, "roadmaps", slug+".diagram.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var f diagramFile
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&f); err != nil {
		return nil, err
	}
	if len(f.Stages) == 0 {
		return nil, fmt.Errorf("нет этапов")
	}
	d := &domain.Diagram{}
	seen := map[string]bool{}
	for si, s := range f.Stages {
		if strings.TrimSpace(s.Title) == "" {
			return nil, fmt.Errorf("этап %d: пустое название", si+1)
		}
		if len(s.Nodes) == 0 {
			return nil, fmt.Errorf("этап %q: нет узлов", s.Title)
		}
		stage := domain.DiagramStage{Title: s.Title, Note: s.Note, Choose: s.Choose}
		for _, n := range s.Nodes {
			if strings.TrimSpace(n.Title) == "" {
				return nil, fmt.Errorf("этап %q: узел без названия", s.Title)
			}
			kind := n.Kind
			if kind == "" {
				kind = domain.NodeCore
			}
			if kind != domain.NodeCore && kind != domain.NodeAlso && kind != domain.NodeOptional {
				return nil, fmt.Errorf("узел %q: неизвестный вид %q", n.Title, kind)
			}
			if n.Href != "" && !strings.HasPrefix(n.Href, "/") && !strings.HasPrefix(n.Href, "https://") {
				return nil, fmt.Errorf("узел %q: ссылка должна начинаться с / или https://", n.Title)
			}
			if !nodeID.MatchString(n.ID) {
				return nil, fmt.Errorf("узел %q: id должен состоять из латинских строчных букв, цифр и дефисов", n.Title)
			}
			if seen[n.ID] {
				return nil, fmt.Errorf("узел %q: id %q повторяется", n.Title, n.ID)
			}
			seen[n.ID] = true
			if n.Href != "" && n.Summary != "" {
				return nil, fmt.Errorf("узел %q: нельзя задавать и href, и summary", n.Title)
			}
			if len([]rune(n.Summary)) > maxSummaryRunes {
				return nil, fmt.Errorf("узел %q: summary длиннее %d символов", n.Title, maxSummaryRunes)
			}
			if len(n.Links) > maxNodeLinks {
				return nil, fmt.Errorf("узел %q: больше %d ссылок", n.Title, maxNodeLinks)
			}
			node := domain.DiagramNode{ID: n.ID, Title: n.Title, Kind: kind, Href: n.Href, Summary: strings.TrimSpace(n.Summary)}
			for _, lk := range n.Links {
				if strings.TrimSpace(lk.Title) == "" || !strings.HasPrefix(lk.URL, "https://") {
					return nil, fmt.Errorf("узел %q: у ссылки нужны название и адрес на https://", n.Title)
				}
				node.Links = append(node.Links, domain.Link{Title: lk.Title, URL: lk.URL})
			}
			stage.Nodes = append(stage.Nodes, node)
		}
		d.Stages = append(d.Stages, stage)
	}
	return d, nil
}

func (l *Loader) Roadmap(slug string) (domain.Roadmap, bool, error) {
	items, err := l.Roadmaps()
	if err != nil {
		return domain.Roadmap{}, false, err
	}
	for _, item := range items {
		if item.Slug == slug {
			return item, true, nil
		}
	}
	return domain.Roadmap{}, false, nil
}

func (l *Loader) Articles() ([]domain.Article, error) {
	return loadMarkdown(l, filepath.Join(l.root, "blog"), parseArticleMeta)
}

func (l *Loader) Article(slug string) (domain.Article, bool, error) {
	items, err := l.Articles()
	if err != nil {
		return domain.Article{}, false, err
	}
	for _, item := range items {
		if item.Slug == slug {
			return item, true, nil
		}
	}
	return domain.Article{}, false, nil
}

func (l *Loader) Render(markdown string) (string, error) {
	var buf bytes.Buffer
	if err := l.md.Convert([]byte(markdown), &buf); err != nil {
		return "", err
	}
	return l.policy.Sanitize(buf.String()), nil
}

func loadMarkdown[T any](l *Loader, dir string, parse func(slug, raw, html string) T) ([]T, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []T
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		html, err := l.Render(string(raw))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", e.Name(), err)
		}
		slug := strings.TrimSuffix(e.Name(), ".md")
		out = append(out, parse(slug, string(raw), html))
	}
	return out, nil
}

func parseRoadmapMeta(slug, raw, html string) domain.Roadmap {
	title := firstHeading(raw, slug)
	intro, body := html, ""
	if i := strings.Index(html, "<h2"); i >= 0 {
		intro, body = html[:i], html[i:]
	}
	return domain.Roadmap{
		Slug:      slug,
		Title:     title,
		Summary:   firstParagraph(raw),
		HTML:      html,
		IntroHTML: intro,
		BodyHTML:  body,
	}
}

func parseArticleMeta(slug, raw, html string) domain.Article {
	title := firstHeading(raw, slug)
	published := metaLine(raw, "Дата:")
	return domain.Article{
		Slug:        slug,
		Title:       title,
		Summary:     firstParagraph(raw),
		PublishedAt: published,
		HTML:        html,
	}
}

func firstHeading(raw, fallback string) string {
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "# "))
		}
	}
	return fallback
}

func firstParagraph(raw string) string {
	var parts []string
	for _, line := range strings.Split(raw, "\n") {
		t := strings.TrimSpace(line)
		if t == "" {
			if len(parts) > 0 {
				break
			}
			continue
		}
		if strings.HasPrefix(t, "#") {
			continue
		}
		if strings.HasPrefix(t, "Дата:") {
			continue
		}
		parts = append(parts, t)
		if !strings.HasPrefix(t, "*") && !strings.HasPrefix(t, "-") {
			break
		}
	}
	summary := strings.Join(parts, " ")
	if len([]rune(summary)) > 180 {
		return string([]rune(summary)[:177]) + "…"
	}
	return summary
}

func metaLine(raw, prefix string) string {
	for _, line := range strings.Split(raw, "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(t, prefix))
		}
	}
	return ""
}
