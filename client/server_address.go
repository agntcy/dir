// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// withServerAddress names the dialled address in an Unavailable error. A connection
// error that reached a listener speaking another protocol only describes the protocol
// mix-up, and the address comes from a context, an environment variable or a flag,
// so it is not always the one the operator has in mind. The code and details are kept.
func withServerAddress(err error, address string) error {
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.Unavailable {
		return err
	}

	proto := st.Proto()
	proto.Message = fmt.Sprintf("%s (server %s)", proto.GetMessage(), address)

	return status.ErrorProto(proto) //nolint:wrapcheck // stays a gRPC status error so callers can read its code
}

func serverAddressUnaryInterceptor(address string) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		return withServerAddress(invoker(ctx, method, req, reply, cc, opts...), address)
	}
}

func serverAddressStreamInterceptor(address string) grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		stream, err := streamer(ctx, desc, cc, method, opts...)
		if err != nil {
			return nil, withServerAddress(err, address)
		}

		return &serverAddressStream{ClientStream: stream, address: address}, nil
	}
}

// serverAddressStream names the address in errors from a stream that loses its
// connection after it was opened.
type serverAddressStream struct {
	grpc.ClientStream

	address string
}

func (s *serverAddressStream) SendMsg(m any) error {
	return withServerAddress(s.ClientStream.SendMsg(m), s.address)
}

func (s *serverAddressStream) RecvMsg(m any) error {
	return withServerAddress(s.ClientStream.RecvMsg(m), s.address)
}
