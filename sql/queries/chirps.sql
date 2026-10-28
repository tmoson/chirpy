-- name: CreatePost :one
INSERT INTO chirps (id, body, user_id)
VALUES(
  gen_random_uuid(),
  $1,
  $2
)
RETURNING *;

-- name: GetAllPosts :many
SELECT *
FROM chirps ORDER BY chirps.created_at;

-- name: GetPost :one
SELECT *
FROM chirps
WHERE id = $1;

-- name: DeletePost :exec
DELETE FROM chirps WHERE id = $1;

-- name: ResetPosts :exec
DELETE FROM chirps *;
