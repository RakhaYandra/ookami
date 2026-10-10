package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/RakhaYandra/ookami/internal/model"
)

var categoryOrder = []model.Category{
	model.CategorySystem,
	model.CategoryDevelopment,
	model.CategoryStorage,
	model.CategoryNetwork,
	model.CategoryGPU,
	model.CategoryServices,
}

func symbolFor(s model.Severity) string {
	switch s {
	case model.SeverityPass:
		return "✓"
	case model.SeverityWarning:
		return "⚠"
	case model.SeverityCritical:
		return "✗"
	case model.SeverityInfo:
		return "ℹ"
	default:
		return "?"
	}
}

func categoryTitle(c model.Category) string {
	switch c {
	case model.CategoryGPU:
		return "GPU"
	}
	s := string(c)
	if s == "" {
		return "Unknown"
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func plural(n int, single, multi string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, single)
	}
	return fmt.Sprintf("%d %s", n, multi)
}

func statusLine(status string) string {
	switch strings.ToLower(status) {
	case "healthy":
		return "Development environment is healthy."
	case "warning":
		return "Development environment needs attention."
	case "critical":
		return "Development environment has critical issues."
	default:
		return status
	}
}

func RenderHuman(w io.Writer, rs []model.Result, score int, status string) error {
	if _, err := fmt.Fprintln(w, "OOKAMI — Developer Machine Doctor"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}
	if len(rs) == 0 {
		if _, err := fmt.Fprintln(w, "No results."); err != nil {
			return err
		}
	} else {
		grouped := map[model.Category][]model.Result{}
		for _, r := range rs {
			grouped[r.Category] = append(grouped[r.Category], r)
		}
		shown := map[model.Category]bool{}
		for _, c := range categoryOrder {
			gr := grouped[c]
			if len(gr) == 0 {
				continue
			}
			shown[c] = true
			if _, err := fmt.Fprintln(w, categoryTitle(c)); err != nil {
				return err
			}
			for _, r := range gr {
				line := r.Title
				if r.Message != "" {
					line += " " + r.Message
				}
				if _, err := fmt.Fprintf(w, "  %s %s\n", symbolFor(r.Severity), strings.TrimSpace(line)); err != nil {
					return err
				}
			}
		}
		for _, r := range rs {
			if shown[r.Category] {
				continue
			}
			shown[r.Category] = true
			if _, err := fmt.Fprintln(w, categoryTitle(r.Category)); err != nil {
				return err
			}
			for _, g := range grouped[r.Category] {
				line := g.Title
				if g.Message != "" {
					line += " " + g.Message
				}
				if _, err := fmt.Fprintf(w, "  %s %s\n", symbolFor(g.Severity), strings.TrimSpace(line)); err != nil {
					return err
				}
			}
		}
	}
	if _, err := fmt.Fprintln(w, "----------------------------------------"); err != nil {
		return err
	}
	warnings, criticals := 0, 0
	for _, r := range rs {
		switch r.Severity {
		case model.SeverityWarning, model.SeverityUnknown:
			warnings++ // unknown counts as warning (explicit)
		case model.SeverityCritical:
			criticals++
		}
	}
	if _, err := fmt.Fprintf(w, "Score: %d/100\n", score); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "%s, %s\n", plural(warnings, "warning", "warnings"), plural(criticals, "critical issue", "critical issues")); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, statusLine(status)); err != nil {
		return err
	}
	return nil
}

func RenderJSON(w io.Writer, rs []model.Result, score int, status string) error {
	if rs == nil {
		rs = []model.Result{}
	}
	return json.NewEncoder(w).Encode(struct {
		Score   int            `json:"score"`
		Status  string         `json:"status"`
		Results []model.Result `json:"results"`
	}{score, status, rs})
}

func RenderQuiet(w io.Writer, rs []model.Result) error {
	for _, r := range rs {
		if r.Severity != model.SeverityWarning && r.Severity != model.SeverityCritical && r.Severity != model.SeverityUnknown {
			continue
		}
		if _, err := fmt.Fprintf(w, "⚠ %s\n", r.Title); err != nil {
			return err
		}
	}
	return nil
}
