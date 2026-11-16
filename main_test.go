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

func TestFindSectionImage(t *testing.T) {
	tests := []struct {
		name string
		html string
		sec  section
		want string
	}{
		{
			"i dag",
			`<html><body><div id="i-dag-0"><noscript><img src="https://svt.se/weather.jpg"></noscript></div></body></html>`,
			sectionIdag,
			"https://svt.se/weather.jpg",
		},
		{
			"i kväll at any position",
			`<html><body><div id="i-dag-0"><noscript><img src="https://svt.se/dag.jpg"></noscript></div>` +
				`<div id="i-kvall-1"><noscript><img src="https://svt.se/kvall.jpg"></noscript></div></body></html>`,
			sectionIkvall,
			"https://svt.se/kvall.jpg",
		},
		{
			"missing div",
			`<html><body><div id="other"><noscript><img src="https://x.com/y.jpg"></noscript></div></body></html>`,
			sectionIdag,
			"",
		},
		{
			"no noscript",
			`<html><body><div id="i-dag-0"><p>hello</p></div></body></html>`,
			sectionIdag,
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, _ := html.Parse(strings.NewReader(tt.html))
			got := findSectionImage(doc, tt.sec)
			if got != tt.want {
				t.Errorf("findSectionImage() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSectionsFor(t *testing.T) {
	stockholm, err := time.LoadLocation("Europe/Stockholm")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		now  time.Time
		want section
	}{
		{"morning CET", time.Date(2026, 11, 16, 9, 0, 0, 0, stockholm), sectionIdag},
		{"12:59 CET", time.Date(2026, 11, 16, 12, 59, 0, 0, stockholm), sectionIdag},
		{"13:00 CET", time.Date(2026, 11, 16, 13, 0, 0, 0, stockholm), sectionIkvall},
		{"13:00 CEST", time.Date(2026, 6, 16, 13, 0, 0, 0, stockholm), sectionIkvall},
		// 11:30 UTC is 13:30 CEST: the cutoff follows Swedish time, not the machine's.
		{"UTC input", time.Date(2026, 6, 16, 11, 30, 0, 0, time.UTC), sectionIkvall},
		{"UTC input morning", time.Date(2026, 6, 16, 10, 30, 0, 0, time.UTC), sectionIdag},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sectionsFor(tt.now)
			if len(got) != 2 || got[0] != tt.want || got[1] == tt.want {
				t.Errorf("sectionsFor() = %v, want %v first with the other as fallback", got, tt.want)
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

func TestFindSectionParagraph(t *testing.T) {
	tests := []struct {
		name string
		html string
		sec  section
		want string
	}{
		{
			"real structure",
			`<div id="i-dag-0"><h2>I DAG</h2><div><div><p>Sol i väst. </p></div>` +
				`<figure><noscript><img src="https://svt.se/x.jpg"></noscript></figure></div></div>` +
				`<div id="i-kvall-1"><h2>I KVÄLL</h2><div><p>Regn.</p></div></div>`,
			sectionIdag,
			"Sol i väst.",
		},
		{
			"i kväll",
			`<div id="i-dag-0"><h2>I DAG</h2><div><p>Sol.</p></div></div>` +
				`<div id="i-kvall-1"><h2>I KVÄLL</h2><div><p>Regn.</p></div></div>`,
			sectionIkvall,
			"Regn.",
		},
		{
			"without i-dag-0 wrapper",
			`<p>Ingress</p><h2>I DAG</h2><p>Blåsigt.</p>`,
			sectionIdag,
			"Blåsigt.",
		},
		{
			"missing h2",
			`<div id="i-dag-0"><p>Sol.</p></div>`,
			sectionIdag,
			"",
		},
		{
			"missing p before next section",
			`<h2>I DAG</h2><div></div><h2>I KVÄLL</h2><p>Regn.</p>`,
			sectionIdag,
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, _ := html.Parse(strings.NewReader(tt.html))
			if got := findSectionParagraph(doc, tt.sec); got != tt.want {
				t.Errorf("findSectionParagraph() = %q, want %q", got, tt.want)
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
