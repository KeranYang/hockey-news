# hockey-news

A Go tool that monitors Oakville hockey sites hourly and sends an email the moment new U8-relevant content appears — so Lucas stays on the ice and off the waitlist.

**Sites monitored:**
- [Oakville Rangers](https://oakvillerangers.ca)
- [Canlan Sports Oakville](https://www.canlansports.com/locations/ca/on/oakville/)
- [Oakville Hockey Academy](https://oakvillehockeyacademy.com)

## How it works

Every hour, the tool fetches the news/listing pages from each site and passes the raw HTML to Claude Haiku, which extracts articles and filters for U8 relevance in a single call. Each returned URL is verified with a HEAD request to catch hallucinated links. If anything new and relevant is found, an email goes out immediately. Already-seen articles are tracked in `seen.json` so you never get the same article twice.

Because extraction is LLM-based rather than CSS-selector-based, the scraper stays functional even when sites redesign their layouts.

## Setup

### 1. Install Go

```bash
brew install go
```

### 2. Install dependencies

```bash
go mod tidy
```

### 3. Configure credentials

Create a `.env` file in the project root:

```env
SMTP_USER=you@gmail.com
SMTP_PASSWORD=your-gmail-app-password
EMAIL_TO=you@gmail.com,someone@example.com
ANTHROPIC_API_KEY=sk-ant-...
```

`SMTP_PASSWORD` must be a [Gmail App Password](https://myaccount.google.com/apppasswords), not your regular Gmail password. Generate one at myaccount.google.com → Security → 2-Step Verification → App passwords.

`EMAIL_TO` accepts one or more comma-separated addresses.

### 4. Run manually

```bash
go run .
```

The first run will treat everything from the last 7 days as new and send one email. After that, only genuinely new articles trigger an alert.

## GitHub Actions (automated)

The workflow runs every hour via GitHub Actions — free for public repositories. Seen articles are persisted between runs using GitHub Actions cache.

To deploy, push to GitHub and add the following repository secrets:

| Secret | Description |
|---|---|
| `SMTP_USER` | Your Gmail address |
| `SMTP_PASSWORD` | Gmail App Password |
| `EMAIL_TO` | Recipient address(es) |
| `ANTHROPIC_API_KEY` | Anthropic API key |

You can also trigger a run manually from the Actions tab.

## Sample email

**Subject:** `just posted: U8 Winter House League Registration is Open`

---

<table width="100%" cellpadding="0" cellspacing="0"><tr><td>
<div style="font-family:Arial,sans-serif;color:#333;max-width:700px;padding:20px">
  <h2 style="color:#00508A;border-bottom:3px solid #C22033;padding-bottom:10px">What's new at the rink</h2>
  <p>These just went up in the last hour across the Oakville hockey sites — figured you'd want to know sooner rather than later. Lucas probably just wants to know if there's ice time, but here we are.</p>
  <div style="margin-bottom:24px;border-left:4px solid #00508A;padding-left:14px">
    <h3 style="margin:0 0 4px 0;font-size:16px"><a href="#" style="color:#00508A;text-decoration:none">U8 Winter House League Registration is Open</a></h3>
    <div style="color:#888;font-size:13px;margin-bottom:6px">Oakville Rangers · April 19, 2026</div>
    <p style="margin:0;line-height:1.6">Registration for the U8 Winter House League is now open. Sessions run Saturday mornings at Sixteen Mile Sports Complex starting November 2. Spots are limited to 80 players across four teams.</p>
  </div>
  <div style="margin-bottom:24px;border-left:4px solid #00508A;padding-left:14px">
    <h3 style="margin:0 0 4px 0;font-size:16px"><a href="#" style="color:#00508A;text-decoration:none">Learn to Skate Program — Fall Session Announced</a></h3>
    <div style="color:#888;font-size:13px;margin-bottom:6px">Oakville Hockey Academy · April 19, 2026</div>
    <p style="margin:0;line-height:1.6">The fall Learn to Skate program is back, designed for young players looking to build confidence on the ice. Open to ages 4–8, with beginner and intermediate streams available.</p>
  </div>
  <div style="margin-top:40px;font-size:12px;color:#aaa;border-top:1px solid #eee;padding-top:12px">
    HockeyNews — built by Keran Yang to keep Lucas on the ice and off the waitlist. © 2026.
  </div>
</div>
</td></tr></table>

---

## Adding a new site

1. Create a new `.go` file (e.g. `mysite.go`) with a struct implementing the `Scraper` interface:
   - `Name() string`
   - `SiteURL() string`
   - `FetchArticles(client *http.Client, since time.Time) ([]Article, error)`
2. Inside `FetchArticles`, fetch the site's news or listing page, then pass the HTML body to `extractArticles(client, body, siteURL, name, since)` — that handles LLM extraction, relevance filtering, and URL validation.
3. Register it in `main.go` by appending an instance to the `scrapers` slice.
