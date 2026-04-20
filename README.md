# url-shortener

Tiny URL shortener in pure Go — stdlib only, in-memory storage.
The kind of thing you hack in one evening.

## run

```bash
go run .
```

Env vars:
- `PORT` — default `8080`
- `BASE_URL` — base for returned short URLs, default `http://localhost:PORT`

## endpoints

- `POST /shorten` — body `{"url": "..."}` → `{"code": "...", "short": "..."}`
- `GET /{code}` — 302 redirect to the original URL, bumps hit counter
- `GET /stats/{code}` — `{"url": "...", "hits": N, "created_at": "..."}`
- `GET /healthz` — liveness probe

## docker

```bash
docker build -t url-shortener .
docker run --rm -p 8080:8080 url-shortener
```

## notes

- Storage is in-memory, restart wipes everything. SQLite / Redis is the
  obvious next step.
- No auth, no rate-limit. Put it behind Caddy/nginx if exposed.
- Codes are 7 chars from `crypto/rand` → ~8e11 space, collisions ignored
  for now (fine until a few million links).
