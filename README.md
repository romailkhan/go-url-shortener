# go-url-shortener

A blazing fast URL shortener built with Go with Redis as a cache and PostgreSQL as a database.

## Features

- Shorten URLs
- Redirect to the original URL
- Track click statistics
- Password protection
- Link expiration

## API

Base URL: `/api/v1` for JSON; public redirects live at `/s/:code`.

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/shorten` | Create a short link |
| `GET` | `/api/v1/links/:code` | Metadata (clicks, expiry, optional `target_url`) |
| `GET` | `/s/:code` | Redirect to the target (increments click count) |

**`POST /api/v1/shorten`** — JSON body:

- `url` (required) — full URL (`http` / `https` only)
- `custom_alias` (optional) — fixed code (3–32 chars: letters, digits, `-`, `_`)
- `password` (optional) — min 8 characters; required to visit or look up the link
- `expires_in` (optional) — Go duration string, e.g. `24h`, `168h` (max ~366 days)

Response `201`: `code`, `target_url`, `short_path`, `short_url`, `password_protected`, `expires_at`

**Password on redirect / lookup** — send `X-Link-Password: <secret>` or `?password=<secret>` (header preferred).

**Errors** — `404` not found, `410` expired link, `401` missing/wrong password, `409` custom code taken.