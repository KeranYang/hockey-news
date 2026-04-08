# hockey-news

A Go script that scrapes the weekly news from [oakvillerangers.ca](https://oakvillerangers.ca) and emails you a digest.

## Setup

### 1. Install Go

```bash
brew install go
```

### 2. Install dependencies

```bash
cd hockey-news
go mod tidy
```

### 3. Run manually

```bash
go run main.go
```

### 4. Schedule weekly (every Monday at 8am)

Add to your crontab (`crontab -e`):

```cron
0 8 * * 1 cd /path/to/hockey-news && go run main.go >> /tmp/hockey-news.log 2>&1
```

Or build a binary first and use that instead:

```bash
go build -o hockey-news
```

```cron
0 8 * * 1 /path/to/hockey-news/hockey-news >> /tmp/hockey-news.log 2>&1
```
