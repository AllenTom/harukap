package rpc

import (
	"google.golang.org/grpc"
)

type HarukaRPCService struct {
	service    interface{}
	OnRegister func(rpcServer *grpc.Server)
}

func NewHarukaRPCService(service interface{}, OnRegister func(rpcServer *grpc.Server)) *HarukaRPCService {
	return &HarukaRPCService{
		service:    service,
		OnRegister: OnRegister,
	}
}
