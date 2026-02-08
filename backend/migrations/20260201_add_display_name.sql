-- +goose Up
-- Add display_name column to spotify_tokens table
ALTER TABLE spotify_tokens ADD COLUMN display_name TEXT NOT NULL DEFAULT '';

-- Add display_name column to session_participants table
ALTER TABLE session_participants ADD COLUMN display_name TEXT NOT NULL DEFAULT '';

-- +goose Down
-- Remove display_name column from session_participants table
ALTER TABLE session_participants DROP COLUMN display_name;

-- Remove display_name column from spotify_tokens table
ALTER TABLE spotify_tokens DROP COLUMN display_name;
