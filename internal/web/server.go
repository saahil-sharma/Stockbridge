package web

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"stockbridge/internal/analysis"
	"stockbridge/internal/app"
	"stockbridge/internal/data/market"
)

type Server struct {
	analyzer *app.Analyzer
	tmpl     *template.Template
}

type pageData struct {
	Query     string
	Report    *analysis.Summary
	Chart     *chartView
	Error     string
	Generated string
	Loading   bool
}

type chartView struct {
	Ticker      string                 `json:"ticker"`
	Periods     []chartPeriod          `json:"periods"`
	SourceName  string                 `json:"sourceName"`
	SourceURL   string                 `json:"sourceURL"`
	RetrievedAt string                 `json:"retrievedAt"`
	ByPeriod    map[string]chartPeriod `json:"byPeriod"`
}

type chartPeriod struct {
	Label       string       `json:"label"`
	Start       float64      `json:"start"`
	End         float64      `json:"end"`
	Change      float64      `json:"change"`
	ChangePct   float64      `json:"changePct"`
	SourceURL   string       `json:"sourceURL"`
	RetrievedAt string       `json:"retrievedAt"`
	Points      []chartPoint `json:"points"`
}

type chartPoint struct {
	Time  string  `json:"time"`
	Close float64 `json:"close"`
}

func NewServer(analyzer *app.Analyzer) *Server {
	return &Server{
		analyzer: analyzer,
		tmpl: template.Must(template.New("index").Funcs(template.FuncMap{
			"formatMetric": formatMetric,
			"chartJSON":    chartJSON,
		}).Parse(indexHTML)),
	}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	return mux
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	query := strings.TrimSpace(r.URL.Query().Get("ticker"))
	data := pageData{
		Query:     query,
		Generated: time.Now().Format("January 2, 2006 15:04 MST"),
	}
	if query != "" {
		ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
		defer cancel()

		summary, err := s.analyzer.Analyze(ctx, query)
		if err != nil {
			data.Error = err.Error()
		} else {
			data.Report = &summary
			data.Query = summary.Ticker
			if chart, err := s.analyzer.PriceChart(ctx, summary.Ticker); err == nil {
				view := buildChartView(chart)
				data.Chart = &view
			} else {
				data.Report.Notes = append(data.Report.Notes, "Price chart unavailable: "+err.Error())
			}
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func buildChartView(bundle market.Bundle) chartView {
	view := chartView{
		Ticker:      bundle.Ticker,
		SourceName:  bundle.SourceName,
		SourceURL:   bundle.SourceURL,
		RetrievedAt: bundle.RetrievedAt.Format("2006-01-02 15:04:05 MST"),
		ByPeriod:    map[string]chartPeriod{},
	}

	for _, period := range market.DefaultPeriods {
		series, ok := bundle.Series[period]
		if !ok || len(series.Points) < 2 {
			continue
		}
		points := make([]chartPoint, 0, len(series.Points))
		for _, point := range series.Points {
			points = append(points, chartPoint{
				Time:  point.Time.Format(time.RFC3339),
				Close: point.Close,
			})
		}
		start := series.Points[0].Close
		end := series.Points[len(series.Points)-1].Close
		periodView := chartPeriod{
			Label:       string(period),
			Start:       start,
			End:         end,
			Change:      end - start,
			ChangePct:   (end - start) / start * 100,
			SourceURL:   series.SourceURL,
			RetrievedAt: series.RetrievedAt.Format("2006-01-02 15:04:05 MST"),
			Points:      points,
		}
		view.Periods = append(view.Periods, periodView)
		view.ByPeriod[periodView.Label] = periodView
	}

	return view
}

func chartJSON(chart *chartView) template.JS {
	if chart == nil {
		return "{}"
	}
	raw, err := json.Marshal(chart)
	if err != nil {
		return "{}"
	}
	return template.JS(raw)
}

func formatMetric(metric analysis.Metric) string {
	switch metric.Unit {
	case "USD":
		return compactUSD(metric.Value)
	case "USD/shares":
		return "$" + formatFloat(metric.Value) + "/share"
	case "x":
		return formatFloat(metric.Value) + "x"
	default:
		return formatFloat(metric.Value) + " " + metric.Unit
	}
}

func compactUSD(value float64) string {
	abs := value
	if abs < 0 {
		abs = -abs
	}

	switch {
	case abs >= 1_000_000_000_000:
		return "$" + formatFloat(value/1_000_000_000_000) + "T"
	case abs >= 1_000_000_000:
		return "$" + formatFloat(value/1_000_000_000) + "B"
	case abs >= 1_000_000:
		return "$" + formatFloat(value/1_000_000) + "M"
	default:
		return "$" + formatFloat(value)
	}
}

func formatFloat(value float64) string {
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.2f", value), "0"), ".")
}

const indexHTML = `<!doctype html>
<html lang="en">
<head>
	  <meta charset="utf-8">
	  <meta name="viewport" content="width=device-width, initial-scale=1">
	  <title>Stockbridge</title>
	  <script>
	    (function () {
	      const validThemes = [
	        "bank-ledger",
	        "wall-street-terminal",
	        "federal-reserve-archive",
	        "ticker-tape",
	        "mahogany-desk",
	        "blueprint-analyst",
	        "monochrome-dossier",
	        "green-screen-mainframe",
	        "ivory-research-lab",
	        "crisis-room"
	      ];
	      const storedTheme = localStorage.getItem("stockbridge-theme");
	      const legacyTheme = storedTheme === "oldschool" ? "bank-ledger" : (storedTheme === "modern" ? "ivory-research-lab" : storedTheme);
	      document.documentElement.dataset.theme = validThemes.includes(legacyTheme) ? legacyTheme : "bank-ledger";
	    })();
	  </script>
	  <style>
	    @import url("https://fonts.googleapis.com/css2?family=Fira+Code:wght@400;500;600;700&display=swap");

	    :root {
	      --font-main: "Fira Code", monospace;
	      --bg: #e8d8b5;
	      --surface: #fbf3df;
	      --surface-alt: #f7ebcf;
	      --surface-inset: #fff7df;
	      --text: #241b15;
	      --text-strong: #22201c;
	      --muted: #746757;
	      --border: #b9a57b;
	      --border-strong: #6f5636;
	      --accent: #9b7530;
	      --accent-strong: #b08a3c;
	      --positive: #2f6048;
	      --negative: #763432;
	      --warning: #9b7530;
	      --chart-line: var(--positive);
	      --chart-grid: #d8c79c;
	      --button-bg: #22201c;
	      --button-hover: #763432;
	      --button-text: #fbf3df;
	      --focus-ring: #b08a3c;
	      --shadow: rgba(54, 39, 23, 0.18);
	      --paper-frame: rgba(111, 86, 54, 0.28);
	      --error-bg: #f7e2d4;
	      --notes-bg: #f8edcc;
	      --link: var(--positive);
	      --heading-bg: #433327;
	      --heading-text: #fbf3df;
	      --radius: 0;
	      --panel-shadow: 0 10px 24px var(--shadow);
	      --header-shadow: 0 10px 28px var(--shadow);
	      --show-stamps: block;
	      --brand-case: uppercase;
	      --input-case: uppercase;
	      --ink: var(--text);
	      --ink-soft: var(--heading-bg);
	      --paper: var(--surface);
	      --paper-deep: var(--surface-alt);
	      --panel: var(--surface-alt);
	      --line: var(--border);
	      --line-dark: var(--border-strong);
	      --brass: var(--accent);
	      --gold: var(--accent-strong);
	      --burgundy: var(--negative);
	      --forest: var(--positive);
	      --charcoal: var(--button-bg);
	    }
	    * { box-sizing: border-box; }
	    body {
	      margin: 0;
	      color: var(--ink);
	      background: var(--bg);
	      font-family: var(--font-main);
	      font-size: 14px;
	      line-height: 1.5;
	    }
	    .shell {
	      max-width: 1180px;
	      margin: 0 auto;
	      padding: 30px 20px 48px;
	    }
	    header {
	      position: relative;
	      display: grid;
	      gap: 16px;
	      margin-bottom: 22px;
	      padding: 22px;
	      background: var(--paper);
	      border: 1px solid var(--line-dark);
	      box-shadow: var(--header-shadow);
	    }
	    header::before,
	    .summary::before,
	    .chart-panel::before,
	    .notes::before,
	    .sources::before {
	      content: "";
	      position: absolute;
	      inset: 7px;
	      border: 1px solid var(--paper-frame);
	      pointer-events: none;
	    }
	    .brand {
	      display: flex;
	      align-items: baseline;
	      flex-wrap: wrap;
	      gap: 10px 14px;
	      border-bottom: 1px solid var(--line);
	      padding-bottom: 12px;
	    }
	    .header-top {
	      display: flex;
	      justify-content: space-between;
	      align-items: flex-start;
	      gap: 16px;
	    }
	    .header-actions {
	      display: flex;
	      align-items: flex-start;
	      gap: 10px;
	      flex: 0 0 auto;
	    }
	    .brand strong {
	      color: var(--paper);
	      background: var(--charcoal);
	      border: 1px solid var(--line-dark);
	      padding: 5px 12px;
	      font-size: 22px;
	      letter-spacing: 0.04em;
	      text-transform: var(--brand-case);
	    }
	    .brand span {
	      color: var(--burgundy);
	      font-weight: 700;
	      text-transform: uppercase;
	      letter-spacing: 0.08em;
	      font-size: 12px;
	    }
	    .deskline {
	      color: var(--muted);
	      font-size: 12px;
	      font-weight: 700;
	      letter-spacing: 0.08em;
	      text-transform: uppercase;
	    }
	    form {
	      display: flex;
	      flex-wrap: wrap;
	      gap: 10px;
	      max-width: 640px;
	      padding: 12px;
	      background: var(--panel);
	      border: 1px solid var(--line);
	      box-shadow: inset 0 0 0 1px rgba(255, 255, 255, 0.35);
	    }
	    .form-label {
	      width: 100%;
	      color: var(--burgundy);
	      font-size: 12px;
	      font-weight: 800;
	      letter-spacing: 0.12em;
	      text-transform: uppercase;
	    }
	    input[type="search"] {
	      flex: 1 1 260px;
	      min-height: 46px;
	      border: 1px solid var(--line-dark);
	      border-radius: 0;
	      padding: 0 14px;
	      font-family: var(--font-main);
	      font-size: 18px;
	      font-weight: 600;
	      letter-spacing: 0.04em;
	      text-transform: var(--input-case);
	      color: var(--ink);
	      background: var(--surface-inset);
	      box-shadow: inset 0 1px 3px rgba(54, 39, 23, 0.16);
	    }
	    button {
	      min-height: 46px;
	      border: 1px solid var(--line-dark);
	      border-radius: 0;
	      padding: 0 18px;
	      background: var(--charcoal);
	      color: var(--paper);
	      font-family: var(--font-main);
	      font-weight: 800;
	      letter-spacing: 0.05em;
	      text-transform: uppercase;
	      cursor: pointer;
	    }
	    button:hover {
	      background: var(--button-hover);
	    }
	    button:focus-visible,
	    input[type="search"]:focus-visible,
	    a:focus-visible {
	      outline: 2px solid var(--focus-ring);
	      outline-offset: 2px;
	    }
	    .theme-toggle {
	      width: 116px;
	      min-height: 36px;
	      padding: 0 12px;
	      white-space: nowrap;
	      font-size: 12px;
	    }
	    .watchlist-toggle {
	      width: 132px;
	      min-height: 36px;
	      padding: 0 12px;
	      white-space: nowrap;
	      font-size: 12px;
	    }
	    .theme-control {
	      position: relative;
	      flex: 0 0 auto;
	    }
	    .watchlist-control {
	      position: relative;
	      flex: 0 0 auto;
	    }
	    .theme-menu {
	      position: absolute;
	      top: calc(100% + 8px);
	      right: 0;
	      z-index: 20;
	      display: none;
	      width: 310px;
	      max-height: min(520px, calc(100vh - 120px));
	      overflow-y: auto;
	      padding: 10px;
	      background: var(--paper);
	      border: 1px solid var(--line-dark);
	      box-shadow: 0 12px 26px var(--shadow);
	    }
	    .theme-menu.open {
	      display: grid;
	      gap: 8px;
	    }
	    .theme-menu-title {
	      color: var(--muted);
	      font-size: 11px;
	      font-weight: 800;
	      letter-spacing: 0.1em;
	      text-transform: uppercase;
	    }
	    .theme-option {
	      display: flex;
	      align-items: center;
	      justify-content: space-between;
	      width: 100%;
	      min-height: 40px;
	      padding: 0 10px;
	      background: var(--panel);
	      color: var(--ink);
	      border: 1px solid var(--line);
	      font-size: 12px;
	      text-align: left;
	    }
	    .theme-option:hover,
	    .theme-option.active {
	      background: var(--burgundy);
	      color: var(--paper);
	      border-color: var(--line-dark);
	    }
	    .theme-option-check {
	      visibility: hidden;
	      font-weight: 900;
	    }
	    .theme-option.active .theme-option-check {
	      visibility: visible;
	    }
	    .watchlist-panel {
	      position: absolute;
	      top: calc(100% + 8px);
	      right: 0;
	      z-index: 25;
	      display: none;
	      width: 310px;
	      padding: 14px;
	      background: var(--paper);
	      border: 1px solid var(--line-dark);
	      box-shadow: 0 14px 30px var(--shadow);
	    }
	    .watchlist-panel.open {
	      display: block;
	    }
	    .watchlist-title {
	      margin: 0 0 10px;
	    }
	    .watchlist-items {
	      display: grid;
	      gap: 8px;
	    }
	    .watchlist-row {
	      display: grid;
	      grid-template-columns: 1fr 38px;
	      gap: 8px;
	      align-items: center;
	    }
	    .watchlist-link,
	    .watchlist-remove {
	      min-height: 38px;
	      padding: 0 10px;
	      font-size: 12px;
	    }
	    .watchlist-link {
	      width: 100%;
	      background: var(--panel);
	      color: var(--ink);
	      border: 1px solid var(--line);
	      text-align: left;
	    }
	    .watchlist-remove {
	      width: 38px;
	      padding: 0;
	      background: transparent;
	      color: var(--burgundy);
	      border: 1px solid var(--line);
	      font-size: 18px;
	      line-height: 1;
	    }
	    .watchlist-link:hover,
	    .watchlist-link:focus {
	      background: var(--forest);
	      border-color: var(--forest);
	      color: var(--paper);
	    }
	    .watchlist-remove:hover,
	    .watchlist-remove:focus {
	      background: var(--burgundy);
	      border-color: var(--burgundy);
	      color: var(--paper);
	    }
	    .watchlist-empty {
	      margin: 0;
	      color: var(--muted);
	      font-size: 13px;
	    }
	    .error {
	      border: 1px solid var(--burgundy);
	      border-left-width: 6px;
	      background: var(--error-bg);
	      color: var(--burgundy);
	      padding: 14px 16px;
	      margin: 18px 0;
	      font-weight: 700;
	      box-shadow: 0 8px 18px var(--shadow);
	    }
	    .empty {
	      background: var(--paper);
	      border: 1px solid var(--line);
	      padding: 26px;
	      color: var(--muted);
	      box-shadow: 0 8px 18px var(--shadow);
	    }
	    .report {
	      display: grid;
	      gap: 18px;
	    }
	    .summary, .chart-panel, .notes, .sources {
	      position: relative;
	      background: var(--paper);
	      border: 1px solid var(--line-dark);
	      padding: 22px;
	      box-shadow: var(--panel-shadow);
	    }
	    h1, h2 {
	      margin: 0;
	      letter-spacing: 0;
	    }
	    h1 {
	      color: var(--ink);
	      font-size: 25px;
	      line-height: 1.2;
	      letter-spacing: 0.03em;
	      text-transform: uppercase;
	    }
	    .company-heading {
	      display: flex;
	      align-items: center;
	      gap: 10px;
	      flex-wrap: wrap;
	    }
	    .watchlist-star {
	      display: inline-flex;
	      align-items: center;
	      justify-content: center;
	      width: 40px;
	      min-width: 40px;
	      height: 40px;
	      min-height: 40px;
	      padding: 0;
	      background: transparent;
	      border: 1px solid var(--line);
	      color: var(--line);
	      font-family: var(--font-main);
	      font-size: 22px;
	      line-height: 1;
	      letter-spacing: 0;
	      text-transform: none;
	    }
	    .watchlist-star:hover,
	    .watchlist-star.active {
	      background: var(--panel);
	      border-color: var(--forest);
	      color: var(--forest);
	    }
	    .dossier-label {
	      display: inline-block;
	      margin-bottom: 10px;
	      color: var(--burgundy);
	      border: 1px solid var(--burgundy);
	      padding: 3px 8px;
	      font-size: 11px;
	      font-weight: 800;
	      letter-spacing: 0.12em;
	      text-transform: uppercase;
	      transform: rotate(-0.4deg);
	    }
	    h2 {
	      display: inline-block;
	      color: var(--paper);
	      background: var(--ink-soft);
	      border: 1px solid var(--line-dark);
	      border-bottom: 3px double var(--gold);
	      padding: 5px 11px;
	      font-size: 13px;
	      text-transform: uppercase;
	      margin-bottom: 12px;
	      letter-spacing: 0.09em;
	    }
	    .meta {
	      display: grid;
	      grid-template-columns: repeat(auto-fit, minmax(210px, 1fr));
	      gap: 10px;
	      margin-top: 16px;
	    }
	    .meta div {
	      background: var(--panel);
	      border: 1px solid var(--line);
	      padding: 10px 12px;
	    }
	    .label {
	      display: block;
	      color: var(--muted);
	      font-size: 12px;
	      font-weight: 800;
	      text-transform: uppercase;
	      margin-bottom: 4px;
	      letter-spacing: 0.08em;
	    }
	    .metrics {
	      width: 100%;
	      border-collapse: collapse;
	      background: var(--paper);
	      border: 1px solid var(--line-dark);
	      overflow: hidden;
	      box-shadow: 0 8px 18px var(--shadow);
	    }
	    .metrics th, .metrics td {
	      text-align: left;
	      border-bottom: 1px solid var(--line);
	      padding: 11px 12px;
	      vertical-align: top;
	    }
	    .metrics th {
	      background: var(--ink-soft);
	      color: var(--paper);
	      font-size: 12px;
	      text-transform: uppercase;
	      letter-spacing: 0.08em;
	    }
	    .metrics tr:last-child td { border-bottom: 0; }
	    .metric-name { font-weight: 800; }
	    .metric-value {
	      font-weight: 900;
	      color: var(--forest);
	      white-space: nowrap;
	    }
	    .metric-value.positive { color: var(--forest); }
	    .metric-value.caution { color: var(--brass); }
	    .metric-detail {
	      color: var(--muted);
	      font-size: 13px;
	    }
	    .chart-panel {
	      padding: 22px;
	    }
	    .chart-head {
	      display: flex;
	      justify-content: space-between;
	      align-items: flex-start;
      gap: 16px;
      margin-bottom: 12px;
    }
	    .chart-price {
	      font-size: 28px;
	      line-height: 1;
	      font-weight: 900;
	      color: var(--ink);
      text-align: right;
      white-space: nowrap;
    }
	    .chart-change {
	      margin-top: 4px;
	      font-weight: 900;
	      color: var(--forest);
	      text-align: right;
	    }
	    .chart-change.down { color: var(--burgundy); }
	    .range-tabs {
	      display: flex;
	      flex-wrap: wrap;
	      gap: 8px;
	      margin: 10px 0 14px;
    }
    .range-tab {
	      min-height: 34px;
	      border: 1px solid var(--line);
	      border-radius: 0;
	      padding: 0 13px;
	      background: var(--panel);
	      color: var(--ink);
	      font-family: var(--font-main);
	      font-weight: 900;
	      letter-spacing: 0.04em;
	      cursor: pointer;
	    }
	    .range-tab.active {
	      background: var(--burgundy);
	      border-color: var(--line-dark);
	      color: var(--paper);
	    }
	    .chart-wrap {
	      position: relative;
	      width: 100%;
	      min-height: 330px;
	      background: var(--surface-inset);
	      border: 1px solid var(--line);
	      overflow: hidden;
	      box-shadow: inset 0 0 0 1px rgba(111, 86, 54, 0.14);
	    }
    .chart-wrap svg {
      display: block;
      width: 100%;
      height: 330px;
    }
	    .chart-grid {
	      stroke: var(--chart-grid);
	      stroke-width: 1;
	    }
    .chart-axis-label {
      fill: var(--muted);
      font-size: 11px;
      font-weight: 700;
    }
	    .chart-line {
	      fill: none;
	      stroke-width: 2.5;
	      stroke-linecap: round;
	      stroke-linejoin: round;
	    }
    .chart-area {
      opacity: 0.12;
    }
	    .chart-dot {
	      stroke: var(--paper);
	      stroke-width: 2;
	    }
    .chart-source {
      margin-top: 10px;
      color: var(--muted);
      font-size: 13px;
    }
	    .notes {
	      border-color: var(--brass);
	      background: var(--notes-bg);
	    }
    ul {
      margin: 0;
      padding-left: 20px;
    }
    li { margin: 8px 0; }
	    .sources a {
	      color: var(--link);
	      overflow-wrap: anywhere;
	    }
    .source {
      padding: 10px 0;
      border-bottom: 1px solid var(--line);
    }
	    .source:last-child { border-bottom: 0; }
	    html[data-theme="wall-street-terminal"] {
	      --bg: #090d0c;
	      --surface: #111817;
	      --surface-alt: #17211f;
	      --surface-inset: #0c1211;
	      --text: #e8efe4;
	      --text-strong: #f4f0dc;
	      --muted: #94a69a;
	      --border: #2f453c;
	      --border-strong: #516758;
	      --accent: #b08a4a;
	      --accent-strong: #d0aa5b;
	      --positive: #68a878;
	      --negative: #b36a5e;
	      --warning: #c89a4c;
	      --chart-grid: #24362f;
	      --button-bg: #1f332b;
	      --button-hover: #2e523f;
	      --button-text: #f1efd8;
	      --focus-ring: #d0aa5b;
	      --shadow: rgba(0, 0, 0, 0.42);
	      --paper-frame: rgba(104, 168, 120, 0.22);
	      --error-bg: #261817;
	      --notes-bg: #171c17;
	      --heading-bg: #1f332b;
	    }
	    html[data-theme="federal-reserve-archive"] {
	      --bg: #dfe4e9;
	      --surface: #f7f8f6;
	      --surface-alt: #eceff1;
	      --surface-inset: #ffffff;
	      --text: #15253a;
	      --text-strong: #0c1f35;
	      --muted: #657384;
	      --border: #b7c0c9;
	      --border-strong: #65778b;
	      --accent: #3f678f;
	      --accent-strong: #244b73;
	      --positive: #2f6f5d;
	      --negative: #8a3d4b;
	      --warning: #8f6a33;
	      --chart-grid: #c8d1da;
	      --button-bg: #1f3f63;
	      --button-hover: #2e5d88;
	      --button-text: #f7f8f6;
	      --focus-ring: #507aa5;
	      --shadow: rgba(27, 45, 69, 0.14);
	      --paper-frame: rgba(31, 63, 99, 0.16);
	      --error-bg: #f2e2e4;
	      --notes-bg: #f4f1e8;
	      --heading-bg: #1f3f63;
	    }
	    html[data-theme="ticker-tape"] {
	      --bg: #e4dcc9;
	      --surface: #f6f0df;
	      --surface-alt: #ebe3cf;
	      --surface-inset: #fbf7e9;
	      --text: #1f1f1b;
	      --text-strong: #11110f;
	      --muted: #6c685e;
	      --border: #beb29a;
	      --border-strong: #6f695c;
	      --accent: #7c7464;
	      --accent-strong: #35322d;
	      --positive: #4f7854;
	      --negative: #9a4a42;
	      --warning: #9a7442;
	      --chart-grid: #d1c7ae;
	      --button-bg: #2f2d28;
	      --button-hover: #5b574b;
	      --button-text: #f6f0df;
	      --focus-ring: #9a4a42;
	      --shadow: rgba(33, 29, 22, 0.12);
	      --paper-frame: rgba(31, 31, 27, 0.18);
	      --error-bg: #efe0d8;
	      --notes-bg: #f0ead8;
	      --heading-bg: #2f2d28;
	    }
	    html[data-theme="mahogany-desk"] {
	      --bg: #23140f;
	      --surface: #f3e5c8;
	      --surface-alt: #e3cda7;
	      --surface-inset: #fff0cf;
	      --text: #2a160e;
	      --text-strong: #1a0d08;
	      --muted: #725d48;
	      --border: #9b7445;
	      --border-strong: #4b2819;
	      --accent: #b08a43;
	      --accent-strong: #d0a750;
	      --positive: #416a4f;
	      --negative: #792f2e;
	      --warning: #a87932;
	      --chart-grid: #c7a978;
	      --button-bg: #3a1c12;
	      --button-hover: #682e22;
	      --button-text: #f3e5c8;
	      --focus-ring: #d0a750;
	      --shadow: rgba(15, 7, 4, 0.34);
	      --paper-frame: rgba(75, 40, 25, 0.28);
	      --error-bg: #ead2c6;
	      --notes-bg: #ead9b8;
	      --heading-bg: #3a1c12;
	    }
	    html[data-theme="blueprint-analyst"] {
	      --bg: #081d34;
	      --surface: #0f2b48;
	      --surface-alt: #163a5d;
	      --surface-inset: #09243e;
	      --text: #e1eef6;
	      --text-strong: #f3fbff;
	      --muted: #9bb5c9;
	      --border: #4b7191;
	      --border-strong: #8db6d1;
	      --accent: #7db7ce;
	      --accent-strong: #b4d7e7;
	      --positive: #7fb99a;
	      --negative: #d08b83;
	      --warning: #d0b06b;
	      --chart-grid: #2f5472;
	      --button-bg: #17496f;
	      --button-hover: #1f638e;
	      --button-text: #f3fbff;
	      --focus-ring: #b4d7e7;
	      --shadow: rgba(0, 10, 22, 0.38);
	      --paper-frame: rgba(180, 215, 231, 0.22);
	      --error-bg: #2b252d;
	      --notes-bg: #102f4c;
	      --heading-bg: #17496f;
	    }
	    html[data-theme="monochrome-dossier"] {
	      --bg: #d8d8d4;
	      --surface: #fbfbf8;
	      --surface-alt: #ececea;
	      --surface-inset: #ffffff;
	      --text: #111111;
	      --text-strong: #000000;
	      --muted: #666666;
	      --border: #b9b9b5;
	      --border-strong: #3b3b3b;
	      --accent: #555555;
	      --accent-strong: #222222;
	      --positive: #2f2f2f;
	      --negative: #111111;
	      --warning: #5c5c5c;
	      --chart-grid: #d0d0cc;
	      --button-bg: #111111;
	      --button-hover: #333333;
	      --button-text: #fbfbf8;
	      --focus-ring: #000000;
	      --shadow: rgba(0, 0, 0, 0.12);
	      --paper-frame: rgba(0, 0, 0, 0.16);
	      --error-bg: #eeeeec;
	      --notes-bg: #f2f2ef;
	      --heading-bg: #111111;
	    }
	    html[data-theme="green-screen-mainframe"] {
	      --bg: #03110b;
	      --surface: #071b12;
	      --surface-alt: #0b2719;
	      --surface-inset: #04150d;
	      --text: #bde7bd;
	      --text-strong: #d8ffd1;
	      --muted: #79a978;
	      --border: #245c35;
	      --border-strong: #63a86d;
	      --accent: #97c15b;
	      --accent-strong: #d0b55f;
	      --positive: #79d07e;
	      --negative: #c6805f;
	      --warning: #d0b55f;
	      --chart-grid: #174429;
	      --button-bg: #10351f;
	      --button-hover: #1a5732;
	      --button-text: #d8ffd1;
	      --focus-ring: #97c15b;
	      --shadow: rgba(0, 0, 0, 0.48);
	      --paper-frame: rgba(121, 208, 126, 0.22);
	      --error-bg: #1f1710;
	      --notes-bg: #0b2114;
	      --heading-bg: #10351f;
	    }
	    html[data-theme="ivory-research-lab"] {
	      --bg: #eee9dc;
	      --surface: #fffdf4;
	      --surface-alt: #f4f0e4;
	      --surface-inset: #ffffff;
	      --text: #20242a;
	      --text-strong: #111827;
	      --muted: #6d7278;
	      --border: #d1c9b8;
	      --border-strong: #817866;
	      --accent: #7b8796;
	      --accent-strong: #33445a;
	      --positive: #52715c;
	      --negative: #954f4b;
	      --warning: #9b7b45;
	      --chart-grid: #ddd6c7;
	      --button-bg: #33445a;
	      --button-hover: #465d78;
	      --button-text: #fffdf4;
	      --focus-ring: #7b8796;
	      --shadow: rgba(32, 36, 42, 0.1);
	      --paper-frame: rgba(51, 68, 90, 0.13);
	      --error-bg: #f2e5df;
	      --notes-bg: #f7f1df;
	      --heading-bg: #33445a;
	    }
	    html[data-theme="crisis-room"] {
	      --bg: #15191d;
	      --surface: #20262c;
	      --surface-alt: #2b333a;
	      --surface-inset: #171d22;
	      --text: #f0ede6;
	      --text-strong: #fff8ec;
	      --muted: #aeb7bd;
	      --border: #4c5962;
	      --border-strong: #7a858c;
	      --accent: #d19a48;
	      --accent-strong: #e2b661;
	      --positive: #6ea36f;
	      --negative: #c15b55;
	      --warning: #d19a48;
	      --chart-grid: #39434b;
	      --button-bg: #7d302f;
	      --button-hover: #9d403c;
	      --button-text: #fff8ec;
	      --focus-ring: #e2b661;
	      --shadow: rgba(0, 0, 0, 0.36);
	      --paper-frame: rgba(209, 154, 72, 0.18);
	      --error-bg: #331f20;
	      --notes-bg: #2e2920;
	      --heading-bg: #7d302f;
	    }
	    html[data-theme] {
	      --ink: var(--text);
	      --ink-soft: var(--heading-bg);
	      --paper: var(--surface);
	      --paper-deep: var(--surface-alt);
	      --panel: var(--surface-alt);
	      --line: var(--border);
	      --line-dark: var(--border-strong);
	      --brass: var(--accent);
	      --gold: var(--accent-strong);
	      --burgundy: var(--negative);
	      --forest: var(--positive);
	      --charcoal: var(--button-bg);
	    }
	    @media (max-width: 720px) {
	      form { flex-direction: column; }
	      button { width: 100%; }
	      .theme-toggle { width: 116px; }
	      .header-actions { flex-wrap: wrap; justify-content: flex-end; }
	      .watchlist-toggle { width: 132px; }
	      .watchlist-remove { width: 38px; }
	      .watchlist-panel { right: 0; width: min(310px, calc(100vw - 40px)); }
	      .theme-menu { right: 0; width: min(310px, calc(100vw - 40px)); }
	      .metrics { display: block; overflow-x: auto; }
	    }
  </style>
</head>
<body>
	  <main class="shell">
	    <header>
	      <div class="header-top">
	        <div class="brand">
	          <strong>Stockbridge</strong>
	          <span>Market Ledger / Fundamental Analysis</span>
	        </div>
	        <div class="header-actions">
	          <div class="watchlist-control">
	            <button id="watchlist-toggle" class="watchlist-toggle" type="button" aria-haspopup="true" aria-expanded="false" aria-controls="watchlist-panel">Watchlist</button>
	            <section id="watchlist-panel" class="watchlist-panel" aria-labelledby="watchlist-title">
	              <h2 id="watchlist-title" class="watchlist-title">Saved Tickers</h2>
	              <div id="watchlist-items" class="watchlist-items"></div>
	            </section>
	          </div>
	          <div class="theme-control">
	            <button id="theme-toggle" class="theme-toggle" type="button" aria-haspopup="menu" aria-expanded="false">Theme</button>
	            <div id="theme-menu" class="theme-menu" role="menu" aria-label="Theme options">
	              <div class="theme-menu-title">Choose theme</div>
	              <button class="theme-option" type="button" role="menuitemradio" aria-checked="true" data-theme-choice="bank-ledger">
	                <span>Bank Ledger</span>
	                <span class="theme-option-check">Selected</span>
	              </button>
	              <button class="theme-option" type="button" role="menuitemradio" aria-checked="false" data-theme-choice="wall-street-terminal">
	                <span>Wall Street Terminal</span>
	                <span class="theme-option-check">Selected</span>
	              </button>
	              <button class="theme-option" type="button" role="menuitemradio" aria-checked="false" data-theme-choice="federal-reserve-archive">
	                <span>Federal Reserve Archive</span>
	                <span class="theme-option-check">Selected</span>
	              </button>
	              <button class="theme-option" type="button" role="menuitemradio" aria-checked="false" data-theme-choice="ticker-tape">
	                <span>Ticker Tape</span>
	                <span class="theme-option-check">Selected</span>
	              </button>
	              <button class="theme-option" type="button" role="menuitemradio" aria-checked="false" data-theme-choice="mahogany-desk">
	                <span>Mahogany Desk</span>
	                <span class="theme-option-check">Selected</span>
	              </button>
	              <button class="theme-option" type="button" role="menuitemradio" aria-checked="false" data-theme-choice="blueprint-analyst">
	                <span>Blueprint Analyst</span>
	                <span class="theme-option-check">Selected</span>
	              </button>
	              <button class="theme-option" type="button" role="menuitemradio" aria-checked="false" data-theme-choice="monochrome-dossier">
	                <span>Monochrome Dossier</span>
	                <span class="theme-option-check">Selected</span>
	              </button>
	              <button class="theme-option" type="button" role="menuitemradio" aria-checked="false" data-theme-choice="green-screen-mainframe">
	                <span>Green Screen Mainframe</span>
	                <span class="theme-option-check">Selected</span>
	              </button>
	              <button class="theme-option" type="button" role="menuitemradio" aria-checked="false" data-theme-choice="ivory-research-lab">
	                <span>Ivory Research Lab</span>
	                <span class="theme-option-check">Selected</span>
	              </button>
	              <button class="theme-option" type="button" role="menuitemradio" aria-checked="false" data-theme-choice="crisis-room">
	                <span>Crisis Room</span>
	                <span class="theme-option-check">Selected</span>
	              </button>
	            </div>
	          </div>
	        </div>
	      </div>
	      <div class="deskline">Ticker lookup desk · archival company file · report generated {{.Generated}}</div>
	      <form action="/" method="get">
	        <label class="form-label" for="ticker-input">Ticker lookup</label>
	        <input id="ticker-input" type="search" name="ticker" value="{{.Query}}" placeholder="ENTER TICKER, E.G. AMZN" autocomplete="off" autofocus>
	        <button type="submit">Analyze</button>
	      </form>
    </header>

    {{if .Error}}
      <div class="error">{{.Error}}</div>
    {{end}}

    {{if .Report}}
	      <section class="report">
	        <div class="summary">
	          <div class="dossier-label">Company file</div>
	          <div class="company-heading">
	            <h1>{{.Report.CompanyName}} ({{.Report.Ticker}})</h1>
	            <button id="watchlist-star" class="watchlist-star" type="button" data-ticker="{{.Report.Ticker}}" aria-label="Add {{.Report.Ticker}} to watchlist">☆</button>
	          </div>
          <div class="meta">
            <div><span class="label">CIK</span>{{printf "%010d" .Report.CIK}}</div>
            <div><span class="label">Exchange</span>{{.Report.Listing.Exchange}}</div>
            {{if .Report.Listing.Market}}<div><span class="label">Market</span>{{.Report.Listing.Market}}</div>{{end}}
            <div><span class="label">Security</span>{{.Report.Listing.SecurityName}}</div>
            {{if .Report.Listing.FileCreatedAt}}<div><span class="label">Symbol File</span>{{.Report.Listing.FileCreatedAt}}</div>{{end}}
          </div>
        </div>

        <section>
          <h2>SEC Fundamentals</h2>
          <table class="metrics">
            <thead>
              <tr>
                <th>Metric</th>
                <th>Value</th>
                <th>Period</th>
                <th>Filing</th>
                <th>Concept</th>
              </tr>
            </thead>
            <tbody>
              {{range .Report.Metrics}}
                <tr>
                  <td class="metric-name">{{.Name}}</td>
                  <td class="metric-value {{if or (eq .Name "Net income") (eq .Name "Operating cash flow") (eq .Name "Basic EPS") (eq .Name "Diluted EPS")}}positive{{else if or (eq .Name "Liabilities") (eq .Name "Capital expenditures")}}caution{{end}}">{{formatMetric .}}</td>
                  <td class="metric-detail">{{.Period}}</td>
                  <td class="metric-detail">{{.Form}} {{if .Filed}}filed {{.Filed}}{{end}}</td>
                  <td class="metric-detail">{{.Concept}}</td>
                </tr>
              {{end}}
            </tbody>
          </table>
        </section>

        {{if .Chart}}
          <section class="chart-panel">
            <div class="chart-head">
	              <div>
	                <h2>Price Movement</h2>
	                <div class="metric-detail">Market ledger graph with connected historical close prices.</div>
	              </div>
              <div>
                <div id="chart-price" class="chart-price"></div>
                <div id="chart-change" class="chart-change"></div>
              </div>
            </div>
            <div id="range-tabs" class="range-tabs"></div>
            <div class="chart-wrap">
              <svg id="price-chart" viewBox="0 0 960 330" role="img" aria-label="Historical stock price chart"></svg>
            </div>
            <div id="chart-source" class="chart-source"></div>
          </section>
        {{end}}

        <section class="notes">
          <h2>Notes</h2>
          <ul>
            {{range .Report.Notes}}<li>{{.}}</li>{{end}}
          </ul>
        </section>

        <section class="sources">
          <h2>Sources</h2>
          {{range .Report.Sources}}
            <div class="source">
              <strong>{{.Name}}</strong><br>
              <a href="{{.URL}}" target="_blank" rel="noreferrer">{{.URL}}</a><br>
              <span class="metric-detail">retrieved: {{.RetrievedAt}}</span>
            </div>
          {{end}}
        </section>
      </section>
    {{else if not .Error}}
      <div class="empty">Type a ticker symbol and press Enter to generate a fundamentals report.</div>
    {{end}}
  </main>
  <script>
    (function () {
      const root = document.documentElement;
      const button = document.getElementById("theme-toggle");
      const menu = document.getElementById("theme-menu");
      const options = Array.from(document.querySelectorAll("[data-theme-choice]"));
      const themes = [
        { key: "bank-ledger", label: "Bank Ledger" },
        { key: "wall-street-terminal", label: "Wall Street Terminal" },
        { key: "federal-reserve-archive", label: "Federal Reserve Archive" },
        { key: "ticker-tape", label: "Ticker Tape" },
        { key: "mahogany-desk", label: "Mahogany Desk" },
        { key: "blueprint-analyst", label: "Blueprint Analyst" },
        { key: "monochrome-dossier", label: "Monochrome Dossier" },
        { key: "green-screen-mainframe", label: "Green Screen Mainframe" },
        { key: "ivory-research-lab", label: "Ivory Research Lab" },
        { key: "crisis-room", label: "Crisis Room" }
      ];
      const starButton = document.getElementById("watchlist-star");
      const watchlistButton = document.getElementById("watchlist-toggle");
      const watchlistPanel = document.getElementById("watchlist-panel");
      const watchlistItems = document.getElementById("watchlist-items");
      const searchForm = document.querySelector('form[action="/"]');
      const tickerInput = document.getElementById("ticker-input");
      const watchlistStorageKey = "stockbridge-watchlist";
      const themeStorageKey = "stockbridge-theme";

      function normalizeTheme(theme) {
        if (theme === "oldschool") return "bank-ledger";
        if (theme === "modern") return "ivory-research-lab";
        return String(theme || "").trim();
      }

      function isValidTheme(theme) {
        const normalizedTheme = normalizeTheme(theme);
        return themes.some(function (entry) { return entry.key === normalizedTheme; });
      }

      function getThemeLabel(theme) {
        const normalizedTheme = normalizeTheme(theme);
        const match = themes.find(function (entry) { return entry.key === normalizedTheme; });
        return match ? match.label : "Bank Ledger";
      }

      function getStoredTheme() {
        try {
          const storedTheme = normalizeTheme(localStorage.getItem(themeStorageKey));
          return isValidTheme(storedTheme) ? storedTheme : "bank-ledger";
        } catch (error) {
          return "bank-ledger";
        }
      }

      function saveTheme(theme) {
        const normalizedTheme = normalizeTheme(theme);
        if (!isValidTheme(normalizedTheme)) return;
        try {
          localStorage.setItem(themeStorageKey, normalizedTheme);
        } catch (error) {
          return;
        }
      }

      function applyTheme(theme) {
        const normalizedTheme = isValidTheme(theme) ? normalizeTheme(theme) : "bank-ledger";
        root.dataset.theme = normalizedTheme;
        if (button) {
          button.textContent = "Theme";
          button.setAttribute("aria-label", "Open theme menu. Current theme: " + getThemeLabel(normalizedTheme));
        }
        options.forEach(function (option) {
          const selected = option.dataset.themeChoice === normalizedTheme;
          option.classList.toggle("active", selected);
          option.setAttribute("aria-checked", selected ? "true" : "false");
        });
        if (window.renderChart && window.currentChartPeriod) {
          window.renderChart(window.currentChartPeriod);
        }
      }

      function initializeTheme() {
        const initialTheme = isValidTheme(root.dataset.theme) ? normalizeTheme(root.dataset.theme) : getStoredTheme();
        applyTheme(initialTheme);
        saveTheme(initialTheme);
      }

      function getWatchlist() {
        try {
          const parsed = JSON.parse(localStorage.getItem(watchlistStorageKey) || "[]");
          if (!Array.isArray(parsed)) return [];
          return parsed
            .map(function (ticker) { return String(ticker).trim().toUpperCase(); })
            .filter(Boolean)
            .filter(function (ticker, index, list) { return list.indexOf(ticker) === index; });
        } catch (error) {
          return [];
        }
      }

      function saveWatchlist(list) {
        const normalized = list
          .map(function (ticker) { return String(ticker).trim().toUpperCase(); })
          .filter(Boolean)
          .filter(function (ticker, index, normalizedList) { return normalizedList.indexOf(ticker) === index; });
        localStorage.setItem(watchlistStorageKey, JSON.stringify(normalized));
      }

      function isInWatchlist(ticker) {
        const normalizedTicker = String(ticker || "").trim().toUpperCase();
        return normalizedTicker !== "" && getWatchlist().includes(normalizedTicker);
      }

      function toggleWatchlistTicker(ticker) {
        const normalizedTicker = String(ticker || "").trim().toUpperCase();
        if (!normalizedTicker) return [];
        const list = getWatchlist();
        const nextList = list.includes(normalizedTicker)
          ? list.filter(function (savedTicker) { return savedTicker !== normalizedTicker; })
          : list.concat(normalizedTicker);
        saveWatchlist(nextList);
        return nextList;
      }

      function updateStarState(ticker) {
        if (!starButton) return;
        const normalizedTicker = String(ticker || starButton.dataset.ticker || "").trim().toUpperCase();
        const saved = isInWatchlist(normalizedTicker);
        starButton.dataset.ticker = normalizedTicker;
        starButton.classList.toggle("active", saved);
        starButton.textContent = saved ? "★" : "☆";
        starButton.setAttribute("aria-label", (saved ? "Remove " : "Add ") + normalizedTicker + (saved ? " from watchlist" : " to watchlist"));
      }

      function removeWatchlistTicker(ticker) {
        const normalizedTicker = String(ticker || "").trim().toUpperCase();
        saveWatchlist(getWatchlist().filter(function (savedTicker) { return savedTicker !== normalizedTicker; }));
      }

      function loadTickerFromWatchlist(ticker) {
        const normalizedTicker = String(ticker || "").trim().toUpperCase();
        if (!normalizedTicker) return;
        if (tickerInput) tickerInput.value = normalizedTicker;
        updateStarState(normalizedTicker);
        setWatchlistOpen(false);
        if (searchForm) {
          if (searchForm.requestSubmit) {
            searchForm.requestSubmit();
          } else {
            searchForm.submit();
          }
          return;
        }
        window.location.href = "/?ticker=" + encodeURIComponent(normalizedTicker);
      }

      function renderWatchlist() {
        if (!watchlistItems) return;
        const list = getWatchlist();
        watchlistItems.innerHTML = "";
        if (!list.length) {
          const empty = document.createElement("p");
          empty.className = "watchlist-empty";
          empty.textContent = "No tickers saved yet. Search a stock and click ★ to add it.";
          watchlistItems.appendChild(empty);
          return;
        }
        list.forEach(function (ticker) {
          const row = document.createElement("div");
          row.className = "watchlist-row";

          const tickerButton = document.createElement("button");
          tickerButton.type = "button";
          tickerButton.className = "watchlist-link";
          tickerButton.textContent = ticker;
          tickerButton.setAttribute("aria-label", "Load " + ticker + " report");
          tickerButton.addEventListener("click", function () {
            loadTickerFromWatchlist(ticker);
          });

          const removeButton = document.createElement("button");
          removeButton.type = "button";
          removeButton.className = "watchlist-remove";
          removeButton.textContent = "×";
          removeButton.setAttribute("aria-label", "Remove " + ticker + " from watchlist");
          removeButton.addEventListener("click", function (event) {
            event.stopPropagation();
            removeWatchlistTicker(ticker);
            renderWatchlist();
            if (starButton && starButton.dataset.ticker === ticker) {
              updateStarState(ticker);
            }
          });

          row.appendChild(tickerButton);
          row.appendChild(removeButton);
          watchlistItems.appendChild(row);
        });
      }

      function setMenuOpen(open) {
        if (!button || !menu) return;
        menu.classList.toggle("open", open);
        button.setAttribute("aria-expanded", open ? "true" : "false");
      }

      function setWatchlistOpen(open) {
        if (!watchlistButton || !watchlistPanel) return;
        if (open) renderWatchlist();
        watchlistPanel.classList.toggle("open", open);
        watchlistButton.setAttribute("aria-expanded", open ? "true" : "false");
      }

      initializeTheme();
      if (button && menu) {
        button.addEventListener("click", function () {
          setMenuOpen(!menu.classList.contains("open"));
          setWatchlistOpen(false);
        });
        options.forEach(function (option) {
          option.addEventListener("click", function () {
            applyTheme(option.dataset.themeChoice);
            saveTheme(option.dataset.themeChoice);
            setMenuOpen(false);
          });
        });
        document.addEventListener("click", function (event) {
          if (!menu.contains(event.target) && event.target !== button) {
            setMenuOpen(false);
          }
        });
        document.addEventListener("keydown", function (event) {
          if (event.key === "Escape") {
            setMenuOpen(false);
            button.focus();
          }
        });
      }

      if (starButton) {
        updateStarState(starButton.dataset.ticker);
        starButton.addEventListener("click", function () {
          const ticker = starButton.dataset.ticker;
          toggleWatchlistTicker(ticker);
          updateStarState(ticker);
          renderWatchlist();
        });
      }

      if (watchlistButton && watchlistPanel) {
        renderWatchlist();
        watchlistButton.addEventListener("click", function () {
          setWatchlistOpen(!watchlistPanel.classList.contains("open"));
          setMenuOpen(false);
        });
        document.addEventListener("click", function (event) {
          if (!watchlistPanel.contains(event.target) && event.target !== watchlistButton) {
            setWatchlistOpen(false);
          }
        });
        document.addEventListener("keydown", function (event) {
          if (event.key === "Escape" && watchlistPanel.classList.contains("open")) {
            setWatchlistOpen(false);
            watchlistButton.focus();
          }
        });
      }
    })();
  </script>
  {{if .Chart}}
  <script>
    const chartData = {{chartJSON .Chart}};
    const svg = document.getElementById("price-chart");
    const tabs = document.getElementById("range-tabs");
    const priceEl = document.getElementById("chart-price");
    const changeEl = document.getElementById("chart-change");
    const sourceEl = document.getElementById("chart-source");
	    const money = new Intl.NumberFormat("en-US", { style: "currency", currency: "USD" });

	    function linePath(points) {
	      return points.map((point, index) => (index === 0 ? "M" : "L") + " " + point.x.toFixed(2) + " " + point.y.toFixed(2)).join(" ");
	    }

    function renderChart(label) {
      const period = chartData.byPeriod[label] || chartData.periods[0];
      if (!period || !period.points.length) return;
      window.currentChartPeriod = period.label;

      document.querySelectorAll(".range-tab").forEach((button) => {
        button.classList.toggle("active", button.dataset.period === period.label);
      });

      const width = 960;
      const height = 330;
      const pad = { top: 24, right: 72, bottom: 36, left: 72 };
      const plotW = width - pad.left - pad.right;
      const plotH = height - pad.top - pad.bottom;
      const closes = period.points.map((point) => point.close);
      const min = Math.min(...closes);
      const max = Math.max(...closes);
      const range = Math.max(max - min, 1);
	      const styles = getComputedStyle(document.documentElement);
	      const upColor = styles.getPropertyValue("--positive").trim() || styles.getPropertyValue("--forest").trim() || "#2f6048";
	      const downColor = styles.getPropertyValue("--negative").trim() || styles.getPropertyValue("--burgundy").trim() || "#763432";
	      const lineColor = period.change >= 0 ? upColor : downColor;
      const mapped = period.points.map((point, index) => ({
        x: pad.left + (index / Math.max(period.points.length - 1, 1)) * plotW,
        y: pad.top + ((max - point.close) / range) * plotH,
        close: point.close,
        time: point.time
      }));

      const path = linePath(mapped);
	      const area = path + " L " + mapped[mapped.length - 1].x.toFixed(2) + " " + (height - pad.bottom) + " L " + mapped[0].x.toFixed(2) + " " + (height - pad.bottom) + " Z";
      const rows = 5;
      const grid = [];
      for (let i = 0; i <= rows; i++) {
        const y = pad.top + (i / rows) * plotH;
        const value = max - (i / rows) * range;
	        grid.push('<line class="chart-grid" x1="' + pad.left + '" y1="' + y + '" x2="' + (width - pad.right) + '" y2="' + y + '"></line>');
	        grid.push('<text class="chart-axis-label" x="' + (width - pad.right + 10) + '" y="' + (y + 4) + '">' + money.format(value) + '</text>');
      }

	      const first = mapped[0];
	      const last = mapped[mapped.length - 1];
	      svg.innerHTML =
	        grid.join("") +
	        '<path class="chart-area" d="' + area + '" fill="' + lineColor + '"></path>' +
	        '<path class="chart-line" d="' + path + '" stroke="' + lineColor + '"></path>' +
	        '<circle class="chart-dot" cx="' + first.x + '" cy="' + first.y + '" r="4" fill="' + lineColor + '"></circle>' +
	        '<circle class="chart-dot" cx="' + last.x + '" cy="' + last.y + '" r="5" fill="' + lineColor + '"></circle>' +
	        '<text class="chart-axis-label" x="' + pad.left + '" y="' + (height - 10) + '">' + new Date(period.points[0].time).toLocaleDateString() + '</text>' +
	        '<text class="chart-axis-label" text-anchor="end" x="' + (width - pad.right) + '" y="' + (height - 10) + '">' + new Date(period.points[period.points.length - 1].time).toLocaleDateString() + '</text>';

	      priceEl.textContent = money.format(period.end);
	      changeEl.textContent = (period.change >= 0 ? "+" : "") + money.format(period.change) + " (" + (period.changePct >= 0 ? "+" : "") + period.changePct.toFixed(2) + "%)";
	      changeEl.classList.toggle("down", period.change < 0);
	      sourceEl.textContent = period.label + " range - " + period.points.length + " closes - retrieved " + period.retrievedAt;
	    }

    if (chartData.periods && chartData.periods.length) {
      chartData.periods.forEach((period) => {
        const button = document.createElement("button");
        button.type = "button";
        button.className = "range-tab";
        button.dataset.period = period.label;
        button.textContent = period.label;
        button.addEventListener("click", () => renderChart(period.label));
        tabs.appendChild(button);
      });
      renderChart(chartData.periods[0].label);
    }
    window.renderChart = renderChart;
  </script>
  {{end}}
</body>
</html>`
