package cli

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"stockbridge/internal/app"
	"stockbridge/internal/report"
)

type analyzeOptions struct {
	format string
	output string
}

func newAnalyzeCommand() *cobra.Command {
	var opts analyzeOptions

	cmd := &cobra.Command{
		Use:   "analyze <ticker>",
		Short: "Fetch sourced fundamentals for a stock ticker.",
		Long: `Fetch listing information and SEC company-facts fundamentals for a ticker.

The command resolves the ticker through Nasdaq Trader symbol directory files,
SEC company tickers, and a local curated foreign issuer fallback. It detects
the listing exchange when available, maps the ticker to an SEC CIK when
available, fetches SEC XBRL company-facts data when available, and renders a
sourced report. If a ticker is recognized but standardized SEC fundamentals are
missing, Stockbridge reports what is unavailable instead of fabricating data.

Styled terminal output is the default. It uses color, spacing, bordered notes,
uppercase section headers, and emphasized financial values to make the report
more readable. Terminal apps cannot control the user's actual font family or
font size, so Stockbridge simulates visual hierarchy through layout, contrast,
and text styling.

Use --format text for plain text output.
When --output is provided, Stockbridge writes plain text to the target file so
the report remains portable and easy to diff.`,
		Example: `  stockbridge analyze AMZN
  stockbridge analyze IBM --format text
  stockbridge analyze MSFT --output reports/MSFT.txt`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAnalyze(cmd.Context(), args[0], opts)
		},
	}

	cmd.Flags().StringVar(&opts.format, "format", "styled", "output format: styled or text")
	cmd.Flags().StringVarP(&opts.output, "output", "o", "", "write plain-text report to a file instead of stdout")

	return cmd
}

func runAnalyze(ctx context.Context, rawTicker string, opts analyzeOptions) error {
	format := strings.ToLower(opts.format)
	if format != "styled" && format != "text" {
		return fmt.Errorf("unsupported format %q; currently supported: styled, text", opts.format)
	}

	httpClient := &http.Client{Timeout: 20 * time.Second}
	analyzer := app.NewAnalyzer(httpClient)
	summary, err := analyzer.Analyze(ctx, rawTicker)
	if err != nil {
		return err
	}

	if opts.output == "" {
		fmt.Print(report.RenderStyled(summary))
		return nil
	}

	doc := report.RenderText(summary)
	return os.WriteFile(opts.output, []byte(doc), 0o644)
}
