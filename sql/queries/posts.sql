-- name: CreatePost :one
INSERT INTO posts (id, created_at, updated_at, title, url, description, published_at, feed_id)
VALUES (
	$1,
	$2,
	$3,
	$4,
	$5,
	$6,
	$7,
	$8
)
RETURNING *;

-- name: GetPostsForUser :many
SELECT posts.title, posts.url, posts.description, posts.published_at, feeds.name FROM posts
INNER JOIN feeds
ON posts.feed_id = $1
LIMIT $2;
