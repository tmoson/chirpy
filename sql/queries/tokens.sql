-- name: StoreToken :one
INSERT INTO refresh_tokens(token, expires_at, user_id)
VALUES(
  $1,
  $2,
  $3
)
RETURNING *;

-- name: RevokeToken :exec
UPDATE refresh_tokens
SET revoked_at = now(), updated_at = now()
WHERE token = $1;

-- name: GetToken :one
SELECT * FROM refresh_tokens
WHERE token = $1 AND revoked_at IS NULL;


-- name: GetUserIDFromRefreshToken :one
SELECT
    user_id
FROM refresh_tokens
WHERE token = $1;
