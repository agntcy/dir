# Directory documentation

MkDocs site for [agntcy/dir](https://github.com/agntcy/dir), published at [https://agntcy.github.io/dir/](https://agntcy.github.io/dir/) with a **version selector** (latest release by default, older releases in the dropdown).

## Layout

| Path | Purpose |
|------|---------|
| `content/` | Published Markdown |
| `mkdocs/` | MkDocs config, uv project, theme overrides |
| `Taskfile.yml` | Build, serve, lint, and local mike preview tasks |
| `research/`, `rfc/` | Drafts and experiments (not in site nav) |

## Prerequisites

- [Task](https://taskfile.dev/)
- [uv](https://docs.astral.sh/uv/)
- [lychee](https://github.com/lycheeverse/lychee) (required for `task docs:ci` / `task docs:test`; installed automatically in CI)

Lint checks run on published nav content (`content/index.md` and `content/dir/`), not orphan pages under `content/`.

## Commands (from repo root)

```bash
task docs:run              # live preview while editing (no version dropdown)
task docs:build            # static site → .build/site
task docs:ci               # build + lint (same as PR CI)
task docs:test             # lint only
task docs:mike:deploy-local   # versioned preview on local gh-pages (no push)
task docs:mike:serve       # serve local gh-pages (after deploy-local)
```

From `docs/`:

```bash
task run
task -t Taskfile.yml ci
```

## Version selector (production)

Each release or semver tag deploy:

1. Builds **only that version** with [mike](https://github.com/jimporter/mike).
2. Sets it as the **default** (`mike set-default`) so https://agntcy.github.io/dir/ opens the latest release.
3. Keeps **older versions** in the dropdown (not rebuilt on later tags).
4. Publishes the combined site via **GitHub Actions Pages**; the `gh-pages` branch is updated **only in CI** as a version ledger between releases.

The live site is whatever **deploy-pages** publishes from the CI artifact (run **Docs** and **Deploy artifacts** in the same workflow). The `gh-pages` branch grows as you keep older versions; prune with `mike delete` when retiring doc releases.

Publishing is not available from a local Taskfile push—use GitHub Actions (release, `v*.*.*` tag, or workflow dispatch with a version).

## CI and deploy

- **PRs** (when `docs/**` or docs workflows change): [`.github/workflows/docs-ci.yaml`](../.github/workflows/docs-ci.yaml) runs `task docs:ci`.
- **Releases / tags**: [`.github/workflows/docs-deploy.yml`](../.github/workflows/docs-deploy.yml) runs `task docs:mike:pages-build` with `VERSION` from the tag or release, then deploys `.build/mike-site`.

### Local version-selector preview

```bash
# Optional: fetch existing versions from origin
git fetch origin gh-pages

task docs:mike:deploy-local   # label from [versions] local in mike_versions.ini
task docs:mike:serve          # http://127.0.0.1:8000/ with dropdown
```

GitHub repo **Settings → Pages → Build and deployment** must use **GitHub Actions** as the source.

**Settings → Environments → github-pages → Deployment branches and tags** must allow release tags (e.g. add tag pattern `v*`) or use **No restriction**. Otherwise tag-triggered deploys fail with environment protection errors.
