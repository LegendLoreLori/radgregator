-- name: CreateFeedFollow :one
WITH i_feed_follow AS (
	INSERT INTO feed_follows (id, created_at, updated_at, user_id, feed_id)
	VALUES (
		$1,
		$2,
		$3,
		$4,
		$5
	)
	RETURNING *
)
SELECT i_feed_follow.*, 
users.name as user_name, 
feeds.name as feed_name 
FROM i_feed_follow
INNER JOIN users 
ON i_feed_follow.user_id = users.id
INNER JOIN feeds
ON i_feed_follow.feed_id = feeds.id;

-- name: GetFeedFollowsForUser :many
SELECT users.name as user_name, feeds.name as feed_name 
FROM feed_follows
INNER JOIN users
ON feed_follows.user_id = users.id
INNER JOIN feeds
ON feed_follows.feed_id = feeds.id
WHERE users.name = $1;
