CREATE TABLE projects (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    start_date DATE NOT NULL,
    planned_finish_date DATE NOT NULL,
    CHECK (planned_finish_date >= start_date)
);
