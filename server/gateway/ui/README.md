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
