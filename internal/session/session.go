package session

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"go-it/internal/domain"
)

const (
	cookieName = "goit_session"
	csrfName   = "csrf_token"
)

// MaxSessionAge — максимальная длительность сессии (24 часа).
const MaxSessionAge = 24 * time.Hour

// CleanupInterval — интервал фоновой очистки старых сессий (1 час).
const CleanupInterval = 1 * time.Hour

// Data — состояние пользователя между запросами. Сессия нужна только кейсам и формам заявок
// (токен CSRF и ход прохождения); вход и кабинеты сессий не используют.
type Data struct {
	CSRF      string
	Attempt   domain.AttemptState
	ExpiresAt time.Time
}

type Manager struct {
	secret []byte
	mu     sync.Mutex
	store  map[string]Data
	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}
}

func New(secret string) *Manager {
	m := &Manager{
		secret: []byte(secret),
		store:  map[string]Data{},
		done:   make(chan struct{}),
	}
	m.ctx, m.cancel = context.WithCancel(context.Background())
	go m.cleanupLoop()
	return m
}

// cleanupLoop — периодическая очистка старых сессий.
func (m *Manager) cleanupLoop() {
	ticker := time.NewTicker(CleanupInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			m.Cleanup()
		case <-m.ctx.Done():
			m.Cleanup()
			close(m.done)
			return
		}
	}
}

// Cleanup удаляет все истёкшие сессии из хранилища.
func (m *Manager) Cleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	for id, data := range m.store {
		if !data.ExpiresAt.IsZero() && now.After(data.ExpiresAt) {
			delete(m.store, id)
		}
	}
}

// Delete удаляет сессию с указанным ID из хранилища.
func (m *Manager) Delete(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.store, id)
}

// Stop останавливает фоновый cleanup.
func (m *Manager) Stop() {
	if m.cancel != nil {
		m.cancel()
		<-m.done
	}
}

// clone копирует карты внутри Data. Без этого сохранённая сессия и копия, с которой работает
// обработчик, делят одни и те же карты, и два одновременных запроса одной сессии (двойной клик,
// две вкладки) образуют гонку данных; запись в карту из двух горутин может завершить процесс
// с fatal error «concurrent map writes», который Recoverer не перехватывает.
func (d Data) clone() Data {
	if d.Attempt.Answers != nil {
		answers := make(map[string]string, len(d.Attempt.Answers))
		for k, v := range d.Attempt.Answers {
			answers[k] = v
		}
		d.Attempt.Answers = answers
	}
	return d
}

// Load возвращает сессию из подписанного cookie. Если cookie нет, подпись неверна,
// сессия неизвестна серверу или истекла, возвращается новая (пустая) сессия с новым ID.
// В хранилище новая сессия попадает только после Save: запросы без cookie не оставляют записей.
func (m *Manager) Load(r *http.Request) (id string, data Data) {
	if c, err := r.Cookie(cookieName); err == nil {
		if got, ok := m.unsign(c.Value); ok {
			m.mu.Lock()
			d, found := m.store[got]
			if found && !d.ExpiresAt.IsZero() && time.Now().After(d.ExpiresAt) {
				delete(m.store, got)
				found = false
			}
			m.mu.Unlock()
			if found {
				return got, d.clone()
			}
		}
	}
	return newID(), Data{CSRF: newID()}
}

// Save сохраняет сессию и выдаёт подписанный cookie. Если срок не задан, он ставится
// на MaxSessionAge вперёд, поэтому анонимные сессии тоже очищаются фоновой задачей.
func (m *Manager) Save(w http.ResponseWriter, id string, data Data) {
	if data.ExpiresAt.IsZero() {
		data.ExpiresAt = time.Now().Add(MaxSessionAge)
	}
	m.mu.Lock()
	m.store[id] = data.clone()
	m.mu.Unlock()
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    m.sign(id),
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(MaxSessionAge.Seconds()),
	})
}

func (m *Manager) CheckCSRF(r *http.Request, data Data) bool {
	if r.Method != http.MethodPost && r.Method != http.MethodPut && r.Method != http.MethodPatch && r.Method != http.MethodDelete {
		return true
	}
	token := r.FormValue(csrfName)
	return token != "" && hmac.Equal([]byte(token), []byte(data.CSRF))
}

// Diagnose объясняет, почему запрос не прошёл проверку сессии. Нужна для журнала: из браузера
// «Сессия устарела» выглядит одинаково при перезапуске сервера, чужой cookie и просроченной странице.
func (m *Manager) Diagnose(r *http.Request) string {
	c, err := r.Cookie(cookieName)
	if err != nil {
		return "cookie сессии не получена (браузер не отправил её)"
	}
	id, ok := m.unsign(c.Value)
	if !ok {
		return "подпись cookie неверна (cookie от другой версии или другого SESSION_SECRET)"
	}
	m.mu.Lock()
	d, found := m.store[id]
	m.mu.Unlock()
	switch {
	case !found:
		return "сессия неизвестна серверу (сервер перезапускался или срок жизни вышел)"
	case !d.ExpiresAt.IsZero() && time.Now().After(d.ExpiresAt):
		return "срок жизни сессии вышел"
	}
	return "токен формы не совпал с токеном сессии (страница устарела или открыта в другой вкладке после входа)"
}

// sign возвращает значение cookie вида "<id>.<hmac>".
func (m *Manager) sign(id string) string {
	mac := hmac.New(sha256.New, m.secret)
	mac.Write([]byte(id))
	return id + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// unsign проверяет подпись и возвращает идентификатор сессии.
func (m *Manager) unsign(value string) (string, bool) {
	i := strings.LastIndexByte(value, '.')
	if i <= 0 {
		return "", false
	}
	id := value[:i]
	if len(id) < 16 {
		return "", false
	}
	want := m.sign(id)
	if !hmac.Equal([]byte(want), []byte(value)) {
		return "", false
	}
	return id, true
}

func newID() string {
	b := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		sum := sha256.Sum256([]byte(time.Now().String()))
		return base64.RawURLEncoding.EncodeToString(sum[:])
	}
	return base64.RawURLEncoding.EncodeToString(b)
}
