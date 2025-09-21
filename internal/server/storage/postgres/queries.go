package postgres

// Снимок актуальных (не удалённых) элементов владельца.
const qCreateUser = `INSERT INTO users (id, email, pass_hash, pass_salt) VALUES ($1, $2, $3, $4)`

// Изменения после курсора (updated_at, id) включительно/исключительно.
const qGetUser = `SELECT id, email, pass_hash, pass_salt FROM users WHERE email = $1`

const qIsertItemCounter = `
		INSERT INTO items_counter(owner_id, next_id)
		VALUES ($1, 1)
		ON CONFLICT (owner_id)
		DO UPDATE SET next_id = items_counter.next_id + 1
		RETURNING next_id
	`

const qInsertItem = `
		INSERT INTO items (id, owner_id, human_id, alias, type, payload, meta)
		VALUES ($1, $2, $3, $4, $5, $6, COALESCE($7, '{}'::jsonb))
		RETURNING version
	`

const qSelectNonDeletedItemByID = `
		SELECT id, owner_id, human_id, alias, type, payload, meta, version,
		       deleted_at, updated_at, created_at
		FROM items
		WHERE id = $1 AND owner_id = $2 AND deleted_at IS NULL
	`

const qSelectNonDeletedItemByHuman = `
		SELECT id, owner_id, human_id, alias, type, payload, meta, version,
		       deleted_at, updated_at, created_at
		FROM items
		WHERE owner_id = $1 AND human_id = $2 AND deleted_at IS NULL
	`

const qSelectNonDeletedItemByAlias = `
		SELECT id, owner_id, human_id, alias, type, payload, meta, version,
		       deleted_at, updated_at, created_at
		FROM items
		WHERE owner_id = $1 AND alias = $2 AND deleted_at IS NULL
	`

const qListItems = `
		SELECT id, owner_id, human_id, alias, type, payload, meta, version,
		       deleted_at, updated_at, created_at
		FROM items
		WHERE owner_id = $1 AND deleted_at IS NULL
		ORDER BY updated_at DESC
		LIMIT $2
	`

const qListChanges = `
		SELECT id, owner_id, human_id, alias, type, payload, meta, version,
		       deleted_at, updated_at, created_at
		FROM items
		WHERE owner_id = $1
		  AND (
		       updated_at > $2
		       OR (updated_at = $2 AND id > $3)
		  )
		ORDER BY updated_at ASC, id ASC
		LIMIT $4
	`

const qUpdateItem = `
		UPDATE items
		SET payload   = COALESCE($4, payload),
		    meta      = COALESCE($5, meta),
		    version   = version + 1,
		    updated_at = now()
		WHERE id = $1
		  AND owner_id = $2
		  AND deleted_at IS NULL
		  AND version = $3
		RETURNING id, owner_id, human_id, alias, type, payload, meta, version,
		          deleted_at, updated_at, created_at
	`

const qDeletedItem = `
		UPDATE items
		SET deleted_at = $3,
		    updated_at = $3
		WHERE id = $1 AND owner_id = $2 AND deleted_at IS NULL
	`
