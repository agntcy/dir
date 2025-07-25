// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.5
// 	protoc        (unknown)
// source: objects/v1/skill.proto

package objectsv1

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
	unsafe "unsafe"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// A specific skills that an agent is capable of performing.
// Specs: https://schema.oasf.agntcy.org/skills.
//
// Example (https://schema.oasf.agntcy.org/skills/contextual_comprehension)
type Skill struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// Additional metadata for this skill.
	Annotations map[string]string `protobuf:"bytes,1,rep,name=annotations,proto3" json:"annotations,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	// UID of the category.
	CategoryUid uint64 `protobuf:"varint,2,opt,name=category_uid,json=categoryUid,proto3" json:"category_uid,omitempty"`
	// UID of the class.
	ClassUid uint64 `protobuf:"varint,3,opt,name=class_uid,json=classUid,proto3" json:"class_uid,omitempty"`
	// Optional human-readable name of the category.
	CategoryName *string `protobuf:"bytes,4,opt,name=category_name,json=categoryName,proto3,oneof" json:"category_name,omitempty"`
	// Optional human-readable name of the class.
	ClassName     *string `protobuf:"bytes,5,opt,name=class_name,json=className,proto3,oneof" json:"class_name,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Skill) Reset() {
	*x = Skill{}
	mi := &file_objects_v1_skill_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Skill) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Skill) ProtoMessage() {}

func (x *Skill) ProtoReflect() protoreflect.Message {
	mi := &file_objects_v1_skill_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Skill.ProtoReflect.Descriptor instead.
func (*Skill) Descriptor() ([]byte, []int) {
	return file_objects_v1_skill_proto_rawDescGZIP(), []int{0}
}

func (x *Skill) GetAnnotations() map[string]string {
	if x != nil {
		return x.Annotations
	}
	return nil
}

func (x *Skill) GetCategoryUid() uint64 {
	if x != nil {
		return x.CategoryUid
	}
	return 0
}

func (x *Skill) GetClassUid() uint64 {
	if x != nil {
		return x.ClassUid
	}
	return 0
}

func (x *Skill) GetCategoryName() string {
	if x != nil && x.CategoryName != nil {
		return *x.CategoryName
	}
	return ""
}

func (x *Skill) GetClassName() string {
	if x != nil && x.ClassName != nil {
		return *x.ClassName
	}
	return ""
}

var File_objects_v1_skill_proto protoreflect.FileDescriptor

var file_objects_v1_skill_proto_rawDesc = string([]byte{
	0x0a, 0x16, 0x6f, 0x62, 0x6a, 0x65, 0x63, 0x74, 0x73, 0x2f, 0x76, 0x31, 0x2f, 0x73, 0x6b, 0x69,
	0x6c, 0x6c, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x0a, 0x6f, 0x62, 0x6a, 0x65, 0x63, 0x74,
	0x73, 0x2e, 0x76, 0x31, 0x22, 0xbc, 0x02, 0x0a, 0x05, 0x53, 0x6b, 0x69, 0x6c, 0x6c, 0x12, 0x44,
	0x0a, 0x0b, 0x61, 0x6e, 0x6e, 0x6f, 0x74, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x18, 0x01, 0x20,
	0x03, 0x28, 0x0b, 0x32, 0x22, 0x2e, 0x6f, 0x62, 0x6a, 0x65, 0x63, 0x74, 0x73, 0x2e, 0x76, 0x31,
	0x2e, 0x53, 0x6b, 0x69, 0x6c, 0x6c, 0x2e, 0x41, 0x6e, 0x6e, 0x6f, 0x74, 0x61, 0x74, 0x69, 0x6f,
	0x6e, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x0b, 0x61, 0x6e, 0x6e, 0x6f, 0x74, 0x61, 0x74,
	0x69, 0x6f, 0x6e, 0x73, 0x12, 0x21, 0x0a, 0x0c, 0x63, 0x61, 0x74, 0x65, 0x67, 0x6f, 0x72, 0x79,
	0x5f, 0x75, 0x69, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x04, 0x52, 0x0b, 0x63, 0x61, 0x74, 0x65,
	0x67, 0x6f, 0x72, 0x79, 0x55, 0x69, 0x64, 0x12, 0x1b, 0x0a, 0x09, 0x63, 0x6c, 0x61, 0x73, 0x73,
	0x5f, 0x75, 0x69, 0x64, 0x18, 0x03, 0x20, 0x01, 0x28, 0x04, 0x52, 0x08, 0x63, 0x6c, 0x61, 0x73,
	0x73, 0x55, 0x69, 0x64, 0x12, 0x28, 0x0a, 0x0d, 0x63, 0x61, 0x74, 0x65, 0x67, 0x6f, 0x72, 0x79,
	0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x48, 0x00, 0x52, 0x0c, 0x63,
	0x61, 0x74, 0x65, 0x67, 0x6f, 0x72, 0x79, 0x4e, 0x61, 0x6d, 0x65, 0x88, 0x01, 0x01, 0x12, 0x22,
	0x0a, 0x0a, 0x63, 0x6c, 0x61, 0x73, 0x73, 0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x05, 0x20, 0x01,
	0x28, 0x09, 0x48, 0x01, 0x52, 0x09, 0x63, 0x6c, 0x61, 0x73, 0x73, 0x4e, 0x61, 0x6d, 0x65, 0x88,
	0x01, 0x01, 0x1a, 0x3e, 0x0a, 0x10, 0x41, 0x6e, 0x6e, 0x6f, 0x74, 0x61, 0x74, 0x69, 0x6f, 0x6e,
	0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75,
	0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02,
	0x38, 0x01, 0x42, 0x10, 0x0a, 0x0e, 0x5f, 0x63, 0x61, 0x74, 0x65, 0x67, 0x6f, 0x72, 0x79, 0x5f,
	0x6e, 0x61, 0x6d, 0x65, 0x42, 0x0d, 0x0a, 0x0b, 0x5f, 0x63, 0x6c, 0x61, 0x73, 0x73, 0x5f, 0x6e,
	0x61, 0x6d, 0x65, 0x42, 0x95, 0x01, 0x0a, 0x0e, 0x63, 0x6f, 0x6d, 0x2e, 0x6f, 0x62, 0x6a, 0x65,
	0x63, 0x74, 0x73, 0x2e, 0x76, 0x31, 0x42, 0x0a, 0x53, 0x6b, 0x69, 0x6c, 0x6c, 0x50, 0x72, 0x6f,
	0x74, 0x6f, 0x50, 0x01, 0x5a, 0x2e, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d,
	0x2f, 0x61, 0x67, 0x6e, 0x74, 0x63, 0x79, 0x2f, 0x64, 0x69, 0x72, 0x2f, 0x61, 0x70, 0x69, 0x2f,
	0x6f, 0x62, 0x6a, 0x65, 0x63, 0x74, 0x73, 0x2f, 0x76, 0x31, 0x3b, 0x6f, 0x62, 0x6a, 0x65, 0x63,
	0x74, 0x73, 0x76, 0x31, 0xa2, 0x02, 0x03, 0x4f, 0x58, 0x58, 0xaa, 0x02, 0x0a, 0x4f, 0x62, 0x6a,
	0x65, 0x63, 0x74, 0x73, 0x2e, 0x56, 0x31, 0xca, 0x02, 0x0a, 0x4f, 0x62, 0x6a, 0x65, 0x63, 0x74,
	0x73, 0x5c, 0x56, 0x31, 0xe2, 0x02, 0x16, 0x4f, 0x62, 0x6a, 0x65, 0x63, 0x74, 0x73, 0x5c, 0x56,
	0x31, 0x5c, 0x47, 0x50, 0x42, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0xea, 0x02, 0x0b,
	0x4f, 0x62, 0x6a, 0x65, 0x63, 0x74, 0x73, 0x3a, 0x3a, 0x56, 0x31, 0x62, 0x06, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x33,
})

var (
	file_objects_v1_skill_proto_rawDescOnce sync.Once
	file_objects_v1_skill_proto_rawDescData []byte
)

func file_objects_v1_skill_proto_rawDescGZIP() []byte {
	file_objects_v1_skill_proto_rawDescOnce.Do(func() {
		file_objects_v1_skill_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_objects_v1_skill_proto_rawDesc), len(file_objects_v1_skill_proto_rawDesc)))
	})
	return file_objects_v1_skill_proto_rawDescData
}

var file_objects_v1_skill_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_objects_v1_skill_proto_goTypes = []any{
	(*Skill)(nil), // 0: objects.v1.Skill
	nil,           // 1: objects.v1.Skill.AnnotationsEntry
}
var file_objects_v1_skill_proto_depIdxs = []int32{
	1, // 0: objects.v1.Skill.annotations:type_name -> objects.v1.Skill.AnnotationsEntry
	1, // [1:1] is the sub-list for method output_type
	1, // [1:1] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_objects_v1_skill_proto_init() }
func file_objects_v1_skill_proto_init() {
	if File_objects_v1_skill_proto != nil {
		return
	}
	file_objects_v1_skill_proto_msgTypes[0].OneofWrappers = []any{}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_objects_v1_skill_proto_rawDesc), len(file_objects_v1_skill_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_objects_v1_skill_proto_goTypes,
		DependencyIndexes: file_objects_v1_skill_proto_depIdxs,
		MessageInfos:      file_objects_v1_skill_proto_msgTypes,
	}.Build()
	File_objects_v1_skill_proto = out.File
	file_objects_v1_skill_proto_goTypes = nil
	file_objects_v1_skill_proto_depIdxs = nil
}
