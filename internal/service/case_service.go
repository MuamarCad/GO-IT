package service

import (
	"fmt"
	"sort"

	"go-it/internal/domain"
)

var (
	ErrUnknownCase   = fmt.Errorf("unknown case")
	ErrUnknownOption = fmt.Errorf("unknown option")
	ErrCaseCompleted = fmt.Errorf("case already completed")
	ErrMissingState  = fmt.Errorf("missing attempt state")
	ErrStepMismatch  = fmt.Errorf("cannot skip required steps")
)

type CaseService struct {
	cases map[string]domain.CareerCase
}

func NewCaseService(cases []domain.CareerCase) *CaseService {
	m := make(map[string]domain.CareerCase, len(cases))
	for _, c := range cases {
		m[c.Slug] = c
	}
	return &CaseService{cases: m}
}

func (s *CaseService) List() []domain.CareerCase {
	out := make([]domain.CareerCase, 0, len(s.cases))
	for _, c := range s.cases {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Slug < out[j].Slug })
	return out
}

func (s *CaseService) Get(slug string) (domain.CareerCase, error) {
	c, ok := s.cases[slug]
	if !ok {
		return domain.CareerCase{}, ErrUnknownCase
	}
	return c, nil
}

func (s *CaseService) Start(slug string) (domain.AttemptState, domain.QuestionView, error) {
	c, err := s.Get(slug)
	if err != nil {
		return domain.AttemptState{}, domain.QuestionView{}, err
	}
	if len(c.Questions) == 0 {
		return domain.AttemptState{}, domain.QuestionView{}, fmt.Errorf("case has no questions")
	}
	state := domain.AttemptState{
		Slug:    slug,
		Index:   0,
		Answers: map[string]string{},
	}
	return state, questionView(c, 0, ""), nil
}

func (s *CaseService) Current(state domain.AttemptState) (domain.QuestionView, error) {
	c, err := s.Get(state.Slug)
	if err != nil {
		return domain.QuestionView{}, err
	}
	if state.Completed {
		return domain.QuestionView{Case: c, Completed: true, Total: len(c.Questions)}, nil
	}
	if state.Index < 0 || state.Index >= len(c.Questions) {
		return domain.QuestionView{}, ErrStepMismatch
	}
	return questionView(c, state.Index, ""), nil
}

func (s *CaseService) Answer(state domain.AttemptState, questionID, optionID string) (domain.AttemptState, domain.AnswerOutcome, error) {
	if state.Slug == "" {
		return state, domain.AnswerOutcome{}, ErrMissingState
	}
	c, err := s.Get(state.Slug)
	if err != nil {
		return state, domain.AnswerOutcome{}, err
	}
	if state.Completed {
		return state, domain.AnswerOutcome{}, ErrCaseCompleted
	}
	if state.Index < 0 || state.Index >= len(c.Questions) {
		return state, domain.AnswerOutcome{}, ErrStepMismatch
	}
	q := c.Questions[state.Index]
	if questionID != q.ID {
		return state, domain.AnswerOutcome{}, ErrStepMismatch
	}
	opt, ok := optionByID(q, optionID)
	if !ok {
		view := questionView(c, state.Index, "Выберите один из предложенных вариантов.")
		return state, domain.AnswerOutcome{Next: &view}, ErrUnknownOption
	}
	if state.Answers == nil {
		state.Answers = map[string]string{}
	}
	if _, exists := state.Answers[q.ID]; exists {
		return nextAfter(state, c)
	}
	state.Answers[q.ID] = opt.ID
	state.Index++
	return nextAfter(state, c)
}

func nextAfter(state domain.AttemptState, c domain.CareerCase) (domain.AttemptState, domain.AnswerOutcome, error) {
	if state.Index >= len(c.Questions) {
		state.Completed = true
		state.Index = len(c.Questions)
		return state, domain.AnswerOutcome{Finished: true}, nil
	}
	view := questionView(c, state.Index, "")
	return state, domain.AnswerOutcome{Next: &view}, nil
}

func (s *CaseService) Result(state domain.AttemptState) (domain.CaseResult, error) {
	if state.Slug == "" {
		return domain.CaseResult{}, ErrMissingState
	}
	c, err := s.Get(state.Slug)
	if err != nil {
		return domain.CaseResult{}, err
	}
	if !state.Completed {
		return domain.CaseResult{}, ErrStepMismatch
	}
	details := make([]domain.OptionResult, 0, len(c.Questions))
	score := 0
	var topics []domain.TopicScore
	topicIdx := map[string]int{}
	for _, q := range c.Questions {
		correct, _ := correctOption(q)
		chosenID := ""
		if state.Answers != nil {
			chosenID = state.Answers[q.ID]
		}
		chosen, _ := optionByID(q, chosenID)
		ok := chosen.ID != "" && chosen.Correct
		if ok {
			score++
		}
		if q.Topic != "" {
			i, seen := topicIdx[q.Topic]
			if !seen {
				i = len(topics)
				topicIdx[q.Topic] = i
				topics = append(topics, domain.TopicScore{Topic: q.Topic})
			}
			topics[i].Max++
			if ok {
				topics[i].Score++
			}
		}
		details = append(details, domain.OptionResult{
			Question:    q,
			Chosen:      chosen,
			Correct:     correct,
			IsCorrect:   ok,
			Explanation: chosen.Explanation,
		})
		if details[len(details)-1].Explanation == "" {
			details[len(details)-1].Explanation = correct.Explanation
		}
	}
	return domain.CaseResult{
		Case:       c,
		Score:      score,
		MaxScore:   len(c.Questions),
		Topics:     topics,
		Details:    details,
		Disclaimer: "Результат ознакомительный: это не диагноз и не окончательная рекомендация профессии.",
	}, nil
}

func questionView(c domain.CareerCase, index int, errMsg string) domain.QuestionView {
	return domain.QuestionView{
		Case:     c,
		Question: c.Questions[index],
		Step:     index + 1,
		Total:    len(c.Questions),
		Error:    errMsg,
	}
}

func optionByID(q domain.CaseQuestion, id string) (domain.CaseOption, bool) {
	for _, o := range q.Options {
		if o.ID == id {
			return o, true
		}
	}
	return domain.CaseOption{}, false
}

func correctOption(q domain.CaseQuestion) (domain.CaseOption, bool) {
	for _, o := range q.Options {
		if o.Correct {
			return o, true
		}
	}
	return domain.CaseOption{}, false
}
