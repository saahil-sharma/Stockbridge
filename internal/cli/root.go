package cli

import "github.com/spf13/cobra"

func NewRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stockbridge",
		Short: "Stockbridge analyzes listed U.S. stocks from sourced public data.",
		Long: `Stockbridge is a command-line stock research tool.

It accepts a listed U.S. stock ticker, validates the listing through Nasdaq
Trader symbol directories, fetches SEC company and XBRL company-facts data,
and renders a sourced fundamental snapshot.

Current scope:
  - Supports Nasdaq-listed and other exchange-listed symbols found in Nasdaq
    Trader symbol directory files.
  - Uses SEC company tickers and company facts for fundamentals.
  - Renders styled terminal output by default with strong section hierarchy,
    scan-friendly financial values, and a multi-color information palette.
  - Keeps experimental chart data code aside for the future desktop/web UI.
  - Does not yet include valuation ratios, interactive charts, or news in the
    active CLI report.

Presentation:
  Stockbridge uses terminal styling to make reports easier to scan. Headers,
  important financial values, notes, metadata, and sources receive different
  visual treatment. Terminal apps cannot change the user's real font size or
  font family, so Stockbridge simulates hierarchy with bold text, spacing,
  uppercase section labels, borders, and color.

Stockbridge output is informational and is not personalized financial advice.`,
		Example: `  stockbridge help
  stockbridge about
  stockbridge analyze AMZN
  stockbridge analyze IBM --format text
  stockbridge analyze AMZN --output reports/AMZN.txt`,
	}

	cmd.AddCommand(newAboutCommand(), newAnalyzeCommand())
	return cmd
}
