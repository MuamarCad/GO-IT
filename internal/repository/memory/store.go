package memory

import (
	"context"
	"sort"
	"sync"
	"time"

	"go-it/internal/catalog"
	"go-it/internal/domain"
)

type Store struct {
	mu           sync.Mutex
	nextID       int64
	mentors      map[string]domain.Mentor
	volunteers   map[string]domain.VolunteerTask
	applications []domain.Application
}

func New() *Store {
	s := &Store{
		nextID:     1,
		mentors:    map[string]domain.Mentor{},
		volunteers: map[string]domain.VolunteerTask{},
	}
	for _, m := range catalog.SeedMentors() {
		s.mentors[m.Slug] = m
	}
	for _, v := range catalog.SeedVolunteerTasks() {
		s.volunteers[v.Slug] = v
	}
	return s
}

func (s *Store) ListMentors(context.Context) ([]domain.Mentor, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]domain.Mentor, 0, len(s.mentors))
	for _, m := range s.mentors {
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].DisplayName < out[j].DisplayName })
	return out, nil
}

func (s *Store) GetMentor(_ context.Context, slug string) (domain.Mentor, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	m, ok := s.mentors[slug]
	return m, ok, nil
}

func (s *Store) ListVolunteerTasks(context.Context) ([]domain.VolunteerTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]domain.VolunteerTask, 0, len(s.volunteers))
	for _, v := range s.volunteers {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Title < out[j].Title })
	return out, nil
}

func (s *Store) GetVolunteerTask(_ context.Context, slug string) (domain.VolunteerTask, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.volunteers[slug]
	return v, ok, nil
}

func (s *Store) MentorExists(ctx context.Context, slug string) (bool, error) {
	_, ok, err := s.GetMentor(ctx, slug)
	return ok, err
}

func (s *Store) VolunteerExists(ctx context.Context, slug string) (bool, error) {
	_, ok, err := s.GetVolunteerTask(ctx, slug)
	return ok, err
}

func (s *Store) Create(_ context.Context, app domain.Application) (domain.Application, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	app.ID = s.nextID
	s.nextID++
	app.CreatedAt = time.Now()
	s.applications = append(s.applications, app)
	return app, nil
}
