package web

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"go-it/internal/http/handlers"
)

func NewRouter(h *handlers.Server, staticDir string) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(handlers.LimitBody)

	r.Get("/health/live", h.Live)
	r.Get("/health/ready", h.Ready)

	fileServer := http.FileServer(http.Dir(staticDir))
	r.Handle("/static/*", http.StripPrefix("/static/", fileServer))

	r.Get("/", h.Home)
	r.Get("/about", h.About)
	r.Get("/directions", h.Directions)
	r.Get("/directions/{slug}", h.Direction)
	r.Get("/cases", h.Cases)
	r.Get("/cases/{slug}", h.CaseStart)
	r.Post("/cases/{slug}/answer", h.CaseAnswer)
	r.Get("/cases/{slug}/result", h.CaseResult)
	r.Get("/mentors", h.Mentors)
	r.Get("/mentors/{slug}", h.Mentor)
	r.Post("/mentors/{slug}/applications", h.MentorApply)
	r.Get("/volunteering", h.Volunteering)
	r.Get("/volunteering/{slug}", h.VolunteerTask)
	r.Post("/volunteering/{slug}/applications", h.VolunteerApply)
	r.Get("/roadmaps", h.Roadmaps)
	r.Get("/roadmaps/{slug}", h.Roadmap)
	r.Get("/blog", h.Blog)
	r.Get("/blog/{slug}", h.Article)
	r.NotFound(h.NotFound)
	return r
}
