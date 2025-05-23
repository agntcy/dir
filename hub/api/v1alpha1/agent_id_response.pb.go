// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.5
// 	protoc        (unknown)
// source: saas/v1alpha1/agent_id_response.proto

package v1alpha1

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

type AgentIdentifierResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	RepoVersionId *RepoVersionId         `protobuf:"bytes,2,opt,name=repo_version_id,json=repoVersionId,proto3" json:"repo_version_id,omitempty"`
	Digest        string                 `protobuf:"bytes,3,opt,name=digest,proto3" json:"digest,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *AgentIdentifierResponse) Reset() {
	*x = AgentIdentifierResponse{}
	mi := &file_saas_v1alpha1_agent_id_response_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *AgentIdentifierResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AgentIdentifierResponse) ProtoMessage() {}

func (x *AgentIdentifierResponse) ProtoReflect() protoreflect.Message {
	mi := &file_saas_v1alpha1_agent_id_response_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AgentIdentifierResponse.ProtoReflect.Descriptor instead.
func (*AgentIdentifierResponse) Descriptor() ([]byte, []int) {
	return file_saas_v1alpha1_agent_id_response_proto_rawDescGZIP(), []int{0}
}

func (x *AgentIdentifierResponse) GetRepoVersionId() *RepoVersionId {
	if x != nil {
		return x.RepoVersionId
	}
	return nil
}

func (x *AgentIdentifierResponse) GetDigest() string {
	if x != nil {
		return x.Digest
	}
	return ""
}

var File_saas_v1alpha1_agent_id_response_proto protoreflect.FileDescriptor

var file_saas_v1alpha1_agent_id_response_proto_rawDesc = string([]byte{
	0x0a, 0x25, 0x73, 0x61, 0x61, 0x73, 0x2f, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2f,
	0x61, 0x67, 0x65, 0x6e, 0x74, 0x5f, 0x69, 0x64, 0x5f, 0x72, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73,
	0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x0d, 0x73, 0x61, 0x61, 0x73, 0x2e, 0x76, 0x31,
	0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x1a, 0x23, 0x73, 0x61, 0x61, 0x73, 0x2f, 0x76, 0x31, 0x61,
	0x6c, 0x70, 0x68, 0x61, 0x31, 0x2f, 0x72, 0x65, 0x70, 0x6f, 0x5f, 0x76, 0x65, 0x72, 0x73, 0x69,
	0x6f, 0x6e, 0x5f, 0x69, 0x64, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x77, 0x0a, 0x17, 0x41,
	0x67, 0x65, 0x6e, 0x74, 0x49, 0x64, 0x65, 0x6e, 0x74, 0x69, 0x66, 0x69, 0x65, 0x72, 0x52, 0x65,
	0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x44, 0x0a, 0x0f, 0x72, 0x65, 0x70, 0x6f, 0x5f, 0x76,
	0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x5f, 0x69, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x1c, 0x2e, 0x73, 0x61, 0x61, 0x73, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e,
	0x52, 0x65, 0x70, 0x6f, 0x56, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x49, 0x64, 0x52, 0x0d, 0x72,
	0x65, 0x70, 0x6f, 0x56, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x49, 0x64, 0x12, 0x16, 0x0a, 0x06,
	0x64, 0x69, 0x67, 0x65, 0x73, 0x74, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x64, 0x69,
	0x67, 0x65, 0x73, 0x74, 0x42, 0xac, 0x01, 0x0a, 0x11, 0x63, 0x6f, 0x6d, 0x2e, 0x73, 0x61, 0x61,
	0x73, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x42, 0x14, 0x41, 0x67, 0x65, 0x6e,
	0x74, 0x49, 0x64, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x50, 0x72, 0x6f, 0x74, 0x6f,
	0x50, 0x01, 0x5a, 0x2c, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x63,
	0x69, 0x73, 0x63, 0x6f, 0x2d, 0x65, 0x74, 0x69, 0x2f, 0x61, 0x67, 0x6e, 0x74, 0x63, 0x79, 0x2f,
	0x61, 0x70, 0x69, 0x2f, 0x68, 0x75, 0x62, 0x2f, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31,
	0xa2, 0x02, 0x03, 0x53, 0x58, 0x58, 0xaa, 0x02, 0x0d, 0x53, 0x61, 0x61, 0x73, 0x2e, 0x56, 0x31,
	0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0xca, 0x02, 0x0d, 0x53, 0x61, 0x61, 0x73, 0x5c, 0x56, 0x31,
	0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0xe2, 0x02, 0x19, 0x53, 0x61, 0x61, 0x73, 0x5c, 0x56, 0x31,
	0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x5c, 0x47, 0x50, 0x42, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61,
	0x74, 0x61, 0xea, 0x02, 0x0e, 0x53, 0x61, 0x61, 0x73, 0x3a, 0x3a, 0x56, 0x31, 0x61, 0x6c, 0x70,
	0x68, 0x61, 0x31, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
})

var (
	file_saas_v1alpha1_agent_id_response_proto_rawDescOnce sync.Once
	file_saas_v1alpha1_agent_id_response_proto_rawDescData []byte
)

func file_saas_v1alpha1_agent_id_response_proto_rawDescGZIP() []byte {
	file_saas_v1alpha1_agent_id_response_proto_rawDescOnce.Do(func() {
		file_saas_v1alpha1_agent_id_response_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_saas_v1alpha1_agent_id_response_proto_rawDesc), len(file_saas_v1alpha1_agent_id_response_proto_rawDesc)))
	})
	return file_saas_v1alpha1_agent_id_response_proto_rawDescData
}

var file_saas_v1alpha1_agent_id_response_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_saas_v1alpha1_agent_id_response_proto_goTypes = []any{
	(*AgentIdentifierResponse)(nil), // 0: saas.v1alpha1.AgentIdentifierResponse
	(*RepoVersionId)(nil),           // 1: saas.v1alpha1.RepoVersionId
}
var file_saas_v1alpha1_agent_id_response_proto_depIdxs = []int32{
	1, // 0: saas.v1alpha1.AgentIdentifierResponse.repo_version_id:type_name -> saas.v1alpha1.RepoVersionId
	1, // [1:1] is the sub-list for method output_type
	1, // [1:1] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_saas_v1alpha1_agent_id_response_proto_init() }
func file_saas_v1alpha1_agent_id_response_proto_init() {
	if File_saas_v1alpha1_agent_id_response_proto != nil {
		return
	}
	file_saas_v1alpha1_repo_version_id_proto_init()
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_saas_v1alpha1_agent_id_response_proto_rawDesc), len(file_saas_v1alpha1_agent_id_response_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_saas_v1alpha1_agent_id_response_proto_goTypes,
		DependencyIndexes: file_saas_v1alpha1_agent_id_response_proto_depIdxs,
		MessageInfos:      file_saas_v1alpha1_agent_id_response_proto_msgTypes,
	}.Build()
	File_saas_v1alpha1_agent_id_response_proto = out.File
	file_saas_v1alpha1_agent_id_response_proto_goTypes = nil
	file_saas_v1alpha1_agent_id_response_proto_depIdxs = nil
}
