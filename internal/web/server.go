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
	    const savedTheme = localStorage.getItem("stockbridge-theme") || "oldschool";
	    document.documentElement.dataset.theme = savedTheme;
	  </script>
	  <style>
	    @import url("https://fonts.googleapis.com/css2?family=Fira+Code:wght@400;500;600;700&display=swap");

	    :root {
	      --font-main: "Fira Code", monospace;
	      --ink: #241b15;
	      --ink-soft: #433327;
	      --muted: #746757;
	      --paper: #fbf3df;
	      --paper-deep: #f1e2bd;
	      --panel: #f7ebcf;
	      --bg: #e8d8b5;
	      --line: #b9a57b;
	      --line-dark: #6f5636;
	      --brass: #9b7530;
	      --gold: #b08a3c;
	      --burgundy: #763432;
	      --forest: #2f6048;
	      --charcoal: #22201c;
	      --shadow: rgba(54, 39, 23, 0.18);
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
	      box-shadow: 0 10px 28px var(--shadow);
	    }
	    header::before,
	    .summary::before,
	    .chart-panel::before,
	    .notes::before,
	    .sources::before {
	      content: "";
	      position: absolute;
	      inset: 7px;
	      border: 1px solid rgba(111, 86, 54, 0.28);
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
	    .brand strong {
	      color: var(--paper);
	      background: var(--charcoal);
	      border: 1px solid var(--line-dark);
	      padding: 5px 12px;
	      font-size: 22px;
	      letter-spacing: 0.04em;
	      text-transform: uppercase;
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
	      text-transform: uppercase;
	      color: var(--ink);
	      background: #fff7df;
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
	      background: var(--burgundy);
	    }
	    .theme-toggle {
	      width: 116px;
	      min-height: 36px;
	      padding: 0 12px;
	      white-space: nowrap;
	      font-size: 12px;
	    }
	    .theme-control {
	      position: relative;
	      flex: 0 0 auto;
	    }
	    .theme-menu {
	      position: absolute;
	      top: calc(100% + 8px);
	      right: 0;
	      z-index: 20;
	      display: none;
	      width: 240px;
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
	    .error {
	      border: 1px solid var(--burgundy);
	      border-left-width: 6px;
	      background: #f7e2d4;
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
	      box-shadow: 0 10px 24px var(--shadow);
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
	      background: #fff7df;
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
	      stroke: #d8c79c;
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
	      background: #f8edcc;
	    }
    ul {
      margin: 0;
      padding-left: 20px;
    }
    li { margin: 8px 0; }
	    .sources a {
	      color: var(--forest);
	      overflow-wrap: anywhere;
	    }
    .source {
      padding: 10px 0;
      border-bottom: 1px solid var(--line);
    }
	    .source:last-child { border-bottom: 0; }
	    html[data-theme="modern"] {
	      --ink: #18202f;
	      --ink-soft: #303747;
	      --muted: #657187;
	      --panel: #f7f8fb;
	      --line: #d9deea;
	      --line-dark: #d9deea;
	      --paper: #ffffff;
	      --bg: #eef2f7;
	      --forest: #138a5b;
	      --brass: #a96700;
	      --burgundy: #b64242;
	      --charcoal: #1455d9;
	      --gold: #008fb3;
	      --shadow: rgba(24, 32, 47, 0.08);
	    }
	    html[data-theme="modern"] body {
	      font-family: ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
	      font-size: 16px;
	      background: var(--bg);
	    }
	    html[data-theme="modern"] .shell {
	      padding: 28px 20px 44px;
	    }
	    html[data-theme="modern"] header,
	    html[data-theme="modern"] .summary,
	    html[data-theme="modern"] .chart-panel,
	    html[data-theme="modern"] .notes,
	    html[data-theme="modern"] .sources,
	    html[data-theme="modern"] .empty {
	      border: 1px solid var(--line);
	      border-radius: 8px;
	      box-shadow: none;
	    }
	    html[data-theme="modern"] header {
	      padding: 0;
	      background: transparent;
	      border: 0;
	      gap: 18px;
	      margin-bottom: 24px;
	    }
	    html[data-theme="modern"] header::before,
	    html[data-theme="modern"] .summary::before,
	    html[data-theme="modern"] .chart-panel::before,
	    html[data-theme="modern"] .notes::before,
	    html[data-theme="modern"] .sources::before {
	      display: none;
	    }
	    html[data-theme="modern"] .brand {
	      border-bottom: 0;
	      padding-bottom: 0;
	    }
	    html[data-theme="modern"] .brand strong {
	      background: #1455d9;
	      color: white;
	      border: 0;
	      border-radius: 6px;
	      letter-spacing: 0;
	      text-transform: none;
	    }
	    html[data-theme="modern"] .brand span {
	      color: #6f5bb5;
	      letter-spacing: 0;
	      text-transform: none;
	      font-size: 16px;
	    }
	    html[data-theme="modern"] .deskline,
	    html[data-theme="modern"] .form-label,
	    html[data-theme="modern"] .dossier-label {
	      display: none;
	    }
	    html[data-theme="modern"] form {
	      max-width: 560px;
	      padding: 0;
	      background: transparent;
	      border: 0;
	      box-shadow: none;
	      flex-wrap: nowrap;
	    }
	    html[data-theme="modern"] input[type="search"] {
	      width: 100%;
	      min-height: 46px;
	      border: 1px solid var(--line);
	      border-radius: 6px;
	      background: white;
	      box-shadow: none;
	      font-family: ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
	      font-size: 18px;
	      font-weight: 400;
	      letter-spacing: 0;
	      text-transform: none;
	    }
	    html[data-theme="modern"] button {
	      border: 0;
	      border-radius: 6px;
	      background: #1455d9;
	      color: white;
	      font-family: ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
	      letter-spacing: 0;
	      text-transform: none;
	    }
	    html[data-theme="modern"] button:hover {
	      background: #0f45b0;
	    }
	    html[data-theme="modern"] .theme-menu {
	      background: white;
	      border: 1px solid var(--line);
	      border-radius: 8px;
	      box-shadow: 0 14px 32px rgba(24, 32, 47, 0.12);
	    }
	    html[data-theme="modern"] .theme-option {
	      background: var(--panel);
	      border: 1px solid var(--line);
	      border-radius: 6px;
	      color: var(--ink);
	      font-family: ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
	    }
	    html[data-theme="modern"] .theme-option:hover,
	    html[data-theme="modern"] .theme-option.active {
	      background: #1455d9;
	      border-color: #1455d9;
	      color: white;
	    }
	    html[data-theme="modern"] h1 {
	      color: #008fb3;
	      font-size: 26px;
	      letter-spacing: 0;
	      text-transform: none;
	    }
	    html[data-theme="modern"] h2 {
	      color: white;
	      background: #303747;
	      border: 0;
	      border-bottom: 3px solid #008fb3;
	      border-radius: 4px 4px 0 0;
	      font-size: 15px;
	      letter-spacing: 0;
	    }
	    html[data-theme="modern"] .meta div,
	    html[data-theme="modern"] .range-tab {
	      background: var(--panel);
	      border: 1px solid var(--line);
	      border-radius: 6px;
	    }
	    html[data-theme="modern"] .metrics {
	      border: 1px solid var(--line);
	      border-radius: 8px;
	      box-shadow: none;
	    }
	    html[data-theme="modern"] .metric-value {
	      color: #008fb3;
	    }
	    html[data-theme="modern"] .metric-value.positive,
	    html[data-theme="modern"] .chart-change {
	      color: #138a5b;
	    }
	    html[data-theme="modern"] .metric-value.caution {
	      color: #a96700;
	    }
	    html[data-theme="modern"] .chart-change.down {
	      color: #b64242;
	    }
	    html[data-theme="modern"] .range-tab {
	      border-radius: 999px;
	      font-family: ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
	      letter-spacing: 0;
	    }
	    html[data-theme="modern"] .range-tab.active {
	      background: #1455d9;
	      border-color: #1455d9;
	      color: white;
	    }
	    html[data-theme="modern"] .chart-wrap {
	      background: linear-gradient(180deg, #ffffff 0%, #f7fbff 100%);
	      border: 1px solid var(--line);
	      border-radius: 8px;
	      box-shadow: none;
	    }
	    html[data-theme="modern"] .chart-grid {
	      stroke: #e4e9f2;
	    }
	    html[data-theme="modern"] .chart-dot {
	      stroke: white;
	    }
	    html[data-theme="modern"] .notes {
	      border-color: #e7c576;
	      background: #fffaf0;
	    }
	    html[data-theme="modern"] .sources a {
	      color: #1455d9;
	    }
	    @media (max-width: 720px) {
	      form { flex-direction: column; }
	      html[data-theme="modern"] form { flex-direction: column; }
	      button { width: 100%; }
	      .theme-toggle { width: 116px; }
	      .theme-menu { right: 0; width: min(240px, calc(100vw - 40px)); }
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
	        <div class="theme-control">
	          <button id="theme-toggle" class="theme-toggle" type="button" aria-haspopup="menu" aria-expanded="false">Theme</button>
	          <div id="theme-menu" class="theme-menu" role="menu" aria-label="Theme options">
	            <div class="theme-menu-title">Choose theme</div>
	            <button class="theme-option" type="button" role="menuitemradio" aria-checked="true" data-theme-choice="oldschool">
	              <span>Old School</span>
	              <span class="theme-option-check">Selected</span>
	            </button>
	            <button class="theme-option" type="button" role="menuitemradio" aria-checked="false" data-theme-choice="modern">
	              <span>Original</span>
	              <span class="theme-option-check">Selected</span>
	            </button>
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
	          <h1>{{.Report.CompanyName}} ({{.Report.Ticker}})</h1>
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

      function setTheme(theme) {
        root.dataset.theme = theme;
        localStorage.setItem("stockbridge-theme", theme);
        if (button) {
          button.textContent = "Theme";
          button.setAttribute("aria-label", "Open theme menu");
        }
        options.forEach(function (option) {
          const selected = option.dataset.themeChoice === theme;
          option.classList.toggle("active", selected);
          option.setAttribute("aria-checked", selected ? "true" : "false");
        });
        if (window.renderChart && window.currentChartPeriod) {
          window.renderChart(window.currentChartPeriod);
        }
      }

      function setMenuOpen(open) {
        if (!button || !menu) return;
        menu.classList.toggle("open", open);
        button.setAttribute("aria-expanded", open ? "true" : "false");
      }

      setTheme(root.dataset.theme || "oldschool");
      if (button && menu) {
        button.addEventListener("click", function () {
          setMenuOpen(!menu.classList.contains("open"));
        });
        options.forEach(function (option) {
          option.addEventListener("click", function () {
            setTheme(option.dataset.themeChoice);
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
	      const upColor = styles.getPropertyValue("--forest").trim() || "#2f6048";
	      const downColor = styles.getPropertyValue("--burgundy").trim() || "#763432";
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
