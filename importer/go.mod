module github.com/agntcy/dir/importer

go 1.26.2

// Replace local modules
replace (
	github.com/agntcy/dir/api => ../api
	github.com/agntcy/dir/client => ../client
	github.com/agntcy/dir/runtime/api => ../runtime/api
	github.com/agntcy/dir/utils => ../utils
)

// Cosign does not updated the crypto11 owner
replace github.com/ThalesIgnite/crypto11 => github.com/ThalesGroup/crypto11 v1.6.0

require (
	buf.build/gen/go/agntcy/oasf/protocolbuffers/go v1.36.11-20260416152818-3df7657b1c83.1
	github.com/agntcy/dir/api v1.2.0
	github.com/agntcy/dir/client v1.2.0
	github.com/agntcy/dir/utils v1.2.0
	github.com/agntcy/oasf-sdk/pkg v1.0.5
	github.com/modelcontextprotocol/registry v1.6.0
	github.com/sashabaranov/go-openai v1.41.2
	google.golang.org/protobuf v1.36.11
)

require (
	buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go v1.36.11-20260415201107-50325440f8f2.1 // indirect
	github.com/Masterminds/semver/v3 v3.4.0 // indirect
	github.com/google/jsonschema-go v0.4.2 // indirect
)

require (
	buf.build/gen/go/agntcy/oasf-sdk/protocolbuffers/go v1.36.11-20260416081642-09171af1ac1a.1 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/go-viper/mapstructure/v2 v2.5.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/ipfs/go-cid v0.6.1 // indirect
	github.com/klauspost/cpuid/v2 v2.3.0 // indirect
	github.com/mark3labs/mcp-go v0.47.1
	github.com/minio/sha256-simd v1.0.1 // indirect
	github.com/mitchellh/mapstructure v1.5.1-0.20231216201459-8508981c8b6c // indirect
	github.com/mr-tron/base58 v1.3.0 // indirect
	github.com/multiformats/go-base32 v0.1.0 // indirect
	github.com/multiformats/go-base36 v0.2.0 // indirect
	github.com/multiformats/go-multibase v0.3.0 // indirect
	github.com/multiformats/go-multihash v0.2.3 // indirect
	github.com/multiformats/go-varint v0.1.0 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/pelletier/go-toml/v2 v2.3.0 // indirect
	github.com/sagikazarmark/locafero v0.12.0 // indirect
	github.com/spaolacci/murmur3 v1.1.0 // indirect
	github.com/spf13/afero v1.15.0 // indirect
	github.com/spf13/cast v1.10.0 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	github.com/spf13/viper v1.21.0 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/yosida95/uritemplate/v3 v3.0.2 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/crypto v0.50.0 // indirect
	golang.org/x/net v0.53.0 // indirect
	golang.org/x/sys v0.43.0 // indirect
	golang.org/x/text v0.36.0 // indirect
	golang.org/x/time v0.15.0
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260406210006-6f92a3bedf2d // indirect
	google.golang.org/grpc v1.80.0 // indirect
	gopkg.in/yaml.v3 v3.0.1
	lukechampine.com/blake3 v1.4.1 // indirect
)
