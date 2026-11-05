package content

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
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
	return loadMarkdown(l, filepath.Join(l.root, "roadmaps"), parseRoadmapMeta)
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
	return domain.Roadmap{
		Slug:    slug,
		Title:   title,
		Summary: firstParagraph(raw),
		HTML:    html,
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
