// Code generated by protoc-gen-go. DO NOT EDIT.
// source: libs/labels/protos/string_int_label_map.proto

package object_detection_protos

import (
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	math "math"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion3 // please upgrade the proto package

type StringIntLabelMapItem struct {
	// String name. The most common practice is to set this to a MID or synsets
	// id.
	Name *string `protobuf:"bytes,1,opt,name=name" json:"name,omitempty"`
	// Integer id that maps to the string name above. Label ids should start from
	// 1.
	Id *int32 `protobuf:"varint,2,opt,name=id" json:"id,omitempty"`
	// Human readable string label.
	DisplayName          *string  `protobuf:"bytes,3,opt,name=display_name,json=displayName" json:"display_name,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *StringIntLabelMapItem) Reset()         { *m = StringIntLabelMapItem{} }
func (m *StringIntLabelMapItem) String() string { return proto.CompactTextString(m) }
func (*StringIntLabelMapItem) ProtoMessage()    {}
func (*StringIntLabelMapItem) Descriptor() ([]byte, []int) {
	return fileDescriptor_20df02887ae33272, []int{0}
}

func (m *StringIntLabelMapItem) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_StringIntLabelMapItem.Unmarshal(m, b)
}
func (m *StringIntLabelMapItem) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_StringIntLabelMapItem.Marshal(b, m, deterministic)
}
func (m *StringIntLabelMapItem) XXX_Merge(src proto.Message) {
	xxx_messageInfo_StringIntLabelMapItem.Merge(m, src)
}
func (m *StringIntLabelMapItem) XXX_Size() int {
	return xxx_messageInfo_StringIntLabelMapItem.Size(m)
}
func (m *StringIntLabelMapItem) XXX_DiscardUnknown() {
	xxx_messageInfo_StringIntLabelMapItem.DiscardUnknown(m)
}

var xxx_messageInfo_StringIntLabelMapItem proto.InternalMessageInfo

func (m *StringIntLabelMapItem) GetName() string {
	if m != nil && m.Name != nil {
		return *m.Name
	}
	return ""
}

func (m *StringIntLabelMapItem) GetId() int32 {
	if m != nil && m.Id != nil {
		return *m.Id
	}
	return 0
}

func (m *StringIntLabelMapItem) GetDisplayName() string {
	if m != nil && m.DisplayName != nil {
		return *m.DisplayName
	}
	return ""
}

type StringIntLabelMap struct {
	Item                 []*StringIntLabelMapItem `protobuf:"bytes,1,rep,name=item" json:"item,omitempty"`
	XXX_NoUnkeyedLiteral struct{}                 `json:"-"`
	XXX_unrecognized     []byte                   `json:"-"`
	XXX_sizecache        int32                    `json:"-"`
}

func (m *StringIntLabelMap) Reset()         { *m = StringIntLabelMap{} }
func (m *StringIntLabelMap) String() string { return proto.CompactTextString(m) }
func (*StringIntLabelMap) ProtoMessage()    {}
func (*StringIntLabelMap) Descriptor() ([]byte, []int) {
	return fileDescriptor_20df02887ae33272, []int{1}
}

func (m *StringIntLabelMap) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_StringIntLabelMap.Unmarshal(m, b)
}
func (m *StringIntLabelMap) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_StringIntLabelMap.Marshal(b, m, deterministic)
}
func (m *StringIntLabelMap) XXX_Merge(src proto.Message) {
	xxx_messageInfo_StringIntLabelMap.Merge(m, src)
}
func (m *StringIntLabelMap) XXX_Size() int {
	return xxx_messageInfo_StringIntLabelMap.Size(m)
}
func (m *StringIntLabelMap) XXX_DiscardUnknown() {
	xxx_messageInfo_StringIntLabelMap.DiscardUnknown(m)
}

var xxx_messageInfo_StringIntLabelMap proto.InternalMessageInfo

func (m *StringIntLabelMap) GetItem() []*StringIntLabelMapItem {
	if m != nil {
		return m.Item
	}
	return nil
}

func init() {
	proto.RegisterType((*StringIntLabelMapItem)(nil), "object_detection.protos.StringIntLabelMapItem")
	proto.RegisterType((*StringIntLabelMap)(nil), "object_detection.protos.StringIntLabelMap")
}

func init() {
	proto.RegisterFile("libs/labels/protos/string_int_label_map.proto", fileDescriptor_20df02887ae33272)
}

var fileDescriptor_20df02887ae33272 = []byte{
	// 194 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x74, 0x8e, 0x3f, 0x6b, 0xc3, 0x30,
	0x10, 0xc5, 0x91, 0xed, 0x0e, 0x95, 0x4b, 0xa1, 0x82, 0x52, 0x8f, 0xae, 0x27, 0x2f, 0x95, 0xa1,
	0x1f, 0xa1, 0x9b, 0xa1, 0xed, 0xe0, 0x0c, 0xd9, 0x22, 0x64, 0xeb, 0x08, 0x17, 0xf4, 0x0f, 0xeb,
	0x96, 0x7c, 0xfb, 0x10, 0x25, 0x5b, 0x92, 0xed, 0x78, 0xef, 0xfd, 0x8e, 0x1f, 0xff, 0xb2, 0x38,
	0xa7, 0xc1, 0xea, 0x19, 0x6c, 0x1a, 0xe2, 0x1a, 0x28, 0xa4, 0x21, 0xd1, 0x8a, 0x7e, 0xaf, 0xd0,
	0x93, 0xca, 0x85, 0x72, 0x3a, 0xca, 0xdc, 0x89, 0x8f, 0x30, 0x1f, 0x60, 0x21, 0x65, 0x80, 0x60,
	0x21, 0x0c, 0xfe, 0x92, 0xa7, 0x6e, 0xc7, 0xdf, 0x37, 0x19, 0x1b, 0x3d, 0xfd, 0x9e, 0xa1, 0x3f,
	0x1d, 0x47, 0x02, 0x27, 0x04, 0xaf, 0xbc, 0x76, 0xd0, 0xb0, 0x96, 0xf5, 0xcf, 0x53, 0xbe, 0xc5,
	0x2b, 0x2f, 0xd0, 0x34, 0x45, 0xcb, 0xfa, 0xa7, 0xa9, 0x40, 0x23, 0x3e, 0xf9, 0x8b, 0xc1, 0x14,
	0xad, 0x3e, 0xaa, 0xbc, 0x2d, 0xf3, 0xb6, 0xbe, 0x66, 0xff, 0xda, 0x41, 0xb7, 0xe5, 0x6f, 0x37,
	0xff, 0xc5, 0x0f, 0xaf, 0x90, 0xc0, 0x35, 0xac, 0x2d, 0xfb, 0xfa, 0x5b, 0xca, 0x07, 0x72, 0xf2,
	0xae, 0xd9, 0x94, 0xd9, 0x53, 0x00, 0x00, 0x00, 0xff, 0xff, 0x95, 0x27, 0x7b, 0xd2, 0x01, 0x01,
	0x00, 0x00,
}
