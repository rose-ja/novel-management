package imports

import (
	"archive/zip"
	"bytes"
	"embed"
	"image"
	"image/color"
	"image/png"
)

//go:embed template-v1.0.md example-v1.0.md
var templateFiles embed.FS

// Markdown 返回固定模板 v1.0 的下载内容。
func Markdown() []byte {
	data, _ := templateFiles.ReadFile("template-v1.0.md")
	return data
}

// ExampleMarkdown 返回可解析的示例 Markdown。
func ExampleMarkdown() []byte {
	data, _ := templateFiles.ReadFile("example-v1.0.md")
	return data
}

// ExampleZip 生成包含示例 Markdown 与一张 PNG 的 zip。
func ExampleZip() ([]byte, error) {
	pngData, err := examplePNG()
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	writer := zip.NewWriter(&buf)
	entries := []struct {
		name string
		data []byte
	}{
		{"example.md", ExampleMarkdown()},
		{"images/example.png", pngData},
	}
	for _, entry := range entries {
		file, err := writer.Create(entry.name)
		if err != nil {
			return nil, err
		}
		if _, err := file.Write(entry.data); err != nil {
			return nil, err
		}
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func examplePNG() ([]byte, error) {
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			img.Set(x, y, color.RGBA{R: 120, G: 160, B: 200, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
