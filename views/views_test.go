//	go test ./views/ -v

package views

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRenderHtml(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	RenderHtml(c, http.StatusOK, IndexPage())

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	if !strings.HasPrefix(w.Header().Get("Content-Type"), "text/html") {
		t.Errorf("content type = %q", w.Header().Get("Content-Type"))
	}
}

func TestHeadingCountsTotal(t *testing.T) {
	h := HeadingCounts{1, 2, 0, 0, 0, 1}
	if h.Total() != 4 {
		t.Errorf("Total = %d, want 4", h.Total())
	}
}

func TestLinkStatsTotal(t *testing.T) {
	l := LinkStats{Internal: 3, External: 2, Inaccessible: 1}
	if l.Total() != 5 {
		t.Errorf("Total = %d, want 5", l.Total())
	}
}

func TestResultPage(t *testing.T) {
	var buf bytes.Buffer
	ResultPage(Result{Title: "Example"}).Render(context.Background(), &buf)

	if !strings.Contains(buf.String(), "Example") {
		t.Error("title is missing")
	}
}

func TestErrorPage(t *testing.T) {
	var buf bytes.Buffer
	ErrorPage("", 404, "The page does not exist.", "status 404").
		Render(context.Background(), &buf)

	if !strings.Contains(buf.String(), "404 Not Found") {
		t.Error("status is missing")
	}
}
