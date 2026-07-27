const INDEXNOW_KEY_PATTERN = /^[A-Za-z0-9-]{8,128}$/;

const RESPONSE_HEADERS = {
  'Cache-Control': 'no-store',
  'Content-Type': 'text/plain; charset=utf-8',
  'X-Content-Type-Options': 'nosniff',
  'X-Robots-Tag': 'noindex, nofollow',
};

/**
 * Serve IndexNow's public ownership proof without exposing the configured key
 * at a predictable filename. The key itself remains a server-only environment
 * value; a request must already know the exact key to receive it.
 */
export function createIndexNowKeyResponse(
  requestedKey: string | undefined,
  configuredKey = process.env.INDEXNOW_KEY,
): Response {
  if (
    !requestedKey ||
    !configuredKey ||
    !INDEXNOW_KEY_PATTERN.test(requestedKey) ||
    !INDEXNOW_KEY_PATTERN.test(configuredKey) ||
    requestedKey !== configuredKey
  ) {
    return new Response('Not found.\n', {
      status: 404,
      headers: RESPONSE_HEADERS,
    });
  }

  return new Response(`${configuredKey}\n`, {
    status: 200,
    headers: RESPONSE_HEADERS,
  });
}
