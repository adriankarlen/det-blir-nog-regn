package main

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/net/html"
)

func TestFindNodeByID(t *testing.T) {
	tests := []struct {
		name  string
		html  string
		id    string
		found bool
	}{
		{"found", `<div id="foo">bar</div>`, "foo", true},
		{"not found", `<div id="foo">bar</div>`, "baz", false},
		{"nested", `<div><span id="deep">x</span></div>`, "deep", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, _ := html.Parse(strings.NewReader(tt.html))
			got := findNodeByID(doc, tt.id)
			if (got != nil) != tt.found {
				t.Errorf("findNodeByID(%q) found=%v, want %v", tt.id, got != nil, tt.found)
			}
		})
	}
}

func TestParseImgSrc(t *testing.T) {
	tests := []struct {
		name     string
		fragment string
		want     string
	}{
		{
			"valid img",
			`<img src="https://example.com/image.jpg" alt="weather">`,
			"https://example.com/image.jpg",
		},
		{
			"skips data uri",
			`<img src="data:image/gif;base64,abc"><img src="https://real.com/img.jpg">`,
			"https://real.com/img.jpg",
		},
		{
			"no img",
			`<div>no image here</div>`,
			"",
		},
		{
			"empty",
			"",
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseImgSrc(tt.fragment)
			if got != tt.want {
				t.Errorf("parseImgSrc() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFindIdagImage(t *testing.T) {
	tests := []struct {
		name string
		html string
		want string
	}{
		{
			"full structure",
			`<html><body><div id="i-dag-0"><noscript><img src="https://svt.se/weather.jpg"></noscript></div></body></html>`,
			"https://svt.se/weather.jpg",
		},
		{
			"missing div",
			`<html><body><div id="other"><noscript><img src="https://x.com/y.jpg"></noscript></div></body></html>`,
			"",
		},
		{
			"no noscript",
			`<html><body><div id="i-dag-0"><p>hello</p></div></body></html>`,
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, _ := html.Parse(strings.NewReader(tt.html))
			got := findIdagImage(doc)
			if got != tt.want {
				t.Errorf("findIdagImage() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractText(t *testing.T) {
	input := `<div>hello <span>world</span></div>`
	doc, _ := html.Parse(strings.NewReader(input))
	// Find the div
	div := findNodeByID(doc, "")
	// Just test on the full doc — extractText should concat all text nodes
	got := extractText(doc)
	_ = div
	if !strings.Contains(got, "hello") || !strings.Contains(got, "world") {
		t.Errorf("extractText() = %q, expected to contain 'hello' and 'world'", got)
	}
}

func TestFindHeadline(t *testing.T) {
	tests := []struct {
		name string
		html string
		want string
	}{
		{
			"article heading by class prefix",
			`<html><body><h1 class="svt-news-header__modal-title">Välj ditt område</h1>` +
				`<h1 class="TextArticle__heading___ZyAgV">  Höstvädret
				växlar upp </h1></body></html>`,
			"Höstvädret växlar upp",
		},
		{
			"falls back to og:title",
			`<html><head><meta property="og:title" content="Regn igen"/></head><body><h1 class="Other">Nej</h1></body></html>`,
			"Regn igen",
		},
		{
			"missing",
			`<html><body><h1 class="Other">Nej</h1></body></html>`,
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, _ := html.Parse(strings.NewReader(tt.html))
			if got := findHeadline(doc); got != tt.want {
				t.Errorf("findHeadline() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFindIdagParagraph(t *testing.T) {
	tests := []struct {
		name string
		html string
		want string
	}{
		{
			"real structure",
			`<div id="i-dag-0"><h2>I DAG</h2><div><div><p>Sol i väst. </p></div>` +
				`<figure><noscript><img src="https://svt.se/x.jpg"></noscript></figure></div></div>` +
				`<div id="i-kvall-1"><h2>I KVÄLL</h2><div><p>Regn.</p></div></div>`,
			"Sol i väst.",
		},
		{
			"without i-dag-0 wrapper",
			`<p>Ingress</p><h2>I DAG</h2><p>Blåsigt.</p>`,
			"Blåsigt.",
		},
		{
			"missing h2",
			`<div id="i-dag-0"><p>Sol.</p></div>`,
			"",
		},
		{
			"missing p before next section",
			`<h2>I DAG</h2><div></div><h2>I KVÄLL</h2><p>Regn.</p>`,
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, _ := html.Parse(strings.NewReader(tt.html))
			if got := findIdagParagraph(doc); got != tt.want {
				t.Errorf("findIdagParagraph() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatSummary(t *testing.T) {
	got := formatSummary("Rubrik", "Text.")
	want := "Rubrik\n\nText.\n"
	if got != want {
		t.Errorf("formatSummary() = %q, want %q", got, want)
	}
}

func TestOutputBase(t *testing.T) {
	now := time.Date(2026, 10, 9, 8, 0, 0, 0, time.UTC)
	got := outputBase("out", now)
	want := filepath.Join("out", "vader_2026-10-09")
	if got != want {
		t.Errorf("outputBase() = %q, want %q", got, want)
	}
}
