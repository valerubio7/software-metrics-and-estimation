CREATE TABLE stories (
    id UUID PRIMARY KEY,
    project_id UUID NOT NULL CONSTRAINT stories_project_id_fkey REFERENCES projects(id) ON DELETE RESTRICT,
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    priority TEXT NOT NULL CHECK (priority IN ('alta', 'media', 'baja')),
    status TEXT NOT NULL,
    story_points INTEGER NULL,
    acceptance_criteria TEXT[] NOT NULL,
    CHECK (cardinality(acceptance_criteria) > 0),
    CHECK (array_position(acceptance_criteria, NULL) IS NULL)
);
