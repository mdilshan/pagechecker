package downloader

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"time"
)

// Document holds the response metadata. The body is not kept in memory,
// it is streamed to the read callback of DownloadDoc instead.
type Document struct {
	Url    string
	Status int
}

var client = &http.Client{}

const ACCEPT = "text/html"

var maxBodySize int // bytes, read stops after this

func Init(timeout time.Duration, maxBodyMB int) {
	client.Timeout = timeout
	maxBodySize = maxBodyMB << 20
}

// DownloadDoc fetches url and, if it is HTML, calls read with the body
// stream. The body is closed after read returns, so read must finish
// with it before returning.
func (d *Document) DownloadDoc(
	ctx context.Context, url string, read func(io.Reader) error,
) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("Accept", ACCEPT)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	d.Url = resp.Request.URL.String()
	d.Status = resp.StatusCode

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("status %d", resp.StatusCode)
	}

	var body io.Reader = resp.Body

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" { // server not sent it so let's try to mime sniff
		bodyHead := make([]byte, 512)
		n, err := io.ReadFull(resp.Body, bodyHead)
		if err != nil && err != io.ErrUnexpectedEOF {
			return fmt.Errorf("html detection: %w", err)
		}
		bodyHead = bodyHead[:n]
		contentType = http.DetectContentType(bodyHead)

		// put the sniffed bytes back in front of the body
		body = io.MultiReader(bytes.NewReader(bodyHead), resp.Body)
	}

	mType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return fmt.Errorf("mime detection %w", err)
	}
	if mType != ACCEPT {
		return fmt.Errorf("not HTML: %q", mType)
	}

	return read(io.LimitReader(body, int64(maxBodySize)))
}
