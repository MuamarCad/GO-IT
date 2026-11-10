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
	if view.Step != 1 || view.Total != 20 {
		t.Fatalf("unexpected view %+v", view)
	}

	answers := []struct{ q, o string }{
		{"q1", "q1_b"},
		{"q2", "q2_b"},
		{"q3", "q3_a"},
		{"q4", "q4_a"},
		{"q5", "q5_a"},
		{"q6", "q6_b"},
		{"q7", "q7_a"},
		{"q8", "q8_a"},
		{"q9", "q9_a"},
		{"q10", "q10_a"},
		{"q11", "q11_a"},
		{"q12", "q12_b"},
		{"q13", "q13_a"},
		{"q14", "q14_a"},
		{"q15", "q15_a"},
		{"q16", "q16_a"},
		{"q17", "q17_a"},
		{"q18", "q18_a"},
		{"q19", "q19_a"},
		{"q20", "q20_a"},
	}
	for i, a := range answers {
		var outcome domain.AnswerOutcome
		state, outcome, err = s.Answer(state, a.q, a.o)
		if err != nil {
			t.Fatalf("step %d: %v", i, err)
		}
		if i < 19 && outcome.Finished {
			t.Fatal("finished too early")
		}
		if i == 19 && !outcome.Finished {
			t.Fatal("expected finish after last question")
		}
	}

	res, err := s.Result(state)
	if err != nil {
		t.Fatal(err)
	}
	if res.Score != 20 || res.MaxScore != 20 {
		t.Fatalf("score %d/%d", res.Score, res.MaxScore)
	}
}

func TestBackendCaseHappyPath(t *testing.T) {
	s := NewCaseService(cases.All())
	state, view, err := s.Start("backend")
	if err != nil {
		t.Fatal(err)
	}
	if view.Step != 1 || view.Total != 18 {
		t.Fatalf("unexpected view %+v", view)
	}

	answers := []struct{ q, o string }{
		{"q1", "q1_a"},
		{"q2", "q2_a"},
		{"q3", "q3_a"},
		{"q4", "q4_a"},
		{"q5", "q5_a"},
		{"q6", "q6_a"},
		{"q7", "q7_a"},
		{"q8", "q8_a"},
		{"q9", "q9_a"},
		{"q10", "q10_a"},
		{"q11", "q11_a"},
		{"q12", "q12_a"},
		{"q13", "q13_a"},
		{"q14", "q14_a"},
		{"q15", "q15_a"},
		{"q16", "q16_a"},
		{"q17", "q17_a"},
		{"q18", "q18_a"},
	}
	for i, a := range answers {
		var outcome domain.AnswerOutcome
		state, outcome, err = s.Answer(state, a.q, a.o)
		if err != nil {
			t.Fatalf("step %d: %v", i, err)
		}
		if i < 17 && outcome.Finished {
			t.Fatal("finished too early")
		}
		if i == 17 && !outcome.Finished {
			t.Fatal("expected finish after last question")
		}
	}

	res, err := s.Result(state)
	if err != nil {
		t.Fatal(err)
	}
	if res.Score != 18 || res.MaxScore != 18 {
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
	// Проходим все 20 вопросов: q1, q3, q5, q7 — верно; остальные — неверно
	picks := []struct{ q, o string }{
		{"q1", "q1_b"},  // верно
		{"q2", "q2_a"},  // неверно
		{"q3", "q3_a"},  // верно
		{"q4", "q4_b"},  // неверно
		{"q5", "q5_a"},  // верно
		{"q6", "q6_a"},  // неверно
		{"q7", "q7_a"},  // верно
		{"q8", "q8_b"},  // неверно
		{"q9", "q9_b"},  // неверно
		{"q10", "q10_b"}, // неверно
		{"q11", "q11_b"}, // неверно
		{"q12", "q12_a"}, // неверно
		{"q13", "q13_a"}, // верно
		{"q14", "q14_b"}, // неверно
		{"q15", "q15_b"}, // неверно
		{"q16", "q16_b"}, // неверно
		{"q17", "q17_b"}, // неверно
		{"q18", "q18_b"}, // неверно
		{"q19", "q19_b"}, // неверно
		{"q20", "q20_b"}, // неверно
	}
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
	// 5 правильных из 20
	if res.Score != 5 || res.MaxScore != 20 {
		t.Fatalf("счёт %d из %d", res.Score, res.MaxScore)
	}
	if len(res.Topics) < 3 {
		t.Fatalf("ожидали хотя бы 3 темы, получили %d", len(res.Topics))
	}
	sum := 0
	for _, tp := range res.Topics {
		if tp.Score > tp.Max {
			t.Errorf("тема %q: score > max", tp.Topic)
		}
		sum += tp.Score
	}
	if sum != res.Score {
		t.Fatalf("сумма по темам %d не равна общему счёту %d", sum, res.Score)
	}
}

func TestCaseListIncludesBoth(t *testing.T) {
	s := NewCaseService(cases.All())
	list := s.List()
	if len(list) != 2 {
		t.Fatalf("ожидали 2 кейса, получили %d", len(list))
	}
	slugSet := make(map[string]bool)
	for _, c := range list {
		slugSet[c.Slug] = true
	}
	if !slugSet["backend"] {
		t.Error("отсутствует кейс backend")
	}
	if !slugSet["frontend"] {
		t.Error("отсутствует кейс frontend")
	}
}
