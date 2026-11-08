package httpapi

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestStaticHandlerServesIndexWithoutRedirect(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html>ok</html>"), 0o644); err != nil {
		t.Fatalf("准备静态文件失败：%v", err)
	}

	router := gin.New()
	router.NoRoute(staticHandler(dir))

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d", rec.Code)
	}
	if body := rec.Body.String(); body != "<html>ok</html>" {
		t.Fatalf("响应体不符：%q", body)
	}
}

func TestStaticHandlerRejectsTraversal(t *testing.T) {
	router := gin.New()
	router.NoRoute(staticHandler(t.TempDir()))

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/../../secret.txt", nil))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("期望 404，实际 %d", rec.Code)
	}
}

func TestStaticHandlerServesExportedPageAnd404(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"index.html":    "<html>home</html>",
		"settings.html": "<html>settings</html>",
		"404.html":      "<html>not-found</html>",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("准备静态文件失败：%v", err)
		}
	}

	router := gin.New()
	router.NoRoute(staticHandler(dir))

	cases := []struct {
		path     string
		status   int
		expected string
	}{
		{"/settings", http.StatusOK, "<html>settings</html>"},
		{"/missing", http.StatusNotFound, "<html>not-found</html>"},
	}
	for _, tc := range cases {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tc.path, nil))
		if rec.Code != tc.status {
			t.Fatalf("%s 期望 %d，实际 %d", tc.path, tc.status, rec.Code)
		}
		if rec.Body.String() != tc.expected {
			t.Fatalf("%s 响应体不符：%q", tc.path, rec.Body.String())
		}
		if contentType := rec.Header().Get("Content-Type"); contentType == "" {
			t.Fatalf("%s 缺少 Content-Type", tc.path)
		}
	}
}
