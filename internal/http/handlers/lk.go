package handlers

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"go-it/internal/domain"
	"go-it/internal/http/render"
)

// Личные кабинеты демонстрационные и ничего не хранят: нет входа, cookie, сессий и CSRF.
// Страница кабинета определяется адресом `/lk/<роль>`. Данные условные; настоящий только
// каталог менторов. Решения по заявкам «запоминаются» в самом адресе (параметр h).

// ---------- Общее: навигация и вывод ----------

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

// lkFlash переводит код из ?done= в фиксированный текст. Произвольный текст из адреса не показываем.
func lkFlash(r *http.Request) string {
	switch r.URL.Query().Get("done") {
	case "profile":
		return "Профиль сохранён. В демо-режиме данные нигде не хранятся."
	case "password":
		return "Пароль изменён. В демо-режиме он нигде не хранится."
	}
	return ""
}

func (s *Server) lkRender(w http.ResponseWriter, role, tmpl, active string, data map[string]any, alert string, status int) {
	data["Nav"] = lkNav(role)
	data["RoleLabel"] = roleLabel(role)
	s.Render.HTML(w, tmpl, render.Page{
		Title:  "Личный кабинет · " + roleLabel(role) + " — GO-IT",
		Active: active,
		LK:     true,
		Alert:  alert,
		Data:   data,
	}, status)
}

// ---------- Решения по заявкам в адресе ----------

// decisions хранит решения по демо-заявкам: номер заявки -> "accepted" или "declined".
// Состояние лежит в параметре адреса h (например, h=1:accepted,2:declined), поэтому
// сервер ничего не помнит, а ссылка «Принять» просто ведёт на ту же страницу с новым h.
type decisions map[int]string

// demoApplicationIDs — номера демо-заявок, по которым можно принять решение.
var demoApplicationIDs = map[string]map[int]bool{
	roleMentor:  {1: true, 2: true},
	rolePartner: {1: true, 2: true, 3: true},
}

func parseDecisions(raw, role string) decisions {
	d := decisions{}
	for _, part := range strings.Split(raw, ",") {
		idStr, action, ok := strings.Cut(part, ":")
		if !ok || (action != "accepted" && action != "declined") {
			continue
		}
		if n, err := strconv.Atoi(idStr); err == nil && demoApplicationIDs[role][n] {
			d[n] = action
		}
	}
	return d
}

func (d decisions) href(path string, id int, action string) string {
	next := make(decisions, len(d)+1)
	for k, v := range d {
		next[k] = v
	}
	next[id] = action
	ids := make([]int, 0, len(next))
	for k := range next {
		ids = append(ids, k)
	}
	sort.Ints(ids)
	parts := make([]string, 0, len(ids))
	for _, k := range ids {
		parts = append(parts, strconv.Itoa(k)+":"+next[k])
	}
	return path + "?h=" + url.QueryEscape(strings.Join(parts, ","))
}

func (d decisions) pending(total int) int {
	return total - len(d)
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
// кейсам условные, но согласованы с реальным числом вопросов.
func (s *Server) studentData(ctx context.Context) (map[string]any, error) {
	name := "Алексей Иванов"

	demo := map[string]int{"frontend": 17, "backend": 14} // правильных ответов в демо
	var completed []studentCompletedCase
	done := map[string]bool{}
	for _, slug := range []string{"frontend", "backend"} {
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
	for _, c := range s.CaseSvc.List() {
		if !done[c.Slug] {
			recommended = append(recommended, studentRecommendedCase{c.Slug, c.Title, 0, len(c.Questions), directionLabel(c.Slug)})
			hint = "Рекомендуем начать кейс «" + c.Title + "»"
			break
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
	data, err := s.studentData(r.Context())
	if err != nil {
		s.internal(w)
		return
	}
	if current == "student-settings" {
		s.lkSettings(w, r, data, nil, nil, http.StatusOK)
		return
	}
	tmpl := map[string]string{
		"student-cases":        "lk_student_cases.html",
		"student-roadmap":      "lk_student_roadmap.html",
		"student-mentors":      "lk_student_mentors.html",
		"student-volunteering": "lk_student_volunteering.html",
	}[current]
	if tmpl == "" {
		tmpl = "lk_student.html"
	}
	s.lkRender(w, roleStudent, tmpl, current, data, "", http.StatusOK)
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
func (s *Server) lkSettings(w http.ResponseWriter, r *http.Request, data map[string]any, form, errs map[string]string, status int) {
	if form == nil {
		form = map[string]string{"name": data["UserName"].(string), "email": "", "phone": ""}
	}
	if errs == nil {
		errs = map[string]string{}
	}
	data["Form"] = form
	data["Errors"] = errs
	s.lkRender(w, roleStudent, "lk_settings.html", "student-settings", data, lkFlash(r), status)
}

// LKStudentSettingsPost проверяет формы «Профиль» и «Смена пароля» и показывает результат.
// Данные нигде не сохраняются (демо), поэтому сессия и CSRF-токен не нужны.
func (s *Server) LKStudentSettingsPost(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	data, err := s.studentData(r.Context())
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
			s.lkSettings(w, r, data, form, errs, http.StatusUnprocessableEntity)
			return
		}
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
			s.lkSettings(w, r, data, nil, errs, http.StatusUnprocessableEntity)
			return
		}
		http.Redirect(w, r, "/lk/student/settings?done=password", http.StatusSeeOther)
	default:
		s.lkSettings(w, r, data, nil, nil, http.StatusBadRequest)
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
	AcceptHref  string
	DeclineHref string
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

func (s *Server) lkMentorPage(w http.ResponseWriter, r *http.Request, current string) {
	dec := parseDecisions(r.URL.Query().Get("h"), roleMentor)
	apps := []mentorApplication{
		{ID: 1, StudentName: "Дмитрий П.", Contact: "dmitry@mail.ru", Date: "12 ноя", Time: "15:00", Message: "Хотел бы обсудить работу с React Hooks"},
		{ID: 2, StudentName: "Мария Л.", Contact: "@maria_tg", Date: "14 ноя", Time: "12:00", Message: "Нужна помощь с TypeScript типами"},
	}
	for i := range apps {
		apps[i].Status = dec[apps[i].ID]
		apps[i].AcceptHref = dec.href(r.URL.Path, apps[i].ID, "accepted")
		apps[i].DeclineHref = dec.href(r.URL.Path, apps[i].ID, "declined")
	}
	data := map[string]any{
		"UserName":        "Анна К.",
		"Initials":        "АК",
		"Specialization":  "Frontend",
		"NewApplications": dec.pending(len(apps)),
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
	s.lkRender(w, roleMentor, "lk_mentor.html", current, data, "", http.StatusOK)
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

// ---------- ЛК: партнёр ----------

type partnerApplication struct {
	ID            int
	VolunteerName string
	TaskTitle     string
	Skills        string
	Contact       string
	Message       string
	Status        string // "", "accepted", "declined"
	AcceptHref    string
	DeclineHref   string
}

type partnerTask struct {
	Title             string
	Skills            string
	Scale             string
	Status            string
	ApplicationsCount int
}

func (s *Server) lkPartnerPage(w http.ResponseWriter, r *http.Request, current string) {
	dec := parseDecisions(r.URL.Query().Get("h"), rolePartner)
	apps := []partnerApplication{
		{ID: 1, VolunteerName: "Елена В.", TaskTitle: "Вёрстка лендинга для акции", Skills: "HTML, CSS", Contact: "elena@example.com", Message: "Есть опыт вёрстки, могу помочь на этой неделе"},
		{ID: 2, VolunteerName: "Сергей К.", TaskTitle: "Телеграм-бот для регистрации", Skills: "Python", Contact: "@sergey_tg"},
		{ID: 3, VolunteerName: "Алина Р.", TaskTitle: "Аналитика опросов участников", Skills: "Excel, анализ данных", Contact: "alina@example.com", Message: "Работала с аналогичными данными"},
	}
	accepted := 9
	for i := range apps {
		apps[i].Status = dec[apps[i].ID]
		apps[i].AcceptHref = dec.href(r.URL.Path, apps[i].ID, "accepted")
		apps[i].DeclineHref = dec.href(r.URL.Path, apps[i].ID, "declined")
		if apps[i].Status == "accepted" {
			accepted++
		}
	}
	data := map[string]any{
		"OrganizationName":  "Фонд «Открытый код»",
		"Role":              "Координатор",
		"UserName":          "Фонд «Открытый код»",
		"Initials":          "ОК",
		"NewApplications":   dec.pending(len(apps)),
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
	s.lkRender(w, rolePartner, "lk_partner.html", current, data, "", http.StatusOK)
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
