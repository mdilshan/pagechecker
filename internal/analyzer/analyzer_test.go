package analyzer

import (
	"errors"
	"io"
	"maps"
	"net/url"
	"strings"
	"testing"
	"time"
)

const samplePage = `<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="utf-8">
	<title>Sample Page</title>
</head>
<body>
	<!-- a comment -->
	<h1>Main heading</h1>
	<h2 class="sub">Sub heading</h2>
	<a href="/about">About</a>
	<a href=":nohost">Contact Me</a>
	<a href="https://example.com">External</a>

	<br/>
	<form action="/login" method="post">
		<input type="text" name="user">
		<input type="password" name="pass">
		<button>Login</button>
	</form>
</body>
</html>`

func streamPage(page string, chunk int, failWith error) io.Reader {
	pr, pw := io.Pipe()

	go func() {
		for i := 0; i < len(page); i += chunk {
			if failWith != nil && i >= len(page)/2 {
				pw.CloseWithError(failWith)
				return
			}
			end := min(i+chunk, len(page))
			if _, err := pw.Write([]byte(page[i:end])); err != nil {
				return // reader side closed
			}
			time.Sleep(time.Millisecond)
		}
		pw.Close() // reader gets io.EOF
	}()

	return pr
}

func runAnalyze(t *testing.T, r io.Reader, shouldErr bool) {
	t.Helper()
	base, _ := url.Parse("https://example.org/docs/page.html")
	pData, err := Analyze(r, base)

	if err != nil && !shouldErr {
		t.Fatalf("Expected no error but got error %q", err)
	}

	t.Log(pData)
}

// go test ./internal/analyzer/ -v -run TestAnalyze_Stream
func TestAnalyze_Stream(t *testing.T) {
	runAnalyze(t, streamPage(samplePage, 16, nil), false)
}

// go test ./internal/analyzer/ -v -run TestAnalyze
func TestAnalyze_1StreamBreaks(t *testing.T) {
	broken := errors.New("connection reset")
	runAnalyze(t, streamPage(samplePage, 16, broken), true)
}

// go test ./internal/analyzer/ -v -run TestAnalyze_HeadingsAndLogin
func TestAnalyze_HeadingsAndLogin(t *testing.T) {
	base, _ := url.Parse("https://example.org/")

	tests := []struct {
		name     string
		page     string
		headings map[string]int
		hasLogin bool
	}{
		{
			name: "headings and password field",
			page: `<h1>a</h1><h2>b</h2><h2>c</h2><H6>d</H6>
				<form><input type="text"><input type="PASSWORD"></form>`,
			headings: map[string]int{"h1": 1, "h2": 2, "h6": 1},
			hasLogin: true,
		},
		{
			name:     "no password field",
			page:     `<h3>x</h3><form><input type="search"></form>`,
			headings: map[string]int{"h3": 1},
			hasLogin: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := Analyze(strings.NewReader(tt.page), base)
			if err != nil {
				t.Fatal(err)
			}
			if !maps.Equal(data.Headings, tt.headings) {
				t.Errorf("Headings = %v, want %v", data.Headings, tt.headings)
			}
			if data.HasLoginForm != tt.hasLogin {
				t.Errorf("HasLoginForm = %v, want %v",
					data.HasLoginForm, tt.hasLogin)
			}
		})
	}
}
