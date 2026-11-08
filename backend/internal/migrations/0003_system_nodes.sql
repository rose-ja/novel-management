-- +goose Up
-- CHG-003：体系类别类型收敛、两级节点与人物引用骨架。
-- 类别增加 sort_order（仅记录创建顺序，页面不提供自定义排序）。
-- system_nodes 采用与作品一致的软删除三件套，回收站只展示根记录。
-- character_system_refs 暂不建人物外键，CHG-004 建 characters 后追加。

ALTER TABLE system_categories ALTER COLUMN category_type SET DEFAULT 'normal';
UPDATE system_categories SET category_type = 'normal'
 WHERE category_type NOT IN ('normal', 'era');

ALTER TABLE system_categories
    ADD CONSTRAINT system_categories_type_check
    CHECK (category_type IN ('normal', 'era'));

ALTER TABLE system_categories
    ADD COLUMN sort_order INTEGER NOT NULL DEFAULT 0;

ALTER TABLE system_categories
    ADD CONSTRAINT system_categories_deleted_consistent CHECK (
        (deleted_at IS NULL AND deleted_root_id IS NULL AND deleted_by_action IS NULL)
        OR (deleted_at IS NOT NULL AND deleted_root_id IS NOT NULL)
    );

CREATE TABLE system_nodes (
    id                 TEXT PRIMARY KEY,
    work_id            TEXT NOT NULL REFERENCES works (id) ON DELETE CASCADE,
    category_id        TEXT NOT NULL REFERENCES system_categories (id) ON DELETE RESTRICT,
    parent_id          TEXT REFERENCES system_nodes (id) ON DELETE RESTRICT,
    name               TEXT NOT NULL,
    name_key           TEXT NOT NULL,
    sort_order         INTEGER NOT NULL DEFAULT 0,
    era_start_text     TEXT,
    era_end_text       TEXT,
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
    CONSTRAINT system_nodes_name_len CHECK (char_length(name) BETWEEN 1 AND 200),
    CONSTRAINT system_nodes_era_len CHECK (
        (era_start_text IS NULL OR char_length(era_start_text) <= 100)
        AND (era_end_text IS NULL OR char_length(era_end_text) <= 100)
    ),
    CONSTRAINT system_nodes_deleted_consistent CHECK (
        (deleted_at IS NULL AND deleted_root_id IS NULL AND deleted_by_action IS NULL)
        OR (deleted_at IS NOT NULL AND deleted_root_id IS NOT NULL)
    )
);

CREATE UNIQUE INDEX system_nodes_sibling_name_uq
    ON system_nodes (category_id, COALESCE(parent_id, ''), name_key)
    WHERE deleted_at IS NULL;

CREATE INDEX system_nodes_tree_idx
    ON system_nodes (category_id, parent_id, sort_order)
    WHERE deleted_at IS NULL;

CREATE INDEX system_nodes_deleted_root_idx
    ON system_nodes (deleted_root_id)
    WHERE deleted_root_id IS NOT NULL;

CREATE TABLE character_system_refs (
    id                 TEXT PRIMARY KEY,
    work_id            TEXT NOT NULL REFERENCES works (id) ON DELETE CASCADE,
    character_id       TEXT NOT NULL,
    category_id        TEXT NOT NULL REFERENCES system_categories (id) ON DELETE RESTRICT,
    node_id            TEXT NOT NULL REFERENCES system_nodes (id) ON DELETE RESTRICT,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX char_sys_uq
    ON character_system_refs (character_id, category_id);
CREATE INDEX char_sys_node_idx ON character_system_refs (node_id);
CREATE INDEX char_sys_category_idx ON character_system_refs (category_id);
CREATE INDEX char_sys_work_idx ON character_system_refs (work_id);

-- +goose Down
DROP TABLE IF EXISTS character_system_refs;
DROP TABLE IF EXISTS system_nodes;

ALTER TABLE system_categories DROP CONSTRAINT IF EXISTS system_categories_deleted_consistent;
ALTER TABLE system_categories DROP COLUMN IF EXISTS sort_order;
ALTER TABLE system_categories DROP CONSTRAINT IF EXISTS system_categories_type_check;
ALTER TABLE system_categories ALTER COLUMN category_type SET DEFAULT 'realm';
