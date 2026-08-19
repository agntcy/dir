module github.com/agntcy/dir/utils

go 1.26.5

require (
	buf.build/gen/go/agntcy/oasf-sdk/grpc/go v1.6.2-20260811133823-5281d0c487b5.1
	buf.build/gen/go/agntcy/oasf-sdk/protocolbuffers/go v1.36.12-20260811133823-5281d0c487b5.1
	buf.build/gen/go/agntcy/oasf/protocolbuffers/go v1.36.12-20260721113505-cf7db8888586.1
	github.com/agntcy/dir/api v1.7.0
	github.com/agntcy/oasf-sdk/pkg v1.1.0
	github.com/mitchellh/mapstructure v1.5.1-0.20231216201459-8508981c8b6c
	github.com/rs/zerolog v1.35.1
	github.com/spf13/viper v1.21.0
	github.com/spiffe/go-spiffe/v2 v2.8.1
	github.com/stretchr/testify v1.12.0
	google.golang.org/grpc v1.82.1
	google.golang.org/protobuf v1.36.12
)

require (
	buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go v1.36.12-20260709200747-435963d16310.1 // indirect
	github.com/Masterminds/semver/v3 v3.5.0 // indirect
	github.com/dlclark/regexp2 v1.4.0 // indirect
	github.com/fsnotify/fsnotify v1.10.1 // indirect
	github.com/go-viper/mapstructure/v2 v2.5.0 // indirect
	github.com/google/flatbuffers v23.5.26+incompatible // indirect
	github.com/ipfs/go-cid v0.6.2 // indirect
	github.com/klauspost/cpuid/v2 v2.3.0 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/minio/sha256-simd v1.0.1 // indirect
	github.com/mr-tron/base58 v1.3.0 // indirect
	github.com/multiformats/go-base32 v0.1.0 // indirect
	github.com/multiformats/go-base36 v0.2.0 // indirect
	github.com/multiformats/go-multibase v0.3.0 // indirect
	github.com/multiformats/go-multihash v0.2.3 // indirect
	github.com/multiformats/go-varint v0.1.0 // indirect
	github.com/nlpodyssey/cybertron v0.2.1 // indirect
	github.com/nlpodyssey/gopickle v0.2.0 // indirect
	github.com/nlpodyssey/gotokenizers v0.2.0 // indirect
	github.com/nlpodyssey/spago v1.1.0 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/pelletier/go-toml/v2 v2.3.1 // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	github.com/sagikazarmark/locafero v0.12.0 // indirect
	github.com/spaolacci/murmur3 v1.1.0 // indirect
	github.com/spf13/afero v1.15.0 // indirect
	github.com/spf13/cast v1.10.0 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/crypto v0.53.0 // indirect
	golang.org/x/net v0.56.0 // indirect
	golang.org/x/sync v0.21.0 // indirect
	golang.org/x/sys v0.46.0 // indirect
	golang.org/x/text v0.39.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260618152121-87f3d3e198d3 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	lukechampine.com/blake3 v1.4.1 // indirect
)

replace github.com/agntcy/dir/api => ../api
