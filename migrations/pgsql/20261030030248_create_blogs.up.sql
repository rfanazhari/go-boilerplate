CREATE TABLE blogs (
    id UUID PRIMARY KEY,
    owner_id UUID NOT NULL REFERENCES accounts (id),
    slug TEXT NOT NULL,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    status TEXT NOT NULL,
    deleted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT blogs_status_check CHECK (status IN ('draft', 'published')),
    CONSTRAINT blogs_slug_check CHECK (
        slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$'
        AND char_length(slug) <= 89
    )
);

CREATE UNIQUE INDEX blogs_slug_uidx ON blogs (slug);

CREATE INDEX "blogs_owner_id_INDEX" ON blogs (owner_id);

CREATE INDEX "blogs_public_list_INDEX"
    ON blogs (created_at DESC, id DESC)
    WHERE deleted_at IS NULL AND status = 'published';
