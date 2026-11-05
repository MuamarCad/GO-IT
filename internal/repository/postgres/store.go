package postgres

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"go-it/internal/catalog"
	"go-it/internal/domain"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

type Store struct {
	pool *pgxpool.Pool
}

func Connect(ctx context.Context, databaseURL string) (*Store, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("connect postgres: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	s := &Store{pool: pool}
	if err := s.migrate(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	if err := s.seed(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() {
	if s != nil && s.pool != nil {
		s.pool.Close()
	}
}

func (s *Store) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

func (s *Store) migrate(ctx context.Context) error {
	if _, err := s.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY
		)`); err != nil {
		return fmt.Errorf("schema_migrations: %w", err)
	}
	entries, err := fs.ReadDir(migrationFS, "migrations")
	if err != nil {
		return err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	for _, name := range names {
		var exists bool
		if err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version=$1)`, name).Scan(&exists); err != nil {
			return err
		}
		if exists {
			continue
		}
		body, err := migrationFS.ReadFile("migrations/" + name)
		if err != nil {
			return err
		}
		if _, err := s.pool.Exec(ctx, string(body)); err != nil {
			return fmt.Errorf("migration %s: %w", name, err)
		}
		if _, err := s.pool.Exec(ctx, `INSERT INTO schema_migrations(version) VALUES ($1)`, name); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) seed(ctx context.Context) error {
	for _, m := range catalog.SeedMentors() {
		_, err := s.pool.Exec(ctx, `
			INSERT INTO mentors (slug, display_name, specialization, level, format, topics, description)
			VALUES ($1,$2,$3,$4,$5,$6,$7)
			ON CONFLICT (slug) DO UPDATE SET
				display_name=EXCLUDED.display_name,
				specialization=EXCLUDED.specialization,
				level=EXCLUDED.level,
				format=EXCLUDED.format,
				topics=EXCLUDED.topics,
				description=EXCLUDED.description`,
			m.Slug, m.DisplayName, m.Specialization, m.Level, m.Format, m.Topics, m.Description)
		if err != nil {
			return fmt.Errorf("seed mentor %s: %w", m.Slug, err)
		}
	}
	for _, v := range catalog.SeedVolunteerTasks() {
		_, err := s.pool.Exec(ctx, `
			INSERT INTO volunteer_requests (slug, title, organization, category, skills, scale, description, expected_result)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
			ON CONFLICT (slug) DO UPDATE SET
				title=EXCLUDED.title,
				organization=EXCLUDED.organization,
				category=EXCLUDED.category,
				skills=EXCLUDED.skills,
				scale=EXCLUDED.scale,
				description=EXCLUDED.description,
				expected_result=EXCLUDED.expected_result`,
			v.Slug, v.Title, v.Organization, v.Category, v.Skills, v.Scale, v.Description, v.ExpectedResult)
		if err != nil {
			return fmt.Errorf("seed volunteer %s: %w", v.Slug, err)
		}
	}
	return nil
}

func (s *Store) ListMentors(ctx context.Context) ([]domain.Mentor, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT slug, display_name, specialization, level, format, topics, description
		FROM mentors ORDER BY display_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Mentor
	for rows.Next() {
		var m domain.Mentor
		if err := rows.Scan(&m.Slug, &m.DisplayName, &m.Specialization, &m.Level, &m.Format, &m.Topics, &m.Description); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Store) GetMentor(ctx context.Context, slug string) (domain.Mentor, bool, error) {
	var m domain.Mentor
	err := s.pool.QueryRow(ctx, `
		SELECT slug, display_name, specialization, level, format, topics, description
		FROM mentors WHERE slug=$1`, slug).
		Scan(&m.Slug, &m.DisplayName, &m.Specialization, &m.Level, &m.Format, &m.Topics, &m.Description)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Mentor{}, false, nil
	}
	if err != nil {
		return domain.Mentor{}, false, err
	}
	return m, true, nil
}

func (s *Store) ListVolunteerTasks(ctx context.Context) ([]domain.VolunteerTask, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT slug, title, organization, category, skills, scale, description, expected_result
		FROM volunteer_requests ORDER BY title`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.VolunteerTask
	for rows.Next() {
		var v domain.VolunteerTask
		if err := rows.Scan(&v.Slug, &v.Title, &v.Organization, &v.Category, &v.Skills, &v.Scale, &v.Description, &v.ExpectedResult); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (s *Store) GetVolunteerTask(ctx context.Context, slug string) (domain.VolunteerTask, bool, error) {
	var v domain.VolunteerTask
	err := s.pool.QueryRow(ctx, `
		SELECT slug, title, organization, category, skills, scale, description, expected_result
		FROM volunteer_requests WHERE slug=$1`, slug).
		Scan(&v.Slug, &v.Title, &v.Organization, &v.Category, &v.Skills, &v.Scale, &v.Description, &v.ExpectedResult)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.VolunteerTask{}, false, nil
	}
	if err != nil {
		return domain.VolunteerTask{}, false, err
	}
	return v, true, nil
}

func (s *Store) MentorExists(ctx context.Context, slug string) (bool, error) {
	_, ok, err := s.GetMentor(ctx, slug)
	return ok, err
}

func (s *Store) VolunteerExists(ctx context.Context, slug string) (bool, error) {
	_, ok, err := s.GetVolunteerTask(ctx, slug)
	return ok, err
}

func (s *Store) Create(ctx context.Context, app domain.Application) (domain.Application, error) {
	err := s.pool.QueryRow(ctx, `
		INSERT INTO applications (kind, target_id, name, contact, message, status)
		VALUES ($1,$2,$3,$4,$5,$6)
		RETURNING id, created_at`,
		string(app.Kind), app.TargetID, app.Name, app.Contact, app.Message, app.Status).
		Scan(&app.ID, &app.CreatedAt)
	if err != nil {
		return domain.Application{}, err
	}
	return app, nil
}
