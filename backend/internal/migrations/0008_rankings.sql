-- +goose Up
-- CHG-008：榜单与固定席位。
-- 榜单沿用软删除三件套；席位无独立删除列，可见性由榜单过滤（ADR-004）。
CREATE TABLE rankings (
    id                TEXT PRIMARY KEY,
    work_id           TEXT NOT NULL REFERENCES works (id) ON DELETE CASCADE,
    name              TEXT NOT NULL,
    name_key          TEXT NOT NULL,
    seat_count        INTEGER NOT NULL DEFAULT 100,
    review_status     TEXT NOT NULL DEFAULT 'reviewed'
                      CHECK (review_status IN ('reviewed', 'pending')),
    reviewed_at       TIMESTAMPTZ,
    reviewed_by       TEXT DEFAULT 'local-author',
    version           INTEGER NOT NULL DEFAULT 1 CHECK (version >= 1),
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at        TIMESTAMPTZ,
    deleted_root_id   TEXT,
    deleted_by_action TEXT,
    CONSTRAINT rankings_name_len CHECK (char_length(name) BETWEEN 1 AND 200),
    CONSTRAINT rankings_seat_count_range CHECK (seat_count BETWEEN 1 AND 200),
    CONSTRAINT rankings_deleted_consistent CHECK (
        (deleted_at IS NULL AND deleted_root_id IS NULL AND deleted_by_action IS NULL)
        OR (deleted_at IS NOT NULL AND deleted_root_id IS NOT NULL)
    )
);

CREATE UNIQUE INDEX rankings_active_name_uq
    ON rankings (work_id, name_key) WHERE deleted_at IS NULL;
CREATE INDEX rankings_active_work_idx
    ON rankings (work_id, updated_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX rankings_deleted_root_idx
    ON rankings (deleted_root_id) WHERE deleted_root_id IS NOT NULL;

CREATE TABLE ranking_seats (
    id           TEXT PRIMARY KEY,
    work_id      TEXT NOT NULL REFERENCES works (id) ON DELETE CASCADE,
    ranking_id   TEXT NOT NULL REFERENCES rankings (id) ON DELETE CASCADE,
    seat_no      INTEGER NOT NULL,
    character_id TEXT REFERENCES characters (id) ON DELETE RESTRICT,
    note         TEXT,
    version      INTEGER NOT NULL DEFAULT 1 CHECK (version >= 1),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT ranking_seats_no_range CHECK (seat_no BETWEEN 1 AND 200),
    CONSTRAINT ranking_seats_note_len CHECK (note IS NULL OR char_length(note) <= 500),
    CONSTRAINT seats_no_uq UNIQUE (ranking_id, seat_no)
);
CREATE UNIQUE INDEX seats_character_uq
    ON ranking_seats (ranking_id, character_id) WHERE character_id IS NOT NULL;
CREATE INDEX ranking_seats_character_idx
    ON ranking_seats (character_id) WHERE character_id IS NOT NULL;
CREATE INDEX ranking_seats_work_idx ON ranking_seats (work_id);

-- +goose Down
DROP TABLE IF EXISTS ranking_seats;
DROP TABLE IF EXISTS rankings;
