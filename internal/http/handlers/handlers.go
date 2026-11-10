package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"unicode"

	"github.com/go-chi/chi/v5"

	"go-it/internal/catalog"
	"go-it/internal/content"
	"go-it/internal/domain"
	"go-it/internal/http/render"
	"go-it/internal/service"
	"go-it/internal/session"
)

type Catalog interface {
	ListMentors(ctx context.Context) ([]domain.Mentor, error)
	GetMentor(ctx context.Context, slug string) (domain.Mentor, bool, error)
	ListVolunteerTasks(ctx context.Context) ([]domain.VolunteerTask, error)
	GetVolunteerTask(ctx context.Context, slug string) (domain.VolunteerTask, bool, error)
}

type HealthPinger interface {
	Ping(ctx context.Context) error
}

type Server struct {
	Render   *render.Renderer
	Sessions *session.Manager
	CaseSvc  *service.CaseService
	Apps     *service.ApplicationService
	Catalog  Catalog
	Content  *content.Loader
	ReadyDB  HealthPinger
}

func (s *Server) page(w http.ResponseWriter, r *http.Request, name, title, active string, data any, status int) {
	id, sess := s.Sessions.Load(r)
	s.Sessions.Save(w, id, sess)
	s.Render.HTML(w, name, render.Page{
		Title:  title,
		Active: active,
		CSRF:   sess.CSRF,
		Data:   data,
	}, status)
}

func (s *Server) Home(w http.ResponseWriter, r *http.Request) {
	roadmaps, _ := s.Content.Roadmaps()
	articles, _ := s.Content.Articles()
	mentors, _ := s.Catalog.ListMentors(r.Context())
	tasks, _ := s.Catalog.ListVolunteerTasks(r.Context())
	s.page(w, r, "home.html", "GO-IT — социальный код", "home", map[string]any{
		"Directions": catalog.Directions(),
		"Roadmaps":   roadmaps,
		"Articles":   articles,
		"Mentor":     firstMentor(mentors),
		"Task":       firstTask(tasks),
	}, http.StatusOK)
}

func (s *Server) About(w http.ResponseWriter, r *http.Request) {
	s.page(w, r, "about.html", "О проекте — GO-IT", "about", nil, http.StatusOK)
}

func (s *Server) Directions(w http.ResponseWriter, r *http.Request) {
	s.page(w, r, "directions.html", "Направления — GO-IT", "directions", catalog.Directions(), http.StatusOK)
}

func (s *Server) Direction(w http.ResponseWriter, r *http.Request) {
	d, ok := catalog.DirectionBySlug(chi.URLParam(r, "slug"))
	if !ok {
		s.NotFound(w, r)
		return
	}
	s.page(w, r, "direction.html", d.Title+" — GO-IT", "directions", d, http.StatusOK)
}

func (s *Server) Cases(w http.ResponseWriter, r *http.Request) {
	s.page(w, r, "cases.html", "Первый кейс — GO-IT", "cases", s.CaseSvc.List(), http.StatusOK)
}

func (s *Server) CaseStart(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	id, sess := s.Sessions.Load(r)
	if sess.Attempt.Slug != slug || sess.Attempt.Completed {
		state, view, err := s.CaseSvc.Start(slug)
		if err != nil {
			s.NotFound(w, r)
			return
		}
		sess.Attempt = state
		s.Sessions.Save(w, id, sess)
		s.Render.HTML(w, "case_question.html", render.Page{
			Title:  view.Case.Title + " — GO-IT",
			Active: "cases",
			CSRF:   sess.CSRF,
			Data:   view,
		}, http.StatusOK)
		return
	}
	view, err := s.CaseSvc.Current(sess.Attempt)
	if err != nil {
		s.NotFound(w, r)
		return
	}
	if r.URL.Query().Get("error") == "invalid_answer" {
		view.Error = "Неверный вариант. Выберите ответ ещё раз."
	}
	s.Render.HTML(w, "case_question.html", render.Page{
		Title:  view.Case.Title + " — GO-IT",
		Active: "cases",
		CSRF:   sess.CSRF,
		Data:   view,
	}, http.StatusOK)
}

func (s *Server) CaseAnswer(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if err := r.ParseForm(); err != nil {
		s.fail(w, r, http.StatusBadRequest, "Некорректный запрос", "Не удалось прочитать форму. Вернитесь назад и попробуйте ещё раз.")
		return
	}
	id, sess := s.Sessions.Load(r)
	if !s.Sessions.CheckCSRF(r, sess) {
		s.fail(w, r, http.StatusForbidden, "Сессия устарела", "Обновите страницу и ответьте ещё раз.")
		return
	}
	if sess.Attempt.Slug != slug {
		http.Redirect(w, r, "/cases/"+slug, http.StatusSeeOther)
		return
	}
	state, outcome, err := s.CaseSvc.Answer(sess.Attempt, r.FormValue("question"), r.FormValue("option"))
	sess.Attempt = state
	s.Sessions.Save(w, id, sess)
	if errors.Is(err, service.ErrUnknownOption) && outcome.Next != nil {
		http.Redirect(w, r, "/cases/"+slug+"?error=invalid_answer", http.StatusSeeOther)
		return
	}
	if err != nil {
		http.Redirect(w, r, "/cases/"+slug, http.StatusSeeOther)
		return
	}
	if outcome.Finished {
		http.Redirect(w, r, "/cases/"+slug+"/result", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/cases/"+slug, http.StatusSeeOther)
}

func (s *Server) CaseResult(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	id, sess := s.Sessions.Load(r)
	s.Sessions.Save(w, id, sess)
	if sess.Attempt.Slug != slug || !sess.Attempt.Completed {
		http.Redirect(w, r, "/cases/"+slug, http.StatusSeeOther)
		return
	}
	result, err := s.CaseSvc.Result(sess.Attempt)
	if err != nil {
		http.Redirect(w, r, "/cases/"+slug, http.StatusSeeOther)
		return
	}
	s.Render.HTML(w, "case_result.html", render.Page{
		Title:  "Результат — GO-IT",
		Active: "cases",
		CSRF:   sess.CSRF,
		Data:   result,
	}, http.StatusOK)
}

// formState — состояние формы заявки для повторного показа страницы.
type formState struct {
	Values map[string]string
	Errors map[string]string
	Sent   bool
	Failed bool
}

type applyPage func(w http.ResponseWriter, r *http.Request, form formState, status int)

func (s *Server) Mentors(w http.ResponseWriter, r *http.Request) {
	items, err := s.Catalog.ListMentors(r.Context())
	if err != nil {
		s.internal(w)
		return
	}
	var filters []string
	seen := map[string]bool{}
	for _, m := range items {
		if !seen[m.Specialization] {
			seen[m.Specialization] = true
			filters = append(filters, m.Specialization)
		}
	}
	filter := strings.TrimSpace(r.URL.Query().Get("dir"))
	shown := items
	if filter != "" {
		shown = nil
		for _, m := range items {
			if strings.EqualFold(m.Specialization, filter) {
				shown = append(shown, m)
			}
		}
	}
	s.page(w, r, "mentors.html", "Менторы — GO-IT", "mentors", map[string]any{
		"Items":   shown,
		"Filter":  filter,
		"Filters": filters,
	}, http.StatusOK)
}

func (s *Server) Mentor(w http.ResponseWriter, r *http.Request) {
	s.showMentor(w, r, formState{
		Sent:   r.URL.Query().Get("sent") == "1",
		Failed: r.URL.Query().Get("error") == "1",
	}, http.StatusOK)
}

func (s *Server) showMentor(w http.ResponseWriter, r *http.Request, form formState, status int) {
	m, ok, err := s.Catalog.GetMentor(r.Context(), chi.URLParam(r, "slug"))
	if err != nil {
		s.internal(w)
		return
	}
	if !ok {
		s.NotFound(w, r)
		return
	}
	s.page(w, r, "mentor.html", m.DisplayName+" — GO-IT", "mentors", map[string]any{
		"Mentor": m,
		"Form":   form,
	}, status)
}

func (s *Server) MentorApply(w http.ResponseWriter, r *http.Request) {
	s.submitApp(w, r, domain.KindMentor, "/mentors/", s.showMentor)
}

func (s *Server) Volunteering(w http.ResponseWriter, r *http.Request) {
	items, err := s.Catalog.ListVolunteerTasks(r.Context())
	if err != nil {
		s.internal(w)
		return
	}
	s.page(w, r, "volunteering.html", "Волонтёрство — GO-IT", "volunteering", items, http.StatusOK)
}

func (s *Server) VolunteerTask(w http.ResponseWriter, r *http.Request) {
	s.showTask(w, r, formState{
		Sent:   r.URL.Query().Get("sent") == "1",
		Failed: r.URL.Query().Get("error") == "1",
	}, http.StatusOK)
}

func (s *Server) showTask(w http.ResponseWriter, r *http.Request, form formState, status int) {
	t, ok, err := s.Catalog.GetVolunteerTask(r.Context(), chi.URLParam(r, "slug"))
	if err != nil {
		s.internal(w)
		return
	}
	if !ok {
		s.NotFound(w, r)
		return
	}
	s.page(w, r, "volunteer.html", t.Title+" — GO-IT", "volunteering", map[string]any{
		"Task": t,
		"Form": form,
	}, status)
}

func (s *Server) VolunteerApply(w http.ResponseWriter, r *http.Request) {
	s.submitApp(w, r, domain.KindVolunteer, "/volunteering/", s.showTask)
}

func (s *Server) submitApp(w http.ResponseWriter, r *http.Request, kind domain.ApplicationKind, prefix string, show applyPage) {
	slug := chi.URLParam(r, "slug")
	if err := r.ParseForm(); err != nil {
		s.fail(w, r, http.StatusBadRequest, "Некорректный запрос", "Не удалось прочитать форму. Вернитесь назад и попробуйте ещё раз.")
		return
	}
	id, sess := s.Sessions.Load(r)
	if !s.Sessions.CheckCSRF(r, sess) {
		s.fail(w, r, http.StatusForbidden, "Сессия устарела", "Обновите страницу и отправьте форму ещё раз.")
		return
	}
	s.Sessions.Save(w, id, sess)

	app := domain.Application{
		Kind:     kind,
		TargetID: slug,
		Name:     r.FormValue("name"),
		Contact:  r.FormValue("contact"),
		Message:  r.FormValue("message"),
	}
	_, err := s.Apps.Submit(r.Context(), app)
	switch {
	case err == nil:
		http.Redirect(w, r, prefix+slug+"?sent=1", http.StatusSeeOther)
	case errors.Is(err, service.ErrUnknownTarget):
		s.NotFound(w, r)
	case errors.Is(err, service.ErrInvalidApplication):
		errs := service.FieldErrors(app)
		if len(errs) == 0 {
			s.fail(w, r, http.StatusBadRequest, "Некорректный запрос", "Не удалось обработать заявку.")
			return
		}
		show(w, r, formState{
			Values: map[string]string{"name": app.Name, "contact": app.Contact, "message": app.Message},
			Errors: errs,
		}, http.StatusUnprocessableEntity)
	default:
		http.Redirect(w, r, prefix+slug+"?error=1", http.StatusSeeOther)
	}
}

func (s *Server) Roadmaps(w http.ResponseWriter, r *http.Request) {
	items, err := s.Content.Roadmaps()
	if err != nil {
		s.internal(w)
		return
	}
	s.page(w, r, "roadmaps.html", "Roadmaps — GO-IT", "roadmaps", items, http.StatusOK)
}

func (s *Server) Roadmap(w http.ResponseWriter, r *http.Request) {
	item, ok, err := s.Content.Roadmap(chi.URLParam(r, "slug"))
	if err != nil {
		s.internal(w)
		return
	}
	if !ok {
		s.NotFound(w, r)
		return
	}
	s.page(w, r, "roadmap.html", item.Title+" — GO-IT", "roadmaps", map[string]any{
		"Roadmap":  item,
		"CaseSlug": caseForRoadmap(item.Slug),
	}, http.StatusOK)
}

func (s *Server) Blog(w http.ResponseWriter, r *http.Request) {
	items, err := s.Content.Articles()
	if err != nil {
		s.internal(w)
		return
	}
	s.page(w, r, "blog.html", "Блог — GO-IT", "blog", items, http.StatusOK)
}

func (s *Server) Article(w http.ResponseWriter, r *http.Request) {
	item, ok, err := s.Content.Article(chi.URLParam(r, "slug"))
	if err != nil {
		s.internal(w)
		return
	}
	if !ok {
		s.NotFound(w, r)
		return
	}
	s.page(w, r, "article.html", item.Title+" — GO-IT", "blog", item, http.StatusOK)
}

func (s *Server) Live(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (s *Server) Ready(w http.ResponseWriter, r *http.Request) {
	if s.ReadyDB != nil {
		if err := s.ReadyDB.Ping(r.Context()); err != nil {
			http.Error(w, "not ready", http.StatusServiceUnavailable)
			return
		}
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (s *Server) NotFound(w http.ResponseWriter, r *http.Request) {
	s.page(w, r, "not_found.html", "Страница не найдена — GO-IT", "", nil, http.StatusNotFound)
}

func (s *Server) internal(w http.ResponseWriter) {
	s.Render.HTML(w, "error.html", render.Page{
		Title: "Внутренняя ошибка — GO-IT",
		Data:  errorView{Code: http.StatusInternalServerError, Heading: "Что-то пошло не так", Message: "Это ошибка на нашей стороне. Попробуйте обновить страницу чуть позже."},
	}, http.StatusInternalServerError)
}

type errorView struct {
	Code    int
	Heading string
	Message string
}

// fail показывает оформленную страницу ошибки, не раскрывая технических деталей.
func (s *Server) fail(w http.ResponseWriter, r *http.Request, status int, heading, message string) {
	id, sess := s.Sessions.Load(r)
	s.Sessions.Save(w, id, sess)
	s.Render.HTML(w, "error.html", render.Page{
		Title: heading + " — GO-IT",
		CSRF:  sess.CSRF,
		Data:  errorView{Code: status, Heading: heading, Message: message},
	}, status)
}

// caseForRoadmap возвращает slug кейса для направления с этой дорожной картой или пустую строку.
func caseForRoadmap(roadmapSlug string) string {
	for _, d := range catalog.Directions() {
		if d.RoadmapSlug == roadmapSlug {
			return d.CaseSlug
		}
	}
	return ""
}

func firstMentor(items []domain.Mentor) *domain.Mentor {
	if len(items) == 0 {
		return nil
	}
	return &items[0]
}

func firstTask(items []domain.VolunteerTask) *domain.VolunteerTask {
	if len(items) == 0 {
		return nil
	}
	return &items[0]
}

// ---------- Авторизация и ЛК (демо-страницы) ----------

// Login показывает страницу авторизации.
func (s *Server) Login(w http.ResponseWriter, r *http.Request) {
	id, sess := s.Sessions.Load(r)
	s.Sessions.Save(w, id, sess)
	s.Render.HTML(w, "login.html", render.Page{
		Title:  "Войти — GO-IT",
		Active: "",
		CSRF:   sess.CSRF,
	}, http.StatusOK)
}

// LoginPost обрабатывает форму авторизации (демо-режим).
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

	// Валидация роли.
	role := r.FormValue("role")
	switch role {
	case "student", "mentor", "partner":
	default:
		role = "student"
	}

	// Валидация email (если заполнен).
	if email := r.FormValue("email"); email != "" {
		if err := validateEmail(email); err != nil {
			s.Sessions.Save(w, id, sess)
			s.Render.HTML(w, "login.html", render.Page{
				Title:      "Войти — GO-IT",
				Active:     "",
				CSRF:       sess.CSRF,
				EmailError: err.Error(),
				AlertError: "Проверьте правильность email.",
			}, http.StatusOK)
			return
		}
	}

	// Валидация телефона (если заполнен).
	if phone := r.FormValue("phone"); phone != "" {
		if err := validatePhone(phone); err != nil {
			s.Sessions.Save(w, id, sess)
			s.Render.HTML(w, "login.html", render.Page{
				Title:      "Войти — GO-IT",
				Active:     "",
				CSRF:       sess.CSRF,
				PhoneError: err.Error(),
				AlertError: "Проверьте правильность телефона.",
			}, http.StatusOK)
			return
		}
	}

	// Валидация пароля (мин. 8 символов, хотя бы 1 буква и 1 цифра).
	// Если пароль не заполнен — пропускаем валидацию (демо-режим).
	if password := r.FormValue("password"); password != "" {
		if err := validatePassword(password); err != nil {
			s.Sessions.Save(w, id, sess)
			s.Render.HTML(w, "login.html", render.Page{
				Title:     "Войти — GO-IT",
				Active:    "",
				CSRF:      sess.CSRF,
				AlertError: "Пароль должен содержать минимум 8 символов, буквы и цифры.",
			}, http.StatusOK)
			return
		}
	}

	sess.LoggedIn = true
	sess.Role = role
	s.Sessions.SaveWithExpiry(w, id, sess)

	switch role {
	case "mentor":
		http.Redirect(w, r, "/lk/mentor", http.StatusSeeOther)
	case "partner":
		http.Redirect(w, r, "/lk/partner", http.StatusSeeOther)
	default:
		http.Redirect(w, r, "/lk/student", http.StatusSeeOther)
	}
}

// LoginProvider — переход на соц. авторизацию (VK, Yandex).
func (s *Server) LoginProvider(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")
	role := r.URL.Query().Get("role")
	if role == "" {
		role = "student"
	}
	// Демо: просто перенаправляем в ЛК.
	_ = provider
	_ = role
	id, sess := s.Sessions.Load(r)
	sess.LoggedIn = true
	sess.Role = role
	s.Sessions.SaveWithExpiry(w, id, sess)

	switch role {
	case "mentor":
		http.Redirect(w, r, "/lk/mentor", http.StatusSeeOther)
	case "partner":
		http.Redirect(w, r, "/lk/partner", http.StatusSeeOther)
	default:
		http.Redirect(w, r, "/lk/student", http.StatusSeeOther)
	}
}

// ---------- ЛК: Ученик ----------

// studentRecommendedCase — данные для рекомендации кейса в ЛК ученика.
type studentRecommendedCase struct {
	Slug       string
	Title      string
	CurrentStep int
	TotalSteps int
	Topic      string
}

// studentMentorRecommend — данные для рекомендации ментора в ЛК ученика.
type studentMentorRecommend struct {
	Slug           string
	Name           string
	Specialization string
	Location       string
}

// studentCompletedCase — данные о пройденном кейсе в ЛК ученика.
type studentCompletedCase struct {
	Direction string
	Title     string
	Correct   int
	Total     int
	Percent   int
	Passed    bool
}

func (s *Server) lkStudentPage(w http.ResponseWriter, r *http.Request, current string) {
	id, sess := s.Sessions.Load(r)
	if !sess.LoggedIn {
		http.Redirect(w, r, "/auth", http.StatusSeeOther)
		return
	}
	s.Sessions.Save(w, id, sess)

	// Считаем прогресс из сессии.
	attempt := sess.Attempt
	casesCompleted := 0
	points := 0
	progressPercent := 0
	activeRoadmap := "Не выбран"
	currentLevel := 1
	totalLevels := 5
	nextCaseHint := "Выберите направление и пройдите первый кейс, чтобы начать обучение."

	if attempt.Completed {
		casesCompleted = 1
		// Для MVP: если кейс пройден, начисляем бонусные баллы.
		points = attempt.Index // кол-во отвеченных вопросов = баллы
	} else if attempt.Slug != "" {
		// Ищем кейс в сервисе для расчёта шага.
		if c, err := s.CaseSvc.Get(attempt.Slug); err == nil {
			casesCompleted = 0
			percent := 0
			if totalQ := len(c.Questions); totalQ > 0 {
				percent = (attempt.Index * 100) / totalQ
				if percent > 100 {
					percent = 100
				}
			}
			progressPercent = percent
			// Профильная информация из кейса.
			activeRoadmap = c.Profession
		}
	} else {
		// Нет активного кейса — ищем неначатый.
		if c, err := s.CaseSvc.Get("frontend"); err == nil {
			_ = c
		}
	}

	// Рекомендации: неначатые кейсы.
	var recommendedCases []studentRecommendedCase
	for _, d := range catalog.Directions() {
		if d.CaseSlug != "" {
			if c, err := s.CaseSvc.Get(d.CaseSlug); err == nil {
				rc := studentRecommendedCase{
					Slug:       c.Slug,
					Title:      c.Title,
					TotalSteps: len(c.Questions),
					Topic:      c.Intro,
				}
				// Если это текущий кейс — подставляем шаг.
				if attempt.Slug == c.Slug && !attempt.Completed {
					rc.CurrentStep = attempt.Index + 1
					if rc.CurrentStep > rc.TotalSteps {
						rc.CurrentStep = rc.TotalSteps
					}
				}
				recommendedCases = append(recommendedCases, rc)
			}
		}
	}

	// Рекомендации менторов из каталога.
	var recommendedMentors []studentMentorRecommend
	for _, m := range catalog.SeedMentors() {
		recommendedMentors = append(recommendedMentors, studentMentorRecommend{
			Slug:           m.Slug,
			Name:           m.DisplayName,
			Specialization: m.Specialization,
			Location:       m.Format,
		})
	}

	// Завершённые кейсы: показываем из сессии.
	var completedCases []studentCompletedCase
	if attempt.Completed && attempt.Slug != "" {
		if c, err := s.CaseSvc.Get(attempt.Slug); err == nil {
			correct := 0
			for _, q := range c.Questions {
				if chosenID, ok := attempt.Answers[q.ID]; ok {
					for _, o := range q.Options {
						if o.ID == chosenID && o.Correct {
							correct++
						}
					}
				}
			}
			total := len(c.Questions)
			percent := 0
			if total > 0 {
				percent = (correct * 100) / total
			}
			capProf := c.Profession
			// capitalize first letter
			if len(capProf) > 0 {
				r := []rune(capProf)
				r[0] = unicode.ToUpper(r[0])
				capProf = string(r)
			}
			completedCases = append(completedCases, studentCompletedCase{
				Direction: capProf,
				Title:     c.Title,
				Correct:   correct,
				Total:     total,
				Percent:   percent,
				Passed:    percent >= 60,
			})
		}
	}

	data := map[string]any{
		"UserName":          "Алексей С.",
		"Initials":          "АС",
		"CasesCompleted":    casesCompleted,
		"Points":            points,
		"RoadmapsOpened":    casesCompleted,
		"Consultations":     0,
		"ActiveRoadmap":     activeRoadmap,
		"CurrentLevel":      currentLevel,
		"TotalLevels":       totalLevels,
		"ProgressPercent":   progressPercent,
		"NextCaseHint":      nextCaseHint,
		"RecommendedCases":  recommendedCases,
		"AvailableMentors":  recommendedMentors,
		"CompletedCases":    completedCases,
		"NextConsultation":  nil, // будет при интеграции с расписанием
	}

	templateName := "lk_student.html"
	switch current {
	case "student-cases":
		templateName = "lk_student_cases.html"
	case "student-roadmap":
		templateName = "lk_student_roadmap.html"
	case "student-mentors":
		templateName = "lk_student_mentors.html"
	case "student-volunteering":
		templateName = "lk_student_volunteering.html"
	case "student-settings":
		templateName = "lk_settings.html"
	}

	s.Render.HTML(w, templateName, render.Page{
		Title:  "Личный кабинет · Ученик — GO-IT",
		Active: current,
		CSRF:   sess.CSRF,
		Data:   data,
	}, http.StatusOK)
}

func (s *Server) LKStudent(w http.ResponseWriter, r *http.Request) { s.lkStudentPage(w, r, "student-overview") }
func (s *Server) LKStudentCases(w http.ResponseWriter, r *http.Request) { s.lkStudentPage(w, r, "student-cases") }
func (s *Server) LKStudentRoadmap(w http.ResponseWriter, r *http.Request) { s.lkStudentPage(w, r, "student-roadmap") }
func (s *Server) LKStudentMentors(w http.ResponseWriter, r *http.Request) { s.lkStudentPage(w, r, "student-mentors") }
func (s *Server) LKStudentVolunteering(w http.ResponseWriter, r *http.Request) { s.lkStudentPage(w, r, "student-volunteering") }
func (s *Server) LKStudentSettings(w http.ResponseWriter, r *http.Request) { s.lkStudentPage(w, r, "student-settings") }

// ---------- ЛК: Ментор ----------

type mentorApplication struct {
	ID          int
	StudentName string
	Contact     string
	Date        string
	Time        string
	Message     string
}

type mentorSession struct {
	StudentName string
	Date        string
	Time        string
	Topic       string
}

type mentorStudent struct {
	Initials        string
	Name            string
	Direction       string
	RoadmapsOpened  int
	Progress        int
}

func (s *Server) lkMentorPage(w http.ResponseWriter, r *http.Request, current string) {
	id, sess := s.Sessions.Load(r)
	if !sess.LoggedIn {
		http.Redirect(w, r, "/auth", http.StatusSeeOther)
		return
	}
	s.Sessions.Save(w, id, sess)
	s.Render.HTML(w, "lk_mentor.html", render.Page{
		Title:  "Личный кабинет · Ментор — GO-IT",
		Active: current,
		CSRF:   sess.CSRF,
		Data: map[string]any{
			"UserName":         "Анна К.",
			"Initials":         "АК",
			"Specialization":   "Frontend",
			"NewApplications":  2,
			"TotalStudents":    5,
			"TotalSessions":    18,
			"Rating":           "4.9",
			"Applications": []mentorApplication{
				{ID: 1, StudentName: "Дмитрий П.", Contact: "dmitry@mail.ru", Date: "12 ноя", Time: "15:00", Message: "Хотел бы обсудить работу с React Hooks"},
				{ID: 2, StudentName: "Мария Л.", Contact: "@maria_tg", Date: "14 ноя", Time: "12:00", Message: "Нужна помощь с TypeScript типами"},
			},
			"UpcomingSessions": []mentorSession{
				{StudentName: "Иван С.", Date: "10 ноя", Time: "18:00", Topic: "CSS Grid и Flexbox"},
				{StudentName: "Ольга Н.", Date: "11 ноя", Time: "16:00", Topic: "Деплой фронтенда"},
			},
			"ActiveStudents": []mentorStudent{
				{Initials: "ДС", Name: "Дмитрий С.", Direction: "Frontend", RoadmapsOpened: 1, Progress: 40},
				{Initials: "ОН", Name: "Ольга Н.", Direction: "Frontend", RoadmapsOpened: 2, Progress: 65},
				{Initials: "ПМ", Name: "Пётр М.", Direction: "Backend", RoadmapsOpened: 0, Progress: 10},
			},
		},
	}, http.StatusOK)
}

func (s *Server) LKMentor(w http.ResponseWriter, r *http.Request)          { s.lkMentorPage(w, r, "mentor-overview") }
func (s *Server) LKMentorApplications(w http.ResponseWriter, r *http.Request) { s.lkMentorPage(w, r, "mentor-applications") }
func (s *Server) LKMentorSchedule(w http.ResponseWriter, r *http.Request)  { s.lkMentorPage(w, r, "mentor-schedule") }
func (s *Server) LKMentorStudents(w http.ResponseWriter, r *http.Request)  { s.lkMentorPage(w, r, "mentor-students") }
func (s *Server) LKMentorProfile(w http.ResponseWriter, r *http.Request)   { s.lkMentorPage(w, r, "mentor-profile") }

func (s *Server) LKMentorAccept(w http.ResponseWriter, r *http.Request) {
	s.lkMentorAction(w, r, "accepted")
}
func (s *Server) LKMentorDecline(w http.ResponseWriter, r *http.Request) {
	s.lkMentorAction(w, r, "declined")
}
func (s *Server) lkMentorAction(w http.ResponseWriter, r *http.Request, action string) {
	id, sess := s.Sessions.Load(r)
	if !sess.LoggedIn {
		http.Redirect(w, r, "/auth", http.StatusSeeOther)
		return
	}
	if !s.Sessions.CheckCSRF(r, sess) {
		http.Redirect(w, r, "/lk/mentor/applications", http.StatusSeeOther)
		return
	}
	_ = chi.URLParam(r, "id")
	s.Sessions.Save(w, id, sess)
	msg := "Заявка принята"
	if action == "declined" {
		msg = "Заявка отклонена"
	}
	http.Redirect(w, r, "/lk/mentor/applications?msg="+url.QueryEscape(msg), http.StatusSeeOther)
}

// ---------- ЛК: Партнёр ----------

type partnerApplication struct {
	ID         int
	VolunteerName string
	TaskTitle  string
	Skills     string
	Contact    string
	Message    string
}

type partnerTask struct {
	Title              string
	Skills             string
	Scale              string
	Status             string
	ApplicationsCount  int
}

func (s *Server) lkPartnerPage(w http.ResponseWriter, r *http.Request, current string) {
	id, sess := s.Sessions.Load(r)
	if !sess.LoggedIn {
		http.Redirect(w, r, "/auth", http.StatusSeeOther)
		return
	}
	s.Sessions.Save(w, id, sess)
	s.Render.HTML(w, "lk_partner.html", render.Page{
		Title:  "Личный кабинет · Партнёр — GO-IT",
		Active: current,
		CSRF:   sess.CSRF,
		Data: map[string]any{
			"OrganizationName": "Фонд «Открытый код»",
			"Role":             "Координатор",
			"Initials":         "ОК",
			"NewApplications":  3,
			"ActiveTasks":      4,
			"TotalApplications": 17,
			"Accepted":         9,
			"PublishedTasks":   6,
			"CompletedProjects": 3,
			"TotalVolunteers":  24,
			"Applications": []partnerApplication{
				{ID: 1, VolunteerName: "Елена В.", TaskTitle: "Вёрстка лендинга для акции", Skills: "HTML, CSS", Contact: "elena@google.com", Message: "Есть опыт вёрстки, могу помочь на этой неделе"},
				{ID: 2, VolunteerName: "Сергей К.", TaskTitle: "Телеграм-бот для регистрации", Skills: "Python", Contact: "@sergey_tg"},
				{ID: 3, VolunteerName: "Алина Р.", TaskTitle: "Аналитика опросов участников", Skills: "Excel,数据分析", Contact: "alina@edu.ru", Message: "Работала с аналогичными данными"},
			},
			"Tasks": []partnerTask{
				{Title: "Вёрстка лендинга для акции", Skills: "HTML, CSS", Scale: "Онлайн", Status: "Активна", ApplicationsCount: 5},
				{Title: "Телеграм-бот для регистрации", Skills: "Python, Aiogram", Scale: "Онлайн", Status: "Активна", ApplicationsCount: 3},
				{Title: "Аналитика опросов участников", Skills: "Excel, SQL", Scale: "Удалённо", Status: "Активна", ApplicationsCount: 4},
				{Title: "Презентация для вебинара", Skills: "PowerPoint, дизайн", Scale: "Удалённо", Status: "Активна", ApplicationsCount: 2},
			},
		},
	}, http.StatusOK)
}

func (s *Server) LKPartner(w http.ResponseWriter, r *http.Request)               { s.lkPartnerPage(w, r, "partner-overview") }
func (s *Server) LKPartnerTasks(w http.ResponseWriter, r *http.Request)          { s.lkPartnerPage(w, r, "partner-tasks") }
func (s *Server) LKPartnerApplications(w http.ResponseWriter, r *http.Request)  { s.lkPartnerPage(w, r, "partner-applications") }
func (s *Server) LKPartnerAnalytics(w http.ResponseWriter, r *http.Request)     { s.lkPartnerPage(w, r, "partner-analytics") }
func (s *Server) LKPartnerSettings(w http.ResponseWriter, r *http.Request)      { s.lkPartnerPage(w, r, "partner-settings") }

func (s *Server) LKPartnerAccept(w http.ResponseWriter, r *http.Request) {
	s.lkPartnerAction(w, r, "accepted")
}
func (s *Server) LKPartnerDecline(w http.ResponseWriter, r *http.Request) {
	s.lkPartnerAction(w, r, "declined")
}
func (s *Server) lkPartnerAction(w http.ResponseWriter, r *http.Request, action string) {
	id, sess := s.Sessions.Load(r)
	if !sess.LoggedIn {
		http.Redirect(w, r, "/auth", http.StatusSeeOther)
		return
	}
	if !s.Sessions.CheckCSRF(r, sess) {
		http.Redirect(w, r, "/lk/partner/applications", http.StatusSeeOther)
		return
	}
	_ = chi.URLParam(r, "id")
	s.Sessions.Save(w, id, sess)
	msg := "Отклик принят"
	if action == "declined" {
		msg = "Отклик отклонён"
	}
	http.Redirect(w, r, "/lk/partner/applications?msg="+url.QueryEscape(msg), http.StatusSeeOther)
}

// ---------- Logout ----------

// LKStudentLogout завершает сессию ученика.
func (s *Server) LKStudentLogout(w http.ResponseWriter, r *http.Request) {
	s.doLogout(w, r, "/lk/student")
}

// LKMentorLogout завершает сессию ментора.
func (s *Server) LKMentorLogout(w http.ResponseWriter, r *http.Request) {
	s.doLogout(w, r, "/lk/mentor")
}

// LKPartnerLogout завершает сессию партнёра.
func (s *Server) LKPartnerLogout(w http.ResponseWriter, r *http.Request) {
	s.doLogout(w, r, "/lk/partner")
}

// doLogout — общий обработчик выхода: сбрасывает сессию и редиректит на главную.
func (s *Server) doLogout(w http.ResponseWriter, r *http.Request, redirectOnFail string) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/auth", http.StatusSeeOther)
		return
	}
	id, _ := s.Sessions.Load(r)
	s.Sessions.Delete(id)
	// CSRF проверка не обязательна при logout, но не помешает.
	// Сбрасываем сессию: устанавливаем cookie с MaxAge = -0
	http.SetCookie(w, &http.Cookie{
		Name:     "goit_session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func LimitBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		next.ServeHTTP(w, r)
	})
}

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Request-ID") == "" {
			w.Header().Set("X-Request-ID", strings.TrimSpace(r.RemoteAddr))
		}
		next.ServeHTTP(w, r)
	})
}

// ---------- Валидация ----------

var (
	emailRe     = regexp.MustCompile(`^[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}$`)
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

// validatePassword проверяет пароль: минимальная длина, наличие букв и цифр.
func validatePassword(pw string) error {
	if len(pw) < minPasswordLen {
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
