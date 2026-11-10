package session

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"io"
	"net/http"
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

type Data struct {
	CSRF      string
	Attempt   domain.AttemptState
	LoggedIn  bool
	Role      string // "student", "mentor", "partner"
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

// Stop останавливает фоновый cleanup.
func (m *Manager) Stop() {
	if m.cancel != nil {
		m.cancel()
		<-m.done
	}
}

func (m *Manager) Load(r *http.Request) (id string, data Data) {
	c, err := r.Cookie(cookieName)
	if err != nil || c.Value == "" || !m.valid(c.Value) {
		id = newID()
		data = Data{CSRF: newID()}
		m.mu.Lock()
		m.store[id] = data
		m.mu.Unlock()
		return id, data
	}
	id = c.Value
	m.mu.Lock()
	data, ok := m.store[id]
	if !ok {
		data = Data{CSRF: newID()}
		m.store[id] = data
	}
	m.mu.Unlock()
	return id, data
}

// SaveWithExpiry сохраняет сессию и при LoggedIn=true устанавливает ExpiresAt.
func (m *Manager) SaveWithExpiry(w http.ResponseWriter, id string, data Data) {
	if data.LoggedIn && data.ExpiresAt.IsZero() {
		data.ExpiresAt = time.Now().Add(MaxSessionAge)
	}
	m.Save(w, id, data)
}

func (m *Manager) Save(w http.ResponseWriter, id string, data Data) {
	m.mu.Lock()
	m.store[id] = data
	m.mu.Unlock()
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    id,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int((24 * time.Hour).Seconds()),
	})
}

func (m *Manager) CheckCSRF(r *http.Request, data Data) bool {
	if r.Method != http.MethodPost && r.Method != http.MethodPut && r.Method != http.MethodPatch && r.Method != http.MethodDelete {
		return true
	}
	token := r.FormValue(csrfName)
	return token != "" && hmac.Equal([]byte(token), []byte(data.CSRF))
}

func (m *Manager) valid(id string) bool {
	_, err := base64.RawURLEncoding.DecodeString(id)
	return err == nil && len(id) >= 16
}

func newID() string {
	b := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		sum := sha256.Sum256([]byte(time.Now().String()))
		return base64.RawURLEncoding.EncodeToString(sum[:])
	}
	return base64.RawURLEncoding.EncodeToString(b)
}
