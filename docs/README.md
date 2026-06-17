# Agent Directory Service documentation

MkDocs site for [agntcy/dir](https://github.com/agntcy/dir), published at [https://agntcy.github.io/dir/](https://agntcy.github.io/dir/) with a **version selector** (latest release by default, older releases in the dropdown).

## Layout

| Path | Purpose |
|------|---------|
| `content/` | Published Markdown |
| `mkdocs/` | MkDocs config, uv project, theme overrides |
| `Taskfile.yml` | Build, serve, lint, and local mike preview tasks |

## Prerequisites

- [Task](https://taskfile.dev/)
- [uv](https://docs.astral.sh/uv/)
- [lychee](https://github.com/lycheeverse/lychee) (required for `task docs:ci` / `task docs:test`; installed automatically in CI)

Lint checks run on published nav content (`content/index.md`, `content/community.md`, and `content/dir/`), not orphan pages under `content/`.

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

## CI and deploy

All build, lint, and deploy logic lives in [`Taskfile.yml`](Taskfile.yml). The GitHub Actions workflows are thin: they provision tools (`uv`, `task`, `lychee`) and call task entries. Dependencies come from `mkdocs/pyproject.toml` / `mkdocs/uv.lock` via `task` → `uv sync`.

- [`.github/workflows/reusable-docs.yml`](../.github/workflows/reusable-docs.yml) — reusable pipeline (`workflow_call`) with a `mode` input (`ci` / `deploy-dev` / `deploy-release`); holds the shared setup + task calls.
- [`.github/workflows/docs-ci.yaml`](../.github/workflows/docs-ci.yaml) — on pull requests, calls the reusable pipeline with `mode: ci`.
- [`.github/workflows/docs-deploy.yml`](../.github/workflows/docs-deploy.yml) — on push to `main`, `v*.*.*` tags, releases, and manual dispatch; resolves the mode/version from the event and calls the reusable pipeline.

| Trigger | Mode | Task(s) invoked | Effect |
| --- | --- | --- | --- |
| Pull request (touching `docs/**` or the workflows) | `ci` | `task docs:ci` | Build + lint (codespell, pymarkdown, lychee). No deploy. |
| Push to `main` | `deploy-dev` | `task docs:ci` + `docs:deploy:dev` + `docs:deploy:root-files` | Build + lint, then `mike deploy --push dev` (the `dev` version). **Never touches `latest`.** |
| Release published / `v*.*.*` tag / manual dispatch | `deploy-release` | `task docs:deploy:release VERSION=<v>` + `docs:deploy:root-files` | `mike deploy --push --update-aliases <version> latest` then `mike set-default latest`, so https://agntcy.github.io/dir/ opens the latest release. |

Deploys use [mike](https://github.com/jimporter/mike) to push each version into a subdirectory of the `gh-pages` branch (the version ledger). `dev` is published as its own version (push to `main`) and `latest` is an alias of the newest release. With `alias_type: copy` plus `canonical_version: latest`, the `latest` alias is a full standalone copy of the release version, so both `/dev/` and `/latest/` serve content in place (HTTP 200, no redirect). Older versions stay in the dropdown; prune with `mike delete` when retiring doc releases. The `docs:deploy:*` tasks push to the remote, so they refuse to run outside CI unless `ALLOW_DOCS_PUSH=1` is set; root-file publishing reads its token from the `GH_TOKEN` environment variable (`REPO_SLUG` selects the repo).

> One-time migration: an earlier setup used `alias_type: redirect` and published `dev` as an alias of a `main` version, so `/dev/` and `/latest/` bounced via HTTP redirect. After this change, run `mike delete --push main` once (CI or `ALLOW_DOCS_PUSH=1`) to drop the stale `main` version, then let the next `main` push (`dev`) and release (`latest`) redeploy so the `latest` alias is recreated as an in-place copy.

GitHub Pages serves directly from the `gh-pages` branch (**Settings → Pages → Build and deployment → Deploy from a branch → `gh-pages`**).

### Local version-selector preview

```bash
# Optional: fetch existing versions from origin
git fetch origin gh-pages

task docs:mike:deploy-local   # deploys the current docs to local gh-pages as `dev` (no push)
task docs:mike:serve          # http://127.0.0.1:8000/ with the version dropdown
```
