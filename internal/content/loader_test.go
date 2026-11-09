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

func TestDiagramValidation(t *testing.T) {
	cases := map[string]string{
		"пустой":          `{"stages":[]}`,
		"лишнее поле":     `{"stages":[{"title":"A","nodes":[{"title":"x"}],"oops":1}]}`,
		"нет названия":    `{"stages":[{"title":" ","nodes":[{"title":"x"}]}]}`,
		"нет узлов":       `{"stages":[{"title":"A","nodes":[]}]}`,
		"узел без имени":  `{"stages":[{"title":"A","nodes":[{"title":""}]}]}`,
		"неизвестный вид": `{"stages":[{"title":"A","nodes":[{"title":"x","kind":"big"}]}]}`,
		"опасная ссылка":  `{"stages":[{"title":"A","nodes":[{"title":"x","href":"javascript:alert(1)"}]}]}`,
		"невалидный json": `{"stages":`,
	}
	for name, body := range cases {
		dir := t.TempDir()
		_ = os.MkdirAll(filepath.Join(dir, "roadmaps"), 0o755)
		_ = os.WriteFile(filepath.Join(dir, "roadmaps", "x.md"), []byte("# X\n\nтекст\n"), 0o644)
		_ = os.WriteFile(filepath.Join(dir, "roadmaps", "x.diagram.json"), []byte(body), 0o644)
		if _, err := New(dir).Roadmaps(); err == nil {
			t.Errorf("%s: ожидали ошибку", name)
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
