# ispeed Server

This server runs in both Bun (self-hosted) and Cloudflare Workers.

## Run locally (Bun)

```
cd server
bun run index.ts
```

The server listens on `PORT` or defaults to `8080`.

## Deploy to Cloudflare Workers

```
cd server
wrangler deploy
```

Wrangler outputs the deployed URL. Use it as the base URL for the CLI `-url` flag.

## Endpoints

- `GET /ping`
- `GET /download?size=<bytes>`
- `POST /upload`
