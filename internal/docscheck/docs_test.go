package docscheck

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
	"unicode"
)

var (
	owners = map[string]bool{
		"Tech Lead": true, "Системный аналитик": true, "Backend": true, "Frontend": true,
		"QA": true, "Продуктовый дизайнер": true, "Команда": true,
	}
	generalStatuses = map[string]bool{"Черновик": true, "Действует": true, "Устарел": true, "Архив": true}
	adrStatuses     = map[string]bool{"Предложен": true, "Принят": true, "Заменён": true, "Отклонён": true}
	rfcStatuses     = map[string]bool{"Обсуждается": true, "Принят": true, "Отложен": true, "Отклонён": true}
	reportStatuses  = map[string]bool{"Открыт": true, "Закрыт": true}

	datePattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	adrName     = regexp.MustCompile(`^(\d{4})-[a-z0-9]+(-[a-z0-9]+)*\.md$`)
	rfcDir      = regexp.MustCompile(`^(\d{4})-[a-z0-9]+(-[a-z0-9]+)*$`)
	linkPattern = regexp.MustCompile(`\[[^\]]*\]\(([^)\s]+)\)`)
	inlineCode  = regexp.MustCompile("`[^`]*`")
	metaLine    = regexp.MustCompile(`^\*\*([^*:]+):\*\*\s*(.*)$`)

	// Имена удалённых или переименованных документов. Исторические записи (аудиты, ADR, RFC,
	// таблица перемещений) вправе их упоминать.
	retired = []string{"DOC-STYLE", "SPECIFICATION.md", "AUDIT-REPORT", "TECH-TASKS", "qa-test-plans", "PLAN.md", "DESIGN-REVIEW", "tasks/"}
)

type doc struct {
	rel    string // путь от корня репозитория, с прямыми слэшами
	lines  []string
	meta   map[string]string
	titles []string // заголовки всех уровней вне блоков кода
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod не найден")
		}
		dir = parent
	}
}

func load(t *testing.T, root, rel string) *doc {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	d := &doc{rel: rel, lines: strings.Split(string(raw), "\n"), meta: map[string]string{}}
	inFence := false
	for _, l := range d.lines {
		if strings.HasPrefix(strings.TrimSpace(l), "```") {
			inFence = !inFence
			continue
		}
		if !inFence && strings.HasPrefix(l, "#") {
			d.titles = append(d.titles, l)
		}
	}
	// метаданные: подряд идущие строки «**Поле:** значение» после заголовка и пустой строки
	i := 1
	for i < len(d.lines) && strings.TrimSpace(d.lines[i]) == "" {
		i++
	}
	for ; i < len(d.lines); i++ {
		m := metaLine.FindStringSubmatch(d.lines[i])
		if m == nil {
			break
		}
		d.meta[m[1]] = strings.TrimSpace(m[2])
	}
	return d
}

// docsToCheck возвращает проверяемые документы: корневые файлы и docs/**/*.md без шаблонов
// (кроме каталога шаблонов README.md).
func docsToCheck(t *testing.T, root string) []string {
	t.Helper()
	files := []string{"README.md", "CONTRIBUTING.md", "SECURITY.md", "GIGACODE.md"}
	err := filepath.WalkDir(filepath.Join(root, "docs"), func(p string, e fs.DirEntry, err error) error {
		if err != nil || e.IsDir() || !strings.HasSuffix(p, ".md") {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		rel = filepath.ToSlash(rel)
		if strings.HasPrefix(rel, "docs/templates/") && rel != "docs/templates/README.md" {
			return nil
		}
		files = append(files, rel)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(files)
	return files
}

func allowedStatuses(rel string) (map[string]bool, string) {
	switch {
	case strings.HasPrefix(rel, "docs/architecture/adr/") && rel != "docs/architecture/adr/README.md":
		return adrStatuses, "ADR"
	case strings.HasPrefix(rel, "docs/rfc/"):
		return rfcStatuses, "RFC"
	case strings.HasPrefix(rel, "docs/quality/"):
		return reportStatuses, "отчёт"
	}
	return generalStatuses, "документ"
}

func TestEveryDocumentHasTitleAndMetadata(t *testing.T) {
	root := repoRoot(t)
	for _, rel := range docsToCheck(t, root) {
		d := load(t, root, rel)
		if len(d.lines) == 0 || !strings.HasPrefix(d.lines[0], "# ") {
			t.Errorf("%s: первая строка должна быть заголовком первого уровня «# …»", rel)
			continue
		}
		h1 := 0
		for _, h := range d.titles {
			if strings.HasPrefix(h, "# ") {
				h1++
			}
		}
		if h1 != 1 {
			t.Errorf("%s: заголовков первого уровня %d, должен быть один", rel, h1)
		}
		for _, f := range []string{"Статус", "Владелец", "Обновлено"} {
			if d.meta[f] == "" {
				t.Errorf("%s: нет поля «%s» в блоке метаданных", rel, f)
			}
		}
		if st, kind := allowedStatuses(rel); d.meta["Статус"] != "" && !st[d.meta["Статус"]] {
			t.Errorf("%s: статус %q недопустим для типа «%s»", rel, d.meta["Статус"], kind)
		}
		if o := d.meta["Владелец"]; o != "" && !owners[o] {
			t.Errorf("%s: владелец %q не из списка стандарта", rel, o)
		}
		if u := d.meta["Обновлено"]; u != "" && !datePattern.MatchString(u) {
			t.Errorf("%s: дата %q не в формате ГГГГ-ММ-ДД", rel, u)
		}
		if _, old := d.meta["Версия"]; old {
			t.Errorf("%s: поле «Версия» устарело, используйте «Обновлено» и CHANGELOG", rel)
		}
	}
}

func TestReadmeIsShort(t *testing.T) {
	root := repoRoot(t)
	d := load(t, root, "README.md")
	n := len(d.lines)
	if d.lines[n-1] == "" {
		n--
	}
	if n > 80 {
		t.Errorf("README.md содержит %d строк, максимум 80 (подробности выносите в docs/)", n)
	}
}

// slug строит якорь заголовка так же, как GitHub.
func slug(heading string) string {
	h := strings.TrimSpace(strings.TrimLeft(heading, "#"))
	h = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`).ReplaceAllString(h, "$1")
	h = strings.NewReplacer("`", "", "**", "", "*", "").Replace(h)
	var b strings.Builder
	for _, r := range strings.ToLower(h) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_':
			b.WriteRune(r)
		case r == ' ':
			b.WriteRune('-')
		}
	}
	return b.String()
}

func anchors(d *doc) map[string]bool {
	out := map[string]bool{}
	count := map[string]int{}
	for _, h := range d.titles {
		s := slug(h)
		if n := count[s]; n > 0 {
			out[s+"-"+strconv.Itoa(n)] = true
		} else {
			out[s] = true
		}
		count[s]++
	}
	return out
}

// links возвращает относительные ссылки документа (вне блоков кода).
func links(d *doc) []string {
	var out []string
	inFence := false
	for _, l := range d.lines {
		if strings.HasPrefix(strings.TrimSpace(l), "```") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		l = inlineCode.ReplaceAllString(l, "")
		for _, m := range linkPattern.FindAllStringSubmatch(l, -1) {
			if strings.HasPrefix(m[1], "http") || strings.HasPrefix(m[1], "mailto:") {
				continue
			}
			out = append(out, m[1])
		}
	}
	return out
}

func resolve(from, target string) (path, anchor string) {
	path, anchor, _ = strings.Cut(target, "#")
	if path == "" {
		return from, anchor
	}
	return filepath.ToSlash(filepath.Join(filepath.Dir(from), path)), anchor
}

func TestRelativeLinksAndAnchorsResolve(t *testing.T) {
	root := repoRoot(t)
	cache := map[string]*doc{}
	get := func(rel string) *doc {
		if d, ok := cache[rel]; ok {
			return d
		}
		d := load(t, root, rel)
		cache[rel] = d
		return d
	}
	for _, rel := range docsToCheck(t, root) {
		for _, target := range links(get(rel)) {
			path, anchor := resolve(rel, target)
			info, err := os.Stat(filepath.Join(root, filepath.FromSlash(path)))
			if err != nil {
				t.Errorf("%s: ссылка %q ведёт на несуществующий путь %s", rel, target, path)
				continue
			}
			if anchor == "" || info.IsDir() || !strings.HasSuffix(path, ".md") {
				continue
			}
			if !anchors(get(path))[anchor] {
				t.Errorf("%s: в %s нет заголовка с якорем #%s", rel, path, anchor)
			}
		}
	}
}

func linkedFrom(t *testing.T, root, rel string) map[string]bool {
	t.Helper()
	d := load(t, root, rel)
	out := map[string]bool{}
	for _, target := range links(d) {
		p, _ := resolve(rel, target)
		out[p] = true
	}
	return out
}

func TestIndexesListEveryDocument(t *testing.T) {
	root := repoRoot(t)
	index := linkedFrom(t, root, "docs/README.md")
	adrIndex := linkedFrom(t, root, "docs/architecture/adr/README.md")
	tplIndex := linkedFrom(t, root, "docs/templates/README.md")

	_ = filepath.WalkDir(filepath.Join(root, "docs"), func(p string, e fs.DirEntry, err error) error {
		if err != nil || e.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		rel = filepath.ToSlash(rel)
		if !strings.HasSuffix(rel, ".md") && !strings.HasSuffix(rel, ".json") && !strings.HasSuffix(rel, ".docx") {
			return nil
		}
		switch {
		case rel == "docs/README.md":
		case strings.HasPrefix(rel, "docs/architecture/adr/") && rel != "docs/architecture/adr/README.md":
			if !adrIndex[rel] {
				t.Errorf("ADR %s не перечислен в docs/architecture/adr/README.md", rel)
			}
		case strings.HasPrefix(rel, "docs/templates/") && rel != "docs/templates/README.md":
			if !tplIndex[rel] {
				t.Errorf("шаблон %s не перечислен в docs/templates/README.md", rel)
			}
		case strings.HasPrefix(rel, "docs/rfc/") && !strings.HasSuffix(rel, "README.md"):
			// приложения RFC достаточно связать из самого RFC
		default:
			if !index[rel] {
				t.Errorf("документ %s не перечислен в docs/README.md", rel)
			}
		}
		return nil
	})
}

func TestADRNumberingAndTitles(t *testing.T) {
	root := repoRoot(t)
	entries, err := os.ReadDir(filepath.Join(root, "docs/architecture/adr"))
	if err != nil {
		t.Fatal(err)
	}
	next := 1
	for _, e := range entries {
		if e.Name() == "README.md" {
			continue
		}
		m := adrName.FindStringSubmatch(e.Name())
		if m == nil {
			t.Errorf("ADR %s: имя должно быть NNNN-название.md", e.Name())
			continue
		}
		num := atoi(m[1])
		if num != next {
			t.Errorf("ADR %s: ожидался номер %04d (нумерация сквозная, без пропусков)", e.Name(), next)
		}
		next = num + 1
		d := load(t, root, "docs/architecture/adr/"+e.Name())
		if !strings.HasPrefix(d.lines[0], "# ADR-"+m[1]+": ") {
			t.Errorf("ADR %s: заголовок должен начинаться с «# ADR-%s: »", e.Name(), m[1])
		}
		if !datePattern.MatchString(d.meta["Дата решения"]) {
			t.Errorf("ADR %s: нужно поле «Дата решения» в формате ГГГГ-ММ-ДД", e.Name())
		}
		for _, section := range []string{"## Контекст", "## Решение", "## Рассмотренные альтернативы", "## Последствия"} {
			if !strings.Contains(strings.Join(d.lines, "\n"), "\n"+section+"\n") {
				t.Errorf("ADR %s: нет раздела «%s»", e.Name(), section)
			}
		}
	}
	if next == 1 {
		t.Error("в docs/architecture/adr нет ни одного ADR")
	}
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

func TestRFCLayout(t *testing.T) {
	root := repoRoot(t)
	entries, err := os.ReadDir(filepath.Join(root, "docs/rfc"))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		m := rfcDir.FindStringSubmatch(e.Name())
		if m == nil {
			t.Errorf("RFC %s: имя папки должно быть NNNN-название", e.Name())
			continue
		}
		rel := "docs/rfc/" + e.Name() + "/README.md"
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel))); err != nil {
			t.Errorf("RFC %s: нет README.md", e.Name())
			continue
		}
		if d := load(t, root, rel); !strings.HasPrefix(d.lines[0], "# RFC-"+m[1]+": ") {
			t.Errorf("%s: заголовок должен начинаться с «# RFC-%s: »", rel, m[1])
		}
	}
}

func TestNoReferencesToRetiredDocuments(t *testing.T) {
	root := repoRoot(t)
	historical := func(rel string) bool {
		return rel == "docs/README.md" || strings.HasPrefix(rel, "docs/quality/") ||
			strings.HasPrefix(rel, "docs/architecture/adr/") || strings.HasPrefix(rel, "docs/rfc/")
	}
	for _, rel := range docsToCheck(t, root) {
		if historical(rel) {
			continue
		}
		d := load(t, root, rel)
		for i, l := range d.lines {
			for _, name := range retired {
				if strings.Contains(l, name) {
					t.Errorf("%s:%d: упоминание удалённого или переименованного документа %q", rel, i+1, name)
				}
			}
		}
	}
}

func TestTemplatesExistForEveryDocumentType(t *testing.T) {
	root := repoRoot(t)
	for _, name := range []string{"document", "adr", "rfc", "runbook", "audit-report", "postmortem", "test-plan", "requirement", "pull-request"} {
		if _, err := os.Stat(filepath.Join(root, "docs/templates", name+".md")); err != nil {
			t.Errorf("нет шаблона docs/templates/%s.md", name)
		}
	}
}
