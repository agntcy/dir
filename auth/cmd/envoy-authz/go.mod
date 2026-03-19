module github.com/agntcy/dir/auth/cmd/envoy-authz

go 1.26.1

// Replace local modules
replace (
	github.com/agntcy/dir/auth/authprovider => ../../authprovider
	github.com/agntcy/dir/auth/authzserver => ../../authzserver
)

require (
	github.com/agntcy/dir/auth/authprovider v1.0.0
	github.com/agntcy/dir/auth/authzserver v1.0.0
	github.com/envoyproxy/go-control-plane/envoy v1.37.0
	google.golang.org/grpc v1.79.3
	gopkg.in/yaml.v3 v3.0.1
)

require (
	cel.dev/expr v0.25.1 // indirect
	github.com/ProtonMail/go-crypto v1.4.1 // indirect
	github.com/bmatcuk/doublestar/v4 v4.10.0 // indirect
	github.com/casbin/casbin/v2 v2.135.0 // indirect
	github.com/casbin/govaluate v1.10.0 // indirect
	github.com/cloudflare/circl v1.6.3 // indirect
	github.com/cncf/xds/go v0.0.0-20260202195803-dba9d589def2 // indirect
	github.com/envoyproxy/go-control-plane v0.14.0 // indirect
	github.com/envoyproxy/protoc-gen-validate v1.3.3 // indirect
	github.com/google/go-github/v50 v50.2.0 // indirect
	github.com/google/go-querystring v1.2.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/iancoleman/strcase v0.3.0 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/lyft/protoc-gen-star/v2 v2.0.4 // indirect
	github.com/planetscale/vtprotobuf v0.6.1-0.20240319094008-0393e58bdf10 // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	github.com/spf13/afero v1.15.0 // indirect
	golang.org/x/crypto v0.49.0 // indirect
	golang.org/x/mod v0.33.0 // indirect
	golang.org/x/net v0.52.0 // indirect
	golang.org/x/oauth2 v0.36.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/text v0.35.0 // indirect
	golang.org/x/tools v0.42.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20251202230838-ff82c1b0f217 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260316180232-0b37fe3546d5 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
)
