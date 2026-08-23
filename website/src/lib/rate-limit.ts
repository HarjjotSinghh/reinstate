/**
 * Best-effort rate limiting for `/api/*` with the IETF RateLimit header
 * fields (draft-ietf-httpapi-ratelimit-headers structured fields) plus the
 * older `RateLimit-Limit` / `RateLimit-Remaining` / `RateLimit-Reset`
 * compatibility form, and `Retry-After` on 429.
 *
 * State is an in-memory sliding window per client address inside one
 * function instance, so the quota is advisory across instances; the headers
 * are always emitted so agents can self-throttle.
 */

export interface RateLimitPolicy {
  /** Policy name used as the structured-field key. */
  name: string;
  /** Requests allowed per window. */
  quota: number;
  /** Window length in seconds. */
  windowSeconds: number;
}

export const API_RATE_LIMIT: RateLimitPolicy = { name: 'api', quota: 60, windowSeconds: 60 };

export interface RateLimitDecision {
  allowed: boolean;
  remaining: number;
  /** Seconds until the oldest request in the window expires. */
  resetSeconds: number;
  headers: Record<string, string>;
}

export class SlidingWindowLimiter {
  readonly #hits = new Map<string, number[]>();
  readonly #maxKeys: number;

  constructor(
    readonly policy: RateLimitPolicy,
    options: { maxKeys?: number } = {},
  ) {
    this.#maxKeys = options.maxKeys ?? 10_000;
  }

  /** Records one request for `key` and reports whether it fits the window. */
  hit(key: string, now = Date.now()): RateLimitDecision {
    const windowMs = this.policy.windowSeconds * 1000;
    const cutoff = now - windowMs;
    const timestamps = (this.#hits.get(key) ?? []).filter((stamp) => stamp > cutoff);
    const allowed = timestamps.length < this.policy.quota;
    if (allowed) timestamps.push(now);
    this.#hits.set(key, timestamps);
    this.#evict();

    const remaining = Math.max(0, this.policy.quota - timestamps.length);
    const oldest = timestamps[0];
    const resetSeconds = oldest === undefined ? this.policy.windowSeconds : Math.max(1, Math.ceil((oldest + windowMs - now) / 1000));
    return { allowed, remaining, resetSeconds, headers: rateLimitHeaders(this.policy, remaining, resetSeconds) };
  }

  /** Current window size for a key without recording a hit. */
  count(key: string, now = Date.now()): number {
    const cutoff = now - this.policy.windowSeconds * 1000;
    return (this.#hits.get(key) ?? []).filter((stamp) => stamp > cutoff).length;
  }

  #evict(): void {
    while (this.#hits.size > this.#maxKeys) {
      const oldestKey = this.#hits.keys().next().value;
      if (oldestKey === undefined) break;
      this.#hits.delete(oldestKey);
    }
  }
}

/** Header fields for one decision: IETF structured fields plus the compatibility trio. */
export function rateLimitHeaders(policy: RateLimitPolicy, remaining: number, resetSeconds: number): Record<string, string> {
  return {
    'RateLimit-Policy': `"${policy.name}";q=${policy.quota};w=${policy.windowSeconds}`,
    RateLimit: `"${policy.name}";r=${remaining};t=${resetSeconds}`,
    'RateLimit-Limit': String(policy.quota),
    'RateLimit-Remaining': String(remaining),
    'RateLimit-Reset': String(resetSeconds),
  };
}

/** Client key: first forwarded address, else the socket address, else a shared bucket. */
export function clientKey(request: Request, clientAddress?: string): string {
  const forwarded = request.headers.get('x-forwarded-for')?.split(',')[0]?.trim();
  const real = request.headers.get('x-real-ip')?.trim();
  return forwarded || real || clientAddress || 'anonymous';
}

export const apiLimiter = new SlidingWindowLimiter(API_RATE_LIMIT);
