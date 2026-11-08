-- +goose Up
-- CHG-002：作品、内置类别、系统设置、快照记录与撤销令牌。
-- 作品采用软删除三件套（deleted_at / deleted_root_id / deleted_by_action），
-- 回收站只展示「根本体」记录，因此 deleted_root_id 必须自指。

CREATE TABLE works (
    id                 TEXT PRIMARY KEY,
    name               TEXT NOT NULL,
    name_key           TEXT NOT NULL,
    description        TEXT,
    version            INTEGER NOT NULL DEFAULT 1 CHECK (version >= 1),
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at         TIMESTAMPTZ,
    deleted_root_id    TEXT,
    deleted_by_action  TEXT,
    CONSTRAINT works_name_len CHECK (char_length(name) BETWEEN 1 AND 200),
    CONSTRAINT works_description_len CHECK (description IS NULL OR char_length(description) <= 20000),
    CONSTRAINT works_deleted_consistent CHECK (
        (deleted_at IS NULL AND deleted_root_id IS NULL AND deleted_by_action IS NULL)
        OR (deleted_at IS NOT NULL AND deleted_root_id IS NOT NULL)
    )
);

-- 仅未删除记录受名称唯一约束；软删除后同名可重建。
CREATE UNIQUE INDEX works_name_key_uq ON works (name_key) WHERE deleted_at IS NULL;
CREATE INDEX works_deleted_at_idx ON works (deleted_at) WHERE deleted_at IS NOT NULL;

-- 内置类别：本变更只建最小结构，供创建作品时原子写入「境界」。
CREATE TABLE system_categories (
    id                 TEXT PRIMARY KEY,
    work_id            TEXT NOT NULL REFERENCES works (id) ON DELETE CASCADE,
    name               TEXT NOT NULL,
    name_key           TEXT NOT NULL,
    category_type      TEXT NOT NULL DEFAULT 'realm',
    is_builtin         BOOLEAN NOT NULL DEFAULT false,
    is_enabled         BOOLEAN NOT NULL DEFAULT true,
    version            INTEGER NOT NULL DEFAULT 1 CHECK (version >= 1),
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at         TIMESTAMPTZ,
    deleted_root_id    TEXT,
    deleted_by_action  TEXT
);

CREATE UNIQUE INDEX system_categories_work_name_uq
    ON system_categories (work_id, name_key) WHERE deleted_at IS NULL;
CREATE INDEX system_categories_work_idx ON system_categories (work_id) WHERE deleted_at IS NULL;

-- 单行设置表：当前作品切换即落库，不依赖会话态。
CREATE TABLE app_settings (
    id                 SMALLINT PRIMARY KEY CHECK (id = 1),
    current_work_id    TEXT REFERENCES works (id) ON DELETE SET NULL,
    last_export_at     TIMESTAMPTZ,
    last_restore_at    TIMESTAMPTZ,
    snapshot_keep      SMALLINT NOT NULL DEFAULT 3,
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO app_settings (id) VALUES (1);

-- 快照记录：目录与数据库记录一一对应，status=failed 的记录不被保留策略删除。
CREATE TABLE snapshots (
    id                 TEXT PRIMARY KEY,
    reason             TEXT NOT NULL CHECK (reason IN ('work_delete', 'trash_purge', 'purge_all', 'import', 'cli_restore')),
    path               TEXT NOT NULL,
    db_bytes           BIGINT NOT NULL DEFAULT 0,
    image_count        INTEGER NOT NULL DEFAULT 0,
    status             TEXT NOT NULL DEFAULT 'ok' CHECK (status IN ('ok', 'failed')),
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX snapshots_created_at_idx ON snapshots (created_at DESC);

-- 撤销令牌：一次性、8 秒有效；过期记录在后续撤销时顺带清理。
CREATE TABLE undo_tokens (
    token              TEXT PRIMARY KEY,
    root_type          TEXT NOT NULL,
    root_id            TEXT NOT NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at         TIMESTAMPTZ NOT NULL,
    used_at            TIMESTAMPTZ
);

CREATE INDEX undo_tokens_expires_at_idx ON undo_tokens (expires_at);

-- +goose Down
DROP TABLE IF EXISTS undo_tokens;
DROP TABLE IF EXISTS snapshots;
DROP TABLE IF EXISTS app_settings;
DROP TABLE IF EXISTS system_categories;
DROP TABLE IF EXISTS works;
