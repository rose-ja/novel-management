-- +goose Up
-- CHG-010：导出任务记录。导出在进程内后台执行，状态落库供设置页轮询。
CREATE TABLE export_runs (
    id            TEXT PRIMARY KEY,
    status        TEXT NOT NULL DEFAULT 'queued'
                  CHECK (status IN ('queued', 'running', 'completed', 'failed')),
    include_trash BOOLEAN NOT NULL DEFAULT true,
    file_name     TEXT,
    file_path     TEXT,
    byte_size     BIGINT CHECK (byte_size IS NULL OR byte_size >= 0),
    error_code    TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    finished_at   TIMESTAMPTZ
);

CREATE INDEX export_runs_status_created_idx ON export_runs (status, created_at DESC);
CREATE INDEX export_runs_created_idx ON export_runs (created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS export_runs;
