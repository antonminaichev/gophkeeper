DROP INDEX IF EXISTS uq_items_owner_human_id;
DROP INDEX IF EXISTS uq_items_owner_alias;
DROP INDEX IF EXISTS idx_items_owner_updated;
DROP INDEX IF EXISTS idx_items_owner_deleted;

DROP TABLE IF EXISTS items;
DROP TABLE IF EXISTS items_counter;