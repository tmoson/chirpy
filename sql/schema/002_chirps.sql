-- +goose Up
CREATE TABLE chirps(
  id UUID PRIMARY KEY,
  created_at TIMESTAMP DEFAULT now() NOT NULL,
  updated_at TIMESTAMP DEFAULT now() NOT NULL,
  body TEXT NOT NULL,
  user_id UUID NOT NULL,
  CONSTRAINT fk_user_id
  FOREIGN KEY (user_id)
  REFERENCES users(id)
  ON DELETE CASCADE
);

-- +goose Down
DROP TABLE chirps
