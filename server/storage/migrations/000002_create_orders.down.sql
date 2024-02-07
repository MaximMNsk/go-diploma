BEGIN TRANSACTION;

DROP INDEX IF EXISTS unique_order_number;
DROP INDEX IF EXISTS hash_order_status;
DROP TABLE IF EXISTS public.orders;

COMMIT ;