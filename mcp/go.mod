module github.com/agntcy/dir/mcp

go 1.24.5

toolchain go1.24.8

require github.com/modelcontextprotocol/go-sdk v0.8.0

require (
	buf.build/gen/go/agntcy/oasf-sdk/protocolbuffers/go v1.36.9-20250917120021-8b2bf93bf8dc.1 // indirect
	buf.build/gen/go/agntcy/oasf/protocolbuffers/go v1.36.9-20250917090956-ba2d05f62118.1 // indirect
	github.com/agntcy/oasf-sdk/pkg v0.0.7 // indirect
	github.com/ipfs/go-cid v0.5.0 // indirect
	github.com/klauspost/cpuid/v2 v2.2.10 // indirect
	github.com/minio/sha256-simd v1.0.1 // indirect
	github.com/mr-tron/base58 v1.2.0 // indirect
	github.com/multiformats/go-base32 v0.1.0 // indirect
	github.com/multiformats/go-base36 v0.2.0 // indirect
	github.com/multiformats/go-multibase v0.2.0 // indirect
	github.com/multiformats/go-multihash v0.2.3 // indirect
	github.com/multiformats/go-varint v0.0.7 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/spaolacci/murmur3 v1.1.0 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	github.com/xeipuuv/gojsonschema v1.2.0 // indirect
	golang.org/x/crypto v0.38.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	google.golang.org/protobuf v1.36.9 // indirect
	lukechampine.com/blake3 v1.4.0 // indirect
)

require (
	github.com/agntcy/dir/api v0.0.0
	github.com/google/jsonschema-go v0.3.0 // indirect
	github.com/yosida95/uritemplate/v3 v3.0.2 // indirect
)

replace github.com/agntcy/dir/api => ../api
