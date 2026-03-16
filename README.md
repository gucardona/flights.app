# FlightRadar

Google Flights scraper via SerpAPI — Go backend, plain HTML frontend.

## Project structure

```
flightradar/
├── cmd/server/main.go          # HTTP server, routes
├── internal/
│   ├── serpapi/client.go       # SerpAPI HTTP client + types
│   └── search/
│       ├── service.go          # Concurrent job runner, result sorting
│       └── jobs.go             # Builds Job list from request (specific/range/month)
├── static/index.html           # Frontend (no framework, no build step)
├── Dockerfile
└── go.mod
```

## Run locally

```bash
go run ./cmd/server
# → http://localhost:8080
```

Custom port:
```bash
PORT=9000 go run ./cmd/server
```

## Docker

```bash
docker build -t flightradar .
docker run -p 8080:8080 flightradar
```

## API

### POST /api/search

All fields:

```json
{
  "api_key": "your_serpapi_key",
  "origin": "GRU",
  "destinations": ["DBV", "SPU", "BEG"],
  "mode": "specific",

  "adults": 2,
  "travel_class": 1,
  "currency": "BRL",

  // mode=specific
  "outbound_date": "2026-12-26",
  "return_date": "2027-01-05",

  // mode=range
  "outbound_from": "2026-12-20",
  "outbound_to": "2026-12-28",
  "return_from": "2027-01-03",
  "return_to": "2027-01-10",
  "max_combos": 8,

  // mode=month
  "outbound_months": [12],
  "return_months": [1],
  "year": 2026,
  "return_year": 2027,
  "samples_per_month": 4
}
```

Response:

```json
{
  "results": [
    {
      "origin": "GRU",
      "destination": "DBV",
      "outbound_date": "2026-12-26",
      "return_date": "2027-01-05",
      "price": 4200,
      "total_duration": 840,
      "stops": 1,
      "airline": "LATAM Airlines",
      "flight_numbers": "LA8084 / IB3106",
      "dep_time": "22:30",
      "arr_time": "17:55",
      "dep_iata": "GRU",
      "arr_iata": "DBV",
      "price_level": "low",
      "lowest_seen": 3900,
      "is_best": true
    }
  ],
  "total_jobs": 3,
  "successful": 3,
  "errors": []
}
```

### GET /api/regions

Returns the built-in region → IATA airport mapping.

## Travel class codes

| Value | Class            |
|-------|------------------|
| 1     | Economy          |
| 2     | Premium Economy  |
| 3     | Business         |
| 4     | First            |

## SerpAPI free tier

100 searches/month. Each Job in a search = 1 SerpAPI credit.
Keep `max_combos` and `samples_per_month` low to save credits.

## Your Balkans trip example

Mode: **month**
- Origin: `GRU`
- Region: Balkans → `DBV, SPU, ZAG, BEG, SKP, TGD`
- Outbound months: December, year 2026
- Return months: January, year 2027
- Samples/month: 4
- Adults: 2, Currency: BRL
