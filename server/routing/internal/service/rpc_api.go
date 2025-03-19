package service

import "context"

const (
	EchoService         = "EchoRPCAPI"
	EchoServiceFuncEcho = "Echo"
)

type EchoRPCAPI struct {
	service *Service
}

type Envelope struct {
	Message string
}

func (r *EchoRPCAPI) Echo(ctx context.Context, in Envelope, out *Envelope) error {
	*out = r.service.ReceiveEcho(in)
	return nil
}
