package render

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"go-it/internal/domain"
)

// Если у направления нет кейса, кнопка «Пройти кейс» скрыта и показан честный текст.
// Сейчас такого направления в каталоге нет, но шаблон должен уметь это состояние.
func TestDirectionPageWithoutCase(t *testing.T) {
	if err := os.Chdir("../../.."); err != nil {
		t.Fatal(err)
	}
	r, err := New("templates")
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	d := domain.Direction{Slug: "x", Title: "X", Icon: "X", Summary: "s", RoadmapSlug: "frontend"}
	if err := r.Execute(&buf, "direction.html", Page{Data: d}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if strings.Contains(out, `href="/cases/`+"x") || !strings.Contains(out, "в разработке") {
		t.Fatal("для направления без кейса кнопки быть не должно")
	}
}
