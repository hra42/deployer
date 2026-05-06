package ui

import (
	"fmt"
	"os"
)

const TotalPhases = 6

type Status string

const (
	StatusOK      Status = "ok"
	StatusSkipped Status = "skipped"
	StatusFailed  Status = "failed"
	StatusSkipRun Status = "not run"
)

type SummaryItem struct {
	Name   string
	Status Status
	Detail string
}

func Phase(n int, label string) {
	fmt.Printf("\n==> [%d/%d] %s\n", n, TotalPhases, label)
}

func OK(msg string) {
	fmt.Printf("    ✓ %s\n", msg)
}

func Info(msg string) {
	fmt.Printf("    %s\n", msg)
}

func Warn(msg string) {
	fmt.Fprintf(os.Stderr, "    ! %s\n", msg)
}

func Failf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "    ✗ "+format+"\n", args...)
}

func Summary(items []SummaryItem) {
	fmt.Printf("\n── Summary ──\n")
	for _, it := range items {
		glyph := "·"
		switch it.Status {
		case StatusOK:
			glyph = "✓"
		case StatusSkipped:
			glyph = "!"
		case StatusFailed:
			glyph = "✗"
		case StatusSkipRun:
			glyph = "·"
		}
		if it.Detail != "" {
			fmt.Printf("    %s %-22s %s — %s\n", glyph, it.Name, it.Status, it.Detail)
		} else {
			fmt.Printf("    %s %-22s %s\n", glyph, it.Name, it.Status)
		}
	}
}
