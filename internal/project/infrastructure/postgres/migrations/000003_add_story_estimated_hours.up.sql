ALTER TABLE stories
    ADD COLUMN estimated_hours NUMERIC(7,2) NULL
    CONSTRAINT stories_estimated_hours_positive CHECK (estimated_hours > 0);

ALTER TABLE stories
    ADD CONSTRAINT stories_status_check
    CHECK (status IN ('pendiente', 'en_progreso', 'completada'));
