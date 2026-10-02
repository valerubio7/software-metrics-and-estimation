DROP TABLE sprint_stories;
ALTER TABLE sprints DROP CONSTRAINT sprints_id_project_id_key;
ALTER TABLE stories DROP CONSTRAINT stories_id_project_id_key;
