# Desktop And Backend Transition

Stockbridge is moving toward a desktop app and website architecture. The CLI work should become the data and analysis backend rather than the final user interface.

## Current Backend Capabilities

- Ticker normalization and listed-symbol validation.
- Nasdaq Trader symbol directory lookups.
- SEC ticker-to-CIK lookup.
- SEC company-facts fetching and normalization into report metrics.
- Styled and plain text report rendering.
- Experimental historical chart-fetching code under `internal/data/market/`.

## Near-Term Direction

- Keep CLI output focused on fundamentals while the UI direction changes.
- Extract the current `analyze` flow into a reusable service layer that returns structured report data instead of formatted terminal text.
- Keep renderers separate from data fetching and analysis.
- Treat charting as a frontend concern. The backend should expose clean historical price series, not terminal-specific chart state.

## Proposed Next Structure

```text
internal/app/
  analyze.go      # orchestration service for ticker -> structured report

internal/domain/
  report.go       # frontend-neutral report models
  chart.go        # frontend-neutral historical price models

internal/http/
  server.go       # future API for desktop/web clients
  handlers.go

internal/report/
  text.go
  styled.go       # CLI-only rendering
```

## First Backend Refactor

Move the logic currently inside `internal/cli/analyze.go` into an `internal/app.Analyzer`. The CLI should call the analyzer, then choose a renderer. The future desktop app or website can call the same analyzer and render the returned structured data itself.
