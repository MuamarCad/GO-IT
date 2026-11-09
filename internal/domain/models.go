package domain

import "time"

type Direction struct {
	Slug        string
	Title       string
	ShortTitle  string
	Icon        string
	Summary     string
	RoadmapSlug string
	CaseSlug    string
}

type CaseOption struct {
	ID          string `json:"id"`
	Text        string `json:"text"`
	Correct     bool   `json:"correct"`
	Explanation string `json:"explanation"`
}

type CaseQuestion struct {
	ID      string       `json:"id"`
	Topic   string       `json:"topic"`
	Text    string       `json:"text"`
	Options []CaseOption `json:"options"`
}

type CareerCase struct {
	Slug        string         `json:"slug"`
	Title       string         `json:"title"`
	Profession  string         `json:"profession"`
	Intro       string         `json:"intro"`
	RoadmapSlug string         `json:"roadmapSlug"`
	Questions   []CaseQuestion `json:"questions"`
}

type AttemptState struct {
	Slug      string            `json:"slug"`
	Index     int               `json:"index"`
	Answers   map[string]string `json:"answers"`
	Completed bool              `json:"completed"`
}

type QuestionView struct {
	Case      CareerCase
	Question  CaseQuestion
	Step      int
	Total     int
	Error     string
	Completed bool
}

type AnswerOutcome struct {
	Next     *QuestionView
	Finished bool
}

type OptionResult struct {
	Question    CaseQuestion
	Chosen      CaseOption
	Correct     CaseOption
	IsCorrect   bool
	Explanation string
}

// TopicScore — результат по одной теме кейса (для разбивки на странице результата).
type TopicScore struct {
	Topic string
	Score int
	Max   int
}

type CaseResult struct {
	Case       CareerCase
	Score      int
	MaxScore   int
	Topics     []TopicScore
	Details    []OptionResult
	Disclaimer string
}

type Mentor struct {
	Slug           string
	DisplayName    string
	Specialization string
	Level          string
	Format         string
	Topics         string
	Description    string
}

type VolunteerTask struct {
	Slug           string
	Title          string
	Organization   string
	Category       string
	Skills         string
	Scale          string
	Description    string
	ExpectedResult string
}

type ApplicationKind string

const (
	KindMentor    ApplicationKind = "mentor"
	KindVolunteer ApplicationKind = "volunteer"
)

type Application struct {
	ID        int64
	Kind      ApplicationKind
	TargetID  string
	Name      string
	Contact   string
	Message   string
	Status    string
	CreatedAt time.Time
}

type Roadmap struct {
	Slug      string
	Title     string
	Summary   string
	HTML      string
	IntroHTML string   // заголовок и вводный текст до первого <h2>
	BodyHTML  string   // остальные разделы
	Diagram   *Diagram // nil, если схемы нет
}

// Узлы схемы.
const (
	NodeCore     = "core"     // основа, без неё дальше трудно
	NodeAlso     = "also"     // вариант на выбор
	NodeOptional = "optional" // по желанию
)

// Diagram — визуальная схема маршрута: последовательные этапы, в каждом несколько тем.
type Diagram struct {
	Stages []DiagramStage
}

type DiagramStage struct {
	Title  string
	Note   string
	Choose bool // из узлов этапа достаточно выбрать один
	Nodes  []DiagramNode
}

// DiagramNode — тема на схеме. Если задан Summary, по клику открывается боковая панель
// с кратким описанием и ссылками на материалы. Если задан Href, узел просто ведёт по ссылке.
type DiagramNode struct {
	ID      string // стабильный идентификатор, используется в адресе #t-<id>
	Title   string
	Kind    string
	Href    string
	Summary string
	Links   []Link
}

// Link — ссылка на внешний материал по теме.
type Link struct {
	Title string
	URL   string
}

type Article struct {
	Slug        string
	Title       string
	Summary     string
	PublishedAt string
	HTML        string
}
