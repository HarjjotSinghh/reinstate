import { describe, expect, it } from 'vitest';
import { htmlToMarkdown } from './html-to-markdown';

const page = (main: string, head = '') => `<!doctype html><html><head><title>Install Reinstate | Reinstate</title>
<meta name="description" content="Install &amp; configure Reinstate.">${head}</head>
<body><header><nav><a href="/">Home</a></nav></header><main>${main}</main><footer><a href="/privacy">Privacy</a></footer></body></html>`;

describe('htmlToMarkdown', () => {
  it('keeps the outline, absolute links, fenced code with language, and drops chrome', () => {
    const html = page(`
      <nav aria-label="Breadcrumb"><a href="/docs">Docs</a></nav>
      <aside><nav aria-label="Documentation"><a href="/docs/faq">FAQ</a></nav></aside>
      <article>
        <h1>Install and sync Reinstate</h1>
        <svg viewBox="0 0 10 10"><text>decoration</text></svg>
        <p>Reinstate finds <em>local</em> sessions. See <a href="/docs/faq">the FAQ</a> and <a href="https://example.com/x">external</a>.</p>
        <button>Copy</button>
        <script>console.log('no')</script>
        <h2 id="install">Install</h2>
        <pre class="astro-code github-dark-default" data-language="sh" tabindex="0"><code><span class="line"><span style="color:#F00">curl</span><span> -fsSL</span> https://reinstate.dev/install.sh | sh</span></code></pre>
        <ul><li>one</li><li>two <code>rein doctor</code></li></ul>
        <div aria-hidden="true">hidden decoration</div>
        <p data-markdown="skip">skipped</p>
      </article>`);

    const result = htmlToMarkdown(html, { url: 'https://reinstate.dev/docs/getting-started' });

    expect(result.title).toBe('Install and sync Reinstate');
    expect(result.description).toBe('Install & configure Reinstate.');
    expect(result.markdown).toMatch(/^# Install and sync Reinstate\n\n> Install & configure Reinstate\.\n/);
    expect(result.markdown).toContain('Reinstate finds *local* sessions. See [the FAQ](https://reinstate.dev/docs/faq) and [external](https://example.com/x).');
    expect(result.markdown).toContain('## Install');
    expect(result.markdown).toContain('```sh\ncurl -fsSL https://reinstate.dev/install.sh | sh\n```');
    expect(result.markdown).toContain('- one\n- two `rein doctor`');
    expect(result.markdown).toContain('Source: https://reinstate.dev/docs/getting-started');
    for (const absent of ['Copy', 'console.log', 'decoration', 'skipped', 'Privacy', 'Home', 'FAQ](https://reinstate.dev/docs/faq)\n']) {
      expect(result.markdown, absent).not.toContain(absent);
    }
    expect(result.markdown).not.toMatch(/\n{3,}/);
  });

  it('renders GFM tables and definition lists', () => {
    const html = page(`
      <h1>Facts</h1>
      <dl><div><dt>License</dt><dd><a href="https://www.apache.org/licenses/LICENSE-2.0">Apache-2.0</a></dd></div>
      <div><dt>Language</dt><dd>Go</dd></div></dl>
      <table><thead><tr><th>Area</th><th>Role</th></tr></thead>
      <tbody><tr><td><code>cmd/reinstate</code></td><td>CLI entry point</td></tr></tbody></table>`);

    const { markdown } = htmlToMarkdown(html, { url: 'https://reinstate.dev/open-source' });

    expect(markdown).toContain('- **License:** [Apache-2.0](https://www.apache.org/licenses/LICENSE-2.0)');
    expect(markdown).toContain('- **Language:** Go');
    expect(markdown).toContain('| Area | Role |');
    expect(markdown).toContain('| `cmd/reinstate` | CLI entry point |');
  });

  it('falls back to the document title when the main content has no H1', () => {
    const html = page('<p>Just a paragraph.</p>');
    const { markdown, title } = htmlToMarkdown(html, { url: 'https://reinstate.dev/x' });
    expect(title).toBe('Install Reinstate | Reinstate');
    expect(markdown.startsWith('# Install Reinstate | Reinstate\n\n> Install & configure Reinstate.\n\nJust a paragraph.')).toBe(true);
  });

  it('records the modification date from article metadata', () => {
    const html = page('<h1>Post</h1><p>Body</p>', '<meta property="article:modified_time" content="2026-08-16T00:00:00.000Z">');
    const { markdown } = htmlToMarkdown(html, { url: 'https://reinstate.dev/blog/post' });
    expect(markdown).toContain('Last modified: 2026-08-16.');
  });

  it('keeps image alt text with absolute sources and drops unsafe link schemes', () => {
    const html = page('<h1>Art</h1><p><img src="/brand/mark.png" alt="Reinstate mark"> <a href="javascript:alert(1)">x</a> <img src="/decor.png" alt=""></p>');
    const { markdown } = htmlToMarkdown(html, { url: 'https://reinstate.dev/' });
    expect(markdown).toContain('![Reinstate mark](https://reinstate.dev/brand/mark.png)');
    expect(markdown).not.toContain('javascript:');
    expect(markdown).not.toContain('decor.png');
  });
});
