-- +goose Up
-- CHG-011：飞书模板导入任务。上传与校验状态落库，提交在单事务内完成。
CREATE TABLE import_runs (
    id               TEXT PRIMARY KEY,
    status           TEXT NOT NULL DEFAULT 'uploaded'
                     CHECK (status IN ('uploaded', 'validating', 'validated', 'running', 'completed', 'failed')),
    stage            TEXT,
    markdown_name    TEXT NOT NULL,
    archive_name     TEXT,
    markdown_path    TEXT NOT NULL,
    archive_path     TEXT,
    markdown_bytes   BIGINT NOT NULL CHECK (markdown_bytes >= 0),
    archive_bytes    BIGINT CHECK (archive_bytes IS NULL OR archive_bytes >= 0),
    template_version TEXT,
    work_name        TEXT,
    work_id          TEXT,
    snapshot_id      TEXT,
    counts_json      TEXT,
    errors_json      TEXT,
    warnings_json    TEXT,
    images_json      TEXT,
    report_json      TEXT,
    error_count      INTEGER NOT NULL DEFAULT 0 CHECK (error_count >= 0),
    error_code       TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    finished_at      TIMESTAMPTZ
);

CREATE INDEX import_runs_status_created_idx ON import_runs (status, created_at DESC);
CREATE INDEX import_runs_created_idx ON import_runs (created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS import_runs;
