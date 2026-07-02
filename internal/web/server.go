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
	Query   string
	Report  *analysis.Summary
	Chart   *chartView
	Error   string
	Loading bool
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
	data := pageData{Query: query}
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
  <style>
    :root {
      --ink: #18202f;
      --muted: #657187;
      --panel: #f7f8fb;
      --line: #d9deea;
      --blue: #1455d9;
      --cyan: #008fb3;
      --green: #138a5b;
      --amber: #a96700;
      --lavender: #6f5bb5;
      --danger: #b64242;
      --paper: #ffffff;
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      color: var(--ink);
      background: #eef2f7;
      font-family: ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
    }
    .shell {
      max-width: 1180px;
      margin: 0 auto;
      padding: 28px 20px 44px;
    }
    header {
      display: grid;
      grid-template-columns: minmax(0, 1fr);
      gap: 18px;
      margin-bottom: 24px;
    }
    .brand {
      display: flex;
      align-items: baseline;
      gap: 12px;
    }
    .brand strong {
      background: var(--blue);
      color: white;
      padding: 5px 12px;
      border-radius: 6px;
      font-size: 22px;
      letter-spacing: 0;
    }
    .brand span {
      color: var(--lavender);
      font-weight: 700;
    }
    form {
      display: flex;
      gap: 10px;
      max-width: 560px;
    }
    input[type="search"] {
      width: 100%;
      min-height: 46px;
      border: 1px solid var(--line);
      border-radius: 6px;
      padding: 0 14px;
      font-size: 18px;
      color: var(--ink);
      background: white;
    }
    button {
      min-height: 46px;
      border: 0;
      border-radius: 6px;
      padding: 0 18px;
      background: var(--blue);
      color: white;
      font-weight: 800;
      cursor: pointer;
    }
    .error {
      border-left: 4px solid var(--danger);
      background: #fff5f5;
      color: var(--danger);
      padding: 14px 16px;
      border-radius: 6px;
      margin: 18px 0;
      font-weight: 700;
    }
    .empty {
      background: var(--paper);
      border: 1px solid var(--line);
      border-radius: 8px;
      padding: 26px;
      color: var(--muted);
    }
    .report {
      display: grid;
      gap: 18px;
    }
    .summary {
      background: var(--paper);
      border: 1px solid var(--line);
      border-radius: 8px;
      padding: 22px;
    }
    h1, h2 {
      margin: 0;
      letter-spacing: 0;
    }
    h1 {
      color: var(--cyan);
      font-size: 26px;
      line-height: 1.2;
    }
    h2 {
      display: inline-block;
      color: white;
      background: #303747;
      border-bottom: 3px solid var(--cyan);
      padding: 5px 12px;
      border-radius: 4px 4px 0 0;
      font-size: 15px;
      text-transform: uppercase;
      margin-bottom: 12px;
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
      border-radius: 6px;
      padding: 10px 12px;
    }
    .label {
      display: block;
      color: var(--muted);
      font-size: 12px;
      font-weight: 800;
      text-transform: uppercase;
      margin-bottom: 4px;
    }
    .metrics {
      width: 100%;
      border-collapse: collapse;
      background: var(--paper);
      border: 1px solid var(--line);
      border-radius: 8px;
      overflow: hidden;
    }
    .metrics th, .metrics td {
      text-align: left;
      border-bottom: 1px solid var(--line);
      padding: 11px 12px;
      vertical-align: top;
    }
    .metrics th {
      background: #303747;
      color: white;
      font-size: 12px;
      text-transform: uppercase;
    }
    .metrics tr:last-child td { border-bottom: 0; }
    .metric-name { font-weight: 800; }
    .metric-value {
      font-weight: 900;
      color: var(--cyan);
      white-space: nowrap;
    }
    .metric-value.positive { color: var(--green); }
    .metric-value.caution { color: var(--amber); }
    .metric-detail {
      color: var(--muted);
      font-size: 13px;
    }
    .chart-panel {
      background: var(--paper);
      border: 1px solid var(--line);
      border-radius: 8px;
      padding: 18px;
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
      color: var(--green);
      text-align: right;
    }
    .chart-change.down { color: var(--danger); }
    .range-tabs {
      display: flex;
      flex-wrap: wrap;
      gap: 8px;
      margin: 10px 0 14px;
    }
    .range-tab {
      min-height: 34px;
      border: 1px solid var(--line);
      border-radius: 999px;
      padding: 0 13px;
      background: #f4f7fb;
      color: var(--ink);
      font-weight: 900;
      cursor: pointer;
    }
    .range-tab.active {
      background: var(--blue);
      border-color: var(--blue);
      color: white;
    }
    .chart-wrap {
      position: relative;
      width: 100%;
      min-height: 330px;
      background: linear-gradient(180deg, #ffffff 0%, #f7fbff 100%);
      border: 1px solid var(--line);
      border-radius: 8px;
      overflow: hidden;
    }
    .chart-wrap svg {
      display: block;
      width: 100%;
      height: 330px;
    }
    .chart-grid {
      stroke: #e4e9f2;
      stroke-width: 1;
    }
    .chart-axis-label {
      fill: var(--muted);
      font-size: 11px;
      font-weight: 700;
    }
    .chart-line {
      fill: none;
      stroke-width: 3;
      stroke-linecap: round;
      stroke-linejoin: round;
    }
    .chart-area {
      opacity: 0.12;
    }
    .chart-dot {
      stroke: white;
      stroke-width: 2;
    }
    .chart-source {
      margin-top: 10px;
      color: var(--muted);
      font-size: 13px;
    }
    .notes, .sources {
      background: var(--paper);
      border: 1px solid var(--line);
      border-radius: 8px;
      padding: 18px;
    }
    .notes {
      border-color: #e7c576;
      background: #fffaf0;
    }
    ul {
      margin: 0;
      padding-left: 20px;
    }
    li { margin: 8px 0; }
    .sources a {
      color: var(--blue);
      overflow-wrap: anywhere;
    }
    .source {
      padding: 10px 0;
      border-bottom: 1px solid var(--line);
    }
    .source:last-child { border-bottom: 0; }
    @media (max-width: 720px) {
      form { flex-direction: column; }
      button { width: 100%; }
      .metrics { display: block; overflow-x: auto; }
    }
  </style>
</head>
<body>
  <main class="shell">
    <header>
      <div class="brand">
        <strong>Stockbridge</strong>
        <span>Fundamental Analysis</span>
      </div>
      <form action="/" method="get">
        <input type="search" name="ticker" value="{{.Query}}" placeholder="Enter ticker, e.g. AMZN" autocomplete="off" autofocus>
        <button type="submit">Analyze</button>
      </form>
    </header>

    {{if .Error}}
      <div class="error">{{.Error}}</div>
    {{end}}

    {{if .Report}}
      <section class="report">
        <div class="summary">
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
                <div class="metric-detail">Connected historical close prices across selectable ranges.</div>
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
      const lineColor = period.change >= 0 ? "#138a5b" : "#b64242";
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
  </script>
  {{end}}
</body>
</html>`
