-- +goose Up
-- CHG-004：人物档案核心。人物软删除三件套与作品一致，回收站只展示根记录。
-- character_system_refs 在 0003 仅作骨架且无生产数据，补外键前先清空（设计 D5）。
DELETE FROM character_system_refs;

CREATE TABLE characters (
    id                 TEXT PRIMARY KEY,
    work_id            TEXT NOT NULL REFERENCES works (id) ON DELETE CASCADE,
    name               TEXT NOT NULL,
    name_key           TEXT NOT NULL,
    description        TEXT,
    faction_id         TEXT,
    image_path         TEXT,
    review_status      TEXT NOT NULL DEFAULT 'reviewed'
                       CHECK (review_status IN ('reviewed', 'pending')),
    reviewed_at        TIMESTAMPTZ,
    reviewed_by        TEXT,
    version            INTEGER NOT NULL DEFAULT 1 CHECK (version >= 1),
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at         TIMESTAMPTZ,
    deleted_root_id    TEXT,
    deleted_by_action  TEXT,
    CONSTRAINT characters_name_len CHECK (char_length(name) BETWEEN 1 AND 200),
    CONSTRAINT characters_desc_len CHECK (description IS NULL OR char_length(description) <= 20000),
    CONSTRAINT characters_deleted_consistent CHECK (
        (deleted_at IS NULL AND deleted_root_id IS NULL AND deleted_by_action IS NULL)
        OR (deleted_at IS NOT NULL AND deleted_root_id IS NOT NULL)
    )
);

CREATE UNIQUE INDEX characters_active_name_uq
    ON characters (work_id, name_key) WHERE deleted_at IS NULL;
CREATE INDEX characters_list_idx
    ON characters (work_id, updated_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX characters_deleted_root_idx
    ON characters (deleted_root_id) WHERE deleted_root_id IS NOT NULL;

CREATE TABLE character_names (
    id           TEXT PRIMARY KEY,
    character_id TEXT NOT NULL REFERENCES characters (id) ON DELETE CASCADE,
    work_id      TEXT NOT NULL REFERENCES works (id) ON DELETE CASCADE,
    kind         TEXT NOT NULL CHECK (kind IN ('alias', 'title')),
    value        TEXT NOT NULL,
    value_key    TEXT NOT NULL,
    sort_order   INTEGER NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT character_names_value_len CHECK (char_length(value) BETWEEN 1 AND 100)
);

CREATE UNIQUE INDEX character_names_uq
    ON character_names (character_id, kind, value_key);
CREATE INDEX character_names_search_idx ON character_names (work_id, value_key);

ALTER TABLE character_system_refs
    ADD CONSTRAINT character_system_refs_character_fk
    FOREIGN KEY (character_id) REFERENCES characters (id) ON DELETE CASCADE;

-- +goose Down
ALTER TABLE character_system_refs DROP CONSTRAINT IF EXISTS character_system_refs_character_fk;
DELETE FROM character_system_refs;
DROP TABLE IF EXISTS character_names;
DROP TABLE IF EXISTS characters;
