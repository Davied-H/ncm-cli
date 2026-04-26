package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"unicode"
)

func JSON(w io.Writer, value any) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	return enc.Encode(value)
}

func Table(w io.Writer, headers []string, rows [][]string) error {
	columns := len(headers)
	for _, row := range rows {
		if len(row) > columns {
			columns = len(row)
		}
	}
	if columns == 0 {
		return nil
	}

	widths := make([]int, columns)
	if len(headers) > 0 {
		for i := 0; i < columns; i++ {
			cell := cellAt(headers, i)
			widths[i] = max(widths[i], displayWidth(cell))
		}
	}
	for _, row := range rows {
		for i := 0; i < columns; i++ {
			cell := cellAt(row, i)
			widths[i] = max(widths[i], displayWidth(cell))
		}
	}

	if len(headers) > 0 {
		if err := writeTableRow(w, headers, widths); err != nil {
			return err
		}
	}
	for _, row := range rows {
		if err := writeTableRow(w, row, widths); err != nil {
			return err
		}
	}
	return nil
}

func Text(w io.Writer, format string, args ...any) error {
	_, err := fmt.Fprintf(w, format, args...)
	return err
}

func writeTableRow(w io.Writer, row []string, widths []int) error {
	for i, width := range widths {
		if i > 0 {
			if _, err := io.WriteString(w, "  "); err != nil {
				return err
			}
		}
		cell := cleanCell(cellAt(row, i))
		if _, err := io.WriteString(w, cell); err != nil {
			return err
		}
		if i < len(widths)-1 {
			if _, err := io.WriteString(w, strings.Repeat(" ", width-displayWidth(cell))); err != nil {
				return err
			}
		}
	}
	_, err := io.WriteString(w, "\n")
	return err
}

func cellAt(row []string, index int) string {
	if index >= len(row) {
		return ""
	}
	return cleanCell(row[index])
}

func cleanCell(cell string) string {
	cell = strings.ReplaceAll(cell, "\t", " ")
	cell = strings.ReplaceAll(cell, "\r", " ")
	cell = strings.ReplaceAll(cell, "\n", " ")
	return cell
}

func displayWidth(text string) int {
	width := 0
	for _, r := range text {
		width += runeWidth(r)
	}
	return width
}

func runeWidth(r rune) int {
	if r == 0 || r == '\u200d' || unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Me, r) {
		return 0
	}
	if r < 32 || (r >= 0x7f && r < 0xa0) {
		return 0
	}
	if isWideRune(r) {
		return 2
	}
	return 1
}

func isWideRune(r rune) bool {
	return (r >= 0x1100 && r <= 0x115f) ||
		(r >= 0x2329 && r <= 0x232a) ||
		(r >= 0x2e80 && r <= 0xa4cf) ||
		(r >= 0xac00 && r <= 0xd7a3) ||
		(r >= 0xf900 && r <= 0xfaff) ||
		(r >= 0xfe10 && r <= 0xfe19) ||
		(r >= 0xfe30 && r <= 0xfe6f) ||
		(r >= 0xff00 && r <= 0xff60) ||
		(r >= 0xffe0 && r <= 0xffe6) ||
		(r >= 0x1f300 && r <= 0x1faff)
}
