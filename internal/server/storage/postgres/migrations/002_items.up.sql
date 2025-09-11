CREATE TABLE IF NOT EXISTS items (
  id         UUID PRIMARY KEY,
  owner_id   UUID NOT NULL,

  human_id   BIGINT NOT NULL,
  alias      TEXT NULL,

  type       SMALLINT NOT NULL,
  payload    BYTEA NOT NULL,
  meta       JSONB NOT NULL DEFAULT '{}',
  version    BIGINT NOT NULL DEFAULT 1,

  deleted_at TIMESTAMPTZ NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_items_owner_human_id ON items(owner_id, human_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_items_owner_alias    ON items(owner_id, alias) WHERE alias IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_items_owner_updated ON items(owner_id, updated_at);
CREATE INDEX IF NOT EXISTS idx_items_owner_deleted ON items(owner_id, deleted_at);

CREATE TABLE IF NOT EXISTS items_counter (
  owner_id UUID PRIMARY KEY,
  next_id  BIGINT NOT NULL DEFAULT 1
);