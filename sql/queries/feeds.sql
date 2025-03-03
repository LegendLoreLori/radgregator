-- name: CreateFeed :one
INSERT INTO feeds(id, created_at, updated_at, name, url, user_id)
VALUES (
	$1,
	$2,
	$3,
	$4,
	$5,
	$6
)
RETURNING *;

-- name: GetFeeds :many
SELECT * FROM feeds
ORDER BY created_at;

-- name: GetFeedsUsers :many
SELECT feeds.name, feeds.url, users.name as user_name FROM feeds
LEFT JOIN users
ON feeds.user_id = users.id;

-- name: GetFeed :one
SELECT *  FROM feeds
WHERE url = $1;
