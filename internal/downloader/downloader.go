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

type Document struct {
	Url       string
	Status    int
	Body      []byte
	Truncated bool
}

var client = &http.Client{Timeout: 30 * time.Second}

const ACCEPT = "text/html"

const maxBodySize = 5 << 20 // 5 MiB

func (d *Document) DownloadDoc(
	ctx context.Context, url string,
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

	// read one extra byte to know if the body was cut off
	data, err := io.ReadAll(io.LimitReader(body, maxBodySize+1))
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}

	d.Truncated = len(data) > maxBodySize
	if d.Truncated {
		data = data[:maxBodySize]
	}
	d.Body = data

	return nil
}
