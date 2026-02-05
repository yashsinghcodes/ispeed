const DEFAULT_PORT = 8080;
const DEFAULT_MAX_BYTES = 1024 * 1024 * 1024;
const DEFAULT_READ_LIMIT = 512 * 1024 * 1024;
const DEFAULT_CHUNK_SIZE = 64 * 1024;

function parseSizeParam(request: Request, maxBytes: number): number {
  const url = new URL(request.url);
  const sizeParam = url.searchParams.get("size");
  if (!sizeParam) {
    return maxBytes;
  }
  const parsed = Number.parseInt(sizeParam, 10);
  if (!Number.isFinite(parsed) || parsed <= 0) {
    return maxBytes;
  }
  return Math.min(parsed, maxBytes);
}

function randomStream(totalSize: number, chunkSize: number): ReadableStream<Uint8Array> {
  let remaining = totalSize;
  return new ReadableStream({
    pull(controller) {
      if (remaining <= 0) {
        controller.close();
        return;
      }
      const size = Math.min(remaining, chunkSize);
      const buffer = new Uint8Array(size);
      crypto.getRandomValues(buffer);
      remaining -= size;
      controller.enqueue(buffer);
    },
  });
}

async function handleUpload(request: Request, readLimit: number): Promise<Response> {
  if (!request.body) {
    return new Response("ok", { headers: { "Content-Type": "text/plain" } });
  }
  const reader = request.body.getReader();
  let total = 0;
  while (true) {
    const result = await reader.read();
    if (result.done) {
      break;
    }
    if (result.value) {
      total += result.value.byteLength;
      if (total >= readLimit) {
        await reader.cancel();
        break;
      }
    }
  }
  return new Response("ok", { headers: { "Content-Type": "text/plain" } });
}

function handleDownload(request: Request, maxBytes: number): Response {
  const size = parseSizeParam(request, maxBytes);
  const stream = randomStream(size, DEFAULT_CHUNK_SIZE);
  return new Response(stream, {
    headers: {
      "Content-Type": "application/octet-stream",
      "Content-Length": size.toString(),
    },
  });
}

function handlePing(): Response {
  return new Response("pong", { headers: { "Content-Type": "text/plain" } });
}

async function handler(request: Request): Promise<Response> {
  const url = new URL(request.url);
  if (url.pathname === "/ping") {
    return handlePing();
  }
  if (url.pathname === "/download") {
    return handleDownload(request, DEFAULT_MAX_BYTES);
  }
  if (url.pathname === "/upload") {
    return handleUpload(request, DEFAULT_READ_LIMIT);
  }
  return new Response("not found", { status: 404, headers: { "Content-Type": "text/plain" } });
}

const worker = {
  fetch: handler,
};

export default worker;
