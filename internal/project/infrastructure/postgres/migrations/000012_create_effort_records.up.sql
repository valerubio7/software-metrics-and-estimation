ALTER TABLE project_members
    ADD CONSTRAINT project_members_id_project_id_key UNIQUE (id, project_id);

ALTER TABLE tasks
    ADD CONSTRAINT tasks_id_story_id_project_id_key UNIQUE (id, story_id, project_id);

CREATE TABLE effort_records (
    id UUID PRIMARY KEY,
    project_id UUID NOT NULL,
    story_id UUID NOT NULL,
    task_id UUID NULL,
    member_id UUID NOT NULL,
    work_date DATE NOT NULL,
    description TEXT NOT NULL
        CONSTRAINT effort_records_description_length
            CHECK (char_length(description) BETWEEN 1 AND 500),
    hours NUMERIC(5,2) NOT NULL
        CONSTRAINT effort_records_hours_range CHECK (hours > 0 AND hours <= 24),
    seq BIGINT GENERATED ALWAYS AS IDENTITY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT effort_records_story_project_fkey FOREIGN KEY (story_id, project_id)
        REFERENCES stories (id, project_id) ON DELETE RESTRICT,
    CONSTRAINT effort_records_member_project_fkey FOREIGN KEY (member_id, project_id)
        REFERENCES project_members (id, project_id) ON DELETE RESTRICT,
    CONSTRAINT effort_records_task_story_project_fkey FOREIGN KEY (task_id, story_id, project_id)
        REFERENCES tasks (id, story_id, project_id) MATCH SIMPLE ON DELETE RESTRICT
);

CREATE INDEX effort_records_project_story_idx ON effort_records (project_id, story_id, seq);
CREATE INDEX effort_records_member_idx ON effort_records (member_id);
CREATE INDEX effort_records_task_idx ON effort_records (task_id) WHERE task_id IS NOT NULL;
