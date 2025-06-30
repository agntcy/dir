module main

go 1.24.1

require (
	github.com/agntcy/dir/api v0.2.6
	github.com/agntcy/dir/server v0.2.6
	github.com/opencontainers/go-digest v1.0.0
	github.com/opencontainers/image-spec v1.1.1
	oras.land/oras-go/v2 v2.5.0
)

require (
	github.com/ipfs/go-cid v0.5.0 // indirect
	github.com/klauspost/cpuid/v2 v2.2.10 // indirect
	github.com/minio/sha256-simd v1.0.1 // indirect
	github.com/mr-tron/base58 v1.2.0 // indirect
	github.com/multiformats/go-base32 v0.1.0 // indirect
	github.com/multiformats/go-base36 v0.2.0 // indirect
	github.com/multiformats/go-multibase v0.2.0 // indirect
	github.com/multiformats/go-multihash v0.2.3 // indirect
	github.com/multiformats/go-varint v0.0.7 // indirect
	github.com/spaolacci/murmur3 v1.1.0 // indirect
	golang.org/x/crypto v0.39.0 // indirect
	golang.org/x/sync v0.15.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	google.golang.org/protobuf v1.36.6 // indirect
	lukechampine.com/blake3 v1.4.0 // indirect
)

replace (
	github.com/agntcy/dir/api => ../../api
	github.com/agntcy/dir/server => ../../server
	github.com/agntcy/dir/utils => ../../utils
	github.com/agntcy/dir/poc/sync/utils => ./utils
)
