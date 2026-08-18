// Package html renders a compiled Lawbook to a human-readable HTML document.
// It operates on the Lawbook IR only, never on Markdown directly (PLAN1
// §22-§23).
package html

import (
	"bytes"
	"fmt"
	"html"
	"io"

	"github.com/yuin/goldmark"

	"github.com/athreyac4/agentlaws/internal/model"
)

const style = `<style>
body{font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;max-width:860px;margin:2rem auto;padding:0 1rem;color:#1e1e1e;line-height:1.55}
h1{border-bottom:1px solid #ddd;padding-bottom:.5rem}
.section-id{color:#767676;font-family:ui-monospace,Menlo,monospace;font-size:.85rem;margin-top:-.5rem}
ol.laws{padding-left:1.4rem}
ol.laws>li{margin:.4rem 0}
ol.laws>li p{display:inline;margin:0}
.law-number{color:#098658;font-family:ui-monospace,Menlo,monospace;margin-right:.4rem}
</style>`

// Render writes the HTML representation of book to w.
func Render(w io.Writer, book model.Lawbook) error {
	fmt.Fprintf(w, "<!doctype html>\n<html><head><meta charset=\"utf-8\"><title>%s</title>%s</head><body>\n",
		html.EscapeString(book.Metadata.Title), style)
	fmt.Fprintf(w, "<h1>%s</h1>\n", html.EscapeString(book.Metadata.Title))

	for _, s := range book.Sections {
		level := min(s.Level+1, 6)
		fmt.Fprintf(w, "<h%d id=%q>%s %s</h%d>\n", level, html.EscapeString(s.ID),
			html.EscapeString(s.Number), html.EscapeString(s.Title), level)
		fmt.Fprintf(w, "<p class=\"section-id\">%s</p>\n", html.EscapeString(s.ID))

		if s.Commentary != "" {
			var buf bytes.Buffer
			if err := goldmark.Convert([]byte(s.Commentary), &buf); err != nil {
				return err
			}
			if _, err := w.Write(buf.Bytes()); err != nil {
				return err
			}
		}

		if len(s.Laws) > 0 {
			fmt.Fprint(w, "<ol class=\"laws\">\n")
			for _, law := range s.Laws {
				var buf bytes.Buffer
				if err := goldmark.Convert([]byte(law.Text), &buf); err != nil {
					return err
				}
				fmt.Fprintf(w, "<li id=%q><span class=\"law-number\">%s</span>%s</li>\n",
					html.EscapeString(law.Number), html.EscapeString(law.Number), buf.String())
			}
			fmt.Fprint(w, "</ol>\n")
		}
	}

	fmt.Fprint(w, "</body></html>\n")
	return nil
}
