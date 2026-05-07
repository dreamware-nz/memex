package fetch

import (
	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
)

// HTMLToMarkdown converts an HTML byte slice to a Markdown string. Relative
// hrefs/srcs are resolved against baseURL when it is non-empty.
func HTMLToMarkdown(html []byte, baseURL string) (string, error) {
	if len(html) == 0 {
		return "", nil
	}

	var opts []converter.ConvertOptionFunc
	if baseURL != "" {
		opts = append(opts, converter.WithDomain(baseURL))
	}

	return htmltomarkdown.ConvertString(string(html), opts...)
}
