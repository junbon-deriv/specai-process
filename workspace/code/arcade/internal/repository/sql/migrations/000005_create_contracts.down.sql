-- Remove foreign key constraint from transactions
ALTER TABLE transactions DROP CONSTRAINT IF EXISTS fk_transactions_contract;

-- Drop contracts table
DROP TABLE IF EXISTS contracts CASCADE;
