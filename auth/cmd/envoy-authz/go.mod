module github.com/agntcy/dir/auth/cmd/envoy-authz

go 1.25.6

require (
	github.com/agntcy/dir/auth/authprovider v0.0.0
	github.com/agntcy/dir/auth/authzserver v0.0.0
	github.com/envoyproxy/go-control-plane/envoy v1.36.0
	google.golang.org/grpc v1.78.0
)

require (
	github.com/ProtonMail/go-crypto v1.3.0 // indirect
	github.com/cloudflare/circl v1.6.2 // indirect
	github.com/cncf/xds/go v0.0.0-20260121142036-a486691bba94 // indirect
	github.com/envoyproxy/protoc-gen-validate v1.3.0 // indirect
	github.com/google/go-github/v50 v50.2.0 // indirect
	github.com/google/go-querystring v1.2.0 // indirect
	github.com/planetscale/vtprotobuf v0.6.1-0.20240319094008-0393e58bdf10 // indirect
	golang.org/x/crypto v0.47.0 // indirect
	golang.org/x/net v0.49.0 // indirect
	golang.org/x/oauth2 v0.34.0 // indirect
	golang.org/x/sys v0.40.0 // indirect
	golang.org/x/text v0.33.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260120221211-b8f7ae30c516 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

replace (
	github.com/agntcy/dir/auth/authprovider => ../../authprovider
	github.com/agntcy/dir/auth/authzserver => ../../authzserver
)
