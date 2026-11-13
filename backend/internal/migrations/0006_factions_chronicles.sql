-- +goose Up
-- CHG-006：势力树与势力大事记。势力自引用父级、首领为 0..1 人物，
-- 软删除三件套与人物一致；大事记复用 chronicles，仅放宽 owner_type。
CREATE TABLE factions (
    id                  TEXT PRIMARY KEY,
    work_id             TEXT NOT NULL REFERENCES works (id) ON DELETE CASCADE,
    name                TEXT NOT NULL,
    name_key            TEXT NOT NULL,
    parent_id           TEXT REFERENCES factions (id) ON DELETE RESTRICT,
    leader_character_id TEXT REFERENCES characters (id) ON DELETE RESTRICT,
    description         TEXT,
    review_status       TEXT NOT NULL DEFAULT 'reviewed'
                        CHECK (review_status IN ('reviewed', 'pending')),
    reviewed_at         TIMESTAMPTZ,
    reviewed_by         TEXT,
    version             INTEGER NOT NULL DEFAULT 1 CHECK (version >= 1),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at          TIMESTAMPTZ,
    deleted_root_id     TEXT,
    deleted_by_action   TEXT,
    CONSTRAINT factions_name_len CHECK (char_length(name) BETWEEN 1 AND 200),
    CONSTRAINT factions_desc_len CHECK (description IS NULL OR char_length(description) <= 20000),
    CONSTRAINT factions_deleted_consistent CHECK (
        (deleted_at IS NULL AND deleted_root_id IS NULL AND deleted_by_action IS NULL)
        OR (deleted_at IS NOT NULL AND deleted_root_id IS NOT NULL)
    )
);

CREATE UNIQUE INDEX factions_active_name_uq
    ON factions (work_id, name_key) WHERE deleted_at IS NULL;
CREATE INDEX factions_active_parent_idx
    ON factions (work_id, parent_id) WHERE deleted_at IS NULL;
CREATE INDEX factions_deleted_root_idx
    ON factions (deleted_root_id) WHERE deleted_root_id IS NOT NULL;

-- CHG-004 的 faction_id 仅为占位；历史脏值先置空，避免建外键失败。
UPDATE characters SET faction_id = NULL
 WHERE faction_id IS NOT NULL
   AND NOT EXISTS (SELECT 1 FROM factions f WHERE f.id = characters.faction_id);

ALTER TABLE characters
    ADD CONSTRAINT characters_faction_fk
    FOREIGN KEY (faction_id) REFERENCES factions (id) ON DELETE RESTRICT;
CREATE INDEX characters_faction_idx
    ON characters (work_id, faction_id) WHERE deleted_at IS NULL;

ALTER TABLE chronicles DROP CONSTRAINT IF EXISTS chronicles_owner_type_check;
ALTER TABLE chronicles
    ADD CONSTRAINT chronicles_owner_type_check
    CHECK (owner_type IN ('character', 'faction'));

-- +goose Down
DELETE FROM chronicles WHERE owner_type = 'faction';
ALTER TABLE chronicles DROP CONSTRAINT IF EXISTS chronicles_owner_type_check;
ALTER TABLE chronicles
    ADD CONSTRAINT chronicles_owner_type_check CHECK (owner_type IN ('character'));

ALTER TABLE characters DROP CONSTRAINT IF EXISTS characters_faction_fk;
DROP INDEX IF EXISTS characters_faction_idx;
DROP TABLE IF EXISTS factions;
