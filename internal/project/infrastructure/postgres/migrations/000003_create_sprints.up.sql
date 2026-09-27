CREATE TABLE sprints (
    id UUID PRIMARY KEY,
    project_id UUID NOT NULL CONSTRAINT sprints_project_id_fkey REFERENCES projects(id) ON DELETE RESTRICT,
    sprint_goal TEXT NOT NULL
);
