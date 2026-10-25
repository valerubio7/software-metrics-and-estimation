CREATE TABLE defects (
    id UUID PRIMARY KEY,
    project_id UUID NOT NULL,
    story_id UUID NOT NULL,
    sprint_id UUID NOT NULL,
    description TEXT NOT NULL
        CONSTRAINT defects_description_length
            CHECK (char_length(description) BETWEEN 1 AND 500),
    severity TEXT NOT NULL
        CONSTRAINT defects_severity_check CHECK (severity = ANY (ARRAY['alta'::text, 'media'::text, 'baja'::text])),
    status TEXT NOT NULL
        CONSTRAINT defects_status_check CHECK (status = ANY (ARRAY['detectado'::text, 'en_correccion'::text, 'resuelto'::text])),
    seq BIGINT GENERATED ALWAYS AS IDENTITY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT defects_story_project_fkey FOREIGN KEY (story_id, project_id)
        REFERENCES stories (id, project_id) ON DELETE RESTRICT,
    CONSTRAINT defects_sprint_project_fkey FOREIGN KEY (sprint_id, project_id)
        REFERENCES sprints (id, project_id) ON DELETE RESTRICT
);

CREATE INDEX defects_project_story_idx ON defects (project_id, story_id, seq);
CREATE INDEX defects_project_sprint_idx ON defects (project_id, sprint_id, seq);
CREATE INDEX defects_project_seq_idx ON defects (project_id, seq);
