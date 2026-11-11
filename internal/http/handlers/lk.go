package handlers

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"

	"go-it/internal/domain"
	"go-it/internal/http/render"
	"go-it/internal/session"
)

// Личные кабинеты демонстрационные: данные в них условные (кроме каталога менторов и последнего
// реально пройденного кейса ученика), решения по заявкам живут только в сессии.

// ---------- Общее: навигация, доступ, вывод ----------

type lkNavItem struct {
	Href  string
	Label string
	Icon  string // имя иконки из partials/lk_sidebar.html
	Key   string // значение Page.Active для подсветки
}

func lkNav(role string) []lkNavItem {
	switch role {
	case roleMentor:
		return []lkNavItem{
			{"/lk/mentor", "Обзор", "home", "mentor-overview"},
			{"/lk/mentor/applications", "Заявки", "inbox", "mentor-applications"},
			{"/lk/mentor/schedule", "Расписание", "calendar", "mentor-schedule"},
			{"/lk/mentor/students", "Ученики", "users", "mentor-students"},
			{"/lk/mentor/profile", "Профиль", "user", "mentor-profile"},
		}
	case rolePartner:
		return []lkNavItem{
			{"/lk/partner", "Обзор", "home", "partner-overview"},
			{"/lk/partner/tasks", "Задачи", "check", "partner-tasks"},
			{"/lk/partner/applications", "Отклики", "inbox", "partner-applications"},
			{"/lk/partner/analytics", "Аналитика", "chart", "partner-analytics"},
			{"/lk/partner/settings", "Настройки", "gear", "partner-settings"},
		}
	default:
		return []lkNavItem{
			{"/lk/student", "Обзор", "home", "student-overview"},
			{"/lk/student/cases", "Мои кейсы", "file", "student-cases"},
			{"/lk/student/roadmap", "Roadmap", "map", "student-roadmap"},
			{"/lk/student/mentors", "Менторы", "users", "student-mentors"},
			{"/lk/student/volunteering", "Волонтёрство", "heart", "student-volunteering"},
			{"/lk/student/settings", "Настройки", "gear", "student-settings"},
		}
	}
}

func roleLabel(role string) string {
	switch role {
	case roleMentor:
		return "Ментор"
	case rolePartner:
		return "Партнёр"
	default:
		return "Ученик"
	}
}

// lkGuard пропускает только вошедшего пользователя нужной роли.
// Чужую роль перенаправляем в её собственный кабинет, незнакомую или анонимную — на форму входа.
func (s *Server) lkGuard(w http.ResponseWriter, r *http.Request, role string) (string, session.Data, bool) {
	id, sess := s.Sessions.Load(r)
	switch {
	case !sess.LoggedIn || !validRole(sess.Role):
		http.Redirect(w, r, "/auth", http.StatusSeeOther)
		return "", sess, false
	case sess.Role != role:
		http.Redirect(w, r, lkHome(sess.Role), http.StatusSeeOther)
		return "", sess, false
	}
	return id, sess, true
}

// lkFlash переводит код из ?done= в фиксированный текст. Произвольный текст из адреса не показываем.
func lkFlash(r *http.Request) string {
	switch r.URL.Query().Get("done") {
	case "profile":
		return "Профиль сохранён."
	case "password":
		return "Пароль изменён. В демо-режиме он нигде не хранится."
	case "app-accepted":
		return "Заявка принята."
	case "app-declined":
		return "Заявка отклонена."
	case "resp-accepted":
		return "Отклик принят."
	case "resp-declined":
		return "Отклик отклонён."
	}
	return ""
}

func (s *Server) lkRender(w http.ResponseWriter, id string, sess session.Data, role, tmpl, active string, data map[string]any, alert string, status int) {
	data["Nav"] = lkNav(role)
	data["RoleLabel"] = roleLabel(role)
	data["LogoutAction"] = lkHome(role) + "/logout"
	s.Sessions.Save(w, id, sess)
	s.Render.HTML(w, tmpl, render.Page{
		Title:  "Личный кабинет · " + roleLabel(role) + " — GO-IT",
		Active: active,
		LK:     true,
		CSRF:   sess.CSRF,
		Alert:  alert,
		Data:   data,
	}, status)
}

// ---------- Радар навыков (SVG строится на сервере) ----------

type radarAxis struct {
	X, Y, LX, LY string
	Anchor       string
	Label        string
	Value        int
}

type radarChart struct {
	Rings   []string
	Axes    []radarAxis
	Poly    string
	Summary string
}

const (
	radarCX = 200.0
	radarCY = 150.0
	radarR  = 120.0
)

func radarPoint(i int, radius float64) (float64, float64) {
	angle := (-90 + 60*float64(i)) * math.Pi / 180
	return radarCX + radius*math.Cos(angle), radarCY + radius*math.Sin(angle)
}

func buildRadar(labels []string, values []int) radarChart {
	n := len(labels)
	var rc radarChart
	for _, k := range []float64{1.0 / 3, 2.0 / 3, 1} {
		pts := make([]string, 0, n)
		for i := 0; i < n; i++ {
			x, y := radarPoint(i, radarR*k)
			pts = append(pts, fmt.Sprintf("%.1f,%.1f", x, y))
		}
		rc.Rings = append(rc.Rings, strings.Join(pts, " "))
	}
	poly := make([]string, 0, n)
	parts := make([]string, 0, n)
	for i := 0; i < n; i++ {
		v := values[i]
		if v < 0 {
			v = 0
		}
		if v > 100 {
			v = 100
		}
		x, y := radarPoint(i, radarR)
		dx, dy := radarPoint(i, radarR*float64(v)/100)
		poly = append(poly, fmt.Sprintf("%.1f,%.1f", dx, dy))
		ax := radarAxis{X: fmt.Sprintf("%.1f", x), Y: fmt.Sprintf("%.1f", y), Label: labels[i], Value: v}
		switch {
		case i == 0:
			ax.Anchor, ax.LX, ax.LY = "middle", "200", fmt.Sprintf("%.1f", y-10)
		case float64(i) == float64(n)/2:
			ax.Anchor, ax.LX, ax.LY = "middle", "200", fmt.Sprintf("%.1f", y+20)
		case x > radarCX:
			ax.Anchor, ax.LX, ax.LY = "start", fmt.Sprintf("%.1f", x+10), fmt.Sprintf("%.1f", y+4)
		default:
			ax.Anchor, ax.LX, ax.LY = "end", fmt.Sprintf("%.1f", x-10), fmt.Sprintf("%.1f", y+4)
		}
		rc.Axes = append(rc.Axes, ax)
		parts = append(parts, fmt.Sprintf("%s %d%%", labels[i], v))
	}
	rc.Poly = strings.Join(poly, " ")
	rc.Summary = strings.Join(parts, ", ")
	return rc
}

// ---------- ЛК: ученик ----------

type studentRecommendedCase struct {
	Slug        string
	Title       string
	CurrentStep int // 0, если кейс ещё не начат
	TotalSteps  int
	Topic       string
}

type studentMentorRecommend struct {
	Slug           string
	Name           string
	Specialization string
	Level          string
	Format         string
}

type studentCompletedCase struct {
	Direction string
	Title     string
	Correct   int
	Total     int
	Percent   int
	Passed    bool
}

func directionLabel(slug string) string {
	switch slug {
	case "frontend":
		return "Frontend"
	case "backend":
		return "Backend"
	case "project-manager":
		return "Project Manager"
	case "product-manager":
		return "Product Manager"
	}
	return slug
}

func completedRow(c domain.CareerCase, correct int) studentCompletedCase {
	total := len(c.Questions)
	pct := 0
	if total > 0 {
		pct = correct * 100 / total
	}
	return studentCompletedCase{Direction: directionLabel(c.Slug), Title: c.Title, Correct: correct, Total: total, Percent: pct, Passed: pct >= 60}
}

// studentData собирает данные обзора ученика. Каталог менторов настоящий; результаты по двум
// кейсам условные, но согласованы с реальным числом вопросов; если ученик в этой сессии
// прошёл кейс по-настоящему, его результат заменяет условный.
func (s *Server) studentData(ctx context.Context, sess session.Data) (map[string]any, error) {
	name := sess.Profile.Name
	if name == "" {
		name = "Алексей Иванов"
	}

	demo := map[string]int{"frontend": 17, "backend": 14} // правильных ответов в демо
	var completed []studentCompletedCase
	done := map[string]bool{}
	var real *studentCompletedCase
	if sess.Attempt.Completed {
		if res, err := s.CaseSvc.Result(sess.Attempt); err == nil {
			row := completedRow(res.Case, res.Score)
			real = &row
			done[res.Case.Slug] = true
		}
	}
	if real != nil {
		completed = append(completed, *real)
	}
	for _, slug := range []string{"frontend", "backend"} {
		if done[slug] {
			continue
		}
		if c, err := s.CaseSvc.Get(slug); err == nil {
			completed = append(completed, completedRow(c, demo[slug]))
			done[slug] = true
		}
	}

	points, skillFront, skillBack := 0, 20, 20
	for _, row := range completed {
		points += row.Correct * 5
		switch row.Direction {
		case "Frontend":
			skillFront = row.Percent
		case "Backend":
			skillBack = row.Percent
		}
	}

	var recommended []studentRecommendedCase
	hint := ""
	attempt := sess.Attempt
	if attempt.Slug != "" && !attempt.Completed {
		if c, err := s.CaseSvc.Get(attempt.Slug); err == nil {
			recommended = append(recommended, studentRecommendedCase{c.Slug, c.Title, attempt.Index + 1, len(c.Questions), directionLabel(c.Slug)})
			hint = "Рекомендуем продолжить кейс «" + c.Title + "»"
		}
	}
	if len(recommended) == 0 {
		for _, c := range s.CaseSvc.List() {
			if !done[c.Slug] {
				recommended = append(recommended, studentRecommendedCase{c.Slug, c.Title, 0, len(c.Questions), directionLabel(c.Slug)})
				hint = "Рекомендуем начать кейс «" + c.Title + "»"
				break
			}
		}
	}

	mentors, err := s.Catalog.ListMentors(ctx)
	if err != nil {
		return nil, err
	}
	var recMentors []studentMentorRecommend
	for i, m := range mentors {
		if i == 2 {
			break
		}
		recMentors = append(recMentors, studentMentorRecommend{m.Slug, m.DisplayName, m.Specialization, m.Level, m.Format})
	}
	nextMentor := "Ментор"
	if len(recMentors) > 0 {
		nextMentor = recMentors[0].Name
	}

	return map[string]any{
		"UserName":         name,
		"Initials":         render.Initials(name),
		"CasesCompleted":   len(completed),
		"Points":           points,
		"RoadmapsOpened":   1,
		"Consultations":    3,
		"ActiveRoadmap":    "Frontend",
		"CurrentLevel":     3,
		"TotalLevels":      5,
		"ProgressPercent":  60,
		"NextCaseHint":     hint,
		"RecommendedCases": recommended,
		"AvailableMentors": recMentors,
		"CompletedCases":   completed,
		"NextConsultation": map[string]any{"MentorName": nextMentor, "Date": "12 ноя", "Time": "15:00", "Topic": "CSS Grid и Flexbox"},
		"Radar": buildRadar(
			[]string{"Frontend", "Backend", "PM", "Soft Skills", "Tools", "Design"},
			[]int{skillFront, skillBack, 40, 70, 60, 35},
		),
	}, nil
}

func (s *Server) lkStudentPage(w http.ResponseWriter, r *http.Request, current string) {
	id, sess, ok := s.lkGuard(w, r, roleStudent)
	if !ok {
		return
	}
	data, err := s.studentData(r.Context(), sess)
	if err != nil {
		s.internal(w)
		return
	}
	tmpl := map[string]string{
		"student-cases":        "lk_student_cases.html",
		"student-roadmap":      "lk_student_roadmap.html",
		"student-mentors":      "lk_student_mentors.html",
		"student-volunteering": "lk_student_volunteering.html",
		"student-settings":     "lk_settings.html",
	}[current]
	if tmpl == "" {
		tmpl = "lk_student.html"
	}
	if current == "student-settings" {
		s.lkSettings(w, r, id, sess, data, nil, nil, http.StatusOK)
		return
	}
	s.lkRender(w, id, sess, roleStudent, tmpl, current, data, "", http.StatusOK)
}

func (s *Server) LKStudent(w http.ResponseWriter, r *http.Request) {
	s.lkStudentPage(w, r, "student-overview")
}
func (s *Server) LKStudentCases(w http.ResponseWriter, r *http.Request) {
	s.lkStudentPage(w, r, "student-cases")
}
func (s *Server) LKStudentRoadmap(w http.ResponseWriter, r *http.Request) {
	s.lkStudentPage(w, r, "student-roadmap")
}
func (s *Server) LKStudentMentors(w http.ResponseWriter, r *http.Request) {
	s.lkStudentPage(w, r, "student-mentors")
}
func (s *Server) LKStudentVolunteering(w http.ResponseWriter, r *http.Request) {
	s.lkStudentPage(w, r, "student-volunteering")
}
func (s *Server) LKStudentSettings(w http.ResponseWriter, r *http.Request) {
	s.lkStudentPage(w, r, "student-settings")
}

// lkSettings показывает «Настройки». form и errs заполнены при повторном показе после ошибки.
func (s *Server) lkSettings(w http.ResponseWriter, r *http.Request, id string, sess session.Data, data map[string]any, form, errs map[string]string, status int) {
	if form == nil {
		form = map[string]string{"name": data["UserName"].(string), "email": sess.Profile.Email, "phone": sess.Profile.Phone}
	}
	if errs == nil {
		errs = map[string]string{}
	}
	data["Form"] = form
	data["Errors"] = errs
	alert := lkFlash(r)
	s.lkRender(w, id, sess, roleStudent, "lk_settings.html", "student-settings", data, alert, status)
}

// LKStudentSettingsPost принимает формы «Профиль» и «Смена пароля». Профиль сохраняется
// в сессии; пароль в демо нигде не хранится, поэтому проверяется только новый пароль.
func (s *Server) LKStudentSettingsPost(w http.ResponseWriter, r *http.Request) {
	id, sess, ok := s.lkGuard(w, r, roleStudent)
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		s.fail(w, r, http.StatusBadRequest, "Некорректный запрос", "Не удалось обработать форму.")
		return
	}
	if !s.Sessions.CheckCSRF(r, sess) {
		s.fail(w, r, http.StatusForbidden, "Сессия устарела", "Обновите страницу и отправьте форму ещё раз.")
		return
	}
	data, err := s.studentData(r.Context(), sess)
	if err != nil {
		s.internal(w)
		return
	}

	errs := map[string]string{}
	switch r.FormValue("action") {
	case "profile":
		name := strings.TrimSpace(r.FormValue("name"))
		email := strings.TrimSpace(r.FormValue("email"))
		phone := strings.TrimSpace(r.FormValue("phone"))
		if n := utf8.RuneCountInString(name); n < 2 || n > 100 {
			errs["name"] = "Введите ФИО: от 2 до 100 символов."
		}
		if email != "" {
			if err := validateEmail(email); err != nil {
				errs["email"] = "Проверьте email: " + err.Error() + "."
			}
		}
		if phone != "" {
			if err := validatePhone(phone); err != nil {
				errs["phone"] = "Проверьте телефон: " + err.Error() + "."
			}
		}
		if len(errs) > 0 {
			form := map[string]string{"name": name, "email": email, "phone": phone}
			s.lkSettings(w, r, id, sess, data, form, errs, http.StatusUnprocessableEntity)
			return
		}
		sess.Profile = session.Profile{Name: name, Email: email, Phone: phone}
		s.Sessions.Save(w, id, sess)
		http.Redirect(w, r, "/lk/student/settings?done=profile", http.StatusSeeOther)
	case "password":
		pw, confirm := r.FormValue("password_new"), r.FormValue("password_confirm")
		if err := validatePassword(pw); err != nil {
			errs["password_new"] = "Новый пароль: " + err.Error() + "."
		}
		if pw != confirm {
			errs["password_confirm"] = "Пароли не совпадают."
		}
		if len(errs) > 0 {
			s.lkSettings(w, r, id, sess, data, nil, errs, http.StatusUnprocessableEntity)
			return
		}
		s.Sessions.Save(w, id, sess)
		http.Redirect(w, r, "/lk/student/settings?done=password", http.StatusSeeOther)
	default:
		s.fail(w, r, http.StatusBadRequest, "Некорректный запрос", "Неизвестное действие формы.")
	}
}

// ---------- ЛК: ментор ----------

type mentorApplication struct {
	ID          int
	StudentName string
	Contact     string
	Date        string
	Time        string
	Message     string
	Status      string // "", "accepted", "declined"
}

type mentorSession struct {
	StudentName string
	Date        string
	Time        string
	Topic       string
}

type mentorStudent struct {
	Initials       string
	Name           string
	Direction      string
	RoadmapsOpened int
	Progress       int
}

func handledStatus(sess session.Data, role string, id int) string {
	return sess.Handled[role+":"+strconv.Itoa(id)]
}

func countPending(apps []string) int {
	n := 0
	for _, st := range apps {
		if st == "" {
			n++
		}
	}
	return n
}

func (s *Server) lkMentorPage(w http.ResponseWriter, r *http.Request, current string) {
	id, sess, ok := s.lkGuard(w, r, roleMentor)
	if !ok {
		return
	}
	apps := []mentorApplication{
		{ID: 1, StudentName: "Дмитрий П.", Contact: "dmitry@mail.ru", Date: "12 ноя", Time: "15:00", Message: "Хотел бы обсудить работу с React Hooks"},
		{ID: 2, StudentName: "Мария Л.", Contact: "@maria_tg", Date: "14 ноя", Time: "12:00", Message: "Нужна помощь с TypeScript типами"},
	}
	statuses := make([]string, len(apps))
	for i := range apps {
		apps[i].Status = handledStatus(sess, roleMentor, apps[i].ID)
		statuses[i] = apps[i].Status
	}
	data := map[string]any{
		"UserName":        "Анна К.",
		"Initials":        "АК",
		"Specialization":  "Frontend",
		"NewApplications": countPending(statuses),
		"TotalStudents":   5,
		"TotalSessions":   18,
		"Rating":          "4.9",
		"Applications":    apps,
		"UpcomingSessions": []mentorSession{
			{StudentName: "Иван С.", Date: "13 ноя", Time: "18:00", Topic: "CSS Grid и Flexbox"},
			{StudentName: "Ольга Н.", Date: "14 ноя", Time: "16:00", Topic: "Деплой фронтенда"},
		},
		"ActiveStudents": []mentorStudent{
			{Initials: "ДС", Name: "Дмитрий С.", Direction: "Frontend", RoadmapsOpened: 1, Progress: 40},
			{Initials: "ОН", Name: "Ольга Н.", Direction: "Frontend", RoadmapsOpened: 2, Progress: 65},
			{Initials: "ПМ", Name: "Пётр М.", Direction: "Backend", RoadmapsOpened: 0, Progress: 10},
		},
	}
	s.lkRender(w, id, sess, roleMentor, "lk_mentor.html", current, data, lkFlash(r), http.StatusOK)
}

func (s *Server) LKMentor(w http.ResponseWriter, r *http.Request) {
	s.lkMentorPage(w, r, "mentor-overview")
}
func (s *Server) LKMentorApplications(w http.ResponseWriter, r *http.Request) {
	s.lkMentorPage(w, r, "mentor-applications")
}
func (s *Server) LKMentorSchedule(w http.ResponseWriter, r *http.Request) {
	s.lkMentorPage(w, r, "mentor-schedule")
}
func (s *Server) LKMentorStudents(w http.ResponseWriter, r *http.Request) {
	s.lkMentorPage(w, r, "mentor-students")
}
func (s *Server) LKMentorProfile(w http.ResponseWriter, r *http.Request) {
	s.lkMentorPage(w, r, "mentor-profile")
}
func (s *Server) LKMentorAccept(w http.ResponseWriter, r *http.Request) {
	s.lkDecide(w, r, roleMentor, "accepted")
}
func (s *Server) LKMentorDecline(w http.ResponseWriter, r *http.Request) {
	s.lkDecide(w, r, roleMentor, "declined")
}

// ---------- ЛК: партнёр ----------

type partnerApplication struct {
	ID            int
	VolunteerName string
	TaskTitle     string
	Skills        string
	Contact       string
	Message       string
	Status        string // "", "accepted", "declined"
}

type partnerTask struct {
	Title             string
	Skills            string
	Scale             string
	Status            string
	ApplicationsCount int
}

func (s *Server) lkPartnerPage(w http.ResponseWriter, r *http.Request, current string) {
	id, sess, ok := s.lkGuard(w, r, rolePartner)
	if !ok {
		return
	}
	apps := []partnerApplication{
		{ID: 1, VolunteerName: "Елена В.", TaskTitle: "Вёрстка лендинга для акции", Skills: "HTML, CSS", Contact: "elena@example.com", Message: "Есть опыт вёрстки, могу помочь на этой неделе"},
		{ID: 2, VolunteerName: "Сергей К.", TaskTitle: "Телеграм-бот для регистрации", Skills: "Python", Contact: "@sergey_tg"},
		{ID: 3, VolunteerName: "Алина Р.", TaskTitle: "Аналитика опросов участников", Skills: "Excel, анализ данных", Contact: "alina@example.com", Message: "Работала с аналогичными данными"},
	}
	statuses := make([]string, len(apps))
	accepted := 9
	for i := range apps {
		apps[i].Status = handledStatus(sess, rolePartner, apps[i].ID)
		statuses[i] = apps[i].Status
		if apps[i].Status == "accepted" {
			accepted++
		}
	}
	data := map[string]any{
		"OrganizationName":  "Фонд «Открытый код»",
		"Role":              "Координатор",
		"UserName":          "Фонд «Открытый код»",
		"Initials":          "ОК",
		"NewApplications":   countPending(statuses),
		"ActiveTasks":       4,
		"TotalApplications": 17,
		"Accepted":          accepted,
		"PublishedTasks":    6,
		"CompletedProjects": 3,
		"TotalVolunteers":   24,
		"Applications":      apps,
		"Tasks": []partnerTask{
			{Title: "Вёрстка лендинга для акции", Skills: "HTML, CSS", Scale: "Онлайн", Status: "Активна", ApplicationsCount: 5},
			{Title: "Телеграм-бот для регистрации", Skills: "Python, Aiogram", Scale: "Онлайн", Status: "Активна", ApplicationsCount: 3},
			{Title: "Аналитика опросов участников", Skills: "Excel, SQL", Scale: "Удалённо", Status: "Активна", ApplicationsCount: 4},
			{Title: "Презентация для вебинара", Skills: "PowerPoint, дизайн", Scale: "Удалённо", Status: "Активна", ApplicationsCount: 2},
		},
	}
	s.lkRender(w, id, sess, rolePartner, "lk_partner.html", current, data, lkFlash(r), http.StatusOK)
}

func (s *Server) LKPartner(w http.ResponseWriter, r *http.Request) {
	s.lkPartnerPage(w, r, "partner-overview")
}
func (s *Server) LKPartnerTasks(w http.ResponseWriter, r *http.Request) {
	s.lkPartnerPage(w, r, "partner-tasks")
}
func (s *Server) LKPartnerApplications(w http.ResponseWriter, r *http.Request) {
	s.lkPartnerPage(w, r, "partner-applications")
}
func (s *Server) LKPartnerAnalytics(w http.ResponseWriter, r *http.Request) {
	s.lkPartnerPage(w, r, "partner-analytics")
}
func (s *Server) LKPartnerSettings(w http.ResponseWriter, r *http.Request) {
	s.lkPartnerPage(w, r, "partner-settings")
}
func (s *Server) LKPartnerAccept(w http.ResponseWriter, r *http.Request) {
	s.lkDecide(w, r, rolePartner, "accepted")
}
func (s *Server) LKPartnerDecline(w http.ResponseWriter, r *http.Request) {
	s.lkDecide(w, r, rolePartner, "declined")
}

// demoApplicationIDs — номера демо-заявок, по которым можно принять решение.
var demoApplicationIDs = map[string]map[int]bool{
	roleMentor:  {1: true, 2: true},
	rolePartner: {1: true, 2: true, 3: true},
}

// lkDecide запоминает решение по демо-заявке в сессии. Реальных заявок за этим нет.
func (s *Server) lkDecide(w http.ResponseWriter, r *http.Request, role, action string) {
	id, sess, ok := s.lkGuard(w, r, role)
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		s.fail(w, r, http.StatusBadRequest, "Некорректный запрос", "Не удалось обработать форму.")
		return
	}
	if !s.Sessions.CheckCSRF(r, sess) {
		s.fail(w, r, http.StatusForbidden, "Сессия устарела", "Обновите страницу и повторите действие.")
		return
	}
	n, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil || !demoApplicationIDs[role][n] {
		s.NotFound(w, r)
		return
	}
	if sess.Handled == nil {
		sess.Handled = map[string]string{}
	}
	sess.Handled[role+":"+strconv.Itoa(n)] = action
	s.Sessions.Save(w, id, sess)

	prefix, back := "app-", "/lk/mentor/applications"
	if role == rolePartner {
		prefix, back = "resp-", "/lk/partner/applications"
	}
	http.Redirect(w, r, back+"?done="+url.QueryEscape(prefix+action), http.StatusSeeOther)
}
