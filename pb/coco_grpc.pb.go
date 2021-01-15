// Code generated by protoc-gen-go-grpc. DO NOT EDIT.

package pb

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion7

// WorkerClient is the client API for Worker service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type WorkerClient interface {
	Run(ctx context.Context, in *RunRequest, opts ...grpc.CallOption) (*RunResponse, error)
	Cancel(ctx context.Context, in *CancelRequest, opts ...grpc.CallOption) (*CancelResponse, error)
}

type workerClient struct {
	cc grpc.ClientConnInterface
}

func NewWorkerClient(cc grpc.ClientConnInterface) WorkerClient {
	return &workerClient{cc}
}

func (c *workerClient) Run(ctx context.Context, in *RunRequest, opts ...grpc.CallOption) (*RunResponse, error) {
	out := new(RunResponse)
	err := c.cc.Invoke(ctx, "/pb.Worker/Run", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *workerClient) Cancel(ctx context.Context, in *CancelRequest, opts ...grpc.CallOption) (*CancelResponse, error) {
	out := new(CancelResponse)
	err := c.cc.Invoke(ctx, "/pb.Worker/Cancel", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// WorkerServer is the server API for Worker service.
// All implementations must embed UnimplementedWorkerServer
// for forward compatibility
type WorkerServer interface {
	Run(context.Context, *RunRequest) (*RunResponse, error)
	Cancel(context.Context, *CancelRequest) (*CancelResponse, error)
	mustEmbedUnimplementedWorkerServer()
}

// UnimplementedWorkerServer must be embedded to have forward compatible implementations.
type UnimplementedWorkerServer struct {
}

func (UnimplementedWorkerServer) Run(context.Context, *RunRequest) (*RunResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Run not implemented")
}
func (UnimplementedWorkerServer) Cancel(context.Context, *CancelRequest) (*CancelResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Cancel not implemented")
}
func (UnimplementedWorkerServer) mustEmbedUnimplementedWorkerServer() {}

// UnsafeWorkerServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to WorkerServer will
// result in compilation errors.
type UnsafeWorkerServer interface {
	mustEmbedUnimplementedWorkerServer()
}

func RegisterWorkerServer(s grpc.ServiceRegistrar, srv WorkerServer) {
	s.RegisterService(&_Worker_serviceDesc, srv)
}

func _Worker_Run_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RunRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WorkerServer).Run(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/pb.Worker/Run",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WorkerServer).Run(ctx, req.(*RunRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Worker_Cancel_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CancelRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WorkerServer).Cancel(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/pb.Worker/Cancel",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WorkerServer).Cancel(ctx, req.(*CancelRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _Worker_serviceDesc = grpc.ServiceDesc{
	ServiceName: "pb.Worker",
	HandlerType: (*WorkerServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Run",
			Handler:    _Worker_Run_Handler,
		},
		{
			MethodName: "Cancel",
			Handler:    _Worker_Cancel_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "coco.proto",
}

// FarmClient is the client API for Farm service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type FarmClient interface {
	Waiting(ctx context.Context, in *WaitingRequest, opts ...grpc.CallOption) (*WaitingResponse, error)
	Bye(ctx context.Context, in *ByeRequest, opts ...grpc.CallOption) (*ByeResponse, error)
	Done(ctx context.Context, in *DoneRequest, opts ...grpc.CallOption) (*DoneResponse, error)
	Failed(ctx context.Context, in *FailedRequest, opts ...grpc.CallOption) (*FailedResponse, error)
}

type farmClient struct {
	cc grpc.ClientConnInterface
}

func NewFarmClient(cc grpc.ClientConnInterface) FarmClient {
	return &farmClient{cc}
}

func (c *farmClient) Waiting(ctx context.Context, in *WaitingRequest, opts ...grpc.CallOption) (*WaitingResponse, error) {
	out := new(WaitingResponse)
	err := c.cc.Invoke(ctx, "/pb.Farm/Waiting", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *farmClient) Bye(ctx context.Context, in *ByeRequest, opts ...grpc.CallOption) (*ByeResponse, error) {
	out := new(ByeResponse)
	err := c.cc.Invoke(ctx, "/pb.Farm/Bye", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *farmClient) Done(ctx context.Context, in *DoneRequest, opts ...grpc.CallOption) (*DoneResponse, error) {
	out := new(DoneResponse)
	err := c.cc.Invoke(ctx, "/pb.Farm/Done", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *farmClient) Failed(ctx context.Context, in *FailedRequest, opts ...grpc.CallOption) (*FailedResponse, error) {
	out := new(FailedResponse)
	err := c.cc.Invoke(ctx, "/pb.Farm/Failed", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// FarmServer is the server API for Farm service.
// All implementations must embed UnimplementedFarmServer
// for forward compatibility
type FarmServer interface {
	Waiting(context.Context, *WaitingRequest) (*WaitingResponse, error)
	Bye(context.Context, *ByeRequest) (*ByeResponse, error)
	Done(context.Context, *DoneRequest) (*DoneResponse, error)
	Failed(context.Context, *FailedRequest) (*FailedResponse, error)
	mustEmbedUnimplementedFarmServer()
}

// UnimplementedFarmServer must be embedded to have forward compatible implementations.
type UnimplementedFarmServer struct {
}

func (UnimplementedFarmServer) Waiting(context.Context, *WaitingRequest) (*WaitingResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Waiting not implemented")
}
func (UnimplementedFarmServer) Bye(context.Context, *ByeRequest) (*ByeResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Bye not implemented")
}
func (UnimplementedFarmServer) Done(context.Context, *DoneRequest) (*DoneResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Done not implemented")
}
func (UnimplementedFarmServer) Failed(context.Context, *FailedRequest) (*FailedResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Failed not implemented")
}
func (UnimplementedFarmServer) mustEmbedUnimplementedFarmServer() {}

// UnsafeFarmServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to FarmServer will
// result in compilation errors.
type UnsafeFarmServer interface {
	mustEmbedUnimplementedFarmServer()
}

func RegisterFarmServer(s grpc.ServiceRegistrar, srv FarmServer) {
	s.RegisterService(&_Farm_serviceDesc, srv)
}

func _Farm_Waiting_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(WaitingRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FarmServer).Waiting(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/pb.Farm/Waiting",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FarmServer).Waiting(ctx, req.(*WaitingRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Farm_Bye_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ByeRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FarmServer).Bye(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/pb.Farm/Bye",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FarmServer).Bye(ctx, req.(*ByeRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Farm_Done_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DoneRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FarmServer).Done(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/pb.Farm/Done",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FarmServer).Done(ctx, req.(*DoneRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Farm_Failed_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(FailedRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FarmServer).Failed(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/pb.Farm/Failed",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FarmServer).Failed(ctx, req.(*FailedRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _Farm_serviceDesc = grpc.ServiceDesc{
	ServiceName: "pb.Farm",
	HandlerType: (*FarmServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Waiting",
			Handler:    _Farm_Waiting_Handler,
		},
		{
			MethodName: "Bye",
			Handler:    _Farm_Bye_Handler,
		},
		{
			MethodName: "Done",
			Handler:    _Farm_Done_Handler,
		},
		{
			MethodName: "Failed",
			Handler:    _Farm_Failed_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "coco.proto",
}
