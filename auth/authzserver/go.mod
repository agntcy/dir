module github.com/agntcy/dir/auth/authzserver

go 1.25.6

require (
	github.com/agntcy/dir/auth/authprovider v1.0.0-rc.3
	github.com/casbin/casbin/v2 v2.135.0
	github.com/envoyproxy/go-control-plane/envoy v1.36.0
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260120221211-b8f7ae30c516
	google.golang.org/grpc v1.78.0
)

require (
	github.com/bmatcuk/doublestar/v4 v4.6.1 // indirect
	github.com/casbin/govaluate v1.3.0 // indirect
	github.com/cncf/xds/go v0.0.0-20260121142036-a486691bba94 // indirect
	github.com/envoyproxy/protoc-gen-validate v1.3.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/planetscale/vtprotobuf v0.6.1-0.20240319094008-0393e58bdf10 // indirect
	golang.org/x/net v0.49.0 // indirect
	golang.org/x/sys v0.40.0 // indirect
	golang.org/x/text v0.33.0 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

replace github.com/agntcy/dir/auth/authprovider => ../authprovider
