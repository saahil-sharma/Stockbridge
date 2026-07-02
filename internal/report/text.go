package report

import (
	"fmt"
	"strings"

	"stockbridge/internal/analysis"
)

func RenderText(summary analysis.Summary) string {
	var b strings.Builder

	fmt.Fprintf(&b, "Stockbridge Fundamental Snapshot\n")
	fmt.Fprintf(&b, "================================\n\n")
	fmt.Fprintf(&b, "%s (%s)\n", summary.CompanyName, summary.Ticker)
	if summary.CIK != 0 {
		fmt.Fprintf(&b, "CIK: %010d\n", summary.CIK)
	}
	fmt.Fprintf(&b, "Exchange: %s\n", summary.Listing.Exchange)
	if summary.Listing.Market != "" && summary.Listing.Market != summary.Listing.Exchange {
		fmt.Fprintf(&b, "Market: %s\n", summary.Listing.Market)
	}
	fmt.Fprintf(&b, "Security: %s\n", summary.Listing.SecurityName)
	if summary.Listing.FileCreatedAt != "" {
		fmt.Fprintf(&b, "Symbol directory timestamp: %s\n", summary.Listing.FileCreatedAt)
	}

	fmt.Fprintf(&b, "\nFundamentals\n")
	fmt.Fprintf(&b, "----------------\n")
	if len(summary.Metrics) == 0 {
		fmt.Fprintf(&b, "No supported fundamentals were found.\n")
	} else {
		for _, metric := range summary.Metrics {
			fmt.Fprintf(&b, "%-24s %16s  period=%s source=%s filed=%s concept=%s\n",
				metric.Name+":",
				formatValue(metric.Value, metric.Unit),
				metric.Period,
				metric.Form,
				metric.Filed,
				metric.Concept,
			)
		}
	}

	fmt.Fprintf(&b, "\nNotes\n")
	fmt.Fprintf(&b, "-----\n")
	for _, note := range summary.Notes {
		fmt.Fprintf(&b, "- %s\n", note)
	}

	fmt.Fprintf(&b, "\nSources\n")
	fmt.Fprintf(&b, "-------\n")
	for _, source := range summary.Sources {
		fmt.Fprintf(&b, "- %s\n  %s\n  retrieved: %s\n", source.Name, source.URL, source.RetrievedAt)
	}

	return b.String()
}

func formatValue(value float64, unit string) string {
	switch unit {
	case "USD":
		return compactUSD(value)
	case "USD/shares":
		return fmt.Sprintf("$%.2f/share", value)
	case "x":
		return fmt.Sprintf("%.2fx", value)
	case "%":
		return fmt.Sprintf("%.2f%%", value)
	default:
		return fmt.Sprintf("%.2f %s", value, unit)
	}
}

func compactUSD(value float64) string {
	abs := value
	if abs < 0 {
		abs = -abs
	}

	switch {
	case abs >= 1_000_000_000_000:
		return fmt.Sprintf("$%.2fT", value/1_000_000_000_000)
	case abs >= 1_000_000_000:
		return fmt.Sprintf("$%.2fB", value/1_000_000_000)
	case abs >= 1_000_000:
		return fmt.Sprintf("$%.2fM", value/1_000_000)
	default:
		return fmt.Sprintf("$%.2f", value)
	}
}
