package analyzer

import (
	"errors"
	"io"
	"net/url"
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
