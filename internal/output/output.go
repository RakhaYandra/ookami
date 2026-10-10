package output

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/RakhaYandra/ookami/internal/model"
)

func RenderHuman(w io.Writer, rs []model.Result, score int, status string) error {
	if _, err := fmt.Fprintf(w, "Score: %d (%s)\n", score, status); err != nil {
		return err
	}
	if len(rs) == 0 {
		_, err := fmt.Fprintln(w, "No results.")
		return err
	}
	for _, r := range rs {
		if _, err := fmt.Fprintf(w, "[%s] %s: %s\n", r.Severity, r.Title, r.Message); err != nil {
			return err
		}
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
