package catalog

import "go-it/internal/domain"

func Directions() []domain.Direction {
	return []domain.Direction{
		{
			Slug:        "frontend",
			Title:       "Frontend",
			ShortTitle:  "Frontend",
			Icon:        "</>",
			Summary:     "Интерфейсы, вёрстка, доступность и состояния кнопок.",
			RoadmapSlug: "frontend",
			CaseSlug:    "frontend",
		},
		{
			Slug:        "backend",
			Title:       "Backend",
			ShortTitle:  "Go",
			Icon:        "Go",
			Summary:     "Обработка ошибок, данные и правила, которые проверяет сервер.",
			RoadmapSlug: "backend",
			CaseSlug:    "", // кейс для Backend ещё не написан
		},
		{
			Slug:        "project-manager",
			Title:       "IT Project Manager",
			ShortTitle:  "PM",
			Icon:        "PM",
			Summary:     "Неполные требования, команда с разным пониманием результата.",
			RoadmapSlug: "project-manager",
			CaseSlug:    "", // кейс для PM ещё не написан
		},
	}
}

func DirectionBySlug(slug string) (domain.Direction, bool) {
	for _, d := range Directions() {
		if d.Slug == slug {
			return d, true
		}
	}
	return domain.Direction{}, false
}

func SeedMentors() []domain.Mentor {
	return []domain.Mentor{
		{
			Slug:           "daria-backend",
			DisplayName:    "Дарья П.",
			Specialization: "Backend",
			Level:          "2 курс",
			Format:         "онлайн",
			Topics:         "Go, SQL, HTTP",
			Description:    "Помогаю начинающим разобрать первую программу на Go и понять, как сервер принимает запрос.",
		},
		{
			Slug:           "ivan-frontend",
			DisplayName:    "Иван С.",
			Specialization: "Frontend",
			Level:          "3 курс",
			Format:         "онлайн / кампус",
			Topics:         "HTML, CSS, доступность",
			Description:    "Разбираем вёрстку, состояния форм и базовую доступность на небольших примерах.",
		},
		{
			Slug:           "sofia-pm",
			DisplayName:    "София К.",
			Specialization: "IT Project Manager",
			Level:          "2 курс",
			Format:         "онлайн",
			Topics:         "требования, задачи, коммуникация",
			Description:    "Показываю, как уточнять задачу у заказчика, когда формулировка звучит слишком общо.",
		},
		{
			Slug:           "artem-qa",
			DisplayName:    "Артём Л.",
			Specialization: "Backend",
			Level:          "4 курс",
			Format:         "онлайн",
			Topics:         "проверки, ошибки, логи",
			Description:    "Демо-профиль наставника: как искать причину ошибки по сообщению сервера.",
		},
	}
}

func SeedVolunteerTasks() []domain.VolunteerTask {
	return []domain.VolunteerTask{
		{
			Slug:           "nko-landing",
			Title:          "Лендинг для социального центра",
			Organization:   "Демо-НКО",
			Category:       "website",
			Skills:         "HTML, CSS",
			Scale:          "небольшой объём",
			Description:    "Нужна простая страница с расписанием кружков и контактами. Это демонстрационная задача.",
			ExpectedResult: "Адаптивная страница с разделами «О нас», «Расписание», «Контакты».",
		},
		{
			Slug:           "school-roadmap",
			Title:          "Оформить памятку «С чего начать в IT»",
			Organization:   "Образовательный кружок",
			Category:       "content",
			Skills:         "Markdown, структурирование текста",
			Scale:          "2–3 часа",
			Description:    "Собрать короткую памятку для школьников по мотивам дорожных карт GO-IT.",
			ExpectedResult: "Один файл с этапами обучения и ссылками на бесплатные материалы.",
		},
		{
			Slug:           "library-form",
			Title:          "Форма записи на консультацию",
			Organization:   "Городская библиотека (демо)",
			Category:       "website",
			Skills:         "HTML-формы, валидация",
			Scale:          "небольшой объём",
			Description:    "Сделать понятную форму заявки: имя, контакт, тема вопроса.",
			ExpectedResult: "Форма с подписями полей и сообщениями об ошибках.",
		},
		{
			Slug:           "club-a11y",
			Title:          "Проверить контраст афиши кружка",
			Organization:   "Студенческий клуб",
			Category:       "design",
			Skills:         "контраст, читаемость",
			Scale:          "1 час",
			Description:    "Посмотреть макет объявления и предложить правки по контрасту текста и фона.",
			ExpectedResult: "Список замечаний и два конкретных исправления.",
		},
	}
}
