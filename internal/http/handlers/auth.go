package handlers

import (
	"errors"
	"net/http"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"

	"go-it/internal/http/render"
	"go-it/internal/session"
)

// Авторизация в проекте демонстрационная: учётных записей и паролей нет.
// Любой вход (в том числе с пустыми полями) создаёт сессию выбранной роли — так требует REQ-002.
// Заполненные поля проверяются по формату, чтобы форма вела себя как настоящая.

const (
	roleStudent = "student"
	roleMentor  = "mentor"
	rolePartner = "partner"
)

func validRole(role string) bool {
	return role == roleStudent || role == roleMentor || role == rolePartner
}

// normalizeRole возвращает роль из белого списка, по умолчанию — ученик.
func normalizeRole(v string) string {
	if validRole(v) {
		return v
	}
	return roleStudent
}

func lkHome(role string) string {
	switch role {
	case roleMentor:
		return "/lk/mentor"
	case rolePartner:
		return "/lk/partner"
	default:
		return "/lk/student"
	}
}

// socialProviders — допустимые провайдеры входа (демо: реального OAuth нет).
var socialProviders = map[string]bool{"vk": true, "yandex": true}

func (s *Server) renderLogin(w http.ResponseWriter, sess session.Data, p render.Page, status int) {
	p.Title = "Войти — GO-IT"
	p.CSRF = sess.CSRF
	s.Render.HTML(w, "login.html", p, status)
}

// Login показывает страницу авторизации.
func (s *Server) Login(w http.ResponseWriter, r *http.Request) {
	id, sess := s.Sessions.Load(r)
	s.Sessions.Save(w, id, sess)
	s.renderLogin(w, sess, render.Page{}, http.StatusOK)
}

// LoginPost обрабатывает форму входа. Роль берётся из радиокнопок; кнопки быстрого входа
// («Войти как Ментор/Партнёр») передают роль в поле quick.
func (s *Server) LoginPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.fail(w, r, http.StatusBadRequest, "Некорректный запрос", "Не удалось обработать форму.")
		return
	}
	id, sess := s.Sessions.Load(r)
	if !s.Sessions.CheckCSRF(r, sess) {
		s.fail(w, r, http.StatusForbidden, "Сессия устарела", "Обновите страницу и попробуйте снова.")
		return
	}

	role := normalizeRole(r.FormValue("role"))
	if q := r.FormValue("quick"); validRole(q) {
		role = q
	}

	if email := strings.TrimSpace(r.FormValue("email")); email != "" {
		if err := validateEmail(email); err != nil {
			s.Sessions.Save(w, id, sess)
			s.renderLogin(w, sess, render.Page{EmailError: err.Error(), AlertError: "Проверьте правильность email."}, http.StatusUnprocessableEntity)
			return
		}
	}
	if phone := strings.TrimSpace(r.FormValue("phone")); phone != "" {
		if err := validatePhone(phone); err != nil {
			s.Sessions.Save(w, id, sess)
			s.renderLogin(w, sess, render.Page{PhoneError: err.Error(), AlertError: "Проверьте правильность телефона."}, http.StatusUnprocessableEntity)
			return
		}
	}
	// Пустой пароль допустим (демо). Если пароль введён, он должен быть «настоящим» по правилам сайта.
	if password := r.FormValue("password"); password != "" {
		if err := validatePassword(password); err != nil {
			s.Sessions.Save(w, id, sess)
			s.renderLogin(w, sess, render.Page{AlertError: "Пароль должен содержать минимум 8 символов, буквы и цифры."}, http.StatusUnprocessableEntity)
			return
		}
	}

	s.startSession(w, r, id, sess, role)
}

// LoginProvider — демо-вход через VK / Yandex ID. Только POST с CSRF-токеном:
// вход по GET позволял бы открыть чужой браузер в кабинете по ссылке.
func (s *Server) LoginProvider(w http.ResponseWriter, r *http.Request) {
	if !socialProviders[chi.URLParam(r, "provider")] {
		s.NotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		s.fail(w, r, http.StatusBadRequest, "Некорректный запрос", "Не удалось обработать форму.")
		return
	}
	id, sess := s.Sessions.Load(r)
	if !s.Sessions.CheckCSRF(r, sess) {
		s.fail(w, r, http.StatusForbidden, "Сессия устарела", "Обновите страницу и попробуйте снова.")
		return
	}
	s.startSession(w, r, id, sess, normalizeRole(r.FormValue("role")))
}

// LoginProviderGet: старые GET-ссылки на вход через провайдера больше не входят, а ведут на форму.
func (s *Server) LoginProviderGet(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/auth", http.StatusSeeOther)
}

// startSession выдаёт новый идентификатор сессии (защита от фиксации) и ведёт в кабинет роли.
func (s *Server) startSession(w http.ResponseWriter, r *http.Request, oldID string, sess session.Data, role string) {
	sess.LoggedIn = true
	sess.Role = role
	s.Sessions.Rotate(w, oldID, sess)
	http.Redirect(w, r, lkHome(role), http.StatusSeeOther)
}

// Logout завершает сессию. Выход идемпотентен, поэтому CSRF-токен не требуется;
// GET не разлогинивает, а просто ведёт на форму входа.
func (s *Server) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/auth", http.StatusSeeOther)
		return
	}
	id, _ := s.Sessions.Load(r)
	s.Sessions.Clear(w, id)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// ---------- Валидация ----------

var (
	emailRe      = regexp.MustCompile(`^[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}$`)
	phoneCleanRe = regexp.MustCompile(`[^0-9+]`)
)

const (
	maxEmailLen    = 254
	maxPhoneLen    = 20
	minPasswordLen = 8
)

// validateEmail проверяет email: длина, формат.
func validateEmail(email string) error {
	if len(email) > maxEmailLen {
		return errors.New("email не должен превышать 254 символов")
	}
	if len(email) < 5 {
		return errors.New("email слишком короткий")
	}
	if !emailRe.MatchString(email) {
		return errors.New("некорректный формат email")
	}
	return nil
}

// validatePhone проверяет телефон: допустимые символы, длина.
func validatePhone(phone string) error {
	cleaned := phoneCleanRe.ReplaceAllString(phone, "")
	if len(cleaned) < 7 || len(cleaned) > maxPhoneLen {
		return errors.New("телефон должен содержать от 7 до 20 цифр")
	}
	return nil
}

// validatePassword проверяет пароль: минимальная длина (в символах), наличие букв и цифр.
func validatePassword(pw string) error {
	if utf8.RuneCountInString(pw) < minPasswordLen {
		return errors.New("пароль должен содержать минимум 8 символов")
	}
	var hasLetter, hasDigit bool
	for _, ch := range pw {
		if unicode.IsLetter(ch) {
			hasLetter = true
		}
		if unicode.IsDigit(ch) {
			hasDigit = true
		}
	}
	if !hasLetter || !hasDigit {
		return errors.New("пароль должен содержать буквы и цифры")
	}
	return nil
}
