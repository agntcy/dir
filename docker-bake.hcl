// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

# Documentation available at: https://docs.docker.com/build/bake/

# Docker build args
variable "IMAGE_REPO" { default = "ghcr.io/agntcy" }
variable "IMAGE_TAG" { default = "v0.1.0-rc" }
variable "EXTRA_LDFLAGS" { default = "" }
variable "IMAGE_NAME_SUFFIX" { default = "" }
variable "REGSYNC_VERSION" { default = "v0.11.1" }

function "get_tag" {
  params = [tags, name]
  result = coalescelist(tags, ["${IMAGE_REPO}/${name}${IMAGE_NAME_SUFFIX}:${IMAGE_TAG}"])
}

group "default" {
  targets = [
    "dir-apiserver",
    "dir-ctl",
    "dir-reconciler",
    "dir-runtime-discovery",
    "dir-runtime-server",
    "envoy-authz",
  ]
}

group "coverage" {
  targets = [
    "dir-apiserver-coverage",
    "dir-reconciler-coverage",
  ]
}

target "_common" {
  output = [
    "type=image",
  ]
  platforms = [
    "linux/arm64",
    "linux/amd64",
  ]
  args = {
    EXTRA_LDFLAGS = "${EXTRA_LDFLAGS}"
  }
}

target "docker-metadata-action" {
  tags = []
}


target "dir-apiserver" {
  context = "."
  dockerfile = "./server/Dockerfile"
  target = "production"
  inherits = [
    "_common",
    "docker-metadata-action",
  ]
  tags = get_tag(target.docker-metadata-action.tags, "${target.dir-apiserver.name}")
}

target "dir-apiserver-coverage" {
  context = "."
  dockerfile = "./server/Dockerfile"
  target = "coverage"
  inherits = [
    "_common",
    "docker-metadata-action",
  ]
  tags = get_tag(target.docker-metadata-action.tags, "dir-apiserver")
}

target "dir-ctl" {
  context = "."
  dockerfile = "./cli/Dockerfile"
  inherits = [
    "_common",
    "docker-metadata-action",
  ]
  tags = get_tag(target.docker-metadata-action.tags, "${target.dir-ctl.name}")
}

target "envoy-authz" {
  context = "."
  dockerfile = "./auth/cmd/envoy-authz/Dockerfile"
  inherits = [
    "_common",
    "docker-metadata-action",
  ]
  tags = get_tag(target.docker-metadata-action.tags, "${target.envoy-authz.name}")
}

target "dir-reconciler" {
  context = "."
  dockerfile = "./reconciler/Dockerfile"
  inherits = [
    "_common",
    "docker-metadata-action",
  ]
  args = {
    REGSYNC_VERSION = "${REGSYNC_VERSION}"
  }
  tags = get_tag(target.docker-metadata-action.tags, "${target.dir-reconciler.name}")
}

target "dir-reconciler-coverage" {
  context = "."
  dockerfile = "./reconciler/Dockerfile"
  target = "coverage"
  inherits = [
    "_common",
    "docker-metadata-action",
  ]
  tags = get_tag(target.docker-metadata-action.tags, "dir-reconciler")
}

target "dir-runtime-discovery" {
  context = "."
  dockerfile = "./runtime/discovery/Dockerfile"
  inherits = [
    "_common",
    "docker-metadata-action",
  ]
  tags = get_tag(target.docker-metadata-action.tags, "${target.dir-runtime-discovery.name}")
}

target "dir-runtime-server" {
  context = "."
  dockerfile = "./runtime/server/Dockerfile"
  inherits = [
    "_common",
    "docker-metadata-action",
  ]
  tags = get_tag(target.docker-metadata-action.tags, "${target.dir-runtime-server.name}")
}

target "sdks-test" {
  context = "."
  dockerfile = "./e2e/sdk/Dockerfile"
  depends_on = ["dir-ctl"] # Ensures dir-ctl is built first
  inherits = [
    "_common",
    "docker-metadata-action",
  ]
  args = {
    IMAGE_REPO = "${IMAGE_REPO}"
    IMAGE_TAG = "${IMAGE_TAG}"
  }
  tags = get_tag(target.docker-metadata-action.tags, "${target.sdks-test.name}")
}
