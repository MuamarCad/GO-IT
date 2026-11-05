package service

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"go-it/internal/domain"
)

var (
	ErrInvalidApplication = fmt.Errorf("invalid application")
	ErrUnknownTarget      = fmt.Errorf("unknown target")
)

type ApplicationStore interface {
	Create(ctx context.Context, app domain.Application) (domain.Application, error)
}

type TargetChecker interface {
	MentorExists(ctx context.Context, slug string) (bool, error)
	VolunteerExists(ctx context.Context, slug string) (bool, error)
}

type ApplicationService struct {
	store   ApplicationStore
	targets TargetChecker
}

func NewApplicationService(store ApplicationStore, targets TargetChecker) *ApplicationService {
	return &ApplicationService{store: store, targets: targets}
}

func (s *ApplicationService) Submit(ctx context.Context, app domain.Application) (domain.Application, error) {
	app.Kind = domain.ApplicationKind(strings.TrimSpace(string(app.Kind)))
	app.TargetID = strings.TrimSpace(app.TargetID)
	app.Name = strings.TrimSpace(app.Name)
	app.Contact = strings.TrimSpace(app.Contact)
	app.Message = strings.TrimSpace(app.Message)
	app.Status = "new"

	if err := validateApplication(app); err != nil {
		return domain.Application{}, err
	}

	var exists bool
	var err error
	switch app.Kind {
	case domain.KindMentor:
		exists, err = s.targets.MentorExists(ctx, app.TargetID)
	case domain.KindVolunteer:
		exists, err = s.targets.VolunteerExists(ctx, app.TargetID)
	default:
		return domain.Application{}, fmt.Errorf("%w: unknown kind", ErrInvalidApplication)
	}
	if err != nil {
		return domain.Application{}, err
	}
	if !exists {
		return domain.Application{}, ErrUnknownTarget
	}
	return s.store.Create(ctx, app)
}

func validateApplication(app domain.Application) error {
	if app.Kind != domain.KindMentor && app.Kind != domain.KindVolunteer {
		return fmt.Errorf("%w: kind", ErrInvalidApplication)
	}
	if app.TargetID == "" {
		return fmt.Errorf("%w: target", ErrInvalidApplication)
	}
	errs := FieldErrors(app)
	for _, field := range []string{"name", "contact", "message"} {
		if _, bad := errs[field]; bad {
			return fmt.Errorf("%w: %s", ErrInvalidApplication, field)
		}
	}
	return nil
}

// FieldErrors возвращает понятные пользователю сообщения по полям формы (ключи: name, contact, message).
// Пустая карта означает, что поля корректны. Пробелы по краям игнорируются, как и в Submit.
func FieldErrors(app domain.Application) map[string]string {
	errs := map[string]string{}
	if n := utf8.RuneCountInString(strings.TrimSpace(app.Name)); n < 2 || n > 100 {
		errs["name"] = "Введите имя: от 2 до 100 символов."
	}
	if n := utf8.RuneCountInString(strings.TrimSpace(app.Contact)); n < 3 || n > 254 {
		errs["contact"] = "Укажите контакт для ответа: email или Telegram, от 3 до 254 символов."
	}
	if utf8.RuneCountInString(strings.TrimSpace(app.Message)) > 2000 {
		errs["message"] = "Сообщение слишком длинное: максимум 2000 символов."
	}
	return errs
}
