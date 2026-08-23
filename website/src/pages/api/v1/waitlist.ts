import { createWaitlistRoutes } from '../../../lib/waitlist-api';

export const prerender = false;

/** Canonical, versioned waitlist endpoint. See `/developers#versioning-and-deprecation`. */
export const { GET, POST, ALL } = createWaitlistRoutes();
