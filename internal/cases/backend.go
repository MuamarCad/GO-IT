package cases

import "go-it/internal/domain"

func Backend() domain.CareerCase {
	return domain.CareerCase{
		Slug:        "backend",
		Title:       "Первый день Backend-разработчика",
		Profession:  "backend",
		RoadmapSlug: "backend",
		Intro:       "API-эндпоинт возвращает 500 при валидном запросе. Нужно определить, на каком этапе произошла ошибка — парсинг, бизнес-логика или доступ к базе данных.",
		Questions: []domain.CaseQuestion{
			{
				ID:    "q1",
				Topic: "REST API",
				Text:  "Какой URI для получения списка пользователей корректен?",
				Options: []domain.CaseOption{
					{ID: "q1_a", Text: "GET /users", Correct: true, Explanation: "REST требует существительные во множественном числе для коллекций; GET — стандартный метод чтения."},
					{ID: "q1_b", Text: "GET /getUsers", Explanation: "Глагол в URI нарушает принцип REST: ресурс — это существительное, а не действие."},
					{ID: "q1_c", Text: "POST /users/list", Explanation: "POST предназначен для создания ресурсов, а не чтения; list — лишний сегмент."},
				},
			},
			{
				ID:    "q2",
				Topic: "HTTP методы",
				Text:  "Нужно частично обновить поле email у существующего пользователя. Какой метод?",
				Options: []domain.CaseOption{
					{ID: "q2_a", Text: "PATCH /users/42", Correct: true, Explanation: "PATCH предназначен для частичного обновления; PUT — для полной замены ресурса."},
					{ID: "q2_b", Text: "PUT /users/42", Explanation: "PUT заменяет весь ресурс; PATCH предпочтительнее для частичных изменений."},
					{ID: "q2_c", Text: "POST /users/42", Explanation: "POST используется для создания, а не обновления."},
				},
			},
			{
				ID:    "q3",
				Topic: "HTTP статусы",
				Text:  "Запрос на создание пользователя возвращает статус 200. Как лучше?",
				Options: []domain.CaseOption{
					{ID: "q3_a", Text: "201 Created — ресурс успешно создан", Correct: true, Explanation: "201 — стандартный статус успешного создания ресурса; 200 — OK для чтения/обновления."},
					{ID: "q3_b", Text: "200 OK — работает", Explanation: "200 OK — общий статус, не указывает на создание ресурса."},
					{ID: "q3_c", Text: "204 No Content", Explanation: "204 — нет тела ответа; при создании нужно вернуть Location созданного ресурса."},
				},
			},
			{
				ID:    "q4",
				Topic: "SQL vs NoSQL",
				Text:  "Проект: система учёта заказов с жёсткой схемой, транзакциями и связями. Какой тип БД?",
				Options: []domain.CaseOption{
					{ID: "q4_a", Text: "PostgreSQL — реляционная БД с ACID-транзакциями", Correct: true, Explanation: "PostgreSQL обеспечивает целостность данных, внешние ключи и транзакции — критично для заказов."},
					{ID: "q4_b", Text: "MongoDB — документоориентированная", Explanation: "NoSQL хорош для гибких схем; жёсткая структура и транзакции — сильная сторона SQL."},
					{ID: "q4_c", Text: "Redis — in-memory кэш", Explanation: "Redis — не основная БД; данные могут потеряться при перезапуске."},
				},
			},
			{
				ID:    "q5",
				Topic: "Аутентификация",
				Text:  "Как реализовать stateless-аутентификацию для мобильного API?",
				Options: []domain.CaseOption{
					{ID: "q5_a", Text: "JWT (JSON Web Token) с подписью", Correct: true, Explanation: "JWT — self-contained токен: сервер не хранит состояние; подходит для мобильных и микросервисов."},
					{ID: "q5_b", Text: "Session-файлы на диске", Explanation: "Session-файлы требуют серверного состояния — не stateless; сложно масштабировать."},
					{ID: "q5_c", Text: "Базовая HTTP-аутентификация", Explanation: "Basic Auth передаёт учётные данные при каждом запросе; небезопасно без HTTPS."},
				},
			},
			{
				ID:    "q6",
				Topic: "Миграции БД",
				Text:  "Нужно добавить столбец NOT NULL в существующую таблицу с данными. Правильный порядок?",
				Options: []domain.CaseOption{
					{ID: "q6_a", Text: "Сначала добавить column с DEFAULT, потом установить значения, потом NOT NULL", Correct: true, Explanation: "Постепенное изменение: DEFAULT заполняет существующие строки, затем NOT NULL — безопасная схема."},
					{ID: "q6_b", Text: "Сразу ALTER TABLE ADD COLUMN col NOT NULL", Explanation: "Существующие строки не имеют значения — будет ошибка."},
					{ID: "q6_c", Text: "Очистить таблицу и добавить column", Explanation: "Очистка таблицы уничтожает данные — недопустимо."},
				},
			},
			{
				ID:    "q7",
				Topic: "Middleware",
				Text:  "Нужно добавить заголовок CORS ко всем ответам. Где реализовать?",
				Options: []domain.CaseOption{
					{ID: "q7_a", Text: "Middleware на уровне роутера: Access-Control-Allow-Origin", Correct: true, Explanation: "Middleware — единое место для сквозной логики; заголовки CORS применяются ко всем маршрутам."},
					{ID: "q7_b", Text: "В каждом handler-е отдельно", Explanation: "Дублирование кода в каждом handler — нарушение DRY; middleware — для этого и создан."},
					{ID: "q7_c", Text: "На уровне Nginx", Explanation: "Nginx — опция, но middleware в коде проще тестировать и версионировать вместе с приложением."},
				},
			},
			{
				ID:    "q8",
				Topic: "CORS",
				Text:  "Frontend на localhost:3000, backend на localhost:8080. Запрос блокируется. Решение?",
				Options: []domain.CaseOption{
					{ID: "q8_a", Text: "Настроить Access-Control-Allow-Origin с конкретным origin (не * для auth)", Correct: true, Explanation: "Конкретный origin безопаснее *; для cookies/auth нельзя использовать wildcard."},
					{ID: "q8_b", Text: "Отключить CORS на сервере (без проверки origin)", Explanation: "Отключение CORS без проверки — угроза CSRF/CORS-атакам."},
					{ID: "q8_c", Text: "Добавить прокси в frontend", Explanation: "Прокси — временное решение; правильно настроить CORS или использовать один origin."},
				},
			},
			{
				ID:    "q9",
				Topic: "Обработка ошибок",
				Text:  "База данных недоступна. Что вернёт API?",
				Options: []domain.CaseOption{
					{ID: "q9_a", Text: "503 Service Unavailable + JSON-ошибка без деталей", Correct: true, Explanation: "503 — сервис временно недоступен; без деталей — безопасность: стек-трек не должен утекать."},
					{ID: "q9_b", Text: "200 OK с пустым телом", Explanation: "200 — ошибка сервера не отражена; клиент не узнает о проблеме."},
					{ID: "q9_c", Text: "500 Internal Server Error со стеком", Explanation: "Стек-трек — утечка реализации; 503 предпочтительнее при временной недоступности."},
				},
			},
			{
				ID:    "q10",
				Topic: "Тестирование",
				Text:  "Как тестировать handler, который пишет в базу данных?",
				Options: []domain.CaseOption{
					{ID: "q10_a", Text: "Mock интерфейса БД или тестовая PG в Docker", Correct: true, Explanation: "Mock изолирует тест; тестовая PG — реалистичнее, проверяет SQL-запросы."},
					{ID: "q10_b", Text: "Только ручные тесты через curl", Explanation: "Ручные тесты медленные и ненадёжные; автотесты покрывают регрессии."},
					{ID: "q10_c", Text: "Тестировать не нужно", Explanation: "Без тестов регрессии гарантированы при изменениях."},
				},
			},
			{
				ID:    "q11",
				Topic: "REST API",
				Text:  "Эндпоинт /articles возвращает 10 000 записей. Как улучшить?",
				Options: []domain.CaseOption{
					{ID: "q11_a", Text: "Пагинация: ?page=1&limit=20, LIMIT/OFFSET в запросе", Correct: true, Explanation: "Пагинация ограничивает объём данных, уменьшает нагрузку на БД и сеть."},
					{ID: "q11_b", Text: "Уменьшить limit до 10 000", Explanation: "10 000 — всё ещё много; пагинация нужна на любом масштабе."},
					{ID: "q11_c", Text: "Вернуть только последние 10 записей", Explanation: "Потеря данных: старые записи теряются для клиентов."},
				},
			},
			{
				ID:    "q12",
				Topic: "Безопасность",
				Text:  "Как защититься от SQL-инъекции при запросе WHERE name = '<user_input>'?",
				Options: []domain.CaseOption{
					{ID: "q12_a", Text: "Параметризованные запросы с $1", Correct: true, Explanation: "Параметризованные запросы отделяют данные от SQL: СУБД обрабатывает $1 как значение, не как код."},
					{ID: "q12_b", Text: "Экранировать кавычки", Explanation: "Экранирование — ненадёжно; разные СУБД, разные правила; параметризация — стандарт."},
					{ID: "q12_c", Text: "Проверять input на спецсимволы", Explanation: "Blacklist подход неполон; параметризация — white-list подход."},
				},
			},
			{
				ID:    "q13",
				Topic: "Базы данных",
				Text:  "Перевод средств: спишется у A, начислится у B. Что если второй шаг упадёт?",
				Options: []domain.CaseOption{
					{ID: "q13_a", Text: "Транзакция: BEGIN → UPDATE A → UPDATE B → COMMIT или ROLLBACK", Correct: true, Explanation: "Транзакция гарантирует ACID: либо оба шага, либо ни одного. Rollback откатит спиание."},
					{ID: "q13_b", Text: "Два отдельных запроса без транзакции", Explanation: "Без транзакции возможна потеря средств при отказе второго шага."},
					{ID: "q13_c", Text: "Записать в лог и исправить вручную", Explanation: "Ручные исправления ненадёжны и медленны; транзакции — автоматическая гарантия."},
				},
			},
			{
				ID:    "q14",
				Topic: "HTTP-сервер",
				Text:  "Сервер получает SIGTERM. Как обработать корректно?",
				Options: []domain.CaseOption{
					{ID: "q14_a", Text: "Перестать принимать новые запросы, дождаться активных, закрыть БД-пул", Correct: true, Explanation: "Graceful shutdown: активные запросы доходят до конца, ресурсы (БД-пул) корректно закрыты."},
					{ID: "q14_b", Text: "Ничего, ОС сама убьёт процесс", Explanation: "Без graceful shutdown активные запросы прервутся; БД-пул может остаться открытым."},
					{ID: "q14_c", Text: "Отправить SIGKILL себе", Explanation: "SIGKILL нельзя перехватить; graceful shutdown требует SIGTERM и обработчика."},
				},
			},
			{
				ID:    "q15",
				Topic: "REST API",
				Text:  "API v1 меняет формат ответа, нужно сохранить обратную совместимость. Решение?",
				Options: []domain.CaseOption{
					{ID: "q15_a", Text: "Версионирование: /api/v1/... и /api/v2/...", Correct: true, Explanation: "URL-версионирование — самый понятный подход; клиенты выбирают нужную версию."},
					{ID: "q15_b", Text: "Изменить v1, все обновятся", Explanation: "Массовое обновление клиентов — риск сломать интеграции."},
					{ID: "q15_c", Text: "Добавить заголовок версии", Explanation: "Header-версионирование возможно, но URL-версионирование проще и прозрачнее."},
				},
			},
			{
				ID:    "q16",
				Topic: "Кэширование",
				Text:  "API возвращает данные, которые меняются раз в час. Как оптимизировать?",
				Options: []domain.CaseOption{
					{ID: "q16_a", Text: "Заголовки Cache-Control: max-age=3600, ETag", Correct: true, Explanation: "Cache-Control + ETag: клиент кэширует на час, сервер проверяет обновление через ETag."},
					{ID: "q16_b", Text: "Отключить кэширование", Explanation: "Отключение кэширования увеличивает нагрузку на сервер."},
					{ID: "q16_c", Text: "Использовать WebSocket", Explanation: "WebSocket — для real-time; кэширование подходит для статичных данных."},
				},
			},
			{
				ID:    "q17",
				Topic: "REST API",
				Text:  "Как правильно передать фильтр по дате в REST API?",
				Options: []domain.CaseOption{
					{ID: "q17_a", Text: "GET /orders?created_from=2026-01-01&created_to=2026-11-30", Correct: true, Explanation: "Query-параметры — стандарт REST для фильтрации; именованные параметры для каждого поля."},
					{ID: "q17_b", Text: "GET /orders/2026-01-01/2026-11-30", Explanation: "Сегменты URI для фильтров нарушают семантику URI; фильтры — query, не ресурс."},
					{ID: "q17_c", Text: "POST /orders/filter", Explanation: "POST для создания; фильтрация — чтение, нужен GET."},
				},
			},
			{
				ID:    "q18",
				Topic: "Базы данных",
				Text:  "Запрос WHERE email = '...' на таблице 1 млн строк работает медленно. Что добавить?",
				Options: []domain.CaseOption{
					{ID: "q18_a", Text: "CREATE INDEX idx_users_email ON users(email)", Correct: true, Explanation: "Индекс по email ускоряет поиск: B-tree индекс заменяет full table scan."},
					{ID: "q18_b", Text: "Увеличить размер RAM", Explanation: "RAM помогает, но не решает проблему отсутствия индекса."},
					{ID: "q18_c", Text: "JOIN с другой таблицей", Explanation: "JOIN не ускорит WHERE по тому же полю."},
				},
			},
		},
	}
}
