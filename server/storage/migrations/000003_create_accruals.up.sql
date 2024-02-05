BEGIN TRANSACTION;

CREATE TABLE IF NOT EXISTS public.accruals
(
    id serial PRIMARY KEY,
    user_id int NOT NULL,
    current_balance float DEFAULT 0,
    total_balance float DEFAULT 0
);

COMMIT ;