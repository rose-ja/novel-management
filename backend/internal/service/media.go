package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
	"gorm.io/gorm"

	"novel-management/backend/internal/model"
	"novel-management/backend/internal/repository"
)

// Media 负责图片校验、重编码、缩略图与文件生命周期。
type Media struct {
	db      *gorm.DB
	dataDir string
	log     *slog.Logger
	now     func() time.Time
}

func NewMedia(db *gorm.DB, dataDir string, log *slog.Logger) *Media {
	if log == nil {
		log = slog.Default()
	}
	return &Media{db: db, dataDir: dataDir, log: log, now: time.Now}
}

type encodedImage struct {
	data   []byte
	mime   string
	ext    string
	width  int
	height int
}

// UploadCharacterImage 校验并重编码图片，替换人物当前主图。
func (s *Media) UploadCharacterImage(ctx context.Context, characterID string, reader io.Reader, size int64) (*model.MediaView, error) {
	if size <= 0 || size > model.MaxImageBytes {
		return nil, model.ErrFileTooLarge
	}
	raw, err := io.ReadAll(io.LimitReader(reader, model.MaxImageBytes+1))
	if err != nil {
		return nil, err
	}
	if len(raw) > model.MaxImageBytes {
		return nil, model.ErrFileTooLarge
	}

	character, err := repository.NewCharacters(s.db).Get(ctx, characterID)
	if err != nil {
		return nil, err
	}

	original, thumb, err := encodeImage(raw)
	if err != nil {
		return nil, err
	}

	dir := filepath.Join(s.dataDir, "images", character.WorkID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	assetID := model.NewID()
	originalName := assetID + original.ext
	thumbName := assetID + "_thumb" + thumb.ext
	originalRel := filepath.ToSlash(filepath.Join("images", character.WorkID, originalName))
	thumbRel := filepath.ToSlash(filepath.Join("images", character.WorkID, thumbName))

	if err := writeAtomic(filepath.Join(dir, originalName), original.data); err != nil {
		return nil, err
	}
	if err := writeAtomic(filepath.Join(dir, thumbName), thumb.data); err != nil {
		_ = os.Remove(filepath.Join(dir, originalName))
		return nil, err
	}

	sum := sha256.Sum256(original.data)
	now := s.now().UTC()
	asset := &model.MediaAsset{
		ID:           assetID,
		WorkID:       character.WorkID,
		EntityType:   model.MediaEntityCharacter,
		EntityID:     character.ID,
		OriginalPath: originalRel,
		ThumbPath:    &thumbRel,
		Mime:         original.mime,
		ByteSize:     int64(len(original.data)),
		Width:        &original.width,
		Height:       &original.height,
		SHA256:       hex.EncodeToString(sum[:]),
		Version:      1,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	var old []model.MediaAsset
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 锁住人物行，避免与人物删除/作品整组删除并发时产生「已删人物下的活跃主图」。
		locked, err := repository.NewCharacters(tx).GetForUpdate(ctx, characterID)
		if err != nil {
			return err
		}
		if locked.WorkID != character.WorkID {
			return model.ErrNotFound
		}
		repo := repository.NewMedia(tx)
		if previous, err := repo.GetActiveForUpdate(ctx, model.MediaEntityCharacter, character.ID); err == nil {
			old = append(old, *previous)
			if err := tx.Model(&model.MediaAsset{}).
				Where("id = ? AND deleted_at IS NULL", previous.ID).
				Updates(map[string]any{
					"deleted_at":        now,
					"deleted_root_id":   character.ID,
					"deleted_by_action": model.MediaActionReplace,
					"updated_at":        now,
				}).Error; err != nil {
				return err
			}
		} else if !errors.Is(err, model.ErrNotFound) {
			return err
		}
		return repo.Insert(ctx, asset)
	})
	if err != nil {
		_ = os.Remove(filepath.Join(dir, originalName))
		_ = os.Remove(filepath.Join(dir, thumbName))
		return nil, err
	}
	s.removeAssets(old)
	return mediaView(asset), nil
}

// DeleteCharacterImage 软删除人物主图并在提交后清理文件。
func (s *Media) DeleteCharacterImage(ctx context.Context, characterID string) error {
	now := s.now().UTC()
	var removed *model.MediaAsset
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repo := repository.NewMedia(tx)
		item, err := repo.GetActiveForUpdate(ctx, model.MediaEntityCharacter, characterID)
		if err != nil {
			return err
		}
		removed = item
		return tx.Model(&model.MediaAsset{}).
			Where("id = ? AND deleted_at IS NULL", item.ID).
			Updates(map[string]any{
				"deleted_at":        now,
				"deleted_root_id":   characterID,
				"deleted_by_action": model.MediaActionDelete,
				"updated_at":        now,
			}).Error
	})
	if err != nil {
		return err
	}
	if removed != nil {
		s.removeAssets([]model.MediaAsset{*removed})
	}
	return nil
}

// ServeAsset 返回资产与磁盘绝对路径；variant 为 thumb 时回退原图。
func (s *Media) ServeAsset(ctx context.Context, assetID, variant string) (*model.MediaAsset, string, error) {
	asset, err := repository.NewMedia(s.db).GetActiveByID(ctx, assetID)
	if err != nil {
		return nil, "", err
	}
	rel := asset.OriginalPath
	if variant == "thumb" && asset.ThumbPath != nil && *asset.ThumbPath != "" {
		rel = *asset.ThumbPath
	}
	path, err := s.safePath(rel)
	if err != nil {
		return nil, "", err
	}
	return asset, path, nil
}

// GetCharacterImage 返回人物当前主图；无图返回 nil。
func (s *Media) GetCharacterImage(ctx context.Context, characterID string) (*model.MediaView, error) {
	asset, err := repository.NewMedia(s.db).GetActive(ctx, model.MediaEntityCharacter, characterID)
	if errors.Is(err, model.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return mediaView(asset), nil
}

// ListCharacterImages 批量返回人物主图映射。
func (s *Media) ListCharacterImages(ctx context.Context, ids []string) (map[string]*model.MediaView, error) {
	assets, err := repository.NewMedia(s.db).ListActiveByEntityIDs(ctx, model.MediaEntityCharacter, ids)
	if err != nil {
		return nil, err
	}
	result := make(map[string]*model.MediaView, len(assets))
	for i := range assets {
		result[assets[i].EntityID] = mediaView(&assets[i])
	}
	return result, nil
}

// RemoveFiles 删除相对 DATA_DIR 的文件；越界路径被忽略。
func (s *Media) RemoveFiles(paths []string) {
	for _, rel := range paths {
		if strings.TrimSpace(rel) == "" {
			continue
		}
		path, err := s.safePath(rel)
		if err != nil {
			s.log.Warn("忽略越界图片路径", "path", rel, "error", err)
			continue
		}
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			s.log.Warn("删除图片文件失败", "path", rel, "error", err)
		}
	}
}

func (s *Media) removeAssets(items []model.MediaAsset) {
	paths := make([]string, 0, len(items)*2)
	for _, item := range items {
		paths = append(paths, item.OriginalPath)
		if item.ThumbPath != nil {
			paths = append(paths, *item.ThumbPath)
		}
	}
	s.RemoveFiles(paths)
}

func (s *Media) safePath(rel string) (string, error) {
	if filepath.IsAbs(rel) || strings.HasPrefix(rel, "/") || strings.HasPrefix(rel, `\`) || filepath.VolumeName(rel) != "" {
		return "", fmt.Errorf("禁止绝对路径")
	}
	clean := filepath.Clean(rel)
	if clean == "." || strings.HasPrefix(clean, "..") {
		return "", fmt.Errorf("非法路径")
	}
	path := filepath.Join(s.dataDir, clean)
	inside, err := filepath.Rel(s.dataDir, path)
	if err != nil || strings.HasPrefix(inside, "..") {
		return "", fmt.Errorf("路径越界")
	}
	return path, nil
}

func mediaView(asset *model.MediaAsset) *model.MediaView {
	width, height := 0, 0
	if asset.Width != nil {
		width = *asset.Width
	}
	if asset.Height != nil {
		height = *asset.Height
	}
	return &model.MediaView{
		AssetID:  asset.ID,
		URL:      "/api/v1/media/" + asset.ID,
		ThumbURL: "/api/v1/media/" + asset.ID + "?variant=thumb",
		Width:    width,
		Height:   height,
		ByteSize: asset.ByteSize,
	}
}

func writeAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	file, err := os.CreateTemp(dir, ".upload-*")
	if err != nil {
		return err
	}
	name := file.Name()
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		_ = os.Remove(name)
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		_ = os.Remove(name)
		return err
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(name)
		return err
	}
	if err := os.Rename(name, path); err != nil {
		_ = os.Remove(name)
		return err
	}
	return nil
}

func detectMime(raw []byte) string {
	switch {
	case len(raw) >= 3 && raw[0] == 0xFF && raw[1] == 0xD8 && raw[2] == 0xFF:
		return "image/jpeg"
	case len(raw) >= 8 && bytes.Equal(raw[:8], []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}):
		return "image/png"
	case len(raw) >= 12 && string(raw[0:4]) == "RIFF" && string(raw[8:12]) == "WEBP":
		return "image/webp"
	default:
		return ""
	}
}

func encodeImage(raw []byte) (encodedImage, encodedImage, error) {
	inputMime := detectMime(raw)
	if inputMime == "" {
		return encodedImage{}, encodedImage{}, model.ErrUnsupportedFileType
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		return encodedImage{}, encodedImage{}, model.ErrUnsupportedFileType
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || int64(cfg.Width)*int64(cfg.Height) > model.MaxImagePixels {
		return encodedImage{}, encodedImage{}, model.ErrImageTooLarge
	}
	src, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return encodedImage{}, encodedImage{}, model.ErrUnsupportedFileType
	}
	if longestEdge(src.Bounds()) > model.MaxImageDimension {
		src = scaleToLongEdge(src, model.MaxImageDimension)
	}

	outputMime, ext := "image/png", ".png"
	if inputMime == "image/jpeg" {
		outputMime, ext = "image/jpeg", ".jpg"
	}
	original, err := encode(src, outputMime)
	if err != nil {
		return encodedImage{}, encodedImage{}, err
	}
	thumbSrc := scaleToLongEdge(src, model.ThumbnailLongEdge)
	thumb, err := encode(thumbSrc, outputMime)
	if err != nil {
		return encodedImage{}, encodedImage{}, err
	}
	bounds := src.Bounds()
	thumbBounds := thumbSrc.Bounds()
	return encodedImage{
		data: original, mime: outputMime, ext: ext,
		width: bounds.Dx(), height: bounds.Dy(),
	}, encodedImage{
		data: thumb, mime: outputMime, ext: ext,
		width: thumbBounds.Dx(), height: thumbBounds.Dy(),
	}, nil
}

func encode(src image.Image, mime string) ([]byte, error) {
	var buf bytes.Buffer
	var err error
	if mime == "image/jpeg" {
		err = jpeg.Encode(&buf, src, &jpeg.Options{Quality: 85})
	} else {
		err = png.Encode(&buf, src)
	}
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func longestEdge(bounds image.Rectangle) int {
	if bounds.Dx() > bounds.Dy() {
		return bounds.Dx()
	}
	return bounds.Dy()
}

func scaleToLongEdge(src image.Image, edge int) image.Image {
	bounds := src.Bounds()
	long := longestEdge(bounds)
	if long <= edge {
		return src
	}
	width := bounds.Dx() * edge / long
	height := bounds.Dy() * edge / long
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, bounds, draw.Over, nil)
	return dst
}
