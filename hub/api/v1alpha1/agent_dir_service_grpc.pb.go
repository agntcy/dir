// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             (unknown)
// source: saas/v1alpha1/agent_dir_service.proto

package v1alpha1

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.64.0 or later.
const _ = grpc.SupportPackageIsVersion9

const (
	AgentDirService_PushAgent_FullMethodName = "/saas.v1alpha1.AgentDirService/PushAgent"
	AgentDirService_PullAgent_FullMethodName = "/saas.v1alpha1.AgentDirService/PullAgent"
)

// AgentDirServiceClient is the client API for AgentDirService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
//
// This API is manily for CLIs and the implementation of these APIs should communicate with the Agent Directory
type AgentDirServiceClient interface {
	PushAgent(ctx context.Context, opts ...grpc.CallOption) (grpc.ClientStreamingClient[PushAgentRequest, PushAgentResponse], error)
	PullAgent(ctx context.Context, in *PullAgentRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[PullAgentResponse], error)
}

type agentDirServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewAgentDirServiceClient(cc grpc.ClientConnInterface) AgentDirServiceClient {
	return &agentDirServiceClient{cc}
}

func (c *agentDirServiceClient) PushAgent(ctx context.Context, opts ...grpc.CallOption) (grpc.ClientStreamingClient[PushAgentRequest, PushAgentResponse], error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	stream, err := c.cc.NewStream(ctx, &AgentDirService_ServiceDesc.Streams[0], AgentDirService_PushAgent_FullMethodName, cOpts...)
	if err != nil {
		return nil, err
	}
	x := &grpc.GenericClientStream[PushAgentRequest, PushAgentResponse]{ClientStream: stream}
	return x, nil
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type AgentDirService_PushAgentClient = grpc.ClientStreamingClient[PushAgentRequest, PushAgentResponse]

func (c *agentDirServiceClient) PullAgent(ctx context.Context, in *PullAgentRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[PullAgentResponse], error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	stream, err := c.cc.NewStream(ctx, &AgentDirService_ServiceDesc.Streams[1], AgentDirService_PullAgent_FullMethodName, cOpts...)
	if err != nil {
		return nil, err
	}
	x := &grpc.GenericClientStream[PullAgentRequest, PullAgentResponse]{ClientStream: stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type AgentDirService_PullAgentClient = grpc.ServerStreamingClient[PullAgentResponse]

// AgentDirServiceServer is the server API for AgentDirService service.
// All implementations must embed UnimplementedAgentDirServiceServer
// for forward compatibility.
//
// This API is manily for CLIs and the implementation of these APIs should communicate with the Agent Directory
type AgentDirServiceServer interface {
	PushAgent(grpc.ClientStreamingServer[PushAgentRequest, PushAgentResponse]) error
	PullAgent(*PullAgentRequest, grpc.ServerStreamingServer[PullAgentResponse]) error
	mustEmbedUnimplementedAgentDirServiceServer()
}

// UnimplementedAgentDirServiceServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedAgentDirServiceServer struct{}

func (UnimplementedAgentDirServiceServer) PushAgent(grpc.ClientStreamingServer[PushAgentRequest, PushAgentResponse]) error {
	return status.Errorf(codes.Unimplemented, "method PushAgent not implemented")
}
func (UnimplementedAgentDirServiceServer) PullAgent(*PullAgentRequest, grpc.ServerStreamingServer[PullAgentResponse]) error {
	return status.Errorf(codes.Unimplemented, "method PullAgent not implemented")
}
func (UnimplementedAgentDirServiceServer) mustEmbedUnimplementedAgentDirServiceServer() {}
func (UnimplementedAgentDirServiceServer) testEmbeddedByValue()                         {}

// UnsafeAgentDirServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to AgentDirServiceServer will
// result in compilation errors.
type UnsafeAgentDirServiceServer interface {
	mustEmbedUnimplementedAgentDirServiceServer()
}

func RegisterAgentDirServiceServer(s grpc.ServiceRegistrar, srv AgentDirServiceServer) {
	// If the following call pancis, it indicates UnimplementedAgentDirServiceServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&AgentDirService_ServiceDesc, srv)
}

func _AgentDirService_PushAgent_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(AgentDirServiceServer).PushAgent(&grpc.GenericServerStream[PushAgentRequest, PushAgentResponse]{ServerStream: stream})
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type AgentDirService_PushAgentServer = grpc.ClientStreamingServer[PushAgentRequest, PushAgentResponse]

func _AgentDirService_PullAgent_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(PullAgentRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(AgentDirServiceServer).PullAgent(m, &grpc.GenericServerStream[PullAgentRequest, PullAgentResponse]{ServerStream: stream})
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type AgentDirService_PullAgentServer = grpc.ServerStreamingServer[PullAgentResponse]

// AgentDirService_ServiceDesc is the grpc.ServiceDesc for AgentDirService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var AgentDirService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "saas.v1alpha1.AgentDirService",
	HandlerType: (*AgentDirServiceServer)(nil),
	Methods:     []grpc.MethodDesc{},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "PushAgent",
			Handler:       _AgentDirService_PushAgent_Handler,
			ClientStreams: true,
		},
		{
			StreamName:    "PullAgent",
			Handler:       _AgentDirService_PullAgent_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "saas/v1alpha1/agent_dir_service.proto",
}
