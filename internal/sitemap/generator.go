package sitemap

import (
	"bytes"
	"compress/gzip"
	"encoding/xml"
	"fmt"
	"strings"
	"time"
)

const xmlHeader = `<?xml version="1.0" encoding="UTF-8"?>` + "\n"

func GenerateURLSet(req *SitemapRequest, defaultChangeFreq string, defaultPriority float64) ([]byte, error) {
	set := URLSet{
		XMLNS: URLSetNamespace,
		URLs:  make([]URL, 0, len(req.URLs)),
	}

	for _, u := range req.URLs {
		loc := strings.TrimSpace(u.Loc)
		if loc == "" {
			continue
		}
		if !strings.HasPrefix(loc, "http://") && !strings.HasPrefix(loc, "https://") {
			return nil, fmt.Errorf("invalid URL (must start with http:// or https://): %s", loc)
		}
		if len(loc) > 2048 {
			return nil, fmt.Errorf("URL exceeds max length of 2048 chars")
		}

		changeFreq := u.ChangeFreq
		if changeFreq == "" {
			changeFreq = defaultChangeFreq
		}

		priority := u.Priority
		if priority == 0 {
			priority = defaultPriority
		}
		if priority < 0 {
			priority = 0
		}
		if priority > 1 {
			priority = 1
		}

		set.URLs = append(set.URLs, URL{
			Loc:        xmlEscape(loc),
			LastMod:    u.LastMod,
			ChangeFreq: changeFreq,
			Priority:   priority,
		})
	}

	return marshalXML(set, req.PrettyPrint)
}

func GenerateSitemapIndex(req *SitemapIndexRequest) ([]byte, error) {
	idx := SitemapIndex{
		XMLNS:    URLSetNamespace,
		Sitemaps: make([]Sitemap, 0, len(req.Sitemaps)),
	}

	for _, s := range req.Sitemaps {
		loc := strings.TrimSpace(s.Loc)
		if loc == "" {
			continue
		}
		idx.Sitemaps = append(idx.Sitemaps, Sitemap{
			Loc:     xmlEscape(loc),
			LastMod: s.LastMod,
		})
	}

	if len(idx.Sitemaps) == 0 {
		return nil, fmt.Errorf("sitemap index must contain at least one sitemap")
	}

	return marshalXML(idx, req.PrettyPrint)
}

func Gzip(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	gz.Name = fmt.Sprintf("sitemap-%d.xml", time.Now().Unix())
	gz.ModTime = time.Now()
	if _, err := gz.Write(data); err != nil {
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func Gunzip(data []byte) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	var out bytes.Buffer
	if _, err := out.ReadFrom(r); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func marshalXML(v any, pretty bool) ([]byte, error) {
	var body []byte
	var err error

	if pretty {
		body, err = xml.MarshalIndent(v, "", "  ")
	} else {
		body, err = xml.Marshal(v)
	}
	if err != nil {
		return nil, err
	}

	out := make([]byte, 0, len(xmlHeader)+len(body))
	out = append(out, xmlHeader...)
	out = append(out, body...)
	return out, nil
}

func xmlEscape(s string) string {
	s = strings.ReplaceAll(s, "\n", "")
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "\t", "")
	return s
}
