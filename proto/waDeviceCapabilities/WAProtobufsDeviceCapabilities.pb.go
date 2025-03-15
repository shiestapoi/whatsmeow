// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.5
// 	protoc        v3.6.1
// source: waDeviceCapabilities/WAProtobufsDeviceCapabilities.proto

package waDeviceCapabilities

import (
	reflect "reflect"
	sync "sync"
	unsafe "unsafe"

	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type DeviceCapabilities_ChatLockSupportLevel int32

const (
	DeviceCapabilities_NONE    DeviceCapabilities_ChatLockSupportLevel = 0
	DeviceCapabilities_MINIMAL DeviceCapabilities_ChatLockSupportLevel = 1
	DeviceCapabilities_FULL    DeviceCapabilities_ChatLockSupportLevel = 2
)

// Enum value maps for DeviceCapabilities_ChatLockSupportLevel.
var (
	DeviceCapabilities_ChatLockSupportLevel_name = map[int32]string{
		0: "NONE",
		1: "MINIMAL",
		2: "FULL",
	}
	DeviceCapabilities_ChatLockSupportLevel_value = map[string]int32{
		"NONE":    0,
		"MINIMAL": 1,
		"FULL":    2,
	}
)

func (x DeviceCapabilities_ChatLockSupportLevel) Enum() *DeviceCapabilities_ChatLockSupportLevel {
	p := new(DeviceCapabilities_ChatLockSupportLevel)
	*p = x
	return p
}

func (x DeviceCapabilities_ChatLockSupportLevel) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (DeviceCapabilities_ChatLockSupportLevel) Descriptor() protoreflect.EnumDescriptor {
	return file_waDeviceCapabilities_WAProtobufsDeviceCapabilities_proto_enumTypes[0].Descriptor()
}

func (DeviceCapabilities_ChatLockSupportLevel) Type() protoreflect.EnumType {
	return &file_waDeviceCapabilities_WAProtobufsDeviceCapabilities_proto_enumTypes[0]
}

func (x DeviceCapabilities_ChatLockSupportLevel) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Do not use.
func (x *DeviceCapabilities_ChatLockSupportLevel) UnmarshalJSON(b []byte) error {
	num, err := protoimpl.X.UnmarshalJSONEnum(x.Descriptor(), b)
	if err != nil {
		return err
	}
	*x = DeviceCapabilities_ChatLockSupportLevel(num)
	return nil
}

// Deprecated: Use DeviceCapabilities_ChatLockSupportLevel.Descriptor instead.
func (DeviceCapabilities_ChatLockSupportLevel) EnumDescriptor() ([]byte, []int) {
	return file_waDeviceCapabilities_WAProtobufsDeviceCapabilities_proto_rawDescGZIP(), []int{0, 0}
}

type DeviceCapabilities struct {
	state                protoimpl.MessageState                   `protogen:"open.v1"`
	ChatLockSupportLevel *DeviceCapabilities_ChatLockSupportLevel `protobuf:"varint,1,opt,name=chatLockSupportLevel,enum=WAProtobufsDeviceCapabilities.DeviceCapabilities_ChatLockSupportLevel" json:"chatLockSupportLevel,omitempty"`
	unknownFields        protoimpl.UnknownFields
	sizeCache            protoimpl.SizeCache
}

func (x *DeviceCapabilities) Reset() {
	*x = DeviceCapabilities{}
	mi := &file_waDeviceCapabilities_WAProtobufsDeviceCapabilities_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *DeviceCapabilities) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DeviceCapabilities) ProtoMessage() {}

func (x *DeviceCapabilities) ProtoReflect() protoreflect.Message {
	mi := &file_waDeviceCapabilities_WAProtobufsDeviceCapabilities_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DeviceCapabilities.ProtoReflect.Descriptor instead.
func (*DeviceCapabilities) Descriptor() ([]byte, []int) {
	return file_waDeviceCapabilities_WAProtobufsDeviceCapabilities_proto_rawDescGZIP(), []int{0}
}

func (x *DeviceCapabilities) GetChatLockSupportLevel() DeviceCapabilities_ChatLockSupportLevel {
	if x != nil && x.ChatLockSupportLevel != nil {
		return *x.ChatLockSupportLevel
	}
	return DeviceCapabilities_NONE
}

var File_waDeviceCapabilities_WAProtobufsDeviceCapabilities_proto protoreflect.FileDescriptor

var file_waDeviceCapabilities_WAProtobufsDeviceCapabilities_proto_rawDesc = string([]byte{
	0x0a, 0x38, 0x77, 0x61, 0x44, 0x65, 0x76, 0x69, 0x63, 0x65, 0x43, 0x61, 0x70, 0x61, 0x62, 0x69,
	0x6c, 0x69, 0x74, 0x69, 0x65, 0x73, 0x2f, 0x57, 0x41, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75,
	0x66, 0x73, 0x44, 0x65, 0x76, 0x69, 0x63, 0x65, 0x43, 0x61, 0x70, 0x61, 0x62, 0x69, 0x6c, 0x69,
	0x74, 0x69, 0x65, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x1d, 0x57, 0x41, 0x50, 0x72,
	0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x73, 0x44, 0x65, 0x76, 0x69, 0x63, 0x65, 0x43, 0x61, 0x70,
	0x61, 0x62, 0x69, 0x6c, 0x69, 0x74, 0x69, 0x65, 0x73, 0x22, 0xc9, 0x01, 0x0a, 0x12, 0x44, 0x65,
	0x76, 0x69, 0x63, 0x65, 0x43, 0x61, 0x70, 0x61, 0x62, 0x69, 0x6c, 0x69, 0x74, 0x69, 0x65, 0x73,
	0x12, 0x7a, 0x0a, 0x14, 0x63, 0x68, 0x61, 0x74, 0x4c, 0x6f, 0x63, 0x6b, 0x53, 0x75, 0x70, 0x70,
	0x6f, 0x72, 0x74, 0x4c, 0x65, 0x76, 0x65, 0x6c, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x46,
	0x2e, 0x57, 0x41, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x73, 0x44, 0x65, 0x76, 0x69,
	0x63, 0x65, 0x43, 0x61, 0x70, 0x61, 0x62, 0x69, 0x6c, 0x69, 0x74, 0x69, 0x65, 0x73, 0x2e, 0x44,
	0x65, 0x76, 0x69, 0x63, 0x65, 0x43, 0x61, 0x70, 0x61, 0x62, 0x69, 0x6c, 0x69, 0x74, 0x69, 0x65,
	0x73, 0x2e, 0x43, 0x68, 0x61, 0x74, 0x4c, 0x6f, 0x63, 0x6b, 0x53, 0x75, 0x70, 0x70, 0x6f, 0x72,
	0x74, 0x4c, 0x65, 0x76, 0x65, 0x6c, 0x52, 0x14, 0x63, 0x68, 0x61, 0x74, 0x4c, 0x6f, 0x63, 0x6b,
	0x53, 0x75, 0x70, 0x70, 0x6f, 0x72, 0x74, 0x4c, 0x65, 0x76, 0x65, 0x6c, 0x22, 0x37, 0x0a, 0x14,
	0x43, 0x68, 0x61, 0x74, 0x4c, 0x6f, 0x63, 0x6b, 0x53, 0x75, 0x70, 0x70, 0x6f, 0x72, 0x74, 0x4c,
	0x65, 0x76, 0x65, 0x6c, 0x12, 0x08, 0x0a, 0x04, 0x4e, 0x4f, 0x4e, 0x45, 0x10, 0x00, 0x12, 0x0b,
	0x0a, 0x07, 0x4d, 0x49, 0x4e, 0x49, 0x4d, 0x41, 0x4c, 0x10, 0x01, 0x12, 0x08, 0x0a, 0x04, 0x46,
	0x55, 0x4c, 0x4c, 0x10, 0x02, 0x42, 0x3c, 0x5a, 0x3a, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e,
	0x63, 0x6f, 0x6d, 0x2f, 0x73, 0x68, 0x69, 0x65, 0x73, 0x74, 0x61, 0x70, 0x6f, 0x69, 0x2f, 0x77,
	0x68, 0x61, 0x74, 0x73, 0x6d, 0x65, 0x6f, 0x77, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x77,
	0x61, 0x44, 0x65, 0x76, 0x69, 0x63, 0x65, 0x43, 0x61, 0x70, 0x61, 0x62, 0x69, 0x6c, 0x69, 0x74,
	0x69, 0x65, 0x73,
})

var (
	file_waDeviceCapabilities_WAProtobufsDeviceCapabilities_proto_rawDescOnce sync.Once
	file_waDeviceCapabilities_WAProtobufsDeviceCapabilities_proto_rawDescData []byte
)

func file_waDeviceCapabilities_WAProtobufsDeviceCapabilities_proto_rawDescGZIP() []byte {
	file_waDeviceCapabilities_WAProtobufsDeviceCapabilities_proto_rawDescOnce.Do(func() {
		file_waDeviceCapabilities_WAProtobufsDeviceCapabilities_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_waDeviceCapabilities_WAProtobufsDeviceCapabilities_proto_rawDesc), len(file_waDeviceCapabilities_WAProtobufsDeviceCapabilities_proto_rawDesc)))
	})
	return file_waDeviceCapabilities_WAProtobufsDeviceCapabilities_proto_rawDescData
}

var file_waDeviceCapabilities_WAProtobufsDeviceCapabilities_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_waDeviceCapabilities_WAProtobufsDeviceCapabilities_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_waDeviceCapabilities_WAProtobufsDeviceCapabilities_proto_goTypes = []any{
	(DeviceCapabilities_ChatLockSupportLevel)(0), // 0: WAProtobufsDeviceCapabilities.DeviceCapabilities.ChatLockSupportLevel
	(*DeviceCapabilities)(nil),                   // 1: WAProtobufsDeviceCapabilities.DeviceCapabilities
}
var file_waDeviceCapabilities_WAProtobufsDeviceCapabilities_proto_depIdxs = []int32{
	0, // 0: WAProtobufsDeviceCapabilities.DeviceCapabilities.chatLockSupportLevel:type_name -> WAProtobufsDeviceCapabilities.DeviceCapabilities.ChatLockSupportLevel
	1, // [1:1] is the sub-list for method output_type
	1, // [1:1] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_waDeviceCapabilities_WAProtobufsDeviceCapabilities_proto_init() }
func file_waDeviceCapabilities_WAProtobufsDeviceCapabilities_proto_init() {
	if File_waDeviceCapabilities_WAProtobufsDeviceCapabilities_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_waDeviceCapabilities_WAProtobufsDeviceCapabilities_proto_rawDesc), len(file_waDeviceCapabilities_WAProtobufsDeviceCapabilities_proto_rawDesc)),
			NumEnums:      1,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_waDeviceCapabilities_WAProtobufsDeviceCapabilities_proto_goTypes,
		DependencyIndexes: file_waDeviceCapabilities_WAProtobufsDeviceCapabilities_proto_depIdxs,
		EnumInfos:         file_waDeviceCapabilities_WAProtobufsDeviceCapabilities_proto_enumTypes,
		MessageInfos:      file_waDeviceCapabilities_WAProtobufsDeviceCapabilities_proto_msgTypes,
	}.Build()
	File_waDeviceCapabilities_WAProtobufsDeviceCapabilities_proto = out.File
	file_waDeviceCapabilities_WAProtobufsDeviceCapabilities_proto_goTypes = nil
	file_waDeviceCapabilities_WAProtobufsDeviceCapabilities_proto_depIdxs = nil
}
