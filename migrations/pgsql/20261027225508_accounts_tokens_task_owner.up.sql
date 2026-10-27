CREATE TABLE accounts (
    id UUID PRIMARY KEY,
    email TEXT NOT NULL,
    name TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL,
    status TEXT NOT NULL,
    deleted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT accounts_role_check CHECK (role IN ('admin', 'user')),
    CONSTRAINT accounts_status_check CHECK (status IN ('active', 'non_active'))
);

CREATE UNIQUE INDEX accounts_email_live_uidx
    ON accounts (LOWER(email))
    WHERE deleted_at IS NULL;

CREATE INDEX "accounts_role_INDEX" ON accounts (role);
CREATE INDEX "accounts_status_INDEX" ON accounts (status);

CREATE TABLE credential_tokens (
    id UUID PRIMARY KEY,
    account_id UUID NOT NULL REFERENCES accounts (id),
    token_hash TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX "credential_tokens_account_id_INDEX" ON credential_tokens (account_id);

INSERT INTO accounts (
    id,
    email,
    name,
    password_hash,
    role,
    status,
    created_at,
    updated_at
) VALUES (
    'e0000000-0000-4000-8000-000000000001',
    'admin@example.com',
    'Admin',
    '$2a$12$cF0IcjelC.DMS6qtUT0UF.8Cy8rBstg8TE1dtPMGiPIdA8o2sgNQC',
    'admin',
    'active',
    NOW(),
    NOW()
);

ALTER TABLE tasks ADD COLUMN owner_id UUID;

UPDATE tasks
SET owner_id = 'e0000000-0000-4000-8000-000000000001';

ALTER TABLE tasks
    ALTER COLUMN owner_id SET NOT NULL;

ALTER TABLE tasks
    ADD CONSTRAINT tasks_owner_id_fkey FOREIGN KEY (owner_id) REFERENCES accounts (id);

CREATE INDEX "tasks_owner_id_INDEX" ON tasks (owner_id);
