// Code generated by protoc-gen-go. DO NOT EDIT.
// source: system.proto

/*
Package pb is a generated protocol buffer package.

It is generated from these files:
	system.proto

It has these top-level messages:
	GetSystemInfoRequest
	SystemInfoResponse
*/
package pb

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"
import _ "google.golang.org/genproto/googleapis/api/annotations"

import (
	context "golang.org/x/net/context"
	grpc "google.golang.org/grpc"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

type GetSystemInfoRequest struct {
}

func (m *GetSystemInfoRequest) Reset()                    { *m = GetSystemInfoRequest{} }
func (m *GetSystemInfoRequest) String() string            { return proto.CompactTextString(m) }
func (*GetSystemInfoRequest) ProtoMessage()               {}
func (*GetSystemInfoRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

type SystemInfoResponse struct {
	GoVersion    string `protobuf:"bytes,1,opt,name=go_version,json=goVersion" json:"go_version,omitempty"`
	Os           string `protobuf:"bytes,2,opt,name=os" json:"os,omitempty"`
	Arch         string `protobuf:"bytes,3,opt,name=arch" json:"arch,omitempty"`
	NumGoroutine uint32 `protobuf:"varint,4,opt,name=num_goroutine,json=numGoroutine" json:"num_goroutine,omitempty"`
	Hostname     string `protobuf:"bytes,5,opt,name=hostname" json:"hostname,omitempty"`
	Pid          uint64 `protobuf:"varint,6,opt,name=pid" json:"pid,omitempty"`
	Uid          uint64 `protobuf:"varint,7,opt,name=uid" json:"uid,omitempty"`
	MemAlloc     uint64 `protobuf:"varint,8,opt,name=mem_alloc,json=memAlloc" json:"mem_alloc,omitempty"`
	MemSys       uint64 `protobuf:"varint,9,opt,name=mem_sys,json=memSys" json:"mem_sys,omitempty"`
	NumGc        uint32 `protobuf:"varint,10,opt,name=num_gc,json=numGc" json:"num_gc,omitempty"`
	NumFds       uint32 `protobuf:"varint,11,opt,name=num_fds,json=numFds" json:"num_fds,omitempty"`
}

func (m *SystemInfoResponse) Reset()                    { *m = SystemInfoResponse{} }
func (m *SystemInfoResponse) String() string            { return proto.CompactTextString(m) }
func (*SystemInfoResponse) ProtoMessage()               {}
func (*SystemInfoResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{1} }

func (m *SystemInfoResponse) GetGoVersion() string {
	if m != nil {
		return m.GoVersion
	}
	return ""
}

func (m *SystemInfoResponse) GetOs() string {
	if m != nil {
		return m.Os
	}
	return ""
}

func (m *SystemInfoResponse) GetArch() string {
	if m != nil {
		return m.Arch
	}
	return ""
}

func (m *SystemInfoResponse) GetNumGoroutine() uint32 {
	if m != nil {
		return m.NumGoroutine
	}
	return 0
}

func (m *SystemInfoResponse) GetHostname() string {
	if m != nil {
		return m.Hostname
	}
	return ""
}

func (m *SystemInfoResponse) GetPid() uint64 {
	if m != nil {
		return m.Pid
	}
	return 0
}

func (m *SystemInfoResponse) GetUid() uint64 {
	if m != nil {
		return m.Uid
	}
	return 0
}

func (m *SystemInfoResponse) GetMemAlloc() uint64 {
	if m != nil {
		return m.MemAlloc
	}
	return 0
}

func (m *SystemInfoResponse) GetMemSys() uint64 {
	if m != nil {
		return m.MemSys
	}
	return 0
}

func (m *SystemInfoResponse) GetNumGc() uint32 {
	if m != nil {
		return m.NumGc
	}
	return 0
}

func (m *SystemInfoResponse) GetNumFds() uint32 {
	if m != nil {
		return m.NumFds
	}
	return 0
}

func init() {
	proto.RegisterType((*GetSystemInfoRequest)(nil), "pb.GetSystemInfoRequest")
	proto.RegisterType((*SystemInfoResponse)(nil), "pb.SystemInfoResponse")
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// Client API for SystemInfoService service

type SystemInfoServiceClient interface {
	GetSystemInfo(ctx context.Context, in *GetSystemInfoRequest, opts ...grpc.CallOption) (*SystemInfoResponse, error)
}

type systemInfoServiceClient struct {
	cc *grpc.ClientConn
}

func NewSystemInfoServiceClient(cc *grpc.ClientConn) SystemInfoServiceClient {
	return &systemInfoServiceClient{cc}
}

func (c *systemInfoServiceClient) GetSystemInfo(ctx context.Context, in *GetSystemInfoRequest, opts ...grpc.CallOption) (*SystemInfoResponse, error) {
	out := new(SystemInfoResponse)
	err := grpc.Invoke(ctx, "/pb.SystemInfoService/GetSystemInfo", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Server API for SystemInfoService service

type SystemInfoServiceServer interface {
	GetSystemInfo(context.Context, *GetSystemInfoRequest) (*SystemInfoResponse, error)
}

func RegisterSystemInfoServiceServer(s *grpc.Server, srv SystemInfoServiceServer) {
	s.RegisterService(&_SystemInfoService_serviceDesc, srv)
}

func _SystemInfoService_GetSystemInfo_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetSystemInfoRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SystemInfoServiceServer).GetSystemInfo(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/pb.SystemInfoService/GetSystemInfo",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SystemInfoServiceServer).GetSystemInfo(ctx, req.(*GetSystemInfoRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _SystemInfoService_serviceDesc = grpc.ServiceDesc{
	ServiceName: "pb.SystemInfoService",
	HandlerType: (*SystemInfoServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetSystemInfo",
			Handler:    _SystemInfoService_GetSystemInfo_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "system.proto",
}

func init() { proto.RegisterFile("system.proto", fileDescriptor0) }

var fileDescriptor0 = []byte{
	// 339 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x6c, 0x91, 0xcf, 0x4e, 0xc2, 0x40,
	0x10, 0xc6, 0xd3, 0x02, 0x85, 0x8e, 0xa0, 0xb2, 0x51, 0xd8, 0xa0, 0x26, 0x04, 0x2f, 0x9c, 0x68,
	0xd4, 0x27, 0xf0, 0x22, 0xf1, 0x5a, 0x12, 0x0f, 0x5e, 0x48, 0x69, 0x97, 0xb2, 0x09, 0xbb, 0x53,
	0x3b, 0x5b, 0x12, 0xae, 0xbe, 0x82, 0x6f, 0xe0, 0x2b, 0xf9, 0x0a, 0x3e, 0x88, 0xd9, 0x6d, 0xfc,
	0x17, 0xbd, 0xcd, 0xfc, 0xbe, 0x6f, 0x92, 0x2f, 0xdf, 0x40, 0x97, 0xf6, 0x64, 0x84, 0x9a, 0x15,
	0x25, 0x1a, 0x64, 0x7e, 0xb1, 0x1a, 0x9d, 0xe7, 0x88, 0xf9, 0x56, 0x44, 0x49, 0x21, 0xa3, 0x44,
	0x6b, 0x34, 0x89, 0x91, 0xa8, 0xa9, 0x76, 0x4c, 0x06, 0x70, 0x32, 0x17, 0x66, 0xe1, 0x8e, 0xee,
	0xf5, 0x1a, 0x63, 0xf1, 0x54, 0x09, 0x32, 0x93, 0x57, 0x1f, 0xd8, 0x4f, 0x4a, 0x05, 0x6a, 0x12,
	0xec, 0x02, 0x20, 0xc7, 0xe5, 0x4e, 0x94, 0x24, 0x51, 0x73, 0x6f, 0xec, 0x4d, 0xc3, 0x38, 0xcc,
	0xf1, 0xa1, 0x06, 0xec, 0x10, 0x7c, 0x24, 0xee, 0x3b, 0xec, 0x23, 0x31, 0x06, 0xcd, 0xa4, 0x4c,
	0x37, 0xbc, 0xe1, 0x88, 0x9b, 0xd9, 0x25, 0xf4, 0x74, 0xa5, 0x96, 0x39, 0x96, 0x58, 0x19, 0xa9,
	0x05, 0x6f, 0x8e, 0xbd, 0x69, 0x2f, 0xee, 0xea, 0x4a, 0xcd, 0x3f, 0x19, 0x1b, 0x41, 0x67, 0x83,
	0x64, 0x74, 0xa2, 0x04, 0x6f, 0xb9, 0xe3, 0xaf, 0x9d, 0x1d, 0x43, 0xa3, 0x90, 0x19, 0x0f, 0xc6,
	0xde, 0xb4, 0x19, 0xdb, 0xd1, 0x92, 0x4a, 0x66, 0xbc, 0x5d, 0x93, 0x4a, 0x66, 0xec, 0x0c, 0x42,
	0x25, 0xd4, 0x32, 0xd9, 0x6e, 0x31, 0xe5, 0x1d, 0xc7, 0x3b, 0x4a, 0xa8, 0x5b, 0xbb, 0xb3, 0x21,
	0xb4, 0xad, 0x48, 0x7b, 0xe2, 0xa1, 0x93, 0x02, 0x25, 0xd4, 0x62, 0x4f, 0xec, 0x14, 0x02, 0x17,
	0x2d, 0xe5, 0xe0, 0x32, 0xb5, 0x6c, 0x26, 0xe7, 0xb7, 0x78, 0x9d, 0x11, 0x3f, 0x70, 0xdc, 0xba,
	0xee, 0x32, 0xba, 0x46, 0xe8, 0x7f, 0x77, 0xb4, 0x10, 0xe5, 0x4e, 0xa6, 0x82, 0x3d, 0x42, 0xef,
	0x57, 0xa3, 0x8c, 0xcf, 0x8a, 0xd5, 0xec, 0xbf, 0x92, 0x47, 0x03, 0xab, 0xfc, 0x6d, 0x79, 0x32,
	0x7c, 0x7e, 0x7b, 0x7f, 0xf1, 0xfb, 0xec, 0x28, 0xda, 0x5d, 0x45, 0xf5, 0x43, 0x23, 0xa9, 0xd7,
	0xb8, 0x0a, 0xdc, 0xd3, 0x6e, 0x3e, 0x02, 0x00, 0x00, 0xff, 0xff, 0x4f, 0xa3, 0x01, 0xe1, 0xe6,
	0x01, 0x00, 0x00,
}