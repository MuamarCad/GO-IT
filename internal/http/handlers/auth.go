package handlers

import (
	"errors"
	"net/http"
	"regexp"
	"unicode"
	"unicode/utf8"

	"go-it/internal/http/render"
)

// Вход в проекте только визуальная механика для демонстрации: нет проверок, cookie, сессий и
// CSRF-токенов. Выбранная роль превращается в переход на страницу её кабинета. Кабинеты открыты
// по адресу `/lk/<роль>` и ничего не помнят. Реальной аутентификации нет сознательно (REQ-002 п.1).

const (
	roleStudent = "student"
	roleMentor  = "mentor"
	rolePartner = "partner"
)

func validRole(role string) bool {
	return role == roleStudent || role == roleMentor || role == rolePartner
}

// normalizeRole возвращает роль из белого списка, по умолчанию — ученик. Белый список нужен,
// чтобы адрес перехода всегда строился из фиксированных путей (никаких открытых редиректов).
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

// Login показывает страницу входа. Сессия не создаётся.
func (s *Server) Login(w http.ResponseWriter, r *http.Request) {
	s.Render.HTML(w, "login.html", render.Page{Title: "Войти — GO-IT"}, http.StatusOK)
}

// LoginPost ведёт в кабинет выбранной роли при любом содержимом формы, в том числе пустом.
// Кнопки «Войти как Ментор/Партнёр» передают роль в поле quick.
func (s *Server) LoginPost(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm() // ошибки разбора формы не важны: роль по умолчанию — ученик
	role := normalizeRole(r.FormValue("role"))
	if q := r.FormValue("quick"); validRole(q) {
		role = q
	}
	http.Redirect(w, r, lkHome(role), http.StatusSeeOther)
}

// LoginProvider — демо-вход «через VK / Yandex ID»: провайдер не вызывается, это тот же переход
// в кабинет выбранной роли (GET и POST). Роль берётся из query или формы.
func (s *Server) LoginProvider(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	http.Redirect(w, r, lkHome(normalizeRole(r.FormValue("role"))), http.StatusSeeOther)
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
