module github.com/agntcy/dir/api

go 1.24.5

require (
	buf.build/gen/go/agntcy/oasf-sdk/protocolbuffers/go v1.36.8-20250909112931-52543f366c53.1
	buf.build/gen/go/agntcy/oasf/protocolbuffers/go v1.36.8-20250909115813-c8b395e06aba.1
	github.com/agntcy/oasf-sdk/core v0.0.0-20250909144735-c3026c591d1b
	github.com/agntcy/oasf-sdk/validation v0.0.0-20250909144735-c3026c591d1b
	github.com/multiformats/go-multihash v0.2.3
	github.com/opencontainers/go-digest v1.0.0
	github.com/stretchr/testify v1.10.0
	google.golang.org/grpc v1.74.2
	google.golang.org/protobuf v1.36.8
)

require (
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/klauspost/cpuid/v2 v2.2.10 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/minio/sha256-simd v1.0.1 // indirect
	github.com/mr-tron/base58 v1.2.0 // indirect
	github.com/multiformats/go-base32 v0.1.0 // indirect
	github.com/multiformats/go-base36 v0.2.0 // indirect
	github.com/multiformats/go-multibase v0.2.0 // indirect
	github.com/multiformats/go-varint v0.0.7 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/spaolacci/murmur3 v1.1.0 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20180127040702-4e3ac2762d5f // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	github.com/xeipuuv/gojsonschema v1.2.0 // indirect
	golang.org/x/crypto v0.38.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	lukechampine.com/blake3 v1.4.0 // indirect
)

require (
	github.com/ipfs/go-cid v0.5.0
	golang.org/x/net v0.40.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/text v0.26.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250528174236-200df99c418a // indirect
)

replace github.com/agntcy/dir/server => ../server
