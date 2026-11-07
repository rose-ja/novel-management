package httpapi

import (
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

// staticHandler 托管 Next.js 静态导出产物。
// 支持 /search -> search.html、/foo/ -> foo/index.html 的静态路由约定。
func staticHandler(root string) gin.HandlerFunc {
	if strings.TrimSpace(root) == "" {
		return func(c *gin.Context) {
			abortError(c, http.StatusNotFound, "WEB_NOT_BUILT", "前端构建产物不存在")
		}
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return func(c *gin.Context) {
			abortError(c, http.StatusInternalServerError, "WEB_DIR_INVALID", "前端目录无效")
		}
	}

	return func(c *gin.Context) {
		// 统一斜杠并清理路径，防止 Windows 反斜杠穿越。
		raw := strings.ReplaceAll(c.Request.URL.Path, "\\", "/")
		rel := strings.TrimPrefix(path.Clean("/"+raw), "/")
		if rel == "" || rel == "." {
			rel = "index.html"
		}
		if strings.HasPrefix(rel, "..") || strings.ContainsRune(rel, 0) {
			abortError(c, http.StatusNotFound, "NOT_FOUND", "页面不存在")
			return
		}

		candidates := []string{rel}
		if !strings.Contains(path.Base(rel), ".") {
			candidates = append(candidates, rel+".html", path.Join(rel, "index.html"))
		}

		for _, candidate := range candidates {
			full := filepath.Join(absRoot, filepath.FromSlash(candidate))
			info, statErr := os.Stat(full)
			if statErr != nil || info.IsDir() {
				continue
			}
			// 直接读取文件内容，避免 FileServer 对 /index.html 做 301 造成重定向循环。
			file, openErr := os.Open(full)
			if openErr != nil {
				continue
			}
			http.ServeContent(c.Writer, c.Request, info.Name(), info.ModTime(), file)
			_ = file.Close()
			return
		}

		abortError(c, http.StatusNotFound, "NOT_FOUND", "页面不存在")
	}
}
