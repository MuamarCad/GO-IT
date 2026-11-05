CREATE TABLE IF NOT EXISTS mentors (
    slug TEXT PRIMARY KEY,
    display_name TEXT NOT NULL,
    specialization TEXT NOT NULL,
    level TEXT NOT NULL,
    format TEXT NOT NULL,
    topics TEXT NOT NULL,
    description TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS volunteer_requests (
    slug TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    organization TEXT NOT NULL,
    category TEXT NOT NULL,
    skills TEXT NOT NULL,
    scale TEXT NOT NULL,
    description TEXT NOT NULL,
    expected_result TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS applications (
    id BIGSERIAL PRIMARY KEY,
    kind TEXT NOT NULL
        CHECK (kind IN ('mentor', 'volunteer')),
    target_id TEXT NOT NULL,
    name TEXT NOT NULL,
    contact TEXT NOT NULL,
    message TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'new'
        CHECK (status IN ('new', 'reviewed', 'closed')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_applications_kind_status
    ON applications (kind, status);
CREATE INDEX IF NOT EXISTS idx_applications_created_at
    ON applications (created_at DESC);
