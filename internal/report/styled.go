package report

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

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

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(colorPaper)).
		Background(lipgloss.Color(colorDeepBlue)).
		Padding(0, 2).
		Render("Stockbridge")

	subtitle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(colorLavender)).
		Render("Fundamental Snapshot")

	company := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(colorCyan)).
		Render(fmt.Sprintf("%s (%s)", summary.CompanyName, summary.Ticker))

	meta := []string{
		labelValue("CIK", fmt.Sprintf("%010d", summary.CIK)),
		labelValue("Exchange", summary.Listing.Exchange),
	}
	if summary.Listing.Market != "" && summary.Listing.Market != summary.Listing.Exchange {
		meta = append(meta, labelValue("Market", summary.Listing.Market))
	}
	meta = append(meta, labelValue("Security", summary.Listing.SecurityName))
	if summary.Listing.FileCreatedAt != "" {
		meta = append(meta, labelValue("Symbol file", summary.Listing.FileCreatedAt))
	}

	fmt.Fprintf(&b, "%s %s\n\n%s\n%s\n", title, subtitle, company, strings.Join(meta, "\n"))

	fmt.Fprintf(&b, "\n%s\n", sectionTitle("SEC Fundamentals"))
	if len(summary.Metrics) == 0 {
		fmt.Fprintf(&b, "%s\n", muted("No supported SEC metrics were found."))
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
	text := strings.ToUpper(value)
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(colorPaper)).
		Background(lipgloss.Color(colorPanel)).
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(lipgloss.Color(colorCyan)).
		Padding(0, 2).
		MarginTop(1).
		Render(text)
}

func labelValue(label, value string) string {
	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(colorMuted)).
		Width(12)
	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(colorText))
	return labelStyle.Render(label+":") + " " + valueStyle.Render(value)
}

func metricRow(metric analysis.Metric) string {
	nameStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(colorText)).
		Width(24)
	valueStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(metricColor(metric.Name)).
		Width(16).
		Align(lipgloss.Right)
	detailStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(colorMuted))

	details := fmt.Sprintf("period %s  %s filed %s  %s", metric.Period, metric.Form, metric.Filed, metric.Concept)
	return nameStyle.Render(metric.Name) + " " + valueStyle.Render(formatValue(metric.Value, metric.Unit)) + "  " + detailStyle.Render(details)
}

func metricColor(name string) lipgloss.Color {
	switch name {
	case "Net income", "Operating cash flow", "Basic EPS", "Diluted EPS":
		return lipgloss.Color(colorGreen)
	case "Liabilities", "Capital expenditures":
		return lipgloss.Color(colorAmber)
	case "Assets", "Stockholders' equity", "Cash and equivalents":
		return lipgloss.Color(colorLavender)
	default:
		return lipgloss.Color(colorCyan)
	}
}

func notesBlock(notes []string) string {
	var body strings.Builder
	body.WriteString(sectionTitle("Notes"))
	body.WriteString("\n")
	for _, note := range notes {
		body.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(colorAmber)).Render("• "))
		body.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(colorText)).Render(note))
		body.WriteString("\n")
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(colorAmber)).
		Padding(1, 2).
		Width(96).
		Render(strings.TrimRight(body.String(), "\n"))
}

func sourceRow(source analysis.Source) string {
	name := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(colorText)).
		Render(source.Name)
	url := lipgloss.NewStyle().
		Foreground(lipgloss.Color(colorCyan)).
		Render(source.URL)
	retrieved := muted("retrieved: " + source.RetrievedAt)
	return fmt.Sprintf("%s\n  %s\n  %s", name, url, retrieved)
}

func muted(value string) string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(colorMuted)).Render(value)
}
