module github.com/agntcy/dir/cmd/envoy-authz

go 1.25.2

require (
	github.com/agntcy/dir/pkg/authprovider v0.0.0
	github.com/agntcy/dir/pkg/authzserver v0.0.0
	github.com/envoyproxy/go-control-plane/envoy v1.32.2
	google.golang.org/grpc v1.69.2
)

require (
	github.com/ProtonMail/go-crypto v0.0.0-20230217124315-7d5c6f04bbb8 // indirect
	github.com/cloudflare/circl v1.1.0 // indirect
	github.com/cncf/xds/go v0.0.0-20240905190251-b4127c9b8d78 // indirect
	github.com/envoyproxy/protoc-gen-validate v1.1.0 // indirect
	github.com/google/go-github/v50 v50.2.0 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/planetscale/vtprotobuf v0.6.1-0.20240319094008-0393e58bdf10 // indirect
	golang.org/x/crypto v0.30.0 // indirect
	golang.org/x/net v0.30.0 // indirect
	golang.org/x/oauth2 v0.24.0 // indirect
	golang.org/x/sys v0.28.0 // indirect
	golang.org/x/text v0.21.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20241230172942-26aa7a208def // indirect
	google.golang.org/protobuf v1.36.1 // indirect
)

replace (
	github.com/agntcy/dir/pkg/authprovider => ../../pkg/authprovider
	github.com/agntcy/dir/pkg/authzserver => ../../pkg/authzserver
)
