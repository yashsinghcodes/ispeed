# ispeed

Fast, minimal CLI speed test with a shared Cloudflare/Bun server.

## Install

### Go install (recommended)

```
go install github.com/yashsinghcodes/ispeed@latest
```

Make sure your `GOBIN` or `GOPATH/bin` is on your `PATH`, then run:

```
ispeed
```

### Build from source

```
go build -o ispeed
./ispeed
```

## Usage

```
ispeed -url https://speed.getanswers.pro
```

Options:

- `-url` base server URL (default: `https://speed.getanswers.pro`)
- `-duration` test duration
- `-streams` parallel streams
- `-download-mb` download size per stream in MB
- `-ping-count` ping samples
- `-timeout` request timeout
- `-json` JSON output

## Host your own server

The server is a single TypeScript entrypoint that runs on both Bun and Cloudflare Workers.

### Run locally (Bun)

```
cd server
bun run index.ts
```

The deploy output URL becomes your CLI `-url` value.
