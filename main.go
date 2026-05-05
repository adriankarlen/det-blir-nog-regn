package main

import (
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

func main() {
  outputDir := flag.String("output-dir", ".", "Directory to save the image")
  flag.Parse()

  if err := run(*outputDir); err != nil {
    fmt.Fprintf(os.Stderr, "error: %v\n", err)
    os.Exit(1)
  }
}

func run(outputDir string) error {
  resp, err := http.Get("https://www.svt.se/vader/vader-idag")
  if err != nil {
    return fmt.Errorf("fetching page: %w", err)
  }
  defer resp.Body.Close()

  if resp.StatusCode != http.StatusOK {
    return fmt.Errorf("unexpected status: %s", resp.Status)
  }

  doc, err := html.Parse(resp.Body)
  if err != nil {
    return fmt.Errorf("parsing HTML: %w", err)
  }

  imgURL := findIdagImage(doc)
  if imgURL == "" {
    return fmt.Errorf("could not find 'idag' image on page")
  }

  imgResp, err := http.Get(imgURL)
  if err != nil {
    return fmt.Errorf("downloading image: %w", err)
  }
  defer imgResp.Body.Close()

  if imgResp.StatusCode != http.StatusOK {
    return fmt.Errorf("image download status: %s", imgResp.Status)
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

  if _, err := io.Copy(f, imgResp.Body); err != nil {
    return fmt.Errorf("writing image: %w", err)
  }

  fmt.Printf("saved: %s\n", path)
  return nil
}

// findIdagImage walks the HTML tree, finds the div#i-dag-0, then extracts
// the first real image URL from a <noscript> fallback img tag within it.
func findIdagImage(doc *html.Node) string {
  var idagDiv *html.Node
  var findDiv func(*html.Node)
  findDiv = func(n *html.Node) {
    if idagDiv != nil {
      return
    }
    if n.Type == html.ElementNode && n.Data == "div" {
      for _, a := range n.Attr {
        if a.Key == "id" && a.Val == "i-dag-0" {
          idagDiv = n
          return
        }
      }
    }
    for c := n.FirstChild; c != nil; c = c.NextSibling {
      findDiv(c)
    }
  }
  findDiv(doc)

  if idagDiv == nil {
    return ""
  }

  // Find <noscript> within idagDiv, parse its text content for an img src
  var result string
  var findImg func(*html.Node)
  findImg = func(n *html.Node) {
    if result != "" {
      return
    }
    if n.Type == html.ElementNode && n.Data == "noscript" {
      // noscript content is raw text in the parsed tree
      text := extractText(n)
      if src := extractSrcFromNoscript(text); src != "" {
        result = src
        return
      }
    }
    for c := n.FirstChild; c != nil; c = c.NextSibling {
      findImg(c)
    }
  }
  findImg(idagDiv)

  return result
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

func extractSrcFromNoscript(text string) string {
  // Look for src="..." in the noscript raw text
  idx := strings.Index(text, `src="`)
  if idx == -1 {
    return ""
  }
  start := idx + len(`src="`)
  end := strings.Index(text[start:], `"`)
  if end == -1 {
    return ""
  }
  src := text[start : start+end]
  // Skip placeholder gifs
  if strings.Contains(src, "data:image") {
    return ""
  }
  return src
}
