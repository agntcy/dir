# Release

This document outlines the process for creating a new release for Directory packages.
All code block examples below use version `v1.3.0`; update the version accordingly.

The Directory release is split into two phases:

- API module release: Go module tags only, no root tag and no artifacts.
- Server release: root tag, server module tags, images, Helm charts, CLI binaries, and GitHub Release.

## 1. Prepare API Module Release

Prepare the API module set release branch:

```sh
task release:create:api RELEASE_VERSION=v1.3.0
```

This prepares only the `api` module set:

- `github.com/agntcy/dir/api`
- `github.com/agntcy/dir/client`
- `github.com/agntcy/dir/utils`

Open a pull request from the generated release branch, wait for approval, and merge it into `main`.

## 2. Tag API Modules

After the API release pull request is merged, update your local `main` branch:

```sh
git checkout main
git pull origin main
```

Create and push only the API module tags:

```sh
git tag api/v1.3.0
git tag client/v1.3.0
git tag utils/v1.3.0

git push origin api/v1.3.0 client/v1.3.0 utils/v1.3.0
```

Do not create the root `v1.3.0` tag during this phase. Root tags trigger artifact releases.

## 2a. Optional: Release API Consumers

After the API module tags are pushed, you can update API consumers with the new `api`, `client`, and `utils` tags and create releases for them.

Released API consumers:

- [`dir-importer`](https://github.com/agntcy/dir-importer)
- [`dir-runtime`](https://github.com/agntcy/dir-runtime)

This step is useful when those projects need to publish releases against the new API module versions before the Directory server release. The `cli` and `server` dependency updates can happen later in the server release preparation step.

## 3. Prepare Server Release

Prepare the server module set release branch:

```sh
task release:create:server RELEASE_VERSION=v1.3.0
```

This prepares only the `server` module set:

- `github.com/agntcy/dir/cli`
- `github.com/agntcy/dir/tests`
- `github.com/agntcy/dir/server`
- `github.com/agntcy/dir/reconciler`

If optional API consumer releases were created in step 2a, update the corresponding dependencies as well:

- `github.com/agntcy/dir-importer`
- `github.com/agntcy/dir-runtime`

If the release preparation updates Go module versions, tidy the module files before pushing the release branch:

```sh
task deps:tidy
```

Open a pull request from the generated release branch, wait for approval, and merge it into `main`.

## 4. Create Root Release Tag

After the server release pull request is merged, update your local `main` branch:

```sh
git checkout main
git pull origin main
```

Create and push the root release tag:

```sh
git tag -a v1.3.0
git push origin v1.3.0
```

The root tag triggers the release workflow, which publishes:

- container images
- Helm charts
- CLI binaries
- draft GitHub Release

Please note that the release tag is not necessarily associated with the release preparation commit. For example, if bug fixes were required after this commit, they can be merged and included in the release.

## 5. Publish Release

- Wait until the release workflow completes successfully.
- Navigate to the [Releases page](https://github.com/agntcy/dir/releases) and verify the draft release description and assets.
- Click `Edit` on the draft release, then click `Publish Release`.

Publishing the root GitHub Release creates only the server-side module tags:

- `cli/v1.3.0`
- `tests/v1.3.0`
- `server/v1.3.0`
- `reconciler/v1.3.0`

It does not re-tag `api`, `client`, or `utils`.

Publishing the release also starts the `Post Release` workflow, which pushes the WinGet manifests for `dirctl` to the `agntcy/winget-pkgs` fork (see the next section).

### Publish to WinGet

The release workflow attaches `dirctl-v1.3.0-windows-amd64.zip` (`dirctl.exe` and `LICENSE.md`) to the release. When a stable release is published, `Post Release` renders the `AGNTCY.dirctl` manifests from that zip, validates them on a Windows runner (`winget validate`, then an install from the local manifests and `dirctl version`) and pushes them to the `agntcy/winget-pkgs` fork. Prereleases are skipped.

The manifest links the privacy policy at the release tag (`cli/PRIVACY.md`), because `dirctl` handles credentials and WinGet asks for one. The workflow stops if the tag does not contain the file. Update the policy in the same pull request as any change to what `dirctl` reads, writes or sends.

The GitHub App can push to the fork but cannot open a pull request against `microsoft/winget-pkgs`. To finish:

1. Open the `WinGet` job's summary in the `Post Release` run. It links to the compare page and gives the pull request title: `New package: AGNTCY.dirctl version 1.3.0` for the first submission, `New version: AGNTCY.dirctl version 1.3.0` after that.
2. Open the pull request from that page and let `microsoft/winget-pkgs` validation pass.

To re-run or check a release without pushing, run the `WinGet` workflow manually with the release tag and leave `publish` off.

## 6. Update Hardcoded Versions in the README

Bump the pinned release version in the `README.md` distribution examples (the `docker pull` image tags and the `helm` `--version` flags).
