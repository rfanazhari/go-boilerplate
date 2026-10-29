-- Blog storage (technical design §3.1).
-- Indexes: global slug uniqueness (blogs_slug_key); owner FK lookups (blogs_owner_id_INDEX);
-- public list (blogs_public_list_order_idx, partial published/non-deleted);
-- admin list (blogs_admin_list_order_idx, all rows).

CREATE TABLE blogs (
    id UUID PRIMARY KEY,
    owner_id UUID NOT NULL REFERENCES accounts (id),
    slug TEXT NOT NULL,
    title TEXT NOT NULL,
    content TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL,
    deleted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT blogs_slug_key UNIQUE (slug),
    CONSTRAINT blogs_status_check CHECK (status IN ('draft', 'published')),
    CONSTRAINT blogs_title_length_check CHECK (char_length(title) >= 10 AND char_length(title) <= 200)
);

CREATE INDEX "blogs_owner_id_INDEX" ON blogs (owner_id);

CREATE INDEX blogs_public_list_order_idx
    ON blogs (created_at DESC, id DESC)
    WHERE status = 'published' AND deleted_at IS NULL;

CREATE INDEX blogs_admin_list_order_idx
    ON blogs (created_at DESC, id DESC);
