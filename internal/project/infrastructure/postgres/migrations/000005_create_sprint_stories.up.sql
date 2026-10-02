ALTER TABLE stories ADD CONSTRAINT stories_id_project_id_key UNIQUE (id, project_id);
ALTER TABLE sprints ADD CONSTRAINT sprints_id_project_id_key UNIQUE (id, project_id);

CREATE TABLE sprint_stories (
    sprint_id UUID NOT NULL,
    story_id UUID NOT NULL,
    project_id UUID NOT NULL,
    CONSTRAINT sprint_stories_pkey PRIMARY KEY (sprint_id, story_id),
    CONSTRAINT sprint_stories_sprint_project_fkey FOREIGN KEY (sprint_id, project_id)
        REFERENCES sprints (id, project_id) ON DELETE RESTRICT,
    CONSTRAINT sprint_stories_story_project_fkey FOREIGN KEY (story_id, project_id)
        REFERENCES stories (id, project_id) ON DELETE RESTRICT
);
