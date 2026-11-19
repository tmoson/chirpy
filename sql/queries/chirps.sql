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
FROM chirps
ORDER BY
CASE WHEN @sort_dir::text = 'asc' THEN chirps.created_at END ASC,
CASE WHEN @sort_dir::text = 'desc' THEN chirps.created_at END DESC,
  chirps.created_at ASC;

-- name: GetPost :one
SELECT *
FROM chirps
WHERE id = $1;

-- name: GetAllPostsByAuthor :many
SELECT *
FROM chirps
WHERE user_id = $1
ORDER BY
CASE WHEN @sort_dir::text = 'asc' THEN chirps.created_at END ASC,
CASE WHEN @sort_dir::text = 'desc' THEN chirps.created_at END DESC,
  chirps.created_at ASC;

-- name: DeletePost :exec
DELETE FROM chirps WHERE id = $1;

-- name: ResetPosts :exec
DELETE FROM chirps *;
