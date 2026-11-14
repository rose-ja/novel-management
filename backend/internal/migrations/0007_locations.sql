-- +goose Up
-- CHG-007：地点单棵树、人物-地点关系、地点-势力关系。
-- 地点沿用软删除三件套；关系表不加删除列，可见性由父实体过滤（ADR-004）。
CREATE TABLE locations (
    id                TEXT PRIMARY KEY,
    work_id           TEXT NOT NULL REFERENCES works (id) ON DELETE CASCADE,
    name              TEXT NOT NULL,
    name_key          TEXT NOT NULL,
    parent_id         TEXT REFERENCES locations (id) ON DELETE RESTRICT,
    type_text         TEXT,
    description       TEXT,
    review_status     TEXT NOT NULL DEFAULT 'reviewed'
                      CHECK (review_status IN ('reviewed', 'pending')),
    reviewed_at       TIMESTAMPTZ,
    reviewed_by       TEXT,
    version           INTEGER NOT NULL DEFAULT 1 CHECK (version >= 1),
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at        TIMESTAMPTZ,
    deleted_root_id   TEXT,
    deleted_by_action TEXT,
    CONSTRAINT locations_name_len CHECK (char_length(name) BETWEEN 1 AND 200),
    CONSTRAINT locations_type_len CHECK (type_text IS NULL OR char_length(type_text) <= 200),
    CONSTRAINT locations_desc_len CHECK (description IS NULL OR char_length(description) <= 20000),
    CONSTRAINT locations_deleted_consistent CHECK (
        (deleted_at IS NULL AND deleted_root_id IS NULL AND deleted_by_action IS NULL)
        OR (deleted_at IS NOT NULL AND deleted_root_id IS NOT NULL)
    )
);

CREATE UNIQUE INDEX locations_active_name_uq
    ON locations (work_id, name_key) WHERE deleted_at IS NULL;
CREATE INDEX locations_active_parent_idx
    ON locations (work_id, parent_id) WHERE deleted_at IS NULL;
CREATE INDEX locations_deleted_root_idx
    ON locations (deleted_root_id) WHERE deleted_root_id IS NOT NULL;

CREATE TABLE character_locations (
    id           TEXT PRIMARY KEY,
    work_id      TEXT NOT NULL REFERENCES works (id) ON DELETE CASCADE,
    character_id TEXT NOT NULL REFERENCES characters (id) ON DELETE CASCADE,
    location_id  TEXT NOT NULL REFERENCES locations (id) ON DELETE RESTRICT,
    note         TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT character_locations_note_len CHECK (note IS NULL OR char_length(note) <= 500),
    CONSTRAINT char_loc_uq UNIQUE (character_id, location_id)
);
CREATE INDEX character_locations_location_idx ON character_locations (location_id);

CREATE TABLE location_factions (
    id          TEXT PRIMARY KEY,
    work_id     TEXT NOT NULL REFERENCES works (id) ON DELETE CASCADE,
    location_id TEXT NOT NULL REFERENCES locations (id) ON DELETE CASCADE,
    faction_id  TEXT NOT NULL REFERENCES factions (id) ON DELETE RESTRICT,
    note        TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT location_factions_note_len CHECK (note IS NULL OR char_length(note) <= 500),
    CONSTRAINT loc_fac_uq UNIQUE (location_id, faction_id)
);
CREATE INDEX location_factions_faction_idx ON location_factions (faction_id);

-- +goose Down
DROP TABLE IF EXISTS location_factions;
DROP TABLE IF EXISTS character_locations;
DROP TABLE IF EXISTS locations;
