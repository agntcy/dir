# Release

This document outlines the process for creating a new release for Directory using the [Go MultiMod Releaser](https://github.com/open-telemetry/opentelemetry-go-build-tools/tree/main/multimod). All code block examples provided below correspond to an update to version `v1.0.0`, please update accordingly.

## 1. Update the New Release Version

* Create a new branch for the release version update.
```sh
git checkout -b release/v1.0.0
```

* Modify the `versions.yaml` file to update the version for Directory's module-set. Keep in mind that the same version is applied to all modules.
```diff
  directory:
-    version: v0.0.0
+    version: v1.0.0
```

* Update the SDK versions in `sdk/dir-js/package.json`:
```diff
  {
    "name": "agntcy-dir",
-   "version": "0.0.0",
+   "version": "1.0.0",
    "description": "Directory SDK",
```

* Update the SDK version in `sdk/dir-py/pyproject.toml`:
```diff
  [project]
  name = "agntcy-dir"
- version = "0.0.0"
+ version = "1.0.0"
  description = "Directory SDK"
```

> [!NOTE]
> For release candidates, use versions like `1.0.0-rc.1` for JavaScript and `1.0.0rc1` for Python (no dash). For example: `v1.0.0-rc.1` becomes `1.0.0-rc.1` in package.json and `1.0.0rc1` in pyproject.toml.

* After updating the SDK versions, regenerate the lock files:
```sh
cd sdk/dir-js && npm install && cd ../..
cd sdk/dir-py && uv lock && cd ../..
```

* Commit the changes with a suitable message.
```sh
git add versions.yaml sdk/dir-js/package.json sdk/dir-js/package-lock.json sdk/dir-py/pyproject.toml sdk/dir-py/uv.lock
git commit -m "release: update module set to version v1.0.0"
```

* Run the version verification command to check for any issues.

> [!NOTE]
> If you use `go.work` and `go.work.sum` files in the project, temporarily remove/rename them so it won't interfere with this step.

```sh
task release:verify
```

## 2. Bump All Dependencies to the New Release Version

* Run the following command to update all `go.mod` files to the new release version.
```sh
task release:prepare
```

> [!NOTE]
> If this command fails with `failed: working tree not clean`, please run `git stash --all` and retry.

* Review the changes made in the last commit to ensure correctness.

* Push the branch to the GitHub repository.
```sh
git push origin release/v1.0.0
```

* Create a pull request for these changes with a title like "release: prepare version v1.0.0".

## 3. Create and Push Tags

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

## 4. Publish release

* Wait until the release workflow is completed successfully.

* Navigate to the [Releases page](https://github.com/agntcy/dir/releases) and verify the draft release description as well as the assets listed.

* Once the draft release has been verified, click on `Edit` release and then on `Publish Release`.
