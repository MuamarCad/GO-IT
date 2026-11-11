package handlers

import (
	"context"
	"errors"
	"net/http"
	"strings"

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

// LimitBody ограничивает размер тела запроса (1 МиБ).
func LimitBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		next.ServeHTTP(w, r)
	})
}
