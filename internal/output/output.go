package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

func JSON(w io.Writer, value any) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	return enc.Encode(value)
}

func Table(w io.Writer, headers []string, rows [][]string) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if len(headers) > 0 {
		if _, err := fmt.Fprintln(tw, strings.Join(headers, "\t")); err != nil {
			return err
		}
	}
	for _, row := range rows {
		for len(row) < len(headers) {
			row = append(row, "")
		}
		if _, err := fmt.Fprintln(tw, strings.Join(row, "\t")); err != nil {
			return err
		}
	}
	return tw.Flush()
}

func Text(w io.Writer, format string, args ...any) error {
	_, err := fmt.Fprintf(w, format, args...)
	return err
}
