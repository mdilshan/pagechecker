package analyzer

import (
	"io"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

type PageData struct {
	HTMLVersion    string
	Title          string
	Links          []string
	MalformedLinks []string
}

// links will be only extracted from following configurations
var linkAttrs = map[string]string{
	"a":   "href",
	"img": "src",
}

// Here i would not pass the body []bytes
// On large user base it could exhause memory
// instead, read directly from the response stream
func Analyze(r io.Reader, base *url.URL) (PageData, error) {
	var data PageData
	inHead := false
	inTitle := false

	tokenizer := html.NewTokenizer(r)
	for {
		tk := tokenizer.Next()
		if tk == html.ErrorToken {
			if err := tokenizer.Err(); err != io.EOF {
				return data, err
			}
			return data, nil // done with data extraction
		}

		token := tokenizer.Token()

		switch tk {
		case html.DoctypeToken:
			data.HTMLVersion = doctypeVersion(token.Data)

		case html.StartTagToken, html.SelfClosingTagToken:
			switch token.Data {
			case "head":
				inHead = true
			case "title":
				inTitle = inHead // svgs can also have title. so need head tracking
			}

			if attr, ok := linkAttrs[token.Data]; ok {
				for _, a := range token.Attr {
					if a.Key != attr || a.Val == "" {
						continue
					}
					link, err := base.Parse(strings.TrimSpace(a.Val))
					if err != nil {
						data.MalformedLinks = append(data.MalformedLinks, a.Val)
						continue
					}
					data.Links = append(data.Links, link.String())
				}
			}

		case html.EndTagToken:
			switch token.Data {
			case "head":
				inHead = false
			case "title":
				inTitle = false
			}

		case html.TextToken:
			if inTitle {
				data.Title += token.Data
			}
		}
	}
}

func doctypeVersion(doctype string) string {
	// <!DOCTYPE html> is HTML5, other doctypes carry their version in
	// the content
	if strings.EqualFold(strings.TrimSpace(doctype), "html") {
		return "HTML5"
	}
	return doctype
}
