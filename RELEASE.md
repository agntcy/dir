# Release

This document outlines the process for creating a new release for Directory packages. 
All code block examples provided below correspond to an update to version `v1.0.0`, please update accordingly.

## 1. Create Release branch

Prepare a new release for the desired version by running the following command:

```sh
task release:create RELEASE_VERSION=v1.0.0
```

> [!NOTE]
> For SDK release candidates, versions like `1.0.0-rc.1` becomes `1.0.0-rc.1` in JavaScript package.json
> and `1.0.0rc1` in Python pyproject.toml.

## 2. Create and Push Tags

* After the pull request is approved and merged, update your local main branch.
```sh
git checkout main
git pull origin main
```

* To trigger the release workflow, create and push to the repository a release tag for the last commit.
```sh
git tag -a v1.0.0
git push origin v1.0.0
```

Please note that the release tag is not necessarily associated with the "release: prepare version v1.0.0" commit. For example, if any bug fixes were required after this commit, they can be merged and included in the release.

## 3. Publish release

* Wait until the release workflow is completed successfully.

* Navigate to the [Releases page](https://github.com/agntcy/dir/releases) and verify the draft release description as well as the assets listed.

* Once the draft release has been verified, click on `Edit` release and then on `Publish Release`.
