// Code generated by protoc-gen-go. DO NOT EDIT.
// source: otaru-fe.proto

package pb

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"
import _ "github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger/options"
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

type ListHostsRequest struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ListHostsRequest) Reset()         { *m = ListHostsRequest{} }
func (m *ListHostsRequest) String() string { return proto.CompactTextString(m) }
func (*ListHostsRequest) ProtoMessage()    {}
func (*ListHostsRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_otaru_fe_d01a44a4472fec38, []int{0}
}
func (m *ListHostsRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ListHostsRequest.Unmarshal(m, b)
}
func (m *ListHostsRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ListHostsRequest.Marshal(b, m, deterministic)
}
func (dst *ListHostsRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ListHostsRequest.Merge(dst, src)
}
func (m *ListHostsRequest) XXX_Size() int {
	return xxx_messageInfo_ListHostsRequest.Size(m)
}
func (m *ListHostsRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_ListHostsRequest.DiscardUnknown(m)
}

var xxx_messageInfo_ListHostsRequest proto.InternalMessageInfo

type HostInfo struct {
	Id                   uint32   `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty"`
	Name                 string   `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *HostInfo) Reset()         { *m = HostInfo{} }
func (m *HostInfo) String() string { return proto.CompactTextString(m) }
func (*HostInfo) ProtoMessage()    {}
func (*HostInfo) Descriptor() ([]byte, []int) {
	return fileDescriptor_otaru_fe_d01a44a4472fec38, []int{1}
}
func (m *HostInfo) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_HostInfo.Unmarshal(m, b)
}
func (m *HostInfo) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_HostInfo.Marshal(b, m, deterministic)
}
func (dst *HostInfo) XXX_Merge(src proto.Message) {
	xxx_messageInfo_HostInfo.Merge(dst, src)
}
func (m *HostInfo) XXX_Size() int {
	return xxx_messageInfo_HostInfo.Size(m)
}
func (m *HostInfo) XXX_DiscardUnknown() {
	xxx_messageInfo_HostInfo.DiscardUnknown(m)
}

var xxx_messageInfo_HostInfo proto.InternalMessageInfo

func (m *HostInfo) GetId() uint32 {
	if m != nil {
		return m.Id
	}
	return 0
}

func (m *HostInfo) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

type ListHostsResponse struct {
	Host                 []*HostInfo `protobuf:"bytes,1,rep,name=host,proto3" json:"host,omitempty"`
	XXX_NoUnkeyedLiteral struct{}    `json:"-"`
	XXX_unrecognized     []byte      `json:"-"`
	XXX_sizecache        int32       `json:"-"`
}

func (m *ListHostsResponse) Reset()         { *m = ListHostsResponse{} }
func (m *ListHostsResponse) String() string { return proto.CompactTextString(m) }
func (*ListHostsResponse) ProtoMessage()    {}
func (*ListHostsResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_otaru_fe_d01a44a4472fec38, []int{2}
}
func (m *ListHostsResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ListHostsResponse.Unmarshal(m, b)
}
func (m *ListHostsResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ListHostsResponse.Marshal(b, m, deterministic)
}
func (dst *ListHostsResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ListHostsResponse.Merge(dst, src)
}
func (m *ListHostsResponse) XXX_Size() int {
	return xxx_messageInfo_ListHostsResponse.Size(m)
}
func (m *ListHostsResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_ListHostsResponse.DiscardUnknown(m)
}

var xxx_messageInfo_ListHostsResponse proto.InternalMessageInfo

func (m *ListHostsResponse) GetHost() []*HostInfo {
	if m != nil {
		return m.Host
	}
	return nil
}

func init() {
	proto.RegisterType((*ListHostsRequest)(nil), "pb.ListHostsRequest")
	proto.RegisterType((*HostInfo)(nil), "pb.HostInfo")
	proto.RegisterType((*ListHostsResponse)(nil), "pb.ListHostsResponse")
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// FeServiceClient is the client API for FeService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type FeServiceClient interface {
	ListHosts(ctx context.Context, in *ListHostsRequest, opts ...grpc.CallOption) (*ListHostsResponse, error)
}

type feServiceClient struct {
	cc *grpc.ClientConn
}

func NewFeServiceClient(cc *grpc.ClientConn) FeServiceClient {
	return &feServiceClient{cc}
}

func (c *feServiceClient) ListHosts(ctx context.Context, in *ListHostsRequest, opts ...grpc.CallOption) (*ListHostsResponse, error) {
	out := new(ListHostsResponse)
	err := c.cc.Invoke(ctx, "/pb.FeService/ListHosts", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// FeServiceServer is the server API for FeService service.
type FeServiceServer interface {
	ListHosts(context.Context, *ListHostsRequest) (*ListHostsResponse, error)
}

func RegisterFeServiceServer(s *grpc.Server, srv FeServiceServer) {
	s.RegisterService(&_FeService_serviceDesc, srv)
}

func _FeService_ListHosts_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListHostsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FeServiceServer).ListHosts(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/pb.FeService/ListHosts",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FeServiceServer).ListHosts(ctx, req.(*ListHostsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _FeService_serviceDesc = grpc.ServiceDesc{
	ServiceName: "pb.FeService",
	HandlerType: (*FeServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "ListHosts",
			Handler:    _FeService_ListHosts_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "otaru-fe.proto",
}

func init() { proto.RegisterFile("otaru-fe.proto", fileDescriptor_otaru_fe_d01a44a4472fec38) }

var fileDescriptor_otaru_fe_d01a44a4472fec38 = []byte{
	// 338 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x64, 0x90, 0xc1, 0x4a, 0xeb, 0x40,
	0x14, 0x86, 0x49, 0x5a, 0x2e, 0xb7, 0x73, 0xdb, 0xd2, 0x3b, 0x28, 0x84, 0x20, 0x12, 0xb2, 0x2a,
	0x62, 0x33, 0x36, 0xe2, 0xa6, 0x2b, 0xab, 0x20, 0x16, 0x04, 0x25, 0x6e, 0xdc, 0xc9, 0x24, 0x3d,
	0x4d, 0xa7, 0xb4, 0x73, 0xc6, 0xcc, 0xa4, 0xea, 0x52, 0xc1, 0x17, 0xd0, 0x47, 0xf3, 0x15, 0x7c,
	0x10, 0xc9, 0x54, 0x45, 0xda, 0xd5, 0xcc, 0xf9, 0xcf, 0xcf, 0x37, 0xc3, 0x47, 0xda, 0x68, 0x78,
	0x51, 0xf6, 0x26, 0x10, 0xa9, 0x02, 0x0d, 0x52, 0x57, 0xa5, 0xfe, 0x4e, 0x8e, 0x98, 0xcf, 0x81,
	0x71, 0x25, 0x18, 0x97, 0x12, 0x0d, 0x37, 0x02, 0xa5, 0x5e, 0x35, 0xfc, 0x7d, 0x7b, 0x64, 0xbd,
	0x1c, 0x64, 0x4f, 0xdf, 0xf3, 0x3c, 0x87, 0x82, 0xa1, 0xb2, 0x8d, 0xcd, 0x76, 0x48, 0x49, 0xe7,
	0x42, 0x68, 0x73, 0x8e, 0xda, 0xe8, 0x04, 0xee, 0x4a, 0xd0, 0x26, 0x8c, 0xc8, 0xdf, 0x6a, 0x1e,
	0xc9, 0x09, 0xd2, 0x36, 0x71, 0xc5, 0xd8, 0x73, 0x02, 0xa7, 0xdb, 0x4a, 0x5c, 0x31, 0xa6, 0x94,
	0xd4, 0x25, 0x5f, 0x80, 0xe7, 0x06, 0x4e, 0xb7, 0x91, 0xd8, 0x7b, 0x78, 0x44, 0xfe, 0xff, 0x62,
	0x68, 0x85, 0x52, 0x03, 0x0d, 0x48, 0x7d, 0x8a, 0xda, 0x78, 0x4e, 0x50, 0xeb, 0xfe, 0x8b, 0x9b,
	0x91, 0x4a, 0xa3, 0x6f, 0x68, 0x62, 0x37, 0xf1, 0x2d, 0x69, 0x9c, 0xc1, 0x35, 0x14, 0x4b, 0x91,
	0x01, 0x4d, 0x48, 0xe3, 0x87, 0x41, 0xb7, 0xaa, 0xf6, 0xfa, 0xb7, 0xfc, 0xed, 0xb5, 0x74, 0xf5,
	0x50, 0xe8, 0x3d, 0xbf, 0x7f, 0xbc, 0xb9, 0x94, 0x76, 0xac, 0x8f, 0x65, 0x9f, 0x4d, 0x80, 0x55,
	0x7c, 0x7d, 0xf2, 0xe2, 0xbc, 0x0e, 0x9f, 0x1c, 0x7a, 0x43, 0x9a, 0x97, 0x5f, 0x12, 0x83, 0xe1,
	0xd5, 0x28, 0x3c, 0x25, 0x2d, 0x3b, 0x07, 0xaa, 0xc0, 0x19, 0x64, 0x86, 0xee, 0x4e, 0x8d, 0x51,
	0x7a, 0xc0, 0x58, 0x2e, 0xcc, 0xb4, 0x4c, 0xa3, 0x0c, 0x17, 0x4c, 0x3e, 0xf2, 0x07, 0xc3, 0xac,
	0x7c, 0x9f, 0x96, 0x20, 0xf1, 0xd8, 0x26, 0xda, 0x80, 0xaa, 0xf6, 0x71, 0xad, 0x1f, 0x1d, 0xec,
	0x39, 0x6e, 0xdc, 0xe1, 0x4a, 0xcd, 0x45, 0x66, 0x9d, 0xb2, 0x99, 0x46, 0x39, 0xd8, 0x48, 0xd2,
	0x3f, 0x56, 0xf5, 0xe1, 0x67, 0x00, 0x00, 0x00, 0xff, 0xff, 0xbd, 0x95, 0xf0, 0xfe, 0xcc, 0x01,
	0x00, 0x00,
}
