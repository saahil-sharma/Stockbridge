# Stockbridge

Stockbridge is a Go command-line and web application for researching listed U.S. companies with current listing data, SEC fundamentals, valuation metrics, and historical market prices. Reports identify their sources, retrieval times, missing data, and limitations. They are informational and are not personalized financial advice.

## Data Sources

- Nasdaq Trader symbol directories validate the ticker and listing exchange.
- SEC EDGAR company tickers and company facts provide standardized fundamentals for U.S. reporting companies.
- Yahoo Finance chart data provides recent prices and web charts.
- Financial Modeling Prep (FMP) is an optional fallback for companies without usable SEC facts.

Live reports depend on those external services. Recent news is not yet integrated.

## Local Development

Requirements: Go 1.22.3 or a compatible newer Go 1.22 release, internet access for live reports, and optionally an FMP API key.

Run the web application:

```sh
git clone https://github.com/saahil-sharma/Stockbridge.git
cd Stockbridge
export STOCKBRIDGE_SEC_USER_AGENT="Stockbridge/1.0 your-email@example.com"
make web
```

Open `http://localhost:8080`. To use a different provider-style port:

```sh
PORT=9090 go run ./cmd/stockbridge-web
```

The existing explicit address override remains available:

```sh
go run ./cmd/stockbridge-web -addr 127.0.0.1:9090
```

Run the CLI without installing it:

```sh
./stockbridge help
./stockbridge analyze AMZN
./stockbridge analyze IBM --output IBM.txt
```

To put the CLI wrapper on the current shell's `PATH`, run `source activate.sh` in zsh/bash or `source activate.fish` in fish.

Run all deterministic tests:

```sh
make test
```

## Environment Variables

| Variable | Required | Purpose |
| --- | --- | --- |
| `PORT` | Supplied by Render | HTTP listen port. Defaults to `8080` locally. Do not set it manually on Render. |
| `STOCKBRIDGE_SEC_USER_AGENT` | Yes for public deployment | Identifies the application and a real contact email to SEC EDGAR, for example `Stockbridge/1.0 owner@example.com`. |
| `STOCKBRIDGE_FMP_API_KEY` | Optional, recommended | Enables fallback fundamentals for companies without usable SEC company facts. AMZN and most U.S. SEC filers do not require it. |

The app does not automatically load `.env` files. To use a local one, create `.env` from `.env.example`, fill in your values, and load it into the shell before starting:

```sh
set -a
source .env
set +a
```

Never commit `.env` or real API keys. `.env` and production binaries are ignored by Git.

## Production Build

The web template, CSS, and JavaScript are compiled into the Go binary, so the production build has no separate static asset step.

Exact Render build command:

```sh
go build -tags netgo -ldflags '-s -w' -o stockbridge-web ./cmd/stockbridge-web
```

Exact Render start command:

```sh
./stockbridge-web
```

Build the same web entrypoint locally with `make build-web`, which writes `bin/stockbridge-web`. A direct production check can use:

```sh
go build -tags netgo -ldflags '-s -w' -o /tmp/stockbridge-web ./cmd/stockbridge-web
PORT=8080 STOCKBRIDGE_SEC_USER_AGENT="Stockbridge/1.0 your-email@example.com" /tmp/stockbridge-web
curl --fail http://localhost:8080/health
curl --fail 'http://localhost:8080/?ticker=AMZN'
```

The server binds to all interfaces on `PORT`, applies HTTP read/write/idle timeouts, limits concurrent analyses, and shuts down gracefully on `SIGINT` or `SIGTERM`. `GET /health` returns `200 OK` with `ok` when the process is running.

## Deploy To Render

Render is the selected provider because its native Go web services match Stockbridge's long-running `net/http` server, while providing GitHub deployments, environment secrets, managed HTTPS, health checks, logs, and a free instance option. Vercel would require adapting this process-oriented server to serverless handlers. Cloud Run is capable but requires more Google Cloud and billing setup; Railway and Fly.io do not currently offer a comparable ongoing free web-service tier.

The repository's `render.yaml` is a Render Blueprint with the build command, start command, free instance plan, health check, auto-deploy behavior, shutdown window, and secret placeholders.

1. Push this repository and the deployment changes to the `main` branch on GitHub.
2. Create an account at [Render](https://dashboard.render.com/register), preferably using GitHub.
3. In the Render dashboard, select **New > Blueprint**.
4. Connect GitHub and authorize Render to access `saahil-sharma/Stockbridge` (or the fork you control).
5. Select the repository and keep the Blueprint path as `render.yaml`.
6. Enter `STOCKBRIDGE_SEC_USER_AGENT` with an application name and a real contact email.
7. Enter `STOCKBRIDGE_FMP_API_KEY` if you have one. Leave it empty if you only need SEC-backed fundamentals.
8. Review the `stockbridge` web service and apply the Blueprint. Render builds and starts it automatically.
9. Wait for the deploy and health check to pass, then open the service URL and test `AMZN`.
10. Verify `https://<service-name>.onrender.com/health` returns `ok` and the home page shows current source retrieval times.

Render assigns a public URL in this format:

```text
https://stockbridge.onrender.com
```

If that subdomain is already taken, Render adds a suffix. Render terminates TLS and redirects HTTP traffic to HTTPS automatically.

Future pushes to the connected `main` branch trigger automatic builds and zero-downtime deploys. To redeploy the same commit, open the service in Render and choose **Manual Deploy > Deploy latest commit**. Environment variable changes can be made under **Environment**; save them and choose the dashboard option to rebuild and deploy.

## Free-Tier And Operational Limits

- Render's Free web service is $0 within the workspace allowances, but it spins down after 15 minutes without inbound traffic. The next visitor can wait about one minute for it to start.
- Each workspace currently receives 750 Free instance hours per month. Free services can be suspended after the allowance is exhausted, and outbound bandwidth/build minutes are limited.
- The always-on Starter web-service tier currently starts around $7/month; check [Render pricing](https://render.com/pricing) before upgrading because prices and allowances can change.
- The filesystem is ephemeral. Stockbridge does not require server-side persistence; themes and watchlists stay in each visitor's browser `localStorage`.
- SEC, Nasdaq, Yahoo, and FMP can fail, change behavior, or rate-limit requests. Stockbridge returns a partial report or a retryable public message without exposing upstream URLs, stack traces, or API keys.
- The process limits concurrent analyses, but there is no distributed per-IP rate limiter or cache yet. A heavily promoted public deployment should add edge rate limiting/caching and monitor FMP quota usage.
- The free tier has no production SLA. Use an always-on paid instance for consistently low latency and stronger reliability expectations.

Current provider documentation: [Render Go deployment](https://render.com/docs/deploy-go-nethttp), [web services and HTTPS](https://render.com/docs/web-services), [Blueprint reference](https://render.com/docs/blueprint-spec), and [Free instance limits](https://render.com/docs/free).
