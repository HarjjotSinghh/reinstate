# Rebuilt research diagrams

Vector recreations of the five research PNGs in `assets/`.

- Each numbered `.svg` is editable vector artwork.
- Each matching `.png` is rendered directly from that SVG.
- Each `compare_*.png` places the original and rebuilt render side by side.
- `review.html` presents all five comparisons in one page.

The approved SVGs are published to `assets/` and `website/public/brand/` and
are the canonical diagrams used throughout the repository and website.

Regenerate the derived SVG, PNG, comparison, and review files with:

```bash
cd website
npm run generate:research-diagrams
```
