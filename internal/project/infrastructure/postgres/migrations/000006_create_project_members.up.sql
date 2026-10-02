CREATE TABLE project_members (
    id UUID PRIMARY KEY,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE RESTRICT,
    full_name TEXT NOT NULL,
    email TEXT NULL
);

CREATE INDEX project_members_project_id_idx ON project_members (project_id);

CREATE UNIQUE INDEX project_members_identity_with_email
    ON project_members (project_id, full_name, email)
    WHERE email IS NOT NULL;

CREATE UNIQUE INDEX project_members_identity_without_email
    ON project_members (project_id, full_name)
    WHERE email IS NULL;
