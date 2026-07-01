# Project Instructions

## Project Goal
Build a command-line tool that accepts a listed U.S. stock ticker and produces a practical fundamental analysis report using current company data, financial metrics, and up-to-date market news.

## Operating Rules
- Treat financial data, company facts, prices, filings, analyst estimates, and news as time-sensitive. Verify current information before using it in analysis.
- Prefer primary or reputable sources for market data and news. Examples include SEC filings, company investor relations pages, exchange data, official API providers, and established financial news outlets.
- Cite the sources used for external facts in generated reports.
- Clearly separate reported facts from model-generated interpretation.
- Do not present output as personalized financial advice.
- Call out data freshness, missing data, assumptions, and uncertainty in every report.

## Engineering Style
- Keep changes small, focused, and easy to review.
- Prefer simple, explicit code over broad abstractions until the workflow stabilizes.
- Use structured APIs and parsers for market data, filings, and news when practical.
- Avoid adding external dependencies unless they materially improve correctness, maintainability, or data access.
- Keep secrets such as API keys out of source control. Use environment variables and document required names.
- Design the CLI so it can run in both online and test modes.

## Terminal UI And Formatting
- Treat presentation quality as part of the product, not a final polish pass.
- Styled terminal output should use Bubble Tea-compatible models and Lip Gloss styles.
- Use a distinct but cohesive color palette with multiple roles: deep blue for the product/header, cyan or electric blue for identity and links, green for positive/high-signal financial values, amber for cautionary values, coral for risk/negative values, lavender for secondary accents, and muted gray for metadata.
- Use hierarchy to signal importance: bold high-level labels, larger-feeling section headers through uppercase text, padding, spacing, borders, and contrast. Terminal apps cannot control the user's actual font family or font size, so simulate emphasis with layout and styling.
- Format important numbers and financial data so they are easy to scan: align values, color-code metric types, and keep supporting filing metadata visually secondary.
- Notes, caveats, data freshness, and limitations should be visible and styled as first-class report content.
- Historical chart code is currently sidelined from the active CLI report. Preserve it for the future desktop/web UI, but keep the CLI focused on the fundamentals report until the app/backend boundary is defined.
- Plain text output should remain available for files, logs, and tests, but interactive terminal output should default to styled presentation.

## Expected Project Structure
- `cmd/` for CLI entrypoints.
- `internal/` for application logic that is not intended as a public package.
- `internal/data/` for market data, filings, and news clients.
- `internal/analysis/` for valuation, financial health, risk, and news-sentiment logic.
- `internal/report/` for terminal and file report formatting.
- `docs/` for design notes, data-source decisions, and report methodology.
- `testdata/` for deterministic fixtures.

## Analysis Expectations
- Validate ticker input before making external requests.
- Confirm the ticker's listing exchange before making analysis claims about the company.
- Include core fundamentals such as revenue, earnings, margins, debt, cash flow, valuation ratios, and growth trends when available.
- Include recent material news and explain how it may affect the fundamental view.
- Distinguish between short-term news movement and long-term business fundamentals.
- Include a concise final stance such as bullish, neutral, or bearish only when supported by the data, and explain the reasoning.

## Testing And Verification
- Add tests for ticker validation, data normalization, analysis calculations, and report generation.
- Use fixtures for deterministic tests rather than live network calls.
- Before finishing implementation work, run the relevant test suite and report what was verified.
- If live data or news could not be verified, say so explicitly.

## Collaboration Notes
- Before major design choices, summarize the tradeoff and choose the simpler maintainable option.
- When changing behavior, mention the affected CLI command or report section.
- Keep documentation aligned with the implemented behavior.
