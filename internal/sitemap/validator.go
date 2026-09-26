package sitemap

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"
)

type Validator struct {
	UserAgent    string
	MaxRedirects int
	Client       *http.Client
}

func NewValidator(userAgent string, maxRedirects, timeoutSecs int) *Validator {
	if timeoutSecs <= 0 {
		timeoutSecs = 10
	}
	return &Validator{
		UserAgent:    userAgent,
		MaxRedirects: maxRedirects,
		Client: &http.Client{
			Timeout: time.Duration(timeoutSecs) * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= maxRedirects {
					return errors.New("too many redirects")
				}
				return nil
			},
		},
	}
}

func (v *Validator) ValidateAll(ctx context.Context, urls []string, concurrency int) []ValidationResult {
	results := make([]ValidationResult, len(urls))

	if concurrency <= 0 {
		concurrency = 20
	}
	if concurrency > len(urls) {
		concurrency = len(urls)
	}
	if concurrency == 0 {
		return results
	}

	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	for i, u := range urls {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, u string) {
			defer wg.Done()
			defer func() { <-sem }()
			results[i] = v.validateOne(ctx, u)
		}(i, u)
	}

	wg.Wait()
	return results
}

func (v *Validator) validateOne(ctx context.Context, url string) ValidationResult {
	start := time.Now()
	result := ValidationResult{URL: url}

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	req.Header.Set("User-Agent", v.UserAgent)

	resp, err := v.Client.Do(req)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	resp.Body.Close()

	if resp.StatusCode == http.StatusMethodNotAllowed || resp.StatusCode == http.StatusNotImplemented {
		req2, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err == nil {
			req2.Header.Set("User-Agent", v.UserAgent)
			if resp2, err2 := v.Client.Do(req2); err2 == nil {
				resp2.Body.Close()
				resp = resp2
			}
		}
	}

	result.StatusCode = resp.StatusCode
	result.FinalURL = resp.Request.URL.String()
	result.ContentType = resp.Header.Get("Content-Type")
	result.ResponseTimeMs = int(time.Since(start).Milliseconds())
	result.IsValid = resp.StatusCode >= 200 && resp.StatusCode < 400

	if resp.Request.URL.String() != url {
		result.RedirectCount = 1
	}

	if !result.IsValid {
		result.Error = http.StatusText(resp.StatusCode)
	}
	return result
}
