package service

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"

	"novel-management/backend/internal/model"
)

func testImage(width, height int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 255), G: uint8(y % 255), B: 100, A: 255})
		}
	}
	return img
}

func TestDetectMime(t *testing.T) {
	cases := []struct {
		name string
		data []byte
		want string
	}{
		{"jpeg", []byte{0xFF, 0xD8, 0xFF, 0xE0}, "image/jpeg"},
		{"png", []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}, "image/png"},
		{"webp", []byte("RIFF1234WEBP"), "image/webp"},
		{"text", []byte("hello"), ""},
		{"gif", []byte("GIF89a"), ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := detectMime(tc.data); got != tc.want {
				t.Fatalf("detectMime = %q，期望 %q", got, tc.want)
			}
		})
	}
}

func TestEncodeImageDownscalesAndThumbnail(t *testing.T) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, testImage(1200, 600)); err != nil {
		t.Fatalf("生成测试图失败：%v", err)
	}
	original, thumb, err := encodeImage(buf.Bytes())
	if err != nil {
		t.Fatalf("encodeImage 失败：%v", err)
	}
	if original.mime != "image/png" || original.ext != ".png" {
		t.Fatalf("PNG 输出应为 image/png/.png，实际 %s/%s", original.mime, original.ext)
	}
	if original.width != 1200 || original.height != 600 {
		t.Fatalf("原图尺寸应保持 1200x600，实际 %dx%d", original.width, original.height)
	}
	if thumb.width != model.ThumbnailLongEdge || thumb.height != model.ThumbnailLongEdge/2 {
		t.Fatalf("缩略图应保持比例 320x160，实际 %dx%d", thumb.width, thumb.height)
	}
}

func TestEncodeImageRejectsInvalid(t *testing.T) {
	if _, _, err := encodeImage([]byte("not an image")); err != model.ErrUnsupportedFileType {
		t.Fatalf("文本应返回 UNSUPPORTED_FILE_TYPE，实际 %v", err)
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, testImage(8, 8), nil); err != nil {
		t.Fatalf("生成 JPEG 失败：%v", err)
	}
	if _, _, err := encodeImage(buf.Bytes()); err != nil {
		t.Fatalf("合法 JPEG 不应失败：%v", err)
	}
}

func TestSafePathRejectsEscape(t *testing.T) {
	service := NewMedia(nil, t.TempDir(), nil)
	if _, err := service.safePath("../secret"); err == nil {
		t.Fatal("应拒绝越界路径")
	}
	if _, err := service.safePath("/etc/passwd"); err == nil {
		t.Fatal("应拒绝绝对路径")
	}
	if _, err := service.safePath("images/w/x.png"); err != nil {
		t.Fatalf("合法相对路径不应失败：%v", err)
	}
}
