package main

import (
	"strings"
	"testing"

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
