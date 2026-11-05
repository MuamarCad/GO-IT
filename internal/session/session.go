package session

import (
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

type Data struct {
	CSRF    string
	Attempt domain.AttemptState
}

type Manager struct {
	secret []byte
	mu     sync.Mutex
	store  map[string]Data
}

func New(secret string) *Manager {
	return &Manager{
		secret: []byte(secret),
		store:  map[string]Data{},
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
