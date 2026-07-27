export function isSafeContentCanonical(value: string): boolean {
  try {
    const url = new URL(value);
    return (
      url.origin === 'https://reinstate.dev' &&
      !url.search &&
      !url.hash &&
      (url.pathname === '/' ||
        /^\/[a-z0-9]+(?:[/-][a-z0-9]+)*$/.test(url.pathname))
    );
  } catch {
    return false;
  }
}

export function hasCompleteSocialOverride(data: {
  ogImage?: string;
  ogImageAlt?: string;
}): boolean {
  return Boolean(data.ogImage) === Boolean(data.ogImageAlt);
}
