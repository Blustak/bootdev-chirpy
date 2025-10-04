-- name: AddRefreshToken :one
INSERT INTO refresh_tokens(
    token,
    created_at,
    updated_at,
    user_id,
    expires_at,
    revoked_at
    ) VALUES(
    @token,
    NOW(),
    NOW(),
    @user_id,
    NOW() + interval '60 days',
    NULL
) RETURNING *;


-- name: GetRefreshToken :one
SELECT token,expires_at,revoked_at FROM refresh_tokens WHERE token = @token;

-- name: RevokeRefreshToken :exec
UPDATE refresh_tokens SET revoked_at = NOW(), updated_at = NOW() WHERE token = @token;

-- name: GetUserByRefreshToken :one
SELECT users.id,
refresh_tokens.token,
refresh_tokens.expires_at,
refresh_tokens.revoked_at
FROM users
LEFT JOIN refresh_tokens
ON users.id = refresh_tokens.user_id
WHERE refresh_tokens.token = @token;
