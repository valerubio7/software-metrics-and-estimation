ALTER TABLE stories
    ADD COLUMN seq BIGINT GENERATED ALWAYS AS IDENTITY;

ALTER TABLE stories
    ADD CONSTRAINT stories_project_id_seq_key UNIQUE (project_id, seq);
