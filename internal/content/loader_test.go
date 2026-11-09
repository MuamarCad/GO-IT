package content

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Реальные схемы из content/ валидны и достаточно содержательны.
func TestRealRoadmapDiagrams(t *testing.T) {
	l := New("../../content")
	items, err := l.Roadmaps()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 3 {
		t.Fatalf("ожидали 3 roadmap, получили %d", len(items))
	}
	for _, r := range items {
		if r.Diagram == nil || len(r.Diagram.Stages) < 6 {
			t.Errorf("%s: схема отсутствует или слишком короткая", r.Slug)
			continue
		}
		titles := map[string]bool{}
		for _, s := range r.Diagram.Stages {
			if titles[s.Title] {
				t.Errorf("%s: этап %q повторяется", r.Slug, s.Title)
			}
			titles[s.Title] = true
			if s.Choose {
				for _, n := range s.Nodes {
					if n.Kind != "also" {
						t.Errorf("%s: в этапе «на выбор» %q узел %q не «also»", r.Slug, s.Title, n.Title)
					}
				}
			}
		}
		if r.IntroHTML == "" || !strings.Contains(r.BodyHTML, "<h2") || strings.Contains(r.IntroHTML, "<h2") {
			t.Errorf("%s: введение и основная часть разделены неверно", r.Slug)
		}
	}
}

func loadWith(t *testing.T, diagramJSON string) error {
	t.Helper()
	dir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(dir, "roadmaps"), 0o755)
	_ = os.WriteFile(filepath.Join(dir, "roadmaps", "x.md"), []byte("# X\n\nтекст\n"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "roadmaps", "x.diagram.json"), []byte(diagramJSON), 0o644)
	_, err := New(dir).Roadmaps()
	return err
}

func TestDiagramValidation(t *testing.T) {
	longSummary := strings.Repeat("я", 401)
	cases := map[string]string{
		"пустой":                `{"stages":[]}`,
		"невалидный json":       `{"stages":`,
		"лишнее поле":           `{"stages":[{"title":"A","nodes":[{"id":"a","title":"x"}],"oops":1}]}`,
		"нет названия этапа":    `{"stages":[{"title":" ","nodes":[{"id":"a","title":"x"}]}]}`,
		"нет узлов":             `{"stages":[{"title":"A","nodes":[]}]}`,
		"узел без имени":        `{"stages":[{"title":"A","nodes":[{"id":"a","title":""}]}]}`,
		"неизвестный вид":       `{"stages":[{"title":"A","nodes":[{"id":"a","title":"x","kind":"big"}]}]}`,
		"опасная ссылка узла":   `{"stages":[{"title":"A","nodes":[{"id":"a","title":"x","href":"javascript:alert(1)"}]}]}`,
		"нет id":                `{"stages":[{"title":"A","nodes":[{"title":"x"}]}]}`,
		"плохой id":             `{"stages":[{"title":"A","nodes":[{"id":"Привет x","title":"x"}]}]}`,
		"повтор id":             `{"stages":[{"title":"A","nodes":[{"id":"a","title":"x"}]},{"title":"B","nodes":[{"id":"a","title":"y"}]}]}`,
		"href и summary вместе": `{"stages":[{"title":"A","nodes":[{"id":"a","title":"x","href":"/m","summary":"текст"}]}]}`,
		"длинное summary":       `{"stages":[{"title":"A","nodes":[{"id":"a","title":"x","summary":"` + longSummary + `"}]}]}`,
		"много ссылок":          `{"stages":[{"title":"A","nodes":[{"id":"a","title":"x","summary":"t","links":[` + strings.TrimSuffix(strings.Repeat(`{"title":"l","url":"https://a.b"},`, 6), ",") + `]}]}]}`,
		"ссылка не https":       `{"stages":[{"title":"A","nodes":[{"id":"a","title":"x","summary":"t","links":[{"title":"l","url":"http://a.b"}]}]}]}`,
		"ссылка javascript":     `{"stages":[{"title":"A","nodes":[{"id":"a","title":"x","summary":"t","links":[{"title":"l","url":"javascript:alert(1)"}]}]}]}`,
		"ссылка без названия":   `{"stages":[{"title":"A","nodes":[{"id":"a","title":"x","summary":"t","links":[{"title":" ","url":"https://a.b"}]}]}]}`,
	}
	for name, body := range cases {
		if err := loadWith(t, body); err == nil {
			t.Errorf("%s: ожидали ошибку", name)
		}
	}
}

func TestDiagramNodeWithSummaryAndLinks(t *testing.T) {
	dir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(dir, "roadmaps"), 0o755)
	_ = os.WriteFile(filepath.Join(dir, "roadmaps", "x.md"), []byte("# X\n\nтекст\n"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "roadmaps", "x.diagram.json"), []byte(`{"stages":[{"title":"A","nodes":[
		{"id":"html-basics","title":"HTML","summary":"  Разметка страницы.  ","links":[{"title":"MDN","url":"https://developer.mozilla.org/ru/docs/Web/HTML"}]},
		{"id":"mentors","title":"Наставник","href":"/mentors"}]}]}`), 0o644)
	items, err := New(dir).Roadmaps()
	if err != nil {
		t.Fatal(err)
	}
	nodes := items[0].Diagram.Stages[0].Nodes
	if nodes[0].Summary != "Разметка страницы." || len(nodes[0].Links) != 1 || nodes[0].Links[0].URL != "https://developer.mozilla.org/ru/docs/Web/HTML" {
		t.Fatalf("узел с панелью прочитан неверно: %+v", nodes[0])
	}
	if nodes[1].Href != "/mentors" || nodes[1].Summary != "" {
		t.Fatalf("узел-ссылка прочитан неверно: %+v", nodes[1])
	}
}

// Каждая тема реальных схем либо ведёт по внутренней ссылке, либо имеет описание для панели;
// у большинства тем есть материалы для углубления.
func TestRealDiagramNodesAreComplete(t *testing.T) {
	items, err := New("../../content").Roadmaps()
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range items {
		total, withLinks := 0, 0
		for _, s := range r.Diagram.Stages {
			for _, n := range s.Nodes {
				total++
				if n.Href == "" && n.Summary == "" {
					t.Errorf("%s: у темы %q нет ни описания, ни ссылки", r.Slug, n.Title)
				}
				if len(n.Links) > 0 {
					withLinks++
				}
			}
		}
		if withLinks*100/total < 65 {
			t.Errorf("%s: материалы есть только у %d из %d тем", r.Slug, withLinks, total)
		}
	}
}

func TestRoadmapWithoutDiagramIsFine(t *testing.T) {
	dir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(dir, "roadmaps"), 0o755)
	_ = os.WriteFile(filepath.Join(dir, "roadmaps", "x.md"), []byte("# X\n\nтекст\n"), 0o644)
	items, err := New(dir).Roadmaps()
	if err != nil || len(items) != 1 || items[0].Diagram != nil {
		t.Fatalf("roadmap без схемы должен загружаться: %v", err)
	}
}
