// Package web provides the HTTP layer: router, handlers and request lifecycle.
package web

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"

	"go-it/internal/http/handlers"
	"go-it/internal/http/middleware"
)

// NewRouter creates the Chi HTTP router with the full middleware chain.
// Middleware order: RequestID → RealIP → Recoverer → LimitBody → RequestLogger
func NewRouter(log *slog.Logger, h *handlers.Server, staticDir string) http.Handler {
	r := chi.NewRouter()
	r.Use(chiMiddleware.RequestID)
	r.Use(chiMiddleware.RealIP)
	r.Use(chiMiddleware.Recoverer)
	r.Use(handlers.LimitBody)
	r.Use(middleware.RequestLogger(log))

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

	/* Авторизация и ЛК */
	r.Get("/auth", h.Login)
	r.Post("/auth", h.LoginPost)
	r.Get("/auth/{provider}", h.LoginProvider)
	r.Post("/auth/{provider}", h.LoginProvider)

	/* Student LK */
	r.Get("/lk/student", h.LKStudent)
	r.Get("/lk/student/cases", h.LKStudentCases)
	r.Get("/lk/student/roadmap", h.LKStudentRoadmap)
	r.Get("/lk/student/mentors", h.LKStudentMentors)
	r.Get("/lk/student/volunteering", h.LKStudentVolunteering)
	r.Get("/lk/student/settings", h.LKStudentSettings)
	r.Post("/lk/student/settings", h.LKStudentSettingsPost)

	/* Mentor LK */
	r.Get("/lk/mentor", h.LKMentor)
	r.Get("/lk/mentor/applications", h.LKMentorApplications)
	r.Get("/lk/mentor/schedule", h.LKMentorSchedule)
	r.Get("/lk/mentor/students", h.LKMentorStudents)
	r.Get("/lk/mentor/profile", h.LKMentorProfile)

	/* Partner LK */
	r.Get("/lk/partner", h.LKPartner)
	r.Get("/lk/partner/tasks", h.LKPartnerTasks)
	r.Get("/lk/partner/applications", h.LKPartnerApplications)
	r.Get("/lk/partner/analytics", h.LKPartnerAnalytics)
	r.Get("/lk/partner/settings", h.LKPartnerSettings)

	r.NotFound(h.NotFound)
	return r
}
