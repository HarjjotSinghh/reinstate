## Design system

**Read [`../DESIGN.md`](../DESIGN.md) before writing any markup or CSS.** It is the source of
truth for tokens, typography, illustration, motion, and components, and it is extracted from
the canonical hero at `src/pages/preview/exploded.astro`. Build every new surface to match
that page.

The rules that break the design fastest if ignored:

- **No shadows, no blur, no glass.** Depth comes from outlines, flat colour faces, and
  stacking order. A zero-blur ring (`0 0 0 3px`) is the only permitted `box-shadow`.
- **Light is the default theme, dark is a first-class twin.** Every token needs both values.
  Theme state lives on `html.dark`, persists to `localStorage['rein-theme']`, and is seeded
  by an inline head script.
- **Display type is Funnel Display via `--font-display`; body is Geist; mono is Geist Mono.**
- **Illustration is true axonometric**, projected at build time from `(x, y, z)`, with `3.2px`
  outlines and three flat tones per solid. Pass SVG colours as `fill="var(--token)"`
  attributes, not CSS classes; scoped styles do not reach every SVG presentation property.
- **The page background must equal the scene's wall colour** so the illustration and the page
  read as one environment.

## Development

When starting the dev server, use background mode:

```
astro dev --background
```

Manage the background server with `astro dev stop`, `astro dev status`, and `astro dev logs`.

## Documentation

Full documentation: https://docs.astro.build

Consult these guides before working on related tasks:

- [Adding pages, dynamic routes, or middleware](https://docs.astro.build/en/guides/routing/)
- [Working with Astro components](https://docs.astro.build/en/basics/astro-components/)
- [Using React, Vue, Svelte, or other framework components](https://docs.astro.build/en/guides/framework-components/)
- [Adding or managing content](https://docs.astro.build/en/guides/content-collections/)
- [Adding styles or using Tailwind](https://docs.astro.build/en/guides/styling/)
- [Supporting multiple languages](https://docs.astro.build/en/guides/internationalization/)
