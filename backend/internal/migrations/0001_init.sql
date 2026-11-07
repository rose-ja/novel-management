-- +goose Up
-- 基线迁移：启用中文子串检索依赖的扩展（V-01 的实测在 CHG-009 完成）。
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- +goose Down
-- 扩展可能被其他对象依赖，基线迁移不删除扩展。
SELECT 1;
