package report

import (
	"fmt"
	"strings"

	"stockbridge/internal/analysis"
)

const (
	colorDeepBlue = "24"
	colorCyan     = "39"
	colorGreen    = "42"
	colorAmber    = "214"
	colorLavender = "141"
	colorPaper    = "230"
	colorText     = "252"
	colorMuted    = "244"
	colorPanel    = "236"
)

func RenderStyled(summary analysis.Summary) string {
	var b strings.Builder

	title := ansiStyle("Stockbridge", "1", "38;5;"+colorPaper, "48;5;"+colorDeepBlue, "2")
	subtitle := ansiStyle("Fundamental Snapshot", "38;5;"+colorLavender)
	company := ansiStyle(fmt.Sprintf("%s (%s)", summary.CompanyName, summary.Ticker), "1", "38;5;"+colorCyan)

	meta := []string{}
	if summary.CIK != 0 {
		meta = append(meta, labelValue("CIK", fmt.Sprintf("%010d", summary.CIK)))
	}
	meta = append(meta, labelValue("Exchange", summary.Listing.Exchange))
	if summary.Listing.Market != "" && summary.Listing.Market != summary.Listing.Exchange {
		meta = append(meta, labelValue("Market", summary.Listing.Market))
	}
	meta = append(meta, labelValue("Security", summary.Listing.SecurityName))
	if summary.Listing.FileCreatedAt != "" {
		meta = append(meta, labelValue("Symbol file", summary.Listing.FileCreatedAt))
	}

	fmt.Fprintf(&b, "%s %s\n\n%s\n%s\n", title, subtitle, company, strings.Join(meta, "\n"))

	fmt.Fprintf(&b, "\n%s\n", sectionTitle("Fundamentals"))
	if len(summary.Metrics) == 0 {
		fmt.Fprintf(&b, "%s\n", muted("No supported fundamentals were found."))
	} else {
		for _, metric := range summary.Metrics {
			fmt.Fprintf(&b, "%s\n", metricRow(metric))
		}
	}

	fmt.Fprintf(&b, "\n%s\n", notesBlock(summary.Notes))

	fmt.Fprintf(&b, "\n%s\n", sectionTitle("Sources"))
	for _, source := range summary.Sources {
		fmt.Fprintf(&b, "%s\n", sourceRow(source))
	}

	return b.String()
}

func sectionTitle(value string) string {
	return ansiStyle(strings.ToUpper(value), "1", "38;5;"+colorPaper, "48;5;"+colorPanel)
}

func labelValue(label, value string) string {
	return ansiStyle(label+":", "38;5;"+colorMuted) + " " + ansiStyle(value, "38;5;"+colorText)
}

func metricRow(metric analysis.Metric) string {
	name := ansiStyle(metric.Name, "1", "38;5;"+colorText)
	value := ansiStyle(formatValue(metric.Value, metric.Unit), "1", "38;5;"+metricColorCode(metric.Name))
	details := ansiStyle(metricDetails(metric), "38;5;"+colorMuted)
	return fmt.Sprintf("%-24s %-16s  %s", name, value, details)
}

func metricDetails(metric analysis.Metric) string {
	parts := []string{}
	if metric.Period != "" {
		parts = append(parts, "period "+metric.Period)
	}
	if metric.Form != "" {
		parts = append(parts, metric.Form)
	}
	if metric.Filed != "" {
		parts = append(parts, "filed "+metric.Filed)
	}
	if metric.Concept != "" {
		parts = append(parts, metric.Concept)
	}
	return strings.Join(parts, "  ")
}

func metricColorCode(name string) string {
	switch name {
	case "Net income", "Operating cash flow", "Basic EPS", "Diluted EPS", "Revenue growth", "Net income growth", "Gross margin", "Operating margin", "Net margin":
		return colorGreen
	case "Liabilities", "Capital expenditures":
		return colorAmber
	case "P/E ratio":
		return colorCyan
	case "Assets", "Stockholders' equity", "Cash and equivalents":
		return colorLavender
	default:
		return colorCyan
	}
}

func notesBlock(notes []string) string {
	var body strings.Builder
	body.WriteString(sectionTitle("Notes"))
	body.WriteString("\n")
	for _, note := range notes {
		body.WriteString(ansiStyle("• ", "38;5;"+colorAmber))
		body.WriteString(ansiStyle(note, "38;5;"+colorText))
		body.WriteString("\n")
	}

	return borderBox(strings.TrimRight(body.String(), "\n"), colorAmber)
}

func sourceRow(source analysis.Source) string {
	name := ansiStyle(source.Name, "1", "38;5;"+colorText)
	url := ansiStyle(source.URL, "38;5;"+colorCyan)
	retrieved := muted("retrieved: " + source.RetrievedAt)
	return fmt.Sprintf("%s\n  %s\n  %s", name, url, retrieved)
}

func muted(value string) string {
	return ansiStyle(value, "38;5;"+colorMuted)
}

func ansiStyle(text string, codes ...string) string {
	if len(codes) == 0 || text == "" {
		return text
	}
	return "\x1b[" + strings.Join(codes, ";") + "m" + text + "\x1b[0m"
}

func borderBox(body, borderColor string) string {
	lines := strings.Split(body, "\n")
	width := 0
	for _, line := range lines {
		if len(line) > width {
			width = len(line)
		}
	}
	top := ansiStyle("┌"+strings.Repeat("─", width+2)+"┐", "38;5;"+borderColor)
	bottom := ansiStyle("└"+strings.Repeat("─", width+2)+"┘", "38;5;"+borderColor)

	var b strings.Builder
	b.WriteString(top)
	b.WriteString("\n")
	for _, line := range lines {
		padding := strings.Repeat(" ", width-len(line))
		b.WriteString(ansiStyle("│ ", "38;5;"+borderColor))
		b.WriteString(line)
		b.WriteString(padding)
		b.WriteString(ansiStyle(" │", "38;5;"+borderColor))
		b.WriteString("\n")
	}
	b.WriteString(bottom)
	return b.String()
}
