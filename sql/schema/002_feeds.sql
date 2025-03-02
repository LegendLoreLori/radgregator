-- +goose Up
CREATE TABLE feeds (
	id UUID PRIMARY KEY,
	created_at TIMESTAMP,
	updated_at TIMESTAMP,
	name TEXT NOT NULL,
	url TEXT NOT NULL,
	user_id UUID NOT NULL,
	FOREIGN KEY (user_id) REFERENCES users (id)
	ON DELETE CASCADE
);

-- +goose Down
DROP TABLE feeds;
	
