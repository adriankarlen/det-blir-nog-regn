package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "time/tzdata" // Europe/Stockholm must resolve even without system zoneinfo

	"golang.org/x/net/html"
)

const (
	svtVaderURL    = "https://www.svt.se/vader/vader-idag"
	requestTimeout = 15 * time.Second
	userAgent      = "det-blir-nog-regn/1.0 (weather image fetcher)"

	// From this hour (Swedish time) the I KVÄLL map is preferred over I DAG.
	// SVT drops the I DAG section during the afternoon.
	eveningFromHour = 13
)

// section is one forecast block on the page, e.g. <div id="i-dag-0"><h2>I DAG</h2>.
// The number after the id prefix is the block's position and changes during the day.
type section struct {
	idPrefix string
	heading  string
}

var (
	sectionIdag   = section{idPrefix: "i-dag-", heading: "I DAG"}
	sectionIkvall = section{idPrefix: "i-kvall-", heading: "I KVÄLL"}
)

// sectionsFor returns the sections to try, in order, for the given time:
// I DAG before eveningFromHour Swedish time, otherwise I KVÄLL. The other
// section is kept as a fallback in case the preferred one is missing.
func sectionsFor(now time.Time) []section {
	if loc, err := time.LoadLocation("Europe/Stockholm"); err == nil {
		now = now.In(loc)
	}
	if now.Hour() < eveningFromHour {
		return []section{sectionIdag, sectionIkvall}
	}
	return []section{sectionIkvall, sectionIdag}
}

func main() {
	outputDir := flag.String("output-dir", ".", "Directory to save the image")
	flag.Parse()

	if err := run(context.Background(), *outputDir); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, outputDir string) error {
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	doc, err := fetchAndParse(ctx, svtVaderURL)
	if err != nil {
		return err
	}

	now := time.Now()
	var sec section
	var imgURL string
	for _, candidate := range sectionsFor(now) {
		if imgURL = findSectionImage(doc, candidate); imgURL != "" {
			sec = candidate
			break
		}
	}
	if imgURL == "" {
		return fmt.Errorf("could not find weather image: no div#i-dag-*/div#i-kvall-* with a <noscript> img in page structure")
	}
	fmt.Printf("section: %s\n", sec.heading)

	base := outputBase(outputDir, now)

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("creating output dir: %w", err)
	}

	if err := downloadImage(ctx, imgURL, base+".jpg"); err != nil {
		return err
	}

	// The summary is a nice-to-have: a missing piece of text should not
	// fail the run once the image has been saved.
	headline := findHeadline(doc)
	paragraph := findSectionParagraph(doc, sec)
	if headline == "" || paragraph == "" {
		fmt.Fprintf(os.Stderr, "warning: could not find summary text (headline found: %t, %s paragraph found: %t)\n", headline != "", sec.heading, paragraph != "")
		return nil
	}

	summary := formatSummary(headline, paragraph)
	summaryPath := base + ".txt"
	if err := os.WriteFile(summaryPath, []byte(summary), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "warning: writing summary: %v\n", err)
		return nil
	}

	fmt.Printf("saved: %s\n\n%s", summaryPath, summary)
	return nil
}

// outputBase returns the output path without extension, e.g. dir/vader_2026-10-09.
func outputBase(outputDir string, now time.Time) string {
	return filepath.Join(outputDir, "vader_"+now.Format("2006-01-02"))
}

func newRequest(ctx context.Context, url string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	return req, nil
}

func fetchAndParse(ctx context.Context, url string) (*html.Node, error) {
	req, err := newRequest(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status from %s: %s", url, resp.Status)
	}

	doc, err := html.Parse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parsing HTML: %w", err)
	}
	return doc, nil
}

func downloadImage(ctx context.Context, imgURL, path string) error {
	req, err := newRequest(ctx, imgURL)
	if err != nil {
		return fmt.Errorf("creating image request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("downloading image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("image download status: %s", resp.Status)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return fmt.Errorf("writing image: %w", err)
	}

	fmt.Printf("saved: %s\n", path)
	return nil
}

// findHeadline returns the article heading. It prefers the <h1> whose class
// starts with "TextArticle__heading" (the suffix is a generated hash) and
// falls back to the og:title meta tag.
func findHeadline(doc *html.Node) string {
	h1 := findNode(doc, func(n *html.Node) bool {
		return n.Type == html.ElementNode && n.Data == "h1" && hasClassPrefix(n, "TextArticle__heading")
	})
	if h1 != nil {
		if text := normalizeSpace(extractText(h1)); text != "" {
			return text
		}
	}

	meta := findNode(doc, func(n *html.Node) bool {
		return n.Type == html.ElementNode && n.Data == "meta" && attr(n, "property") == "og:title"
	})
	if meta != nil {
		return normalizeSpace(attr(meta, "content"))
	}
	return ""
}

// findSectionParagraph finds the section's <h2> heading (inside its div when
// present) and returns the text of the first <p> that follows it.
func findSectionParagraph(doc *html.Node, sec section) string {
	scope := findNodeByIDPrefix(doc, sec.idPrefix)
	if scope == nil {
		scope = doc
	}

	var result string
	seenHeading := false
	var walk func(*html.Node) bool
	walk = func(n *html.Node) bool {
		if n.Type == html.ElementNode {
			switch {
			case n.Data == "h2" && !seenHeading:
				if strings.EqualFold(normalizeSpace(extractText(n)), sec.heading) {
					seenHeading = true
					return false // skip the heading's own children
				}
			case n.Data == "h2" && seenHeading:
				return true // next section reached without a paragraph
			case n.Data == "p" && seenHeading:
				if text := normalizeSpace(extractText(n)); text != "" {
					result = text
					return true
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if walk(c) {
				return true
			}
		}
		return false
	}
	walk(scope)
	return result
}

// formatSummary returns the headline, a blank line and the paragraph.
func formatSummary(headline, paragraph string) string {
	return headline + "\n\n" + paragraph + "\n"
}

// findNode performs a depth-first search for the first node matching match.
func findNode(n *html.Node, match func(*html.Node) bool) *html.Node {
	if match(n) {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if found := findNode(c, match); found != nil {
			return found
		}
	}
	return nil
}

func attr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

func hasClassPrefix(n *html.Node, prefix string) bool {
	for _, class := range strings.Fields(attr(n, "class")) {
		if strings.HasPrefix(class, prefix) {
			return true
		}
	}
	return false
}

// normalizeSpace trims and collapses all runs of whitespace into single spaces.
func normalizeSpace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// findSectionImage finds the section's div (e.g. div#i-dag-0), then extracts
// the first real image URL from a <noscript> fallback img tag within it.
func findSectionImage(doc *html.Node, sec section) string {
	div := findNodeByIDPrefix(doc, sec.idPrefix)
	if div == nil {
		return ""
	}
	return findNoscriptImgSrc(div)
}

// findNodeByIDPrefix returns the first element whose id starts with prefix.
func findNodeByIDPrefix(doc *html.Node, prefix string) *html.Node {
	return findNode(doc, func(n *html.Node) bool {
		return n.Type == html.ElementNode && strings.HasPrefix(attr(n, "id"), prefix)
	})
}

// findNodeByID performs a depth-first search for a node with the given id attribute.
func findNodeByID(n *html.Node, id string) *html.Node {
	if n.Type == html.ElementNode {
		for _, a := range n.Attr {
			if a.Key == "id" && a.Val == id {
				return n
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if found := findNodeByID(c, id); found != nil {
			return found
		}
	}
	return nil
}

// findNoscriptImgSrc finds a <noscript> element within n, parses its text
// content as HTML, and returns the first non-placeholder img src.
func findNoscriptImgSrc(n *html.Node) string {
	if n.Type == html.ElementNode && n.Data == "noscript" {
		text := extractText(n)
		return parseImgSrc(text)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if src := findNoscriptImgSrc(c); src != "" {
			return src
		}
	}
	return ""
}

func extractText(n *html.Node) string {
	var sb strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			sb.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return sb.String()
}

// parseImgSrc parses an HTML fragment and returns the src attribute of the
// first <img> element that isn't a data URI placeholder.
func parseImgSrc(fragment string) string {
	doc, err := html.Parse(strings.NewReader(fragment))
	if err != nil {
		return ""
	}
	return findImgSrc(doc)
}

// findImgSrc walks the tree looking for an <img> with a real src.
func findImgSrc(n *html.Node) string {
	if n.Type == html.ElementNode && n.Data == "img" {
		for _, a := range n.Attr {
			if a.Key == "src" && !strings.Contains(a.Val, "data:image") {
				return a.Val
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if src := findImgSrc(c); src != "" {
			return src
		}
	}
	return ""
}
