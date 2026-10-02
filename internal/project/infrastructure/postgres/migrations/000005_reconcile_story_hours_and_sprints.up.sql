DO $$
DECLARE
    has_hours boolean;
    has_hours_check boolean;
    has_status_check boolean;
    has_sprints boolean;
    projects_oid oid;
    constraint_definition text;
BEGIN
    SELECT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = current_schema() AND table_name = 'stories' AND column_name = 'estimated_hours'
    ) INTO has_hours;
    SELECT EXISTS (
        SELECT 1 FROM pg_constraint c
        JOIN pg_class t ON t.oid = c.conrelid
        JOIN pg_namespace n ON n.oid = t.relnamespace
        WHERE n.nspname = current_schema() AND t.relname = 'stories'
          AND c.conname = 'stories_estimated_hours_positive'
    ) INTO has_hours_check;
    SELECT EXISTS (
        SELECT 1 FROM pg_constraint c
        JOIN pg_class t ON t.oid = c.conrelid
        JOIN pg_namespace n ON n.oid = t.relnamespace
        WHERE n.nspname = current_schema() AND t.relname = 'stories'
          AND c.conname = 'stories_status_check'
    ) INTO has_status_check;
    SELECT EXISTS (
        SELECT 1 FROM information_schema.tables
        WHERE table_schema = current_schema() AND table_name = 'sprints'
    ) INTO has_sprints;
    SELECT c.oid INTO projects_oid
    FROM pg_class c
    JOIN pg_namespace n ON n.oid = c.relnamespace
    WHERE n.nspname = current_schema() AND c.relname = 'projects';

    -- Preflight every failure condition before executing the first persistent DDL.
    IF has_hours <> has_hours_check OR has_hours <> has_status_check THEN
        RAISE EXCEPTION 'incompatible partial story estimated-hours schema';
    END IF;

    IF has_hours THEN
        IF NOT EXISTS (
            SELECT 1 FROM information_schema.columns
            WHERE table_schema = current_schema() AND table_name = 'stories'
              AND column_name = 'estimated_hours' AND data_type = 'numeric'
              AND numeric_precision = 7 AND numeric_scale = 2 AND is_nullable = 'YES'
        ) THEN
            RAISE EXCEPTION 'incompatible estimated_hours column';
        END IF;
        SELECT pg_get_constraintdef(c.oid) INTO constraint_definition
        FROM pg_constraint c
        JOIN pg_class t ON t.oid = c.conrelid
        JOIN pg_namespace n ON n.oid = t.relnamespace
        WHERE n.nspname = current_schema() AND t.relname = 'stories'
          AND c.conname = 'stories_estimated_hours_positive';
        IF regexp_replace(
            regexp_replace(lower(constraint_definition), '::(numeric|text)', '', 'g'),
            '[[:space:]()]', '', 'g'
        ) <> 'checkestimated_hours>0' THEN
            RAISE EXCEPTION 'incompatible stories_estimated_hours_positive';
        END IF;
        SELECT pg_get_constraintdef(c.oid) INTO constraint_definition
        FROM pg_constraint c
        JOIN pg_class t ON t.oid = c.conrelid
        JOIN pg_namespace n ON n.oid = t.relnamespace
        WHERE n.nspname = current_schema() AND t.relname = 'stories'
          AND c.conname = 'stories_status_check';
        IF regexp_replace(
            regexp_replace(lower(constraint_definition), '::(numeric|text)', '', 'g'),
            '[[:space:]()]', '', 'g'
        ) <> 'checkstatus=anyarray[''pendiente'',''en_progreso'',''completada'']' THEN
            RAISE EXCEPTION 'incompatible stories_status_check';
        END IF;
    ELSE
        IF EXISTS (SELECT 1 FROM stories WHERE status NOT IN ('pendiente', 'en_progreso', 'completada')) THEN
            RAISE EXCEPTION 'incompatible legacy story status';
        END IF;
    END IF;

    IF has_sprints AND (
        NOT EXISTS (
            SELECT 1 FROM information_schema.columns
            WHERE table_schema = current_schema() AND table_name = 'sprints'
              AND column_name = 'id' AND data_type = 'uuid' AND is_nullable = 'NO'
        ) OR NOT EXISTS (
            SELECT 1 FROM information_schema.columns
            WHERE table_schema = current_schema() AND table_name = 'sprints'
              AND column_name = 'project_id' AND data_type = 'uuid' AND is_nullable = 'NO'
        ) OR NOT EXISTS (
            SELECT 1 FROM information_schema.columns
            WHERE table_schema = current_schema() AND table_name = 'sprints'
              AND column_name = 'sprint_goal' AND data_type = 'text' AND is_nullable = 'NO'
        ) OR NOT EXISTS (
            SELECT 1 FROM pg_constraint c
            JOIN pg_class t ON t.oid = c.conrelid
            JOIN pg_namespace n ON n.oid = t.relnamespace
            WHERE n.nspname = current_schema() AND t.relname = 'sprints'
              AND c.contype = 'p'
        ) OR NOT EXISTS (
            SELECT 1 FROM pg_constraint c
            JOIN pg_class t ON t.oid = c.conrelid
            JOIN pg_namespace n ON n.oid = t.relnamespace
            WHERE n.nspname = current_schema() AND t.relname = 'sprints'
              AND c.contype = 'f' AND c.confrelid = projects_oid
              AND c.confdeltype = 'r'
        )
    ) THEN
        RAISE EXCEPTION 'incompatible pre-existing sprints schema';
    END IF;

    -- All known validation failures have now been checked; only now perform DDL.
    IF NOT has_hours THEN
        ALTER TABLE stories
            ADD COLUMN estimated_hours NUMERIC(7,2) NULL
            CONSTRAINT stories_estimated_hours_positive CHECK (estimated_hours > 0);
        ALTER TABLE stories ADD CONSTRAINT stories_status_check
            CHECK (status IN ('pendiente', 'en_progreso', 'completada'));
    END IF;

    IF NOT has_sprints THEN
        CREATE TABLE sprints (
            id UUID PRIMARY KEY,
            project_id UUID NOT NULL CONSTRAINT sprints_project_id_fkey REFERENCES projects(id) ON DELETE RESTRICT,
            sprint_goal TEXT NOT NULL
        );
    END IF;
END $$;
