package sitemap

import "encoding/xml"

const URLSetNamespace = "http://www.sitemaps.org/schemas/sitemap/0.9"

// ─── Sitemap URL Set ──────────────────────────────────────────

type URLSet struct {
	XMLName xml.Name `xml:"urlset"`
	XMLNS   string   `xml:"xmlns,attr"`
	URLs    []URL    `xml:"url"`
}

type URL struct {
	Loc        string  `xml:"loc"`
	LastMod    string  `xml:"lastmod,omitempty"`
	ChangeFreq string  `xml:"changefreq,omitempty"`
	Priority   float64 `xml:"priority"`
}

// ─── Sitemap Index ────────────────────────────────────────────

type SitemapIndex struct {
	XMLName  xml.Name  `xml:"sitemapindex"`
	XMLNS    string    `xml:"xmlns,attr"`
	Sitemaps []Sitemap `xml:"sitemap"`
}

type Sitemap struct {
	Loc     string `xml:"loc"`
	LastMod string `xml:"lastmod,omitempty"`
}

// ─── Request / Response Types ────────────────────────────────

type SitemapRequest struct {
	URLs        []URLInput `json:"urls"`
	PrettyPrint bool       `json:"pretty_print,omitempty"`
	Compress    bool       `json:"compress,omitempty"`
	Validate    bool       `json:"validate,omitempty"`
}

type URLInput struct {
	Loc        string  `json:"loc"`
	LastMod    string  `json:"lastmod,omitempty"`
	ChangeFreq string  `json:"changefreq,omitempty"`
	Priority   float64 `json:"priority,omitempty"`
}

type SitemapIndexRequest struct {
	Sitemaps    []SitemapInput `json:"sitemaps"`
	PrettyPrint bool           `json:"pretty_print,omitempty"`
	Compress    bool           `json:"compress,omitempty"`
}

type SitemapInput struct {
	Loc     string `json:"loc"`
	LastMod string `json:"lastmod,omitempty"`
}

type ValidateRequest struct {
	URLs        []string `json:"urls"`
	TimeoutSecs int      `json:"timeout_seconds,omitempty"`
	Concurrency int      `json:"concurrency,omitempty"`
}

type ValidationResult struct {
	URL            string `json:"url"`
	FinalURL       string `json:"final_url,omitempty"`
	StatusCode     int    `json:"status_code"`
	RedirectCount  int    `json:"redirect_count"`
	ContentType    string `json:"content_type,omitempty"`
	ResponseTimeMs int    `json:"response_time_ms"`
	IsValid        bool   `json:"is_valid"`
	Error          string `json:"error,omitempty"`
}

type DiscoverRequest struct {
	SitemapURL   string `json:"sitemap_url"`
	IncludeAlt   bool   `json:"include_alternates,omitempty"`
	MaxDepth     int    `json:"max_depth,omitempty"`
}
