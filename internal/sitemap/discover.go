package sitemap

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"time"
)

type DiscoveredURL struct {
	Loc     string `json:"loc"`
	LastMod string `json:"lastmod,omitempty"`
	Source  string `json:"source"`
}

func Discover(ctx context.Context, sitemapURL string, followIndex bool, maxDepth int, userAgent string, timeoutSecs int) ([]DiscoveredURL, error) {
	if maxDepth <= 0 {
		maxDepth = 3
	}
	client := &http.Client{Timeout: time.Duration(timeoutSecs) * time.Second}
	return discoverRecursive(ctx, client, sitemapURL, followIndex, maxDepth, 0, userAgent)
}

func discoverRecursive(ctx context.Context, client *http.Client, url string, followIndex bool, maxDepth, depth int, userAgent string) ([]DiscoveredURL, error) {
	if depth > maxDepth {
		return nil, fmt.Errorf("max depth exceeded")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d fetching %s", resp.StatusCode, url)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 50*1024*1024))
	if err != nil {
		return nil, err
	}

	var urlSet URLSet
	if err := xml.Unmarshal(body, &urlSet); err == nil && len(urlSet.URLs) > 0 {
		results := make([]DiscoveredURL, 0, len(urlSet.URLs))
		for _, u := range urlSet.URLs {
			results = append(results, DiscoveredURL{
				Loc:     u.Loc,
				LastMod: u.LastMod,
				Source:  url,
			})
		}
		return results, nil
	}

	if followIndex {
		var index SitemapIndex
		if err := xml.Unmarshal(body, &index); err == nil && len(index.Sitemaps) > 0 {
			var all []DiscoveredURL
			for _, s := range index.Sitemaps {
				child, err := discoverRecursive(ctx, client, s.Loc, followIndex, maxDepth, depth+1, userAgent)
				if err != nil {
					continue
				}
				all = append(all, child...)
			}
			return all, nil
		}
	}

	return nil, fmt.Errorf("no sitemap URLs found in %s", url)
}
