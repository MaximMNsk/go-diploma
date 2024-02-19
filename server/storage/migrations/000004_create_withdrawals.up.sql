BEGIN TRANSACTION;

CREATE TABLE IF NOT EXISTS public.withdrawals
(
    id serial PRIMARY KEY,
    user_id int NOT NULL,
    order_number TEXT not null,
    created_at TIMESTAMP DEFAULT NOW() NOT NULL,
    sum float DEFAULT 0
);

COMMIT ;