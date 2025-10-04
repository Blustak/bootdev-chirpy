-- name: CreateUser :one
INSERT INTO users(id, created_at, updated_at, email,hashed_password) VALUES(
    gen_random_uuid(),
    NOW(),
    NOW(),
    $1,
    $2
) RETURNING id,created_at,updated_at,email,is_chirpy_red;

-- name: ResetUserTable :exec
DELETE FROM users;

-- name: GetUserByEmail :one
SELECT id,created_at,updated_at,email,is_chirpy_red FROM users WHERE email = @email;

-- name: GetHashedPasswordByID :one
SELECT hashed_password FROM users WHERE id = @id;

-- name: UpdateUser :one
UPDATE users
SET email = @email,
hashed_password = @hashed_password,
updated_at = NOW()
WHERE id = @user_id
RETURNING id,created_at,updated_at,email,is_chirpy_red;

-- name: UpgradeUserToChirpyRed :exec
UPDATE USERS
SET is_chirpy_red = TRUE
WHERE id = @user_id;
