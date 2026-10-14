ALTER TABLE sprints
    ADD COLUMN seq BIGINT GENERATED ALWAYS AS IDENTITY;

ALTER TABLE sprints
    ADD CONSTRAINT sprints_project_id_seq_key UNIQUE (project_id, seq);
