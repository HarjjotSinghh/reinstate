import { createWaitlistRoutes } from '../../lib/waitlist-api';

export const prerender = false;

/**
 * Deprecated unversioned alias of `/api/v1/waitlist`. It keeps working and
 * answers with RFC 9745 `Deprecation` and `Link rel="successor-version"`
 * headers; no `Sunset` date is scheduled.
 */
export const { GET, POST, ALL } = createWaitlistRoutes({ deprecated: true });
