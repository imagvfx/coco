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
	Run(ctx context.Context, in *Task, opts ...grpc.CallOption) (*Empty, error)
}

type workerClient struct {
	cc grpc.ClientConnInterface
}

func NewWorkerClient(cc grpc.ClientConnInterface) WorkerClient {
	return &workerClient{cc}
}

func (c *workerClient) Run(ctx context.Context, in *Task, opts ...grpc.CallOption) (*Empty, error) {
	out := new(Empty)
	err := c.cc.Invoke(ctx, "/pb.Worker/Run", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// WorkerServer is the server API for Worker service.
// All implementations must embed UnimplementedWorkerServer
// for forward compatibility
type WorkerServer interface {
	Run(context.Context, *Task) (*Empty, error)
	mustEmbedUnimplementedWorkerServer()
}

// UnimplementedWorkerServer must be embedded to have forward compatible implementations.
type UnimplementedWorkerServer struct {
}

func (UnimplementedWorkerServer) Run(context.Context, *Task) (*Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Run not implemented")
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
	in := new(Task)
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
		return srv.(WorkerServer).Run(ctx, req.(*Task))
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
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "coco.proto",
}

// FarmClient is the client API for Farm service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type FarmClient interface {
	Waiting(ctx context.Context, in *Here, opts ...grpc.CallOption) (*Empty, error)
}

type farmClient struct {
	cc grpc.ClientConnInterface
}

func NewFarmClient(cc grpc.ClientConnInterface) FarmClient {
	return &farmClient{cc}
}

func (c *farmClient) Waiting(ctx context.Context, in *Here, opts ...grpc.CallOption) (*Empty, error) {
	out := new(Empty)
	err := c.cc.Invoke(ctx, "/pb.Farm/Waiting", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// FarmServer is the server API for Farm service.
// All implementations must embed UnimplementedFarmServer
// for forward compatibility
type FarmServer interface {
	Waiting(context.Context, *Here) (*Empty, error)
	mustEmbedUnimplementedFarmServer()
}

// UnimplementedFarmServer must be embedded to have forward compatible implementations.
type UnimplementedFarmServer struct {
}

func (UnimplementedFarmServer) Waiting(context.Context, *Here) (*Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Waiting not implemented")
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
	in := new(Here)
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
		return srv.(FarmServer).Waiting(ctx, req.(*Here))
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
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "coco.proto",
}
