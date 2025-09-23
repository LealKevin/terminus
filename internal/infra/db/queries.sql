
-- name: CreateWorld :one
INSERT INTO worlds (id, width, height, layout)
VALUES ($1, $2, $3, $4)
RETURNING id, width, height, layout, created_at, updated_at;

-- name: GetWorldByID :one
SELECT id, width, height, layout, created_at, updated_at
FROM worlds
WHERE id = $1;

-- name: ListWorlds :many
SELECT id, width, height, layout, created_at, updated_at
FROM worlds
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: UpdateWorld :one
UPDATE worlds
SET width = $2, height = $3, layout = $4, updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING id, width, height, layout, created_at, updated_at;

-- name: DeleteWorld :exec
DELETE FROM worlds
WHERE id = $1;

-- name: CreatePlayer :one
INSERT INTO players (id, world_id, x, y, health, attack, defense, range)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, world_id, x, y, health, attack, defense, range, created_at, updated_at;

-- name: GetPlayerByID :one
SELECT id, world_id, x, y, health, attack, defense, range, created_at, updated_at
FROM players
WHERE id = $1;

-- name: UpdatePlayer :one
UPDATE players
SET world_id = $8, x = $2, y = $3, health = $4, attack = $5, defense = $6, range = $7, updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING id, world_id, x, y, health, attack, defense, range, created_at, updated_at;

-- name: DeletePlayer :exec
DELETE FROM players
WHERE id = $1;


