# EnvDash — Air Quality & Environment Dashboard Service

A REST web service that aggregates live environmental data — temperature, precipitation, air quality, country information, and currency exchange rates — from multiple open APIs into configurable dashboards. Built for **PROG2005 — Cloud Technologies**.

**Deployed at:** [INSERT YOUR SKYHIGH URL HERE]

**Repository:** https://git.gvk.idi.ntnu.no/course/prog2005/prog2005-2026-workspace/arpitsahoooooo/assignment-2

---

## Table of Contents

- [Overview](#overview)
- [Architecture](#architecture)
- [Project Structure](#project-structure)
- [Firebase](#firebase)
- [API Endpoints](#api-endpoints)
- [Environment Variables](#environment-variables)
- [Testing](#testing)
- [External APIs Used](#external-apis-used)
- [Caching Strategy](#caching-strategy)
- [Advanced Tasks Implemented](#advanced-tasks-implemented)
- [Known Issues & Limitations](#known-issues--limitations)
- [Group Organisation & Contributions](#group-organisation--contributions)
- [AI Assistance Disclosure](#ai-assistance-disclosure)
- [Acknowledgements](#acknowledgements)

---

## Overview

EnvDash exposes a configurable dashboard API where users register countries of interest and the features they want to track. The service fetch data from five external APIs and supports webhook-based notifications for both lifecycle events and threshold event crossings on live measurements.


---

## Architecture

[BRIEF DESCRIPTION OF YOUR PROJECT STRUCTURE — e.g. layered: handlers → services → external clients. 
Mention the use of Firebase Firestore for persistence and stubs for tests.]

### Project Structure

This project uses a conventional Go layout with `cmd/` and `internal/`.

````
assignment-2/
├── cmd/                    
├── internal/
│   ├── clients/             
│   ├── handlers/           
│   ├── middleware/         
│   ├── models/             
│   └── utility/            
├── Dockerfile              
├── docker-compose.yml
├── go.mod
└── README.md
````

### Firebase
The application uses Firebase Firestore for data persistence, while tests rely on the Firestore Emulator to validate database interactions. More details are available in the [Testing](#testing)
section.

---

## Endpoints

All endpoints are versioned under `/envdash/v1/`.

### Landing Page

| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/` | Serve the HTML landing page with basic API information |

### Registrations
| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/envdash/v1/registrations/` | Create new dashboard configuration |
| `GET` | `/envdash/v1/registrations/{id}` | Retrieve specific configuration |
| `GET` | `/envdash/v1/registrations/` | Retrieve all configurations |
| `HEAD` | `/envdash/v1/registrations/` | Headers only (advanced task) |
| `PUT` | `/envdash/v1/registrations/{id}` | Replace configuration |
| `PATCH` | `/envdash/v1/registrations/{id}` | Partial update (advanced task) |
| `DELETE` | `/envdash/v1/registrations/{id}` | Delete configuration |

### Dashboards
| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/envdash/v1/dashboards/{id}` | Retrieve populated live dashboard |

### Notifications (Webhooks)
| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/envdash/v1/notifications/` | Register webhook (lifecycle or threshold) |
| `GET` | `/envdash/v1/notifications/{id}` | Retrieve webhook |
| `GET` | `/envdash/v1/notifications/` | List all webhooks |
| `DELETE` | `/envdash/v1/notifications/{id}` | Delete webhook |

### Status
| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/status/` | Service health and upstream availability |

### Authentication 
| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/envdash/v1/auth/` | Register and obtain API key |
| `DELETE` | `/envdash/v1/auth/{key}` | Revoke API key |

For full request/response examples, see the [assignment specification](LINK TO ASSIGNMENT WIKI IF YOU WANT).

---

## Setup — Local Development

### Prerequisites
- Go [1.25.0](https://go.dev/dl/)
- A Firebase Firestore service account key (`secret.json`)
- An OpenAQ API key (free at [explore.openaq.org](https://explore.openaq.org))

### Steps

1. Clone the repository:
```bash
   git clone [YOUR GITLAB URL]
   cd assignment-2
```

2. Place your Firebase service account key in the project root as `secret.json` (do **not** commit this file).

3. Set required environment variables:
```bash
   export PORT=8080
   export OPENAQ_API_KEY=your-key-here
   export GOOGLE_APPLICATION_CREDENTIALS=./secret.json
```

4. Install dependencies and run:
```bash
   go mod download
   go run ./cmd/[INSERT MAIN PACKAGE]
```

5. Visit [http://localhost:8080/](http://localhost:8080/) for the landing page.

---

## Setup — Docker

### Prerequisites

Create these files locally before starting the container:

- `.env`
- `.secrets/firebase-credential.json`
- The repository already includes 'compose.yml' and 'Dockerfile'

Structure:

```text
.Assignment-2
├── .env
├── .secrets/
│   └── firebase-credential.json
├── Dockerfile
├── compose.yml
└── ...
```

### How to setup environment variables

- Create a `.env` file in the project root before starting the container and add:
```env
 OPENAQ_API_KEY=your_api_key_here
 FIREBASE_PROJECT_ID="assigment-2-4d134"
```

### Compose and build

```bash
`Use either one depending on your plugin`
sudo docker-compose up -d --build
sudo docker compose up --build
```

or

```bash
sudo docker run -p 8080:8080 \
  --env-file .env \
  -e PORT=8080 \
  -e GOOGLE_APPLICATION_CREDENTIALS=/googlecredentials/firebase-credential.json \
  -v "$(pwd)/.secrets:/googlecredentials:ro" \
  assignment-2-api
```

### Check deployment status

```bash
sudo docker ps
sudo docker-compose ps
sudo docker-compose logs -f api
```
What each does:
- `sudo docker ps` shows running containers
- `sudo docker-compose ps` shows the status of services from your compose file
- `sudo docker-compose logs -f api` shows the app logs for the `api` service

### Notes

- The service runs on port `8080`
- Firebase credentials must be available at `.secrets/firebase-credential.json`
- The container reads the credential file from `/googlecredentials/firebase-credential.json`
---

## Setup — SkyHigh Deployment

### Prerequisites
- NTNU VPN connected
- SSH access to your assigned SkyHigh VM
- Docker installed on the VM

### Deployment steps

1. SSH into your VM:
```bash
   ssh ubuntu@[YOUR VM IP]
```

2. Clone the repo on the VM:
```bash
   git clone [YOUR GITLAB URL]
   cd assignment-2
```

3. Securely copy your `secret.json` to the VM (run on your local machine):
```bash
   scp secret.json ubuntu@[VM IP]:~/assignment-2/secret.json
```

4. Set environment variables and run:
```bash
   export OPENAQ_API_KEY=your-key-here
   docker compose up -d --build
```

5. Ensure the VM's security group allows inbound traffic on port [INSERT PORT].

The service is then accessible at `http://[VM IP]:[PORT]/`.

---

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `PORT` | No | `8080` | Port the HTTP server binds to |
| `GOOGLE_APPLICATION_CREDENTIALS` | Yes | — | Path to Firebase service account JSON file |
| `OPENAQ_API_KEY` | Yes | — | API key for OpenAQ v3 |
| `FIRESTORE_EMULATOR_HOST` | No | — | If set, Firestore client connects to this emulator address (used for testing) |
| `GOOGLE_CLOUD_PROJECT` | No | `test-project` | Project ID — only used when running against the emulator |
| [ANY OTHERS YOU HAVE — e.g. CACHE_TTL_HOURS] | | | |

---

## API Endpoints

All endpoints are versioned under `/envdash/v1/`.

### Registrations

Manage dashboard configurations stored by the service.

| Method | Endpoint | Description |
|---|---|---|
| `POST` | `/envdash/v1/registrations/` | Create a new dashboard configuration |
| `GET` | `/envdash/v1/registrations/` | List all stored configurations |
| `GET` | `/envdash/v1/registrations/{id}` | Retrieve a specific configuration by ID |
| `PUT` | `/envdash/v1/registrations/{id}` | Replace an existing configuration |
| `DELETE` | `/envdash/v1/registrations/{id}` | Delete a configuration by ID |
| `HEAD` | `/envdash/v1/registrations/` | Return headers only for the registrations collection *(advanced task)* |
| `PATCH` | `/envdash/v1/registrations/{id}` | Partially update a configuration *(advanced task)* |

### Dashboards

Retrieve populated dashboards using live data from external services.

| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/envdash/v1/dashboards/{id}` | Retrieve a live dashboard for a stored configuration |

### Notifications

Register and manage webhook subscriptions for lifecycle and threshold events.

| Method | Endpoint | Description |
|---|---|---|
| `POST` | `/envdash/v1/notifications/` | Register a new webhook |
| `GET` | `/envdash/v1/notifications/` | List all registered webhooks |
| `GET` | `/envdash/v1/notifications/{id}` | Retrieve a specific webhook registration |
| `DELETE` | `/envdash/v1/notifications/{id}` | Delete a webhook registration |

### Status

Check service health and upstream dependency availability.

| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/envdash/v1/status/` | Retrieve health status and upstream service availability |

### Authentication *(Advanced, if implemented)*

Manage API client registration and key revocation.

| Method | Endpoint | Description |
|---|---|---|
| `POST` | `/envdash/v1/auth/` | Register a client and obtain an API key |
| `DELETE` | `/envdash/v1/auth/{key}` | Revoke an existing API key |

---

## Setup — Local Development

---

## Setup — Docker

---

## Setup — SkyHigh Deployment

In addition to the local Docker setup, deploying to SkyHigh requires:
1. SSH access to your assigned VM
2. Docker installed on the VM
3. Securely copying your `secret.json` to the VM
4. Setting environment variables and running the service with Docker Compose on the VM
5. Ensuring the VM's security group allows inbound traffic on the configured port
6. Accessing the service at `http://[VM IP]:[PORT]/` after deployment
7. Monitoring logs with `docker compose logs -f` for troubleshooting
8. Optionally setting up a process manager like `systemd` or `supervisord` for production deployments to ensure the service restarts on failure and starts on boot.
9. Lastly, the docker file and compose file should be configured to run in production mode, with appropriate environment variables and resource limits for the SkyHigh environment.

---

---

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `PORT` | No | `8080` | Port the HTTP server binds to |
| `GOOGLE_APPLICATION_CREDENTIALS` | Yes | — | Path to Firebase service account JSON file |
| `OPENAQ_API_KEY` | Yes | — | API key for OpenAQ v3 |
| `FIRESTORE_EMULATOR_HOST` | No | — | If set, Firestore client connects to this emulator address (used for testing) |
| `GOOGLE_CLOUD_PROJECT` | No | `test-project` | Project ID — only used when running against the emulator |

---

---

## Testing

### Run all tests
```bash
go test ./...
```

### With coverage
```bash
go test -cover ./...
```

### Note

Enig med forklaring einar?

Coverage for `internal/clients` may appear lower than expected, even though the related behavior is tested through `DashboardHandler`. This is because some client calls are replaced with test stubs, so the handler behavior is verified without always executing the full client implementation.

### Integration tests with Firestore emulator
Integration tests run against a real emulator instance.

1. Install the Firebase CLI, if you already have it you can skip this part:
    ```bash
       npm install -g firebase-tools
    ```

2. Start the Firestore emulator (in a separate terminal):
    ```bash
       firebase emulators:start --only firestore --project test-project
    ```

3. Set the emulator host and run tests:
    ```bash
       export FIRESTORE_EMULATOR_HOST=127.0.0.1:8080
       export GOOGLE_CLOUD_PROJECT=test-project 
       go test ./...
    ```

If `FIRESTORE_EMULATOR_HOST` is unset or the emulator is unreachable, integration tests are skipped automatically — unit tests still run.

---

## External APIs Used

| API | Purpose | Notes |
|-----|---------|-------|
| REST Countries (course-hosted) | Country metadata, capitals, coordinates | `http://129.241.150.113:8080/v3.1` |
| Open-Meteo | Weather forecasts (temperature, precipitation) | Externally hosted — uses caching to limit calls |
| OpenAQ v3 | Air quality measurements (PM2.5, PM10) | Requires API key |
| Nominatim (OSM) | [DESCRIBE WHAT YOU USE IT FOR] | 1 req/sec rate limit, custom `User-Agent` set |
| Currency API (course-hosted) | Exchange rates | `http://129.241.150.113:9090/currency/` |

---

## Caching Strategy

We use a Firestore-backed decorator pattern: `CachedClient` wraps a `RawClient` and implements the flow:
check cache -> return cached if valid -> otherwise fetch live -> store result with TTL.
### Goals achieved
- Reduce repeated calls to external APIs, by reusing latest response.
- Keep cached data fresh using per-endpoint TTLs.
- Fail-safe: if Firestore is unavailable or cache is malformed, the system falls back to live fetches.

### How it works (high-level)
- The `APIClient` interface defines methods used by handlers:
   - `GetCountry(ctx, iso)`, `GetWeather(ctx, lat,lng)`, `GetExchangeRates(ctx, base, targets)`, `GetAirQuality(ctx, iso, capital)`.
- `RawClient` performs actual HTTP requests (via package fetch functions).
- `CachedClient` wraps any `APIClient` and:
   1. Builds a (deterministic) cache key from endpoint + params.
   2. Calls `cache.GetCached(ctx, fsClient, key)` to attempt a read from Firestore.
   3. If a valid (not expired) payload exists and unmarshal correctly -> return it (cache hit).
   4. Otherwise, call underlying `Underlying.Get*` (live fetch).
   5. On success marshal result -> `cache.SetCached(ctx, fsClient, key, payload, ttl)` (best-effort write).
- If the Firestore client is `nil`, cache reads return a miss and writes are skipped (cache disabled mode).

Files:
- Cache helpers: `internal/cache/storeCache.go` (`MakeCacheKey`, `GetCached`, `SetCached`)
- Cache model: `internal/models/cachestruct.go` (`CacheEntry` with Payload/CreatedAt/ExpiresAt)
- Decorator client: `internal/clients/cachedClient.go`
- API interface: `internal/clients/APIClientInterface.go`
- Raw implementation: `internal/clients/rawClient.go`

### Cache key generation
Keys are produced by `MakeCacheKey(endpoint, params)`:

1. JSON‑marshal `params`
2. Concatenate the result with the `endpoint` label
3. Hash the concatenated bytes with SHA‑256
4. Hex‑encode the hash

Final format:
```
cacheKey = hex(sha256(endpoint + json(params)))
```
Benefits
- Produces safe Firestore document IDs (fixed-length, alphanumeric)
- Namespacing by `endpoint` prevents collisions between different endpoints

Note
- If `params` is a map, keys should be sorted before marshaling to avoid different JSON orders producing different keys.

### Firestore schema
Each cache document stored in collection `utility.ApiCacheCollection` contains a `CacheEntry`:
- `payload` ([]byte) — JSON-marshaled bytes of the response (shown as base64 in the Firestore console)
- `createdAt` (timestamp)
- `expiresAt` (timestamp) — used for TTL checks at read time (application-level expiry).



---

## Advanced Tasks Implemented

- `HEAD` method on `GET /registrations/`
- `PATCH` for partial registration updates
- Compound thresholds (high + low bounds in single webhook), and additional threshold operators (`>=`, `<=`, `=`)
- API key authentication system (`/auth/` endpoint with middleware)
- HAR MED DETTE ARPIT? - Automatic cache purging with configurable TTL (NEI VI HAR IKKE CACHE PURGING.....)

## Known Issues & Limitations

DETTE MÅ VI SE OVER

Not that we are awear of

---

## Group Organisation & Contributions

DETTE MÅ VI SE OVER

### Group members

| Name  | Primary responsibilities                        |
|-------|-------------------------------------------------|
| Arpit | Registration endpoints, authentication, caching |
| Bjørn | Dashboard handler, Webhooks                     |
| Einar | Registration endpoints                          |

### How we organised our work

We met often in person and used Discord for coordination. Decisions about scope, design, and divisions of work were made collaboratively at the start of the project, with status check-ins at least once a week.

---

## AI Assistance Disclosure

SKRIV HER

## Acknowledgements

- Course staff: Christopher Frantz, Erlend Rømo, Khai Duong, Katharina Kivle, Torgrim Thorsen