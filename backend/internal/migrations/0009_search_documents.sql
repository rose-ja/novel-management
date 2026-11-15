-- +goose Up
-- CHG-009：检索文档。反规范化当前作品五类实体的可检索文本，配合 pg_trgm GIN 索引。
CREATE TABLE search_documents (
    id            TEXT PRIMARY KEY,
    work_id       TEXT NOT NULL REFERENCES works (id) ON DELETE CASCADE,
    entity_type   TEXT NOT NULL
                  CHECK (entity_type IN ('character', 'faction', 'location', 'ranking', 'system_node')),
    entity_id     TEXT NOT NULL,
    name_text     TEXT NOT NULL,
    name_key      TEXT NOT NULL,
    alias_text    TEXT,
    title_text    TEXT,
    desc_text     TEXT,
    relation_text TEXT,
    review_status TEXT NOT NULL DEFAULT 'reviewed'
                  CHECK (review_status IN ('reviewed', 'pending')),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    is_deleted    BOOLEAN NOT NULL DEFAULT false,
    CONSTRAINT search_entity_uq UNIQUE (work_id, entity_type, entity_id)
);

CREATE INDEX search_work_type_idx
    ON search_documents (work_id, entity_type, is_deleted);

CREATE INDEX search_name_trgm_idx
    ON search_documents USING gin (name_text gin_trgm_ops) WHERE is_deleted = false;
CREATE INDEX search_alias_trgm_idx
    ON search_documents USING gin (alias_text gin_trgm_ops) WHERE is_deleted = false;
CREATE INDEX search_title_trgm_idx
    ON search_documents USING gin (title_text gin_trgm_ops) WHERE is_deleted = false;
CREATE INDEX search_desc_trgm_idx
    ON search_documents USING gin (desc_text gin_trgm_ops) WHERE is_deleted = false;
CREATE INDEX search_relation_trgm_idx
    ON search_documents USING gin (relation_text gin_trgm_ops) WHERE is_deleted = false;

-- +goose Down
DROP TABLE IF EXISTS search_documents;
