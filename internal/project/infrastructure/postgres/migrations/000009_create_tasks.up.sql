CREATE TABLE tasks (
    id UUID PRIMARY KEY,
    project_id UUID NOT NULL,
    sprint_id UUID NOT NULL,
    story_id UUID NOT NULL,
    title TEXT NOT NULL,
    estimated_hours NUMERIC(7,2) NULL
        CONSTRAINT tasks_estimated_hours_positive CHECK (estimated_hours > 0),
    seq BIGINT GENERATED ALWAYS AS IDENTITY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT tasks_sprint_story_fkey FOREIGN KEY (sprint_id, story_id)
        REFERENCES sprint_stories (sprint_id, story_id) ON DELETE RESTRICT,
    CONSTRAINT tasks_sprint_project_fkey FOREIGN KEY (sprint_id, project_id)
        REFERENCES sprints (id, project_id) ON DELETE RESTRICT,
    CONSTRAINT tasks_story_project_fkey FOREIGN KEY (story_id, project_id)
        REFERENCES stories (id, project_id) ON DELETE RESTRICT
);

CREATE INDEX tasks_sprint_story_idx ON tasks (sprint_id, story_id, seq);
