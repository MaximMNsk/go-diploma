BEGIN TRANSACTION;

CREATE TABLE IF NOT EXISTS public.users
(
    id serial PRIMARY KEY,
    login  TEXT  NOT NULL,
    password_hash  TEXT  NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS unique_login
    ON public.users(login);

COMMIT ;