package output

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/RakhaYandra/ookami/internal/model"
	"github.com/RakhaYandra/ookami/internal/scoring"
)

// categoryOrder is the single display order (PRD §34); scoring owns it.
var categoryOrder = scoring.CategoryOrder

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

func suggestedActionLine(rem *model.Remediation) string {
	if rem == nil {
		return ""
	}
	cmd := strings.TrimSpace(strings.Join(append([]string{rem.Command}, rem.Args...), " "))
	if cmd == "" {
		return ""
	}
	desc := rem.Description
	if rem.RequiresSudo {
		if desc != "" {
			desc += ", requires sudo"
		} else {
			desc = "requires sudo"
		}
	}
	return fmt.Sprintf("    → Suggested action: %s (%s)", cmd, desc)
}

func writeResult(w io.Writer, r model.Result, verbose bool) error {
	line := r.Title
	if r.Message != "" {
		line += " " + r.Message
	}
	if _, err := fmt.Fprintf(w, "  %s %s\n", symbolFor(r.Severity), strings.TrimSpace(line)); err != nil {
		return err
	}
	if (r.Severity == model.SeverityWarning || r.Severity == model.SeverityCritical || r.Severity == model.SeverityUnknown) && r.Remediation != nil {
		if line := suggestedActionLine(r.Remediation); line != "" {
			if _, err := fmt.Fprintln(w, line); err != nil {
				return err
			}
		}
	}
	if verbose && len(r.Details) > 0 {
		keys := make([]string, 0, len(r.Details))
		for k := range r.Details {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if _, err := fmt.Fprintf(w, "      %s=%v\n", k, r.Details[k]); err != nil {
				return err
			}
		}
	}
	return nil
}

func writeGroupedResults(w io.Writer, rs []model.Result, verbose bool) error {
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
			if err := writeResult(w, r, verbose); err != nil {
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
			if err := writeResult(w, g, verbose); err != nil {
				return err
			}
		}
	}
	return nil
}

func RenderHuman(w io.Writer, rs []model.Result, score int, status string) error {
	return renderHuman(w, rs, score, status, false)
}

func RenderHumanVerbose(w io.Writer, rs []model.Result, score int, status string) error {
	return renderHuman(w, rs, score, status, true)
}

func renderHuman(w io.Writer, rs []model.Result, score int, status string, verbose bool) error {
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
		if err := writeGroupedResults(w, rs, verbose); err != nil {
			return err
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

// CategoryJSON is one ordered breakdown entry for JSON output.
type CategoryJSON struct {
	Category string `json:"category"`
	Status   string `json:"status"`
	Score    int    `json:"score"`
}

// RenderJSONBreakdown renders score/status/categories/results.
// Categories follow scoring.CategoryOrder (active subset only).
// Deprecated RenderJSON (no categories) still works for compat.
func RenderJSONBreakdown(w io.Writer, rs []model.Result) error {
	if rs == nil {
		rs = []model.Result{}
	}
	score, status, cats := scoring.Breakdown(rs)
	ordered := make([]CategoryJSON, 0, len(cats))
	for _, c := range scoring.CategoryOrder {
		cs, ok := cats[c]
		if !ok {
			continue
		}
		ordered = append(ordered, CategoryJSON{Category: string(c), Status: cs.Status, Score: cs.Score})
	}
	return json.NewEncoder(w).Encode(struct {
		Score      int            `json:"score"`
		Status     string         `json:"status"`
		Categories []CategoryJSON `json:"categories"`
		Results    []model.Result `json:"results"`
	}{score, status, ordered, rs})
}

func RenderQuiet(w io.Writer, rs []model.Result) error {
	for _, r := range rs {
		if r.Severity != model.SeverityWarning && r.Severity != model.SeverityCritical && r.Severity != model.SeverityUnknown {
			continue
		}
		line := r.Title
		if r.Message != "" {
			line += " " + r.Message
		}
		if _, err := fmt.Fprintf(w, "%s %s\n", symbolFor(r.Severity), strings.TrimSpace(line)); err != nil {
			return err
		}
	}
	return nil
}
