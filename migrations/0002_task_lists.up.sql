CREATE TABLE task_lists (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX task_lists_user_id_created_at_idx ON task_lists (user_id, created_at DESC);

ALTER TABLE tasks ADD COLUMN list_id BIGINT REFERENCES task_lists (id) ON DELETE CASCADE;

-- Backfill: give every user with existing tasks a default list and assign
-- their tasks to it.
INSERT INTO task_lists (user_id, name)
SELECT DISTINCT user_id, 'My Tasks' FROM tasks;

UPDATE tasks
SET list_id = task_lists.id
FROM task_lists
WHERE tasks.user_id = task_lists.user_id AND tasks.list_id IS NULL;

ALTER TABLE tasks ALTER COLUMN list_id SET NOT NULL;

CREATE INDEX tasks_list_id_created_at_idx ON tasks (list_id, created_at DESC);
