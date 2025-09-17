# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **API**: Implement Search API for network-wide record discovery using RecordQuery interface (#362)
- **API**: Add initial authorization framework (#330)
- **API**: Add distributed label-based announce/discovery via DHT (#285)
- **API**: Add wildcard search support with * pattern matching (#355)
- **API**: Add max replicasets to keep in deployment (#207)
- **API**: Add sync API (#199)
- **SDK**: Add SDK release process (#216)
- **SDK**: Add more gRPC services (#294)
- **SDK**: Add gRPC client code and example for JavaScript SDK (#248)
- **SDK**: Add sync support (#361)
- **SDK**: Add sign and verification (#337)
- **SDK**: Add testing solution for CI (#269)
- **SDK**: Standardize Python SDK tooling for Directory (#371)
- **Security**: Implement server-side verification with zot (#286)
- **Security**: Use SPIFFE/SPIRE to enable security schema (#210)
- **Security**: Add spire federation support (#295)
- **Storage**: Add storage layer full-index record synchronisation (#227)
- **Storage**: Add post sync verification (#324)
- **Storage**: Enable search on synced records (#310)
- **Storage**: Add fallback to client-side verification (#373)
- **Storage**: Add policy-based publish (#333)
- **Storage**: Add custom type for error handling (#189)
- **Storage**: Add sign and verify gRPC service (#201)
- **Storage**: Add new hub https://hub.agntcy.org/directory (#202)
- **CI**: Add Codecov workflow & docs (#380)
- **CI**: Add npm caching for SDK (#339)
- **CI**: Introduce BSR (#212)

### Changed
- **API**: Switch to generic OASF objects across codebase (#381)
- **API**: Version upgrade of API services (#225)
- **API**: Update sync API and add credential RPC (#217)
- **API**: Refactor domain interfaces to align with OASF schema
- **API**: Rename v1alpha2 to v1 (#258)
- **API**: Migrate record signature to OCI native signature (#250)
- **API**: Store implementations and digest/CID calculation (#238)
- **SDK**: Move and refactor Python SDK code (#229)
- **SDK**: Bump package versions for release (#274)
- **SDK**: Bump versions for release (#249)
- **SDK**: Support streams & update docs (#284)
- **SDK**: Update API code and add example code for Python SDK (#237)
- **Storage**: Standardize and cleanup store providers (#385)
- **Storage**: Improve logging to suppress misleading errors in database and routing layers (#289)
- **Storage**: Refactor E2E Test Suite & Utilities Enhancement (#268)
- **Storage**: Refactor e2e tests multiple OASF versions (#278)
- **Storage**: Refactor: remove semantic tags keep only CID tag (#265)
- **Storage**: Refactor: remove generated OASF objects (#356)
- **Storage**: Refactor: remove builder artifacts and build cmd usages (#329)
- **Storage**: Refactor: remove agent refs (#331)
- **Storage**: Refactor: remove redundant proto files (#219)
- **Storage**: Refactor: remove Legacy List API and Migrate to RecordQuery-Based System (#342)
- **Storage**: Refactor: remove Python code generation (#215)
- **Documentation**: Update usage documentation for record generation (#287)
- **Documentation**: Add and update README for refactored SDKs (#273)
- **Documentation**: Update README to reflect new usage documentation link and remove USAGE.md file (#332)
- **Documentation**: Update documentation setup (#394)
- **CI**: Update codebase with proto namespace changes (#398)
- **CI**: Update CI task gen to ignore buf lock file changes (#275)
- **CI**: Update brew formula version (#372, #263, #257, #247)
- **CI**: Bump Go (#221)
- **CI**: Pin OASF buf version (#312)

### Fixed
- **API**: Resolve buf proto API namespacing issues (#393)
- **API**: Add sync testdata (#396)
- **API**: Update apiserver.env to use new config values (#406)
- **API**: Suppress command usage display on runtime errors (#290)
- **API**: Quick-fix for e2e CLI cmd state handling (#270)
- **API**: Fix/CI task gen (#271)
- **SDK**: Fix Python SDK imports and tests (#403)
- **SDK**: Fix codeowners file (#404)
- **Storage**: Add separate maintainers for hub CLI directory (#375)
- **Storage**: Update agent directory default location (#226)
- **CI**: Allow dir-hub-maintainers release (#402)

### Dependencies
- **chore(deps)**: Bump github.com/go-viper/mapstructure/v2 from 2.3.0 to 2.4.0 (#314)
- **chore(deps)**: Bump github.com/go-viper/mapstructure/v2 from 2.2.1 to 2.3.0 (#200)

## [v0.2.13] - Previous Release

[Full Changelog](https://github.com/agntcy/dir/compare/v0.2.12...v0.2.13)

---

## [v0.2.12] - Previous Release

[Full Changelog](https://github.com/agntcy/dir/compare/v0.2.11...v0.2.12)

---

## [v0.2.11] - Previous Release

[Full Changelog](https://github.com/agntcy/dir/compare/v0.2.10...v0.2.11)

---

## [v0.2.10] - Previous Release

[Full Changelog](https://github.com/agntcy/dir/compare/v0.2.9...v0.2.10)

---

## [v0.2.9] - Previous Release

[Full Changelog](https://github.com/agntcy/dir/compare/v0.2.8...v0.2.9)

---

## [v0.2.8] - Previous Release

[Full Changelog](https://github.com/agntcy/dir/compare/v0.2.7...v0.2.8)

---

## [v0.2.7] - Previous Release

[Full Changelog](https://github.com/agntcy/dir/compare/v0.2.6...v0.2.7)

---

## [v0.2.6] - Previous Release

[Full Changelog](https://github.com/agntcy/dir/compare/v0.2.5...v0.2.6)

---

## [v0.2.5] - Previous Release

[Full Changelog](https://github.com/agntcy/dir/compare/v0.2.4...v0.2.5)

---

## [v0.2.4] - Previous Release

[Full Changelog](https://github.com/agntcy/dir/compare/v0.2.3...v0.2.4)

---

## [v0.2.3] - Previous Release

[Full Changelog](https://github.com/agntcy/dir/compare/v0.2.2...v0.2.3)

---

## [v0.2.2] - Previous Release

[Full Changelog](https://github.com/agntcy/dir/compare/v0.2.1...v0.2.2)

---

## [v0.2.1] - Previous Release

[Full Changelog](https://github.com/agntcy/dir/compare/v0.2.0...v0.2.1)

---

## [v0.2.0] - Previous Release

[Full Changelog](https://github.com/agntcy/dir/compare/v0.1.6...v0.2.0)

---

## [v0.1.6] - Previous Release

[Full Changelog](https://github.com/agntcy/dir/compare/v0.1.5...v0.1.6)

---

## [v0.1.5] - Previous Release

[Full Changelog](https://github.com/agntcy/dir/compare/v0.1.4...v0.1.5)

---

## [v0.1.4] - Previous Release

[Full Changelog](https://github.com/agntcy/dir/compare/v0.1.3...v0.1.4)

---

## [v0.1.3] - Previous Release

[Full Changelog](https://github.com/agntcy/dir/compare/v0.1.2...v0.1.3)

---

## [v0.1.2] - Previous Release

[Full Changelog](https://github.com/agntcy/dir/compare/v0.1.1...v0.1.2)

---

## [v0.1.1] - Previous Release

[Full Changelog](https://github.com/agntcy/dir/compare/v0.1.0...v0.1.1)

---

## [v0.1.0] - Initial Release

[Full Changelog](https://github.com/agntcy/dir/releases/tag/v0.1.0)

---

## Legend

- **Added** for new features
- **Changed** for changes in existing functionality
- **Deprecated** for soon-to-be removed features
- **Removed** for now removed features
- **Fixed** for any bug fixes
- **Security** for vulnerability fixes
