package sitemap

import (
	"strings"
	"testing"
)

func TestGenerateURLSet_Basic(t *testing.T) {
	req := &SitemapRequest{
		URLs: []URLInput{
			{Loc: "https://example.com/", Priority: 1},
			{Loc: "https://example.com/about", ChangeFreq: "monthly"},
		},
		PrettyPrint: true,
	}
	out, err := GenerateURLSet(req, "weekly", 0.5)
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if !strings.Contains(s, "<urlset") || !strings.Contains(s, "https://example.com/") {
		t.Fatalf("unexpected output:\n%s", s)
	}
}

func TestGenerateURLSet_RejectsBadURL(t *testing.T) {
	req := &SitemapRequest{URLs: []URLInput{{Loc: "ftp://example.com/"}}}
	if _, err := GenerateURLSet(req, "weekly", 0.5); err == nil {
		t.Fatal("expected error for non-http URL")
	}
}

func TestGenerateRobots_Basic(t *testing.T) {
	delay := 10
	req := &RobotsRequest{
		Rules:           []RobotsRule{{UserAgent: "*", Allow: []string{"/"}, Disallow: []string{"/admin/"}, CrawlDelay: &delay}},
		Sitemaps:        []string{"https://example.com/sitemap.xml"},
		Host:            "example.com",
		IncludeComments: true,
	}
	out, err := GenerateRobots(req)
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	for _, want := range []string{"User-agent: *", "Disallow: /admin/", "Sitemap: https://example.com/sitemap.xml", "Host: example.com"} {
		if !strings.Contains(s, want) {
			t.Fatalf("missing %q in:\n%s", want, s)
		}
	}
}

func TestGzip_RoundTrip(t *testing.T) {
	raw := []byte("<xml>hello</xml>")
	gz, err := Gzip(raw)
	if err != nil {
		t.Fatal(err)
	}
	back, err := Gunzip(gz)
	if err != nil {
		t.Fatal(err)
	}
	if string(back) != string(raw) {
		t.Fatalf("roundtrip mismatch: %q", back)
	}
}
