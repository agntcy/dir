// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

# Documentation available at: https://docs.docker.com/build/bake/

# Docker build args
variable "IMAGE_REPO" { default = "ghcr.io/agntcy" }
variable "IMAGE_TAG" { default = "v0.1.0-rc" }
variable "BUILD_LDFLAGS" { default = "-s -w -extldflags -static" }
variable "IMAGE_NAME_SUFFIX" { default = "" }
variable "MCP_SCANNER_VERSION" { default = "4.7.1" }
variable "SKILL_SCANNER_VERSION" { default = "2.0.8" }
variable "A2A_SCANNER_VERSION" { default = "1.0.1" }

function "get_tag" {
  params = [tags, name]
  result = coalescelist(tags, ["${IMAGE_REPO}/${name}${IMAGE_NAME_SUFFIX}:${IMAGE_TAG}"])
}

group "default" {
  targets = [
    "dir-apiserver",
    "dir-ctl",
    "dir-reconciler",
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
    BUILD_LDFLAGS = "${BUILD_LDFLAGS}"
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

target "dir-reconciler" {
  context = "."
  dockerfile = "./reconciler/Dockerfile"
  inherits = [
    "_common",
    "docker-metadata-action",
  ]
  args = {
    MCP_SCANNER_VERSION   = "${MCP_SCANNER_VERSION}"
    SKILL_SCANNER_VERSION = "${SKILL_SCANNER_VERSION}"
    A2A_SCANNER_VERSION   = "${A2A_SCANNER_VERSION}"
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
