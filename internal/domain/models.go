// Package domain defines the core business types for the GO-IT educational platform.
package domain

import "time"

// Direction represents an IT learning track (e.g. Frontend, Backend, PM).
// It holds the slug, title, icon, summary, and references to the
// associated roadmap and career case.
type Direction struct {
	Slug        string
	Title       string
	ShortTitle  string
	Icon        string
	Summary     string
	RoadmapSlug string
	CaseSlug    string
}

// CaseOption is a single answer option for a career-case question.
type CaseOption struct {
	ID          string `json:"id"`
	Text        string `json:"text"`
	Correct     bool   `json:"correct"`
	Explanation string `json:"explanation"`
}

// CaseQuestion represents one question within a career case, with its
// topic and a set of answer options.
type CaseQuestion struct {
	ID      string       `json:"id"`
	Topic   string       `json:"topic"`
	Text    string       `json:"text"`
	Options []CaseOption `json:"options"`
}

// CareerCase is an interactive case study that guides the user through
// a series of topic-based questions. Each case belongs to a profession
// and links to a roadmap and a set of questions.
type CareerCase struct {
	Slug        string         `json:"slug"`
	Title       string         `json:"title"`
	Profession  string         `json:"profession"`
	Intro       string         `json:"intro"`
	RoadmapSlug string         `json:"roadmapSlug"`
	Questions   []CaseQuestion `json:"questions"`
}

// AttemptState tracks the user's progress through a case in the current
// session — current question index, accumulated answers, and completion flag.
type AttemptState struct {
	Slug      string            `json:"slug"`
	Index     int               `json:"index"`
	Answers   map[string]string `json:"answers"`
	Completed bool              `json:"completed"`
}

// QuestionView holds the current question together with the case context
// and metadata (step number, total questions, any error message, and
// completion status) needed for rendering the question UI.
type QuestionView struct {
	Case      CareerCase
	Question  CaseQuestion
	Step      int
	Total     int
	Error     string
	Completed bool
}

// AnswerOutcome represents the result of submitting an answer — whether
// there is a next question to display or the case is finished.
type AnswerOutcome struct {
	Next     *QuestionView
	Finished bool
}

// OptionResult holds the detailed result for a single answered option,
// including whether the answer was correct and the explanation text.
type OptionResult struct {
	Question    CaseQuestion
	Chosen      CaseOption
	Correct     CaseOption
	IsCorrect   bool
	Explanation string
}

// TopicScore holds the score breakdown for one topic within a case result.
type TopicScore struct {
	Topic string
	Score int
	Max   int
}

// CaseResult aggregates the final result of a case attempt, including
// total score, per-topic breakdowns, and detailed per-question outcomes.
type CaseResult struct {
	Case       CareerCase
	Score      int
	MaxScore   int
	Topics     []TopicScore
	Details    []OptionResult
	Disclaimer string
}

// Mentor describes a mentor profile on the platform.
type Mentor struct {
	Slug           string
	DisplayName    string
	Specialization string
	Level          string
	Format         string
	Topics         string
	Description    string
}

// VolunteerTask describes a volunteer opportunity available on the platform.
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

// ApplicationKind distinguishes between mentorship and volunteer applications.
type ApplicationKind string

const (
	// KindMentor identifies a mentorship application.
	KindMentor ApplicationKind = "mentor"
	// KindVolunteer identifies a volunteer application.
	KindVolunteer ApplicationKind = "volunteer"
)

// Application represents a user submission for either mentorship or
// volunteering, including its status and contact details.
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

// Roadmap defines a structured learning path with an optional diagram
// (visual node graph) and HTML content for introduction and body sections.
type Roadmap struct {
	Slug      string
	Title     string
	Summary   string
	HTML      string
	IntroHTML string   // heading and introductory text before the first <h2>
	BodyHTML  string   // remaining content sections
	Diagram   *Diagram // nil when no diagram is defined
}

const (
	// NodeCore is a mandatory topic — without it progressing further is difficult.
	NodeCore = "core"
	// NodeAlso is an optional alternative topic the learner can choose.
	NodeAlso = "also"
	// NodeOptional is a topic the learner can explore at will.
	NodeOptional = "optional"
)

// Diagram is a visual representation of a learning roadmap: a sequence
// of stages, each containing a set of topics (nodes).
type Diagram struct {
	Stages []DiagramStage
}

// DiagramStage groups related topics within a single step of a diagram.
type DiagramStage struct {
	Title  string
	Note   string
	Choose bool   // when true, selecting one node in the stage is sufficient
	Nodes  []DiagramNode
}

// DiagramNode represents a single topic node in a diagram. When Summary
// is set, clicking the node opens a side panel with a brief description
// and linked resources. When Href is set, the node is a plain link.
type DiagramNode struct {
	ID      string // stable identifier, used in URL fragment #t-<id>
	Title   string
	Kind    string
	Href    string
	Summary string
	Links   []Link
}

// Link points to an external resource related to a topic.
type Link struct {
	Title string
	URL   string
}

// Article is a blog post on the platform.
type Article struct {
	Slug        string
	Title       string
	Summary     string
	PublishedAt string
	HTML        string
}
