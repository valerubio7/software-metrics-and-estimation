ALTER TABLE defects
    ADD COLUMN resolution_sprint_id UUID;

ALTER TABLE defects
    ADD CONSTRAINT defects_resolution_sprint_project_fkey
        FOREIGN KEY (resolution_sprint_id, project_id)
        REFERENCES sprints (id, project_id) ON DELETE RESTRICT;

CREATE INDEX defects_project_resolution_sprint_idx
    ON defects (project_id, resolution_sprint_id)
    WHERE resolution_sprint_id IS NOT NULL;
