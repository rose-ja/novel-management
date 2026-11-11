-- +goose Up
-- CHG-005：人物主图与大事记。图片落库为重编码后的 jpeg/png；大事记为人物子内容。
ALTER TABLE characters DROP COLUMN IF EXISTS image_path;

CREATE TABLE media_assets (
    id                TEXT PRIMARY KEY,
    work_id           TEXT NOT NULL REFERENCES works (id) ON DELETE CASCADE,
    entity_type       TEXT NOT NULL CHECK (entity_type IN ('character')),
    entity_id         TEXT NOT NULL,
    original_path     TEXT NOT NULL,
    thumb_path        TEXT,
    mime              TEXT NOT NULL CHECK (mime IN ('image/jpeg', 'image/png')),
    byte_size         BIGINT NOT NULL CHECK (byte_size > 0 AND byte_size <= 10485760),
    width             INTEGER,
    height            INTEGER,
    sha256            CHAR(64) NOT NULL,
    version           INTEGER NOT NULL DEFAULT 1 CHECK (version >= 1),
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at        TIMESTAMPTZ,
    deleted_root_id   TEXT,
    deleted_by_action TEXT,
    CONSTRAINT media_deleted_consistent CHECK (
        (deleted_at IS NULL AND deleted_root_id IS NULL AND deleted_by_action IS NULL)
        OR (deleted_at IS NOT NULL AND deleted_root_id IS NOT NULL)
    )
);

CREATE UNIQUE INDEX media_one_per_entity_uq
    ON media_assets (entity_type, entity_id) WHERE deleted_at IS NULL;
CREATE INDEX media_work_deleted_idx ON media_assets (work_id, deleted_at);
CREATE INDEX media_deleted_root_idx
    ON media_assets (deleted_root_id) WHERE deleted_root_id IS NOT NULL;

CREATE TABLE chronicles (
    id                TEXT PRIMARY KEY,
    work_id           TEXT NOT NULL REFERENCES works (id) ON DELETE CASCADE,
    owner_type        TEXT NOT NULL CHECK (owner_type IN ('character')),
    owner_id          TEXT NOT NULL,
    text              TEXT NOT NULL,
    time_label        TEXT,
    sort_order        INTEGER NOT NULL DEFAULT 0,
    version           INTEGER NOT NULL DEFAULT 1 CHECK (version >= 1),
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at        TIMESTAMPTZ,
    deleted_root_id   TEXT,
    deleted_by_action TEXT,
    CONSTRAINT chronicles_text_len CHECK (char_length(text) BETWEEN 1 AND 2000),
    CONSTRAINT chronicles_time_label_len CHECK (time_label IS NULL OR char_length(time_label) <= 200),
    CONSTRAINT chronicles_deleted_consistent CHECK (
        (deleted_at IS NULL AND deleted_root_id IS NULL AND deleted_by_action IS NULL)
        OR (deleted_at IS NOT NULL AND deleted_root_id IS NOT NULL)
    )
);

CREATE INDEX chronicles_owner_idx ON chronicles (owner_type, owner_id, sort_order);
CREATE INDEX chronicles_work_idx ON chronicles (work_id);
CREATE INDEX chronicles_deleted_root_idx
    ON chronicles (deleted_root_id) WHERE deleted_root_id IS NOT NULL;

-- +goose Down
DROP TABLE IF EXISTS chronicles;
DROP TABLE IF EXISTS media_assets;
ALTER TABLE characters ADD COLUMN IF NOT EXISTS image_path TEXT;
