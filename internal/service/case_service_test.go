package service

import (
	"errors"
	"testing"

	"go-it/internal/cases"
	"go-it/internal/domain"
)

func TestCaseHappyPath(t *testing.T) {
	s := NewCaseService(cases.All())
	state, view, err := s.Start("frontend")
	if err != nil {
		t.Fatal(err)
	}
	if view.Step != 1 || view.Total != 4 {
		t.Fatalf("unexpected view %+v", view)
	}

	answers := []struct{ q, o string }{
		{"q1", "q1_b"},
		{"q2", "q2_b"},
		{"q3", "q3_b"},
		{"q4", "q4_b"},
	}
	for i, a := range answers {
		var outcome domain.AnswerOutcome
		state, outcome, err = s.Answer(state, a.q, a.o)
		if err != nil {
			t.Fatalf("step %d: %v", i, err)
		}
		if i < 3 && outcome.Finished {
			t.Fatal("finished too early")
		}
		if i == 3 && !outcome.Finished {
			t.Fatal("expected finish")
		}
	}

	res, err := s.Result(state)
	if err != nil {
		t.Fatal(err)
	}
	if res.Score != 4 || res.MaxScore != 4 {
		t.Fatalf("score %d/%d", res.Score, res.MaxScore)
	}
}

func TestUnknownOption(t *testing.T) {
	s := NewCaseService(cases.All())
	state, _, _ := s.Start("frontend")
	_, _, err := s.Answer(state, "q1", "nope")
	if !errors.Is(err, ErrUnknownOption) {
		t.Fatalf("got %v", err)
	}
}

func TestCannotSkip(t *testing.T) {
	s := NewCaseService(cases.All())
	state, _, _ := s.Start("frontend")
	_, _, err := s.Answer(state, "q2", "q2_b")
	if !errors.Is(err, ErrStepMismatch) {
		t.Fatalf("got %v", err)
	}
}

func TestRepeatDoesNotDoubleScore(t *testing.T) {
	s := NewCaseService(cases.All())
	state, _, _ := s.Start("frontend")
	state, _, err := s.Answer(state, "q1", "q1_b")
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = s.Answer(state, "q1", "q1_a")
	if !errors.Is(err, ErrStepMismatch) {
		t.Fatalf("expected mismatch, got %v", err)
	}
}

func TestUnknownCase(t *testing.T) {
	s := NewCaseService(cases.All())
	_, _, err := s.Start("nope")
	if !errors.Is(err, ErrUnknownCase) {
		t.Fatalf("got %v", err)
	}
}

func TestResultRequiresCompletion(t *testing.T) {
	s := NewCaseService(cases.All())
	state, _, _ := s.Start("frontend")
	_, err := s.Result(state)
	if !errors.Is(err, ErrStepMismatch) {
		t.Fatalf("got %v", err)
	}
}

func TestMissingState(t *testing.T) {
	s := NewCaseService(cases.All())
	_, _, err := s.Answer(domain.AttemptState{}, "q1", "q1_b")
	if !errors.Is(err, ErrMissingState) {
		t.Fatalf("got %v", err)
	}
}

func TestResultTopicBreakdown(t *testing.T) {
	svc := NewCaseService(cases.All())
	state, _, err := svc.Start("frontend")
	if err != nil {
		t.Fatal(err)
	}
	// q1 верно, q2 неверно, q3 верно, q4 неверно
	picks := []struct{ q, o string }{{"q1", "q1_b"}, {"q2", "q2_a"}, {"q3", "q3_b"}, {"q4", "q4_a"}}
	for _, p := range picks {
		state, _, err = svc.Answer(state, p.q, p.o)
		if err != nil {
			t.Fatal(err)
		}
	}
	res, err := svc.Result(state)
	if err != nil {
		t.Fatal(err)
	}
	if res.Score != 2 || res.MaxScore != 4 {
		t.Fatalf("счёт %d из %d", res.Score, res.MaxScore)
	}
	if len(res.Topics) != 4 {
		t.Fatalf("ожидали 4 темы, получили %d", len(res.Topics))
	}
	sum := 0
	for _, tp := range res.Topics {
		if tp.Max != 1 {
			t.Errorf("тема %q: max=%d", tp.Topic, tp.Max)
		}
		sum += tp.Score
	}
	if sum != res.Score {
		t.Fatalf("сумма по темам %d не равна общему счёту %d", sum, res.Score)
	}
}
