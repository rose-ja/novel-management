package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"novel-management/backend/internal/model"
	"novel-management/backend/internal/search"
)

// Review 处理可校对实体的单条与批量确认。
type Review struct {
	db  *gorm.DB
	now func() time.Time
}

func NewReview(db *gorm.DB) *Review {
	return &Review{db: db, now: time.Now}
}

var reviewTables = map[string]string{
	"characters":   "characters",
	"factions":     "factions",
	"locations":    "locations",
	"rankings":     "rankings",
	"system-nodes": "system_nodes",
}

func reviewTable(resource string) (string, bool) {
	table, ok := reviewTables[strings.TrimSpace(resource)]
	return table, ok
}

// Confirm 确认单条实体已校对；已确认时幂等返回当前状态。
func (s *Review) Confirm(ctx context.Context, resource, id string) (*model.ReviewResult, error) {
	table, ok := reviewTable(resource)
	if !ok {
		return nil, &model.ValidationError{Field: "entity_type", Message: "不支持的校对类型"}
	}
	var result *model.ReviewResult
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row struct {
			ID           string
			WorkID       string
			ReviewStatus string
		}
		if err := tx.Table(table).Select("id, work_id, review_status").
			Where("id = ? AND deleted_at IS NULL", id).Take(&row).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return model.ErrReviewNotFound
			}
			return err
		}
		now := s.now().UTC()
		if row.ReviewStatus != model.ReviewStatusReviewed {
			if err := tx.Table(table).Where("id = ? AND deleted_at IS NULL", id).
				Updates(map[string]any{
					"review_status": model.ReviewStatusReviewed,
					"reviewed_at":   now,
					"reviewed_by":   model.LocalAuthor,
					"version":       gorm.Expr("version + 1"),
					"updated_at":    now,
				}).Error; err != nil {
				return err
			}
		}
		result = &model.ReviewResult{
			EntityType: resource, ID: id, ReviewStatus: model.ReviewStatusAPIConfirmed,
			ReviewedAt: &now, ReviewedBy: ptr(model.LocalAuthor),
		}
		var current struct {
			ReviewedAt *time.Time
			ReviewedBy *string
			Version    int
		}
		if err := tx.Table(table).Select("reviewed_at, reviewed_by, version").Where("id = ?", id).Take(&current).Error; err != nil {
			return err
		}
		if row.ReviewStatus == model.ReviewStatusReviewed {
			result.ReviewedAt = current.ReviewedAt
			result.ReviewedBy = current.ReviewedBy
		}
		result.Version = current.Version
		return search.NewRefresher(tx).RefreshWork(ctx, row.WorkID)
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// Batch 批量确认；已确认项计入 skipped，任一 ID 不存在则整批失败。
func (s *Review) Batch(ctx context.Context, input model.ReviewBatchInput) (*model.ReviewBatchResult, error) {
	table, ok := reviewTable(input.EntityType)
	if !ok {
		return nil, &model.ValidationError{Field: "entity_type", Message: "不支持的校对类型"}
	}
	ids := uniqueStrings(input.IDs)
	if len(ids) == 0 && (input.WorkID == nil || *input.WorkID == "") {
		return nil, &model.ValidationError{Field: "ids", Message: "请选择要确认的条目"}
	}
	result := &model.ReviewBatchResult{Items: []model.ReviewResult{}}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if len(ids) == 0 {
			query := tx.Table(table).Where("work_id = ? AND deleted_at IS NULL AND review_status = ?",
				*input.WorkID, model.ReviewStatusPending)
			if input.CategoryID != nil && *input.CategoryID != "" {
				query = query.Where("category_id = ?", *input.CategoryID)
			}
			if err := query.Pluck("id", &ids).Error; err != nil {
				return err
			}
			if len(ids) == 0 {
				return nil
			}
		}
		var rows []struct {
			ID           string
			WorkID       string
			ReviewStatus string
		}
		if err := tx.Table(table).Select("id, work_id, review_status").
			Where("id IN ? AND deleted_at IS NULL", ids).Find(&rows).Error; err != nil {
			return err
		}
		if len(rows) != len(ids) && len(input.IDs) > 0 {
			return model.ErrReviewNotFound
		}
		works := map[string]bool{}
		pending := make([]string, 0, len(rows))
		for _, row := range rows {
			works[row.WorkID] = true
			if row.ReviewStatus != model.ReviewStatusReviewed {
				pending = append(pending, row.ID)
			} else {
				result.Skipped++
			}
		}
		now := s.now().UTC()
		if len(pending) > 0 {
			if err := tx.Table(table).Where("id IN ? AND deleted_at IS NULL", pending).
				Updates(map[string]any{
					"review_status": model.ReviewStatusReviewed,
					"reviewed_at":   now,
					"reviewed_by":   model.LocalAuthor,
					"version":       gorm.Expr("version + 1"),
					"updated_at":    now,
				}).Error; err != nil {
				return err
			}
			result.Updated = len(pending)
		}
		var updated []struct {
			ID           string
			WorkID       string
			ReviewStatus string
			ReviewedAt   *time.Time
			ReviewedBy   *string
			Version      int
		}
		if err := tx.Table(table).Select("id, work_id, review_status, reviewed_at, reviewed_by, version").
			Where("id IN ?", ids).Find(&updated).Error; err != nil {
			return err
		}
		for _, row := range updated {
			result.Items = append(result.Items, model.ReviewResult{
				EntityType: input.EntityType, ID: row.ID,
				ReviewStatus: model.APIReviewStatus(row.ReviewStatus),
				ReviewedAt:   row.ReviewedAt, ReviewedBy: row.ReviewedBy, Version: row.Version,
			})
		}
		for workID := range works {
			if err := search.NewRefresher(tx).RefreshWork(ctx, workID); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}
