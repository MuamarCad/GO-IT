package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"go-it/internal/domain"
	"go-it/internal/repository/memory"
)

func TestApplicationSubmit(t *testing.T) {
	store := memory.New()
	svc := NewApplicationService(store, store)
	app, err := svc.Submit(context.Background(), domain.Application{
		Kind:     domain.KindMentor,
		TargetID: "ivan-frontend",
		Name:     "Анна",
		Contact:  "anna@example.com",
		Message:  "Нужна помощь с формой",
	})
	if err != nil {
		t.Fatal(err)
	}
	if app.ID == 0 || app.Status != "new" {
		t.Fatalf("%+v", app)
	}
}

func TestApplicationUnknownTarget(t *testing.T) {
	store := memory.New()
	svc := NewApplicationService(store, store)
	_, err := svc.Submit(context.Background(), domain.Application{
		Kind:     domain.KindVolunteer,
		TargetID: "missing",
		Name:     "Анна",
		Contact:  "anna@example.com",
	})
	if !errors.Is(err, ErrUnknownTarget) {
		t.Fatalf("got %v", err)
	}
}

func TestApplicationValidation(t *testing.T) {
	store := memory.New()
	svc := NewApplicationService(store, store)
	_, err := svc.Submit(context.Background(), domain.Application{
		Kind:     domain.KindMentor,
		TargetID: "ivan-frontend",
		Name:     "A",
		Contact:  "x",
	})
	if !errors.Is(err, ErrInvalidApplication) {
		t.Fatalf("got %v", err)
	}
}

func TestFieldErrors(t *testing.T) {
	ok := domain.Application{Name: "Анна", Contact: "anna@example.com"}
	if errs := FieldErrors(ok); len(errs) != 0 {
		t.Fatalf("корректная заявка: %v", errs)
	}
	bad := FieldErrors(domain.Application{Name: " A ", Contact: "", Message: strings.Repeat("я", 2001)})
	for _, f := range []string{"name", "contact", "message"} {
		if bad[f] == "" {
			t.Errorf("нет сообщения для поля %s", f)
		}
	}
}
