DROP INDEX IF EXISTS defects_project_resolution_sprint_idx;

ALTER TABLE defects
    DROP CONSTRAINT IF EXISTS defects_resolution_sprint_project_fkey,
    DROP COLUMN IF EXISTS resolution_sprint_id;
