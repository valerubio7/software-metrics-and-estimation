DROP TABLE IF EXISTS effort_records;
ALTER TABLE tasks DROP CONSTRAINT IF EXISTS tasks_id_story_id_project_id_key;
ALTER TABLE project_members DROP CONSTRAINT IF EXISTS project_members_id_project_id_key;
