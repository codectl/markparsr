// Package check holds the README checks terraform-docs has no opinion on:
// required files, additional section headings, and URL liveness.
package check

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"mvdan.cc/xurls/v2"
)

// maxURLConcurrency bounds in-flight HTTP requests during URLs.
const maxURLConcurrency = 5

// providerDocsPrefix marks terraform-docs generated links, which are trusted
// and skipped: they are numerous, slow, and rate-limited.
const providerDocsPrefix = "https://registry.terraform.io/providers/"

// Files reports every name under dir that is missing or empty.
func Files(dir string, names ...string) error {
	var errs []error
	for _, name := range names {
		path := name
		if !filepath.IsAbs(path) {
			path = filepath.Join(dir, name)
		}
		info, err := os.Stat(path)
		switch {
		case errors.Is(err, os.ErrNotExist):
			errs = append(errs, fmt.Errorf("file missing: %s", name))
		case err != nil:
			errs = append(errs, fmt.Errorf("file %s: %w", name, err))
		case info.Size() == 0:
			errs = append(errs, fmt.Errorf("file empty: %s", name))
		}
	}
	return errors.Join(errs...)
}

// Sections reports every name that has no `## name` heading in readme.
func Sections(readme []byte, names ...string) error {
	var errs []error
	for _, name := range names {
		if !hasHeading(readme, name) {
			errs = append(errs, fmt.Errorf("section missing: ## %s", name))
		}
	}
	return errors.Join(errs...)
}

// URLs fetches every URL in readme and reports those that fail or do not
// return 200. Links into the Terraform provider registry are skipped.
func URLs(ctx context.Context, client *http.Client, readme []byte) error {
	var urls []string
	for _, u := range xurls.Strict().FindAllString(string(readme), -1) {
		if !strings.HasPrefix(u, providerDocsPrefix) {
			urls = append(urls, u)
		}
	}

	errs := make([]error, len(urls))
	sem := make(chan struct{}, maxURLConcurrency)
	var wg sync.WaitGroup
	for i, u := range urls {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			errs[i] = fetch(ctx, client, u)
		}()
	}
	wg.Wait()
	return errors.Join(errs...)
}

func hasHeading(readme []byte, name string) bool {
	want := []byte("## " + name)
	for line := range bytes.Lines(readme) {
		if bytes.Equal(bytes.TrimRight(line, "\r\n"), want) {
			return true
		}
	}
	return false
}

func fetch(ctx context.Context, client *http.Client, url string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("url %s: %w", url, err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("url %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("url %s: status %d", url, resp.StatusCode)
	}
	return nil
}
