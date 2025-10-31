// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

# Documentation available at: https://docs.docker.com/build/bake/

# Docker build args
variable "IMAGE_REPO" { default = "ghcr.io/agntcy" }
variable "IMAGE_TAG" { default = "v0.1.0-rc" }
variable "EXTRA_LDFLAGS" { default = "" }

function "get_tag" {
  params = [tags, name]
  result = coalescelist(tags, ["${IMAGE_REPO}/${name}:${IMAGE_TAG}"])
}

group "default" {
  targets = [
    "dir-apiserver",
    "dir-ctl",
  ]
}

group "coverage" {
  targets = [
    "dir-apiserver-coverage",
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

target "sdks-test" {
  context = "."
  dockerfile = "./e2e/sdk/Dockerfile"
  depends_on = ["dir-ctl"] # Ensures dir-ctl is built first
  inherits = [
    "_common",
    "docker-metadata-action",
  ]
  tags = get_tag(target.docker-metadata-action.tags, "${target.sdks-test.name}")
}
