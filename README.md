# hockey-news

A Go script that scrapes weekly news from Oakville hockey sites and emails a U8-focused digest every Monday morning.

**Sites monitored:**
- [Oakville Rangers](https://oakvillerangers.ca)
- [Canlan Sports Oakville](https://www.canlansports.com/locations/ca/on/oakville/)
- [Oakville Hockey Academy](https://oakvillehockeyacademy.com)

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
```

`SMTP_PASSWORD` must be a [Gmail App Password](https://myaccount.google.com/apppasswords), not your regular Gmail password.
`EMAIL_TO` accepts one or more comma-separated addresses.

### 4. Run manually

```bash
go run .
```

### 5. Schedule weekly (every Monday at 8am)

Build a binary first:

```bash
go build -o hockey-news .
```

Then add to your crontab (`crontab -e`):

```cron
0 8 * * 1 cd /path/to/hockey-news && ./hockey-news >> /tmp/hockey-news.log 2>&1
```

Check the log after the first run:

```bash
cat /tmp/hockey-news.log
```

## Adding a new site

1. Create a new `.go` file (e.g. `mysite.go`) with a struct implementing the `Scraper` interface:
   - `Name() string`
   - `SiteURL() string`
   - `FetchArticles(client *http.Client, since time.Time) ([]Article, error)`
2. Register it in `main.go` by appending an instance to the `scrapers` slice.
