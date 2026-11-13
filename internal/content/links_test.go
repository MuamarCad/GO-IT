package content

import (
	"net/http"
	"os"
	"sort"
	"testing"
	"time"
)

// TestLinksAlive проверяет, что внешние ссылки на материалы открываются.
// Нужен интернет, поэтому по умолчанию тест пропускается. Запуск: make check-links.
// Ответы 403/429 (сайт не любит ботов) считаются предупреждением, а не ошибкой.
func TestLinksAlive(t *testing.T) {
	if os.Getenv("CHECK_LINKS") != "1" {
		t.Skip("проверка ссылок выключена; запустите: make check-links")
	}
	items, err := New("../../content").Roadmaps()
	if err != nil {
		t.Fatal(err)
	}
	urls := map[string][]string{}
	for _, r := range items {
		for _, s := range r.Diagram.Stages {
			for _, n := range s.Nodes {
				for _, l := range n.Links {
					urls[l.URL] = append(urls[l.URL], r.Slug+"/"+n.ID)
				}
			}
		}
	}
	list := make([]string, 0, len(urls))
	for u := range urls {
		list = append(list, u)
	}
	sort.Strings(list)

	client := &http.Client{Timeout: 20 * time.Second}
	for _, u := range list {
		req, _ := http.NewRequest(http.MethodGet, u, nil)
		req.Header.Set("User-Agent", "Mozilla/5.0 (GO-IT link checker)")
		res, err := client.Do(req)
		if err != nil {
			t.Errorf("НЕ ОТКРЫВАЕТСЯ %s (%v): %v", u, urls[u], err)
			continue
		}
		res.Body.Close()
		switch {
		case res.StatusCode >= 200 && res.StatusCode < 400:
		case res.StatusCode == http.StatusForbidden || res.StatusCode == http.StatusTooManyRequests:
			t.Logf("предупреждение: %s ответил %d (возможно, блокирует ботов) %v", u, res.StatusCode, urls[u])
		default:
			t.Errorf("ПЛОХОЙ ОТВЕТ %d: %s %v", res.StatusCode, u, urls[u])
		}
	}
	t.Logf("проверено ссылок: %d", len(list))
}
