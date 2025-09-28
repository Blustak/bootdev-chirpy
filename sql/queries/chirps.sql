-- name: AddChirp :one
INSERT INTO chirps(id,created_at,updated_at,body,user_id) VALUES(
    gen_random_uuid(),
    NOW(),
    NOW(),
    @chirp_body,
    @id
) RETURNING *;

-- name: GetAllChirps :many
SELECT * FROM chirps ORDER BY created_at ASC;
