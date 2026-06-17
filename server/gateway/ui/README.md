# Gateway UI

SvelteKit-based web dashboard for the dirctl HTTP gateway. The built static
assets are embedded into the Go binary via `//go:embed static` and served
alongside the REST API.

## Prerequisites

- Node.js 20+
- npm 10+

## Development

Start the Vite dev server with hot module replacement. API calls are proxied
to the daemon running on `localhost:8889`:

```bash
cd server/gateway/ui
npm install
npm run dev
```

Open `http://localhost:5173` in your browser. The daemon must be running
separately (`dirctl daemon start` or `task cli:compile && ./.bin/dirctl daemon start`).

## Build for Production

Build the static output into `server/gateway/static/`:

```bash
task ui:build
```

Or manually:

```bash
cd server/gateway/ui
npm ci
npm run build
```

The output lands in `server/gateway/static/` which is the `//go:embed` target.
After building the UI, recompile the Go binary to embed the fresh assets:

```bash
task cli:compile
```

### Static asset caching

The Go gateway sets HTTP cache and compression headers for embedded UI assets:

- `/_app/immutable/*` — `Cache-Control: public, max-age=31536000, immutable`; gzip at `BestSpeed` when the client accepts gzip and the asset is a compressible text type (CSS, JS, HTML, JSON, SVG)
- `/_app/version.json` — `Cache-Control: no-cache` for SvelteKit deployment update checks
- `/` and SPA fallbacks — `Cache-Control: no-cache` so deploys pick up new hashed bundles
- Missing `/_app/*` assets return 404 (not the SPA shell) so broken deploys are visible
- Inter font files are bundled via `@fontsource/inter` (Latin subset; no Google Fonts at runtime); woff/woff2 fonts are served uncompressed

When exposing the HTTP gateway through F5 NGINX Ingress, enable edge gzip and upstream
keepalive via `httpGateway.ingress.annotations` (see chart defaults in
`install/charts/dir/apiserver/values.yaml`).

## Full rebuild (UI + CLI binary)

```bash
task ui:build && task cli:compile
```

## Project Structure

```
server/gateway/ui/
  src/
    routes/           SvelteKit file-based routes
      +page.svelte    AI Catalog dashboard (home page)
      +layout.svelte  Shared layout (imports CSS)
      +layout.ts      Static rendering config
    lib/
      components/     Reusable Svelte components
        AgentCard.svelte
        DetailModal.svelte
        FilterSidebar.svelte
        MediaTypeBadge.svelte
        Pagination.svelte
        StarRating.svelte
        VerifiedBadge.svelte
      api.ts          API fetch helpers
      types.ts        TypeScript interfaces
      utils.ts        Shared utility functions
    app.html          HTML shell
    app.css           Tailwind CSS with brand tokens
  svelte.config.js    SvelteKit config (static adapter)
  vite.config.ts      Vite config (Tailwind plugin, API proxy)
```

## Adding a New Page

Create a new route file:

```bash
mkdir -p src/routes/config
touch src/routes/config/+page.svelte
```

The page will be available at `/config` automatically via SvelteKit's
file-based router. The Go gateway serves `index.html` for any unknown path
(SPA fallback), so client-side routing works without server changes.

## Type Checking

```bash
npm run check
```

## Formatting

```bash
npm run format
```
