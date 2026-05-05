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

	"golang.org/x/net/html"
)

const (
	svtVaderURL    = "https://www.svt.se/vader/vader-idag"
	requestTimeout = 15 * time.Second
	userAgent      = "det-blir-nog-regn/1.0 (weather image fetcher)"
)

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

	imgURL := findIdagImage(doc)
	if imgURL == "" {
		return fmt.Errorf("could not find weather image: div#i-dag-0 or <noscript> img not found in page structure")
	}

	return downloadImage(ctx, imgURL, outputDir)
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

func downloadImage(ctx context.Context, imgURL, outputDir string) error {
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

	filename := fmt.Sprintf("vader_%s.jpg", time.Now().Format("2006-01-02"))
	path := filepath.Join(outputDir, filename)

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("creating output dir: %w", err)
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

// findIdagImage walks the HTML tree, finds div#i-dag-0, then extracts
// the first real image URL from a <noscript> fallback img tag within it.
func findIdagImage(doc *html.Node) string {
	idagDiv := findNodeByID(doc, "i-dag-0")
	if idagDiv == nil {
		return ""
	}
	return findNoscriptImgSrc(idagDiv)
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
