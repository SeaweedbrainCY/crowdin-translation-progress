// Command crowdin-badge fetches per-language translation progress from the
// Crowdin API and renders it as a single SVG "status card", written to
// badges/crowdin-status.svg.
//
// Required env vars:
//
//	CROWDIN_TOKEN       Crowdin API token (personal or project token) with
//	                     read access to the project and projects translations
//	CROWDIN_PROJECT_ID  Numeric Crowdin project ID.
//
// Optional env vars:
//
//		CROWDIN_API_BASE    			Override API base, default https://api.crowdin.com
//		OUTPUT_PATH         			Override output path, default badges/crowdin-status.svg
//	 MINIMUM_TRANSLATION_PROGRESS 	Minimum translation progress to be include. Default: 10%
package main

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type progressResponse struct {
	Data []struct {
		Data struct {
			LanguageID          string  `json:"languageId"`
			TranslationProgress float64 `json:"translationProgress"`
			Language            struct {
				Name string `json:"name"`
			} `json:"language"`
		} `json:"data"`
	} `json:"data"`
	Pagination struct {
		Offset int `json:"offset"`
		Limit  int `json:"limit"`
	} `json:"pagination"`
}

type projectResponse struct {
	Data struct {
		SourceLanguage struct {
			Name string `json:"name"`
		} `json:"sourceLanguage"`
	} `json:"data"`
}

type langProgress struct {
	Name    string
	Percent int
}

func fetchSoureLang(base, projectID, token string) (string, error) {
	client := &http.Client{}
	url := fmt.Sprintf("%s/api/v2/projects/%s", base, projectID)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}

	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("crowdin api returned %d: %s", resp.StatusCode, string(body))
	}

	var parsed projectResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", err
	}
	return parsed.Data.SourceLanguage.Name, nil
}

func fetchProgress(base, projectID, token string, minimumProgress int) ([]langProgress, error) {
	client := &http.Client{}
	var results []langProgress
	offset := 0
	const limit = 100

	for {
		url := fmt.Sprintf("%s/api/v2/projects/%s/languages/progress?limit=%d&offset=%d",
			base, projectID, limit, offset)

		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("crowdin api returned %d: %s", resp.StatusCode, string(body))
		}

		var parsed progressResponse
		if err := json.Unmarshal(body, &parsed); err != nil {
			return nil, err
		}

		for _, item := range parsed.Data {
			if item.Data.TranslationProgress >= float64(minimumProgress) {
				results = append(results, langProgress{
					Name:    item.Data.Language.Name,
					Percent: int(item.Data.TranslationProgress),
				})
			}
		}

		if len(parsed.Data) < limit {
			break
		}
		offset += limit
	}

	return results, nil
}

func colorFor(pct int) string {
	switch {
	case pct < 40:
		return "#f85149"
	case pct < 75:
		return "#d29922"
	default:
		return "#3fb950"
	}
}

func renderSVG(langs []langProgress) string {
	const (
		width   = 420
		pad     = 24
		rowH    = 34
		headerH = 56
		footerH = 30
		barX    = 150
		barW    = 190
		barH    = 10
		radius  = 14
	)

	height := headerH + len(langs)*rowH + footerH + pad

	var rows strings.Builder
	y := headerH + 10
	for _, l := range langs {
		fillW := barW * l.Percent / 100
		color := colorFor(l.Percent)
		fmt.Fprintf(&rows, `
  <text x="%d" y="%d" font-family="'Segoe UI', Ubuntu, sans-serif" font-size="13" fill="#c9d1d9">%s</text>
  <rect x="%d" y="%d" width="%d" height="%d" rx="%d" fill="#21262d"/>
  <rect x="%d" y="%d" width="%d" height="%d" rx="%d" fill="%s">
    <animate attributeName="width" from="0" to="%d" dur="0.8s" fill="freeze" />
  </rect>
  <text x="%d" y="%d" text-anchor="end" font-family="'Segoe UI', Ubuntu, sans-serif" font-size="13" font-weight="600" fill="%s">%d%%</text>`,
			pad, y+13, html.EscapeString(l.Name),
			barX, y+3, barW, barH, barH/2,
			barX, y+3, fillW, barH, barH/2, color, fillW,
			width-pad, y+13, color, l.Percent,
		)
		y += rowH
	}

	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d">
  <defs>
    <clipPath id="rounded">
      <rect x="0" y="0" width="%d" height="%d" rx="%d"/>
    </clipPath>
  </defs>
  <g clip-path="url(#rounded)">
    <rect x="0" y="0" width="%d" height="%d" fill="#0d1117"/>
    <rect x="0.5" y="0.5" width="%d" height="%d" rx="%d" fill="none" stroke="#30363d" stroke-width="1"/>
    <text x="%d" y="30" font-family="'Segoe UI', Ubuntu, sans-serif" font-size="16" font-weight="700" fill="#58a6ff">Translation Status</text>
    <text x="%d" y="30" text-anchor="end" font-family="'Segoe UI', Ubuntu, sans-serif" font-size="11" fill="#8b949e">via Crowdin</text>
    <line x1="%d" y1="%d" x2="%d" y2="%d" stroke="#21262d" stroke-width="1"/>%s
    <text x="%d" y="%d" font-family="'Segoe UI', Ubuntu, sans-serif" font-size="10" fill="#6e7681">SeaweedbrainCY/crowdin-translation-progress · updated automatically</text>
  </g>
</svg>`,
		width, height, width, height,
		width, height, radius,
		width, height,
		width-1, height-1, radius,
		pad,
		width-pad,
		pad, headerH-14, width-pad, headerH-14,
		rows.String(),
		pad, height-10,
	)
}

// firstNonEmpty returns the first non-empty string, used to prefer a plain
// env var (for local/manual runs) over the GitHub Action INPUT_* equivalent
// (set automatically when running as a container action), or vice versa.
func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func main() {
	token := firstNonEmpty(os.Getenv("CROWDIN_TOKEN"), os.Getenv("INPUT_CROWDIN_TOKEN"))
	projectID := firstNonEmpty(os.Getenv("CROWDIN_PROJECT_ID"), os.Getenv("INPUT_CROWDIN_PROJECT_ID"))

	if token == "" || projectID == "" {
		fmt.Fprintln(os.Stderr, "CROWDIN_TOKEN and CROWDIN_PROJECT_ID must be set")
		os.Exit(1)
	}

	base := firstNonEmpty(os.Getenv("CROWDIN_API_BASE"), os.Getenv("INPUT_CROWDIN_API_BASE"))
	if base == "" {
		base = "https://api.crowdin.com"
	}

	outputPath := firstNonEmpty(os.Getenv("OUTPUT_PATH"), os.Getenv("INPUT_OUTPUT_PATH"))
	if outputPath == "" {
		outputPath = "badges/crowdin-status.svg"
	}

	minimumProgress, minProgressErr := strconv.Atoi(firstNonEmpty(os.Getenv("MINIMUM_TRANSLATION_PROGRESS"), os.Getenv("INPUT_MINIMUM_TRANSLATION_PROGRESS")))
	if minProgressErr != nil {
		minimumProgress = 10
	}

	langs, err := fetchProgress(base, projectID, token, minimumProgress)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to fetch progress: %v\n", err)
		os.Exit(1)
	}

	sourceLangName, err := fetchSoureLang(base, projectID, token)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to fetch project: %v\n", err)
		os.Exit(1)
	}
	langs = append(langs, langProgress{
		Name:    sourceLangName,
		Percent: 100,
	})

	if len(langs) == 0 {
		fmt.Fprintln(os.Stderr, "no languages returned by crowdin api")
		os.Exit(1)
	}

	sort.Slice(langs, func(i, j int) bool {
		if langs[i].Percent != langs[j].Percent {
			return langs[i].Percent > langs[j].Percent
		}
		return langs[i].Name < langs[j].Name
	})

	svg := renderSVG(langs)

	if dir := filepath.Dir(outputPath); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "failed to create output dir: %v\n", err)
			os.Exit(1)
		}
	}

	if err := os.WriteFile(outputPath, []byte(svg), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write svg: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("wrote %s (%d languages)\n", outputPath, len(langs))
	writeGithubOutput("svg_path", outputPath)
}

// writeGithubOutput appends a key=value pair to $GITHUB_OUTPUT if it's set,
// so the value is usable as a step output when running inside a GitHub Action.
// No-op (and non-fatal) outside of Actions.
func writeGithubOutput(key, value string) {
	path := os.Getenv("GITHUB_OUTPUT")
	if path == "" {
		return
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "%s=%s\n", key, value)
}
