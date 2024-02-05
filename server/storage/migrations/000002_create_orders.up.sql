BEGIN TRANSACTION;

CREATE TABLE IF NOT EXISTS public.orders
(
    id serial PRIMARY KEY,
    created_at TIMESTAMP DEFAULT NOW() NOT NULL,
    user_id int NOT NULL,
    number TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'NEW',
    accrual float DEFAULT 0
);

CREATE UNIQUE INDEX IF NOT EXISTS unique_order_number
    ON public.orders(number);


COMMIT ;