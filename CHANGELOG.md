# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v0.5.1] - 2025-11-17

### Key Highlights

This release focuses on operational improvements and deployment enhancements, including:

**Helm & Deployment Improvements**
- External Secrets Operator integration for secure credential management
- SPIRE ClusterSPIFFEID DNS name templates support for external access
- Improved TLS certificate SAN configuration for production deployments

**MCP Enhancements**
- OASF schema exploration tools for AI-assisted record enrichment
- Hierarchical domain and skill navigation capabilities

**Dependencies & Stability**
- OASF SDK upgrade to v0.0.11 with latest schema improvements
- SDK testing fixes for X.509 authentication mode

### Compatibility Matrix

| Component              | Version | Compatible With             |
| ---------------------- | ------- | --------------------------- |
| **dir-apiserver**      | v0.5.1  | oasf v0.3.x, v0.7.x, v0.8.x |
| **dirctl**             | v0.5.1  | dir-apiserver >= v0.5.0     |
| **dir-go**             | v0.5.1  | dir-apiserver >= v0.5.0     |
| **dir-py**             | v0.5.1  | dir-apiserver >= v0.5.0     |
| **dir-js**             | v0.5.1  | dir-apiserver >= v0.5.0     |
| **helm-charts/dir**    | v0.5.1  | dir-apiserver >= v0.5.0     |
| **helm-charts/dirctl** | v0.5.1  | dirctl >= v0.5.0            |

### Added
- **Helm**: External Secrets Operator integration for credential management (#691)
- **Helm**: DNS name templates support for SPIRE ClusterSPIFFEID (#681)
- **MCP**: OASF schema domain and skill exploration tools for enricher workflow (#680)
- **Importer**: OASF SDK translator for MCP Registry conversion with deduplication and debug diagnostics (#624)

### Changed
- **Dependencies**: Bump OASF SDK to v0.0.11 (#679)
- **CI**: Update upload-artifacts version (#682)
- **CI**: Update brew formula version (#684)
- **Docs**: Update readme versions (#685)

### Fixed
- **SDK**: Use X.509 auth mode for testing (#678)

[Full Changelog](https://github.com/agntcy/dir/compare/v0.5.0...v0.5.1)

---

## [v0.5.0] - 2025-11-12

### Key Highlights

This release focuses on extending API functionalities, improving
operational reliability, strengthening security capabilities, and
adding MCP (Model Context Protocol) integrations support, including:

**MCP Integrations**
- MCP registry importer for automated OASF record ingestion
- MCP server implementation with OASF and Directory tools
- Added support for MCP server announce and discovery via DHT

**API & Client Improvements**
- Event listener RPC for real-time updates across services
- gRPC connection management and streaming enhancements
- Rate limiting at application layer for improved stability
- Health checks migrated from HTTP to gRPC

**Security & Reliability**
- Simplified TLS-based authentication support for SDKs
- Panic recovery middleware and structured logging for gRPC
- Critical resource leak fixes and improved context handling
- Enhanced security scanning with CodeQL workflows

**Developer Experience**
- MCP tooling for easy record management and API access
- LLM-based enrichment for OASF records
- Simplified SDK integration in secure environments
- Unified CLI output formats with --output flag and JSONL support

### Compatibility Matrix

| Component              | Version | Compatible With             |
| ---------------------- | ------- | --------------------------- |
| **dir-apiserver**      | v0.5.0  | oasf v0.3.x, v0.7.x, v0.8.x |
| **dirctl**             | v0.5.0  | dir-apiserver >= v0.5.0     |
| **dir-go**             | v0.5.0  | dir-apiserver >= v0.5.0     |
| **dir-py**             | v0.5.0  | dir-apiserver >= v0.5.0     |
| **dir-js**             | v0.5.0  | dir-apiserver >= v0.5.0     |
| **helm-charts/dir**    | v0.5.0  | dir-apiserver >= v0.5.0     |
| **helm-charts/dirctl** | v0.5.0  | dirctl >= v0.5.0            |

### Added
- **MCP**: Add tools for record management and API operations (#465, #574, #619, #611, #650, #660)
- **MCP**: Registry importer for automated record ingestion (#544, #568)
- **MCP**: LLM-based record enrichment for OASF records (#646)
- **API**: Event listener RPC exposure (#537)
- **API**: Structured gRPC request/response logging interceptor (#566)
- **API**: Panic recovery middleware for gRPC handlers (#573)
- **API**: Application-layer rate limiting for gRPC (#593)
- **API**: Readiness checks for apiserver services (#582)
- **API**: gRPC connection management (#647)
- **Security**: Container security scanning workflow (#547)
- **Security**: CodeQL security workflows (#584)
- **Security**: TLS token-based authentication for Go client/CLI (#606)
- **Helm**: Routing service deployment configuration (#599)
- **Helm**: Extra environment variables support in dir chart (#605)
- **Helm**: Zot configuration and authentication (#576)
- **CLI**: Unified output formats with --output flag and JSONL support (#587)
- **CLI**: MCP dirctl subcommand (#660)
- **Testing**: Unit test coverage for all Go modules (#555)
- **Testing**: E2E test coverage (#591)

### Changed
- **API**: Push/pull API refactoring and improvements (#585, #595)
- **Storage**: Store Capability Interfaces refactoring (#562)
- **SDK**: Migration to local generated proto stubs (#569, #588)
- **CLI**: Hub sign/verify commands restoration (#612)
- **Helm**: Ingress deployment fixes (#600, #601)
- **Security**: Security fixes and dependency updates (#602)
- **Health**: Health checks migrated from HTTP to gRPC (#597)
- **Dependencies**: Bump OASF SDK to v0.0.8 and v0.0.9 (#603, #640)
- **Dependencies**: Bump Zot to latest version (#578, #579)
- **Dependencies**: Set SPIRE version in taskfile (#583)
- **CI**: Update container tags for security scans (#558)
- **CI**: Tag for SDK/JS package releases (#617, #621)

### Fixed
- **Client**: Critical resource leaks and context handling issues (#577)
- **Client**: Push stream hanging with multiple errors (#644)
- **Security**: JWT auth test and authentication fixes (#545)
- **API**: Rate limit E2E tests (#598)
- **API**: Hub API updates (#595)
- **API**: MCP search limit handling (#623)
- **Storage**: OCI E2E concurrent issues and healthcheck service (#620)
- **SDK**: SPIFFE sign test (#592)
- **SDK**: Add proto stubs to repository (#588)
- **SDK**: JS package release prefix for RC tags (#621, #625)
- **CLI**: Sign and verify options cleanup (#673)
- **CLI**: API key methods improvements (#659)

### Removed
- **MCP**: Remove dedicated MCP artifacts (#663)

[Full Changelog](https://github.com/agntcy/dir/compare/v0.4.0...v0.5.0)

---

## [v0.4.0] - 2025-10-15

### Key Highlights

This release delivers improvements to application layer via generic referrers,
security schema with JWT/X.509 support, network capabilities, operational stability, 
and developer experience, with a focus on:

**Security & Authentication**
- Added JWT/X.509 over TLS versus mTLS for flexible authentication
- Authenticated PeerID integration with libp2p transport
- Secure credential management for zot sync operations
- OCI 1.1 referrers specification migration for signature attachments

**Networking & Connectivity**
- GossipSub implementation for efficient label announcements
- Connection Manager implementation with removal of custom peer discovery
- Improved routing search with better peer address handling
- Locator-based record search capabilities across the network

**Generic Record Referrers**
- Support for storage and handling of generic record referrers
- Referrer object encoding across server and SDK components
- Application support via referrer (e.g., signatures, attestations)

**Developer Experience & Tooling**
- Streaming package for Golang gRPC client functionality
- Standardized CLI output formats and command improvements
- Reusable setup-dirctl GitHub Action for CI/CD workflows

**Quality & Stability Improvements**
- Comprehensive test suite enhancements including SPIRE tests
- E2E network test stability with environment warm-up phases
- Bug fixes for API key formatting, file permissions, and documentation

### Compatibility Matrix

| Component              | Version | Compatible With                              |
| ---------------------- | ------- | -------------------------------------------- |
| **dir-apiserver**      | v0.4.0  | oasf v0.3.x, v0.7.x                          |
| **dirctl**             | v0.4.0  | dir-apiserver >= v0.3.0, oasf v0.3.x, v0.7.x |
| **dir-go**             | v0.4.0  | dir-apiserver >= v0.3.0, oasf v0.3.x, v0.7.x |
| **dir-py**             | v0.4.0  | dir-apiserver >= v0.3.0, oasf v0.3.x, v0.7.x |
| **dir-js**             | v0.4.0  | dir-apiserver >= v0.3.0, oasf v0.3.x, v0.7.x |
| **helm-charts/dir**    | v0.4.0  | dir-apiserver >= v0.3.0                      |
| **helm-charts/dirctl** | v0.4.0  | dirctl >= v0.3.0                             |

### Added
- **Networking**: GossipSub for efficient label announcements (#472)
- **Networking**: AutoRelay, Hole Punching, and mDNS for better peer connectivity (#503)
- **Networking**: Connection Manager implementation (#495)
- **Security**: JWT authentication and TLS communication support (#492)
- **Security**: Authenticated PeerID from libp2p transport (#502)
- **Storage**: Generic record referrers support (#451, #480)
- **Storage**: Referrer encoding capabilities (#491, #499)
- **API**: Server validator functionality (#456)
- **SDK**: Streaming package for gRPC client (#527)
- **CI**: Zot dependency for server config (#444)
- **CI**: Reusable setup-dirctl GitHub Action (#441)
- **CI**: SPIRE tests action (#488)
- **CI**: Local tap for testing homebrew formulae (#437)

### Changed
- **Security**: Rename mTLS to x.509 for clarity (#508)
- **Storage**: Secure credential management for zot sync operations (#457)
- **API**: Unify extensions to modules architecture (#463)
- **CLI**: Standardize CLI output formats (#509)
- **CLI**: Update search command to use routing search command flags (#521)
- **Docs**: Update SDK source and release links (#434)
- **CI**: Improvements to pipeline and taskfile setup (#438, #442)
- **CI**: Bump Go, golangci-lint and libp2p (#534)
- **CI**: Bump OASF SDK to versions 0.0.6 and 0.0.7 (#481, #489)
- **CI**: Bump brew tap to v0.3.0 (#435)

### Fixed
- **Networking**: Empty peer addresses in routing search results (#513)
- **Networking**: Locator-based remote record search (#469)
- **Networking**: E2E network test flakiness with environment warm-up (#516)
- **Security**: Auth mode mTLS issues (#517)
- **Security**: Cosign signature attachment migration to OCI 1.1 referrers spec (#464)
- **Security**: Signature verification against expected payload (#493)
- **Storage**: Demo testdata and flaky test removal (#504)
- **SDK**: JavaScript package release issues (#432)
- **CLI**: API key list format (#494)
- **Docs**: Remove broken links from README (#458)
- **Docs**: Fix push and pull documentation (#436)
- **CI**: Support multiple federation artifacts via charts (#519)
- **CI**: Enable push to buf registry on merge (#484)
- **CI**: Bug report template to allow version input (#460)
- **CI**: File permissions for hub/api/v1alpha1/* (#518)

### Removed
- **Networking**: Custom peer discovery (replaced with Connection Manager) (#495)
- **CI**: Useless files in hub/ directory (#522)

[Full Changelog](https://github.com/agntcy/dir/compare/v0.3.0...v0.4.0)

---

## [v0.3.0] - 2025-09-19

### Key Highlights

This release delivers foundational improvements to security schema,
storage services, network capabilities, and user/developer experience,
with a focus on:

**Zero-Trust Security Architecture**
- X.509-based SPIFFE/SPIRE identity framework with federation support
- Policy-based authorization framework with fine-grained access controls
- Secure mTLS communication across all services
- OCI-native PKI with client- and server-side verification capabilities

**Content Standardization**
- Unified Core v1 API with multi-version support for OASF objects
- Deterministic CID generation using canonical JSON marshaling
- Cross-language and service consistency with CIDv1 record addressing
- OCI-native object storage and relationship management

**Search & Discovery**
- Local search with wildcard and pattern matching support
- Network-wide record discovery with prefix-based search capabilities  
- DHT-based routing for distributed service announcement and discovery

**Data Synchronization**
- Full and partial index synchronization with CID selection
- Automated sync workflows for simple data migration and replication
- Post-sync verification checks and search capabilities across records

**Developer Experience**
- Native Client SDKs for Golang, JavaScript, TypeScript, and Python
- Standardized CLI and SDK tooling with consistent interfaces
- Decoupled signing workflows for easier usage and integration
- Kubernetes deployment with SPIRE and Federation support

### Compatibility Matrix

The following matrix shows compatibility between different component versions:

| Core Component    | Version | Compatible With                                |
| ----------------- | ------- | ---------------------------------------------- |
| **dir-apiserver** | v0.3.0  | oasf v0.3.x, oasf v0.7.x                       |
| **dirctl**        | v0.3.0  | dir-apiserver v0.3.0, oasf v0.3.x, oasf v0.7.x |
| **dir-go**        | v0.3.0  | dir-apiserver v0.3.0, oasf v0.3.x, oasf v0.7.x |
| **dir-py**        | v0.3.0  | dir-apiserver v0.3.0, oasf v0.3.x, oasf v0.7.x |
| **dir-js**        | v0.3.0  | dir-apiserver v0.3.0, oasf v0.3.x, oasf v0.7.x |

#### Helm Chart Compatibility

| Helm Chart                 | Version | Deploys Component    | Minimum Requirements |
| -------------------------- | ------- | -------------------- | -------------------- |
| **dir/helm-charts/dir**    | v0.3.0  | dir-apiserver v0.3.0 | Kubernetes 1.20+     |
| **dir/helm-charts/dirctl** | v0.3.0  | dirctl v0.3.0        | Kubernetes 1.20+     |

#### Compatibility Notes

- **Full OASF support** is available across all core components
- **dir-apiserver v0.3.0** introduces breaking changes to the API layer
- **dirctl v0.3.0** introduces breaking changes to the CLI usage
- **dir-go v0.3.0** introduces breaking changes to the SDK usage
- Older versions of **dir-apiserver** are **not compatible** with **dir-apiserver v0.3.0**
- Older versions of client components are **not compatible** with **dir-apiserver v0.3.0**
- Older versions of helm charts are **not compatible** with **dir-apiserver v0.3.0**
- Data must be manually migrated from older **dir-apiserver** versions to **dir-apiserver v0.3.0**

#### Migration Guide

Data from the OCI storage layer in the Directory can be migrated by repushing via new API endpoints.
For example:

```bash
repo=localhost:5000/dir
for tag in $(oras repo tags $repo); do
    digest=$(oras resolve $repo:$tag)
    oras blob fetch --output - $repo@$digest | dirctl push --stdin
done
```

### Added
- **API**: Implement Search API for network-wide record discovery using RecordQuery interface (#362)
- **API**: Add initial authorization framework (#330)
- **API**: Add distributed label-based announce/discovery via DHT (#285)
- **API**: Add wildcard search support with pattern matching (#355)
- **API**: Add max replicasets to keep in deployment (#207)
- **API**: Add sync API (#199)
- **CI**: Add Codecov workflow & docs (#380)
- **CI**: Introduce BSR (#212)
- **SDK**: Add SDK release process (#216)
- **SDK**: Add more gRPC services (#294)
- **SDK**: Add gRPC client code and example for JavaScript SDK (#248)
- **SDK**: Add sync support (#361)
- **SDK**: Add sign and verification (#337)
- **SDK**: Add testing solution for CI (#269)
- **SDK**: Standardize Python SDK tooling for Directory (#371)
- **SDK**: Add TypeScript/JavaScript DIR Client SDK (#407)
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
- **Storage**: Add cid-based synchronisation support (#401)
- **Storage**: Add rich manifest annotations (#236)

### Changed
- **API**: Switch to generic OASF objects across codebase (#381)
- **API**: Version upgrade of API services (#225)
- **API**: Update sync API and add credential RPC (#217)
- **API**: Refactor domain interfaces to align with OASF schema (#397)
- **API**: Rename v1alpha2 to v1 (#258)
- **CI**: Find better place for proto APIs (#384)
- **CI**: Reduce flaky jobs for SDK (#339)
- **CI**: Update codebase with proto namespace changes (#398)
- **CI**: Update CI task gen to ignore buf lock file changes (#275)
- **CI**: Update brew formula version (#372, #263, #257, #247)
- **CI**: Bump Go (#221)
- **CI**: Update Directory proto imports for SDKs (#421)
- **CI**: Bump OASF SDK version to v0.0.5 (#424)
- **Documentation**: Update usage documentation for record generation (#287)
- **Documentation**: Add and update README for refactored SDKs (#273)
- **Documentation**: Update README to reflect new usage documentation link and remove USAGE.md file (#332)
- **Documentation**: Update documentation setup (#394)
- **SDK**: Move and refactor Python SDK code (#229)
- **SDK**: Bump package versions for release (#274)
- **SDK**: Bump versions for release (#249)
- **SDK**: Support streams & update docs (#284)
- **SDK**: Update API code and add example code for Python SDK (#237)
- **Storage**: Migrate record signature to OCI native signature (#250)
- **Storage**: Store implementations and digest/CID calculation (#238)
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

### Fixed
- **API**: Resolve buf proto API namespacing issues (#393)
- **API**: Add sync testdata (#396)
- **API**: Update apiserver.env to use new config values (#406)
- **API**: Suppress command usage display on runtime errors (#290)
- **API**: Quick-fix for e2e CLI cmd state handling (#270)
- **API**: Fix/CI task gen (#271)
- **CI**: Allow dir-hub-maintainers release (#402)
- **SDK**: Fix Python SDK imports and tests (#403)
- **SDK**: Fix codeowners file (#404)
- **SDK**: Flaky SDK CICD tests (#422)
- **Storage**: Add separate maintainers for hub CLI directory (#375)
- **Storage**: Update agent directory default location (#226)
- **Storage**: Flaky e2e test and restructure test suites (#416)
- **Storage**: E2E sync test cleanup (#423)

[Full Changelog](https://github.com/agntcy/dir/compare/v0.2.13...v0.3.0)

---

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
