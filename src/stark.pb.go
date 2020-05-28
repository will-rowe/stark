//  STARK-DB schema
//  ------------------
//
//  Projects:
//    *
//
//  Samples:
//    * samples are the biological entities that sequencing experiments are based on
//    * any number of samples can be used in a project
//    * samples can be used across multiple projects
//
//  Libraries:
//    * a library is made from one Sample and is to be used on one sequencing platform
//    * a library contains information such as platform, kit etc.
//    * libraries must be aliquoted before being run on a sequencer
//    * multiple library aliquots can be pooled and run together on the same sequencer
//

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.23.0
// 	protoc        v3.11.4
// source: stark.proto

package stark

import (
	proto "github.com/golang/protobuf/proto"
	timestamp "github.com/golang/protobuf/ptypes/timestamp"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// This is a compile-time assertion that a sufficiently up-to-date version
// of the legacy proto package is being used.
const _ = proto.ProtoPackageIsVersion4

//
//Status is currently un-used.
type Status int32

const (
	Status_UN_INITIALIZED Status = 0
	Status_untagged       Status = 1
	Status_tagged         Status = 2
)

// Enum value maps for Status.
var (
	Status_name = map[int32]string{
		0: "UN_INITIALIZED",
		1: "untagged",
		2: "tagged",
	}
	Status_value = map[string]int32{
		"UN_INITIALIZED": 0,
		"untagged":       1,
		"tagged":         2,
	}
)

func (x Status) Enum() *Status {
	p := new(Status)
	*p = x
	return p
}

func (x Status) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (Status) Descriptor() protoreflect.EnumDescriptor {
	return file_stark_proto_enumTypes[0].Descriptor()
}

func (Status) Type() protoreflect.EnumType {
	return &file_stark_proto_enumTypes[0]
}

func (x Status) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use Status.Descriptor instead.
func (Status) EnumDescriptor() ([]byte, []int) {
	return file_stark_proto_rawDescGZIP(), []int{0}
}

//
//Comments are used to describe message history.
type Comment struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Timestamp   *timestamp.Timestamp `protobuf:"bytes,1,opt,name=timestamp,proto3" json:"timestamp,omitempty"`     // timestamp for change
	Text        string               `protobuf:"bytes,2,opt,name=text,proto3" json:"text,omitempty"`               // description of the change
	PreviousCID string               `protobuf:"bytes,3,opt,name=previousCID,proto3" json:"previousCID,omitempty"` // used to rollback the Record and undo the commented change
}

func (x *Comment) Reset() {
	*x = Comment{}
	if protoimpl.UnsafeEnabled {
		mi := &file_stark_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Comment) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Comment) ProtoMessage() {}

func (x *Comment) ProtoReflect() protoreflect.Message {
	mi := &file_stark_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Comment.ProtoReflect.Descriptor instead.
func (*Comment) Descriptor() ([]byte, []int) {
	return file_stark_proto_rawDescGZIP(), []int{0}
}

func (x *Comment) GetTimestamp() *timestamp.Timestamp {
	if x != nil {
		return x.Timestamp
	}
	return nil
}

func (x *Comment) GetText() string {
	if x != nil {
		return x.Text
	}
	return ""
}

func (x *Comment) GetPreviousCID() string {
	if x != nil {
		return x.PreviousCID
	}
	return ""
}

//
//Project is used to aggregate Nanopore sequencing data.
//A single project is used per STARKDB; where the project is used to describe database metadata.
type Project struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// reserved:
	Uuid    string     `protobuf:"bytes,1,opt,name=uuid,proto3" json:"uuid,omitempty"`       // universally unique id
	History []*Comment `protobuf:"bytes,2,rep,name=history,proto3" json:"history,omitempty"` // describes the history of the project - can be used to get timestamps for creation and last updated
	// user updateable:
	Alias       string            `protobuf:"bytes,5,opt,name=alias,proto3" json:"alias,omitempty"`                                                                                                 // the project name / human readable id
	Description string            `protobuf:"bytes,6,opt,name=description,proto3" json:"description,omitempty"`                                                                                     // a short description of the project
	Samples     map[string]string `protobuf:"bytes,7,rep,name=samples,proto3" json:"samples,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`     // all samples used in this project (map links sample UUIDs to a metadata location)
	Libraries   map[string]string `protobuf:"bytes,8,rep,name=libraries,proto3" json:"libraries,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"` // all libraries used in this project (map links library UUIDs to a metadata location)
}

func (x *Project) Reset() {
	*x = Project{}
	if protoimpl.UnsafeEnabled {
		mi := &file_stark_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Project) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Project) ProtoMessage() {}

func (x *Project) ProtoReflect() protoreflect.Message {
	mi := &file_stark_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Project.ProtoReflect.Descriptor instead.
func (*Project) Descriptor() ([]byte, []int) {
	return file_stark_proto_rawDescGZIP(), []int{1}
}

func (x *Project) GetUuid() string {
	if x != nil {
		return x.Uuid
	}
	return ""
}

func (x *Project) GetHistory() []*Comment {
	if x != nil {
		return x.History
	}
	return nil
}

func (x *Project) GetAlias() string {
	if x != nil {
		return x.Alias
	}
	return ""
}

func (x *Project) GetDescription() string {
	if x != nil {
		return x.Description
	}
	return ""
}

func (x *Project) GetSamples() map[string]string {
	if x != nil {
		return x.Samples
	}
	return nil
}

func (x *Project) GetLibraries() map[string]string {
	if x != nil {
		return x.Libraries
	}
	return nil
}

//
//Record describes a Nanopore run, linking it to parent samples and libraries.
type Record struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// reserved:
	Uuid                  string     `protobuf:"bytes,1,opt,name=uuid,proto3" json:"uuid,omitempty"`                                   // universally unique id for the record
	PreviousCID           string     `protobuf:"bytes,2,opt,name=previousCID,proto3" json:"previousCID,omitempty"`                     // the last known CID this record was pulled from
	History               []*Comment `protobuf:"bytes,3,rep,name=history,proto3" json:"history,omitempty"`                             // describes the history of the record - can be used to get timestamps for creation and last updated
	Encrypted             bool       `protobuf:"varint,4,opt,name=encrypted,proto3" json:"encrypted,omitempty"`                        // set true to indicate if fields have been encrypted
	SequencerOutputDirCID string     `protobuf:"bytes,5,opt,name=sequencerOutputDirCID,proto3" json:"sequencerOutputDirCID,omitempty"` // the CID of the output directory (field 9)
	Status                Status     `protobuf:"varint,6,opt,name=status,proto3,enum=stark.Status" json:"status,omitempty"`            // describes if untagged/tagged/etc.
	// user updateable:
	Alias                   string            `protobuf:"bytes,7,opt,name=alias,proto3" json:"alias,omitempty"`                                                                                                              // the record name / human readable id
	Description             string            `protobuf:"bytes,8,opt,name=description,proto3" json:"description,omitempty"`                                                                                                  // a short description of the record
	LocalSequencerOutputDir string            `protobuf:"bytes,9,opt,name=localSequencerOutputDir,proto3" json:"localSequencerOutputDir,omitempty"`                                                                          // where the sequencer is outputing data for this record
	LinkedSamples           map[string]string `protobuf:"bytes,10,rep,name=linkedSamples,proto3" json:"linkedSamples,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`     // all samples linked to this record (map relates sample UUIDs to a metadata location)
	LinkedLibraries         map[string]string `protobuf:"bytes,11,rep,name=linkedLibraries,proto3" json:"linkedLibraries,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"` // all libraries linked to this record (map relates library UUIDs to a metadata location)
	Barcodes                map[string]int32  `protobuf:"bytes,12,rep,name=barcodes,proto3" json:"barcodes,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"varint,2,opt,name=value,proto3"`              // all barcodes used by this record (map links library UUID to barcode for that library)
}

func (x *Record) Reset() {
	*x = Record{}
	if protoimpl.UnsafeEnabled {
		mi := &file_stark_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Record) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Record) ProtoMessage() {}

func (x *Record) ProtoReflect() protoreflect.Message {
	mi := &file_stark_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Record.ProtoReflect.Descriptor instead.
func (*Record) Descriptor() ([]byte, []int) {
	return file_stark_proto_rawDescGZIP(), []int{2}
}

func (x *Record) GetUuid() string {
	if x != nil {
		return x.Uuid
	}
	return ""
}

func (x *Record) GetPreviousCID() string {
	if x != nil {
		return x.PreviousCID
	}
	return ""
}

func (x *Record) GetHistory() []*Comment {
	if x != nil {
		return x.History
	}
	return nil
}

func (x *Record) GetEncrypted() bool {
	if x != nil {
		return x.Encrypted
	}
	return false
}

func (x *Record) GetSequencerOutputDirCID() string {
	if x != nil {
		return x.SequencerOutputDirCID
	}
	return ""
}

func (x *Record) GetStatus() Status {
	if x != nil {
		return x.Status
	}
	return Status_UN_INITIALIZED
}

func (x *Record) GetAlias() string {
	if x != nil {
		return x.Alias
	}
	return ""
}

func (x *Record) GetDescription() string {
	if x != nil {
		return x.Description
	}
	return ""
}

func (x *Record) GetLocalSequencerOutputDir() string {
	if x != nil {
		return x.LocalSequencerOutputDir
	}
	return ""
}

func (x *Record) GetLinkedSamples() map[string]string {
	if x != nil {
		return x.LinkedSamples
	}
	return nil
}

func (x *Record) GetLinkedLibraries() map[string]string {
	if x != nil {
		return x.LinkedLibraries
	}
	return nil
}

func (x *Record) GetBarcodes() map[string]int32 {
	if x != nil {
		return x.Barcodes
	}
	return nil
}

var File_stark_proto protoreflect.FileDescriptor

var file_stark_proto_rawDesc = []byte{
	0x0a, 0x0b, 0x73, 0x74, 0x61, 0x72, 0x6b, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x05, 0x73,
	0x74, 0x61, 0x72, 0x6b, 0x1a, 0x1f, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x79, 0x0a, 0x07, 0x43, 0x6f, 0x6d, 0x6d, 0x65, 0x6e, 0x74,
	0x12, 0x38, 0x0a, 0x09, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x52,
	0x09, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x12, 0x12, 0x0a, 0x04, 0x74, 0x65,
	0x78, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x74, 0x65, 0x78, 0x74, 0x12, 0x20,
	0x0a, 0x0b, 0x70, 0x72, 0x65, 0x76, 0x69, 0x6f, 0x75, 0x73, 0x43, 0x49, 0x44, 0x18, 0x03, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x0b, 0x70, 0x72, 0x65, 0x76, 0x69, 0x6f, 0x75, 0x73, 0x43, 0x49, 0x44,
	0x22, 0xed, 0x02, 0x0a, 0x07, 0x50, 0x72, 0x6f, 0x6a, 0x65, 0x63, 0x74, 0x12, 0x12, 0x0a, 0x04,
	0x75, 0x75, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x75, 0x75, 0x69, 0x64,
	0x12, 0x28, 0x0a, 0x07, 0x68, 0x69, 0x73, 0x74, 0x6f, 0x72, 0x79, 0x18, 0x02, 0x20, 0x03, 0x28,
	0x0b, 0x32, 0x0e, 0x2e, 0x73, 0x74, 0x61, 0x72, 0x6b, 0x2e, 0x43, 0x6f, 0x6d, 0x6d, 0x65, 0x6e,
	0x74, 0x52, 0x07, 0x68, 0x69, 0x73, 0x74, 0x6f, 0x72, 0x79, 0x12, 0x14, 0x0a, 0x05, 0x61, 0x6c,
	0x69, 0x61, 0x73, 0x18, 0x05, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x61, 0x6c, 0x69, 0x61, 0x73,
	0x12, 0x20, 0x0a, 0x0b, 0x64, 0x65, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x18,
	0x06, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0b, 0x64, 0x65, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x69,
	0x6f, 0x6e, 0x12, 0x35, 0x0a, 0x07, 0x73, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x73, 0x18, 0x07, 0x20,
	0x03, 0x28, 0x0b, 0x32, 0x1b, 0x2e, 0x73, 0x74, 0x61, 0x72, 0x6b, 0x2e, 0x50, 0x72, 0x6f, 0x6a,
	0x65, 0x63, 0x74, 0x2e, 0x53, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79,
	0x52, 0x07, 0x73, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x73, 0x12, 0x3b, 0x0a, 0x09, 0x6c, 0x69, 0x62,
	0x72, 0x61, 0x72, 0x69, 0x65, 0x73, 0x18, 0x08, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x1d, 0x2e, 0x73,
	0x74, 0x61, 0x72, 0x6b, 0x2e, 0x50, 0x72, 0x6f, 0x6a, 0x65, 0x63, 0x74, 0x2e, 0x4c, 0x69, 0x62,
	0x72, 0x61, 0x72, 0x69, 0x65, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x09, 0x6c, 0x69, 0x62,
	0x72, 0x61, 0x72, 0x69, 0x65, 0x73, 0x1a, 0x3a, 0x0a, 0x0c, 0x53, 0x61, 0x6d, 0x70, 0x6c, 0x65,
	0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75,
	0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02,
	0x38, 0x01, 0x1a, 0x3c, 0x0a, 0x0e, 0x4c, 0x69, 0x62, 0x72, 0x61, 0x72, 0x69, 0x65, 0x73, 0x45,
	0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01,
	0x22, 0xe7, 0x05, 0x0a, 0x06, 0x52, 0x65, 0x63, 0x6f, 0x72, 0x64, 0x12, 0x12, 0x0a, 0x04, 0x75,
	0x75, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x75, 0x75, 0x69, 0x64, 0x12,
	0x20, 0x0a, 0x0b, 0x70, 0x72, 0x65, 0x76, 0x69, 0x6f, 0x75, 0x73, 0x43, 0x49, 0x44, 0x18, 0x02,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x0b, 0x70, 0x72, 0x65, 0x76, 0x69, 0x6f, 0x75, 0x73, 0x43, 0x49,
	0x44, 0x12, 0x28, 0x0a, 0x07, 0x68, 0x69, 0x73, 0x74, 0x6f, 0x72, 0x79, 0x18, 0x03, 0x20, 0x03,
	0x28, 0x0b, 0x32, 0x0e, 0x2e, 0x73, 0x74, 0x61, 0x72, 0x6b, 0x2e, 0x43, 0x6f, 0x6d, 0x6d, 0x65,
	0x6e, 0x74, 0x52, 0x07, 0x68, 0x69, 0x73, 0x74, 0x6f, 0x72, 0x79, 0x12, 0x1c, 0x0a, 0x09, 0x65,
	0x6e, 0x63, 0x72, 0x79, 0x70, 0x74, 0x65, 0x64, 0x18, 0x04, 0x20, 0x01, 0x28, 0x08, 0x52, 0x09,
	0x65, 0x6e, 0x63, 0x72, 0x79, 0x70, 0x74, 0x65, 0x64, 0x12, 0x34, 0x0a, 0x15, 0x73, 0x65, 0x71,
	0x75, 0x65, 0x6e, 0x63, 0x65, 0x72, 0x4f, 0x75, 0x74, 0x70, 0x75, 0x74, 0x44, 0x69, 0x72, 0x43,
	0x49, 0x44, 0x18, 0x05, 0x20, 0x01, 0x28, 0x09, 0x52, 0x15, 0x73, 0x65, 0x71, 0x75, 0x65, 0x6e,
	0x63, 0x65, 0x72, 0x4f, 0x75, 0x74, 0x70, 0x75, 0x74, 0x44, 0x69, 0x72, 0x43, 0x49, 0x44, 0x12,
	0x25, 0x0a, 0x06, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x18, 0x06, 0x20, 0x01, 0x28, 0x0e, 0x32,
	0x0d, 0x2e, 0x73, 0x74, 0x61, 0x72, 0x6b, 0x2e, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x52, 0x06,
	0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x12, 0x14, 0x0a, 0x05, 0x61, 0x6c, 0x69, 0x61, 0x73, 0x18,
	0x07, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x61, 0x6c, 0x69, 0x61, 0x73, 0x12, 0x20, 0x0a, 0x0b,
	0x64, 0x65, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x08, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x0b, 0x64, 0x65, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x38,
	0x0a, 0x17, 0x6c, 0x6f, 0x63, 0x61, 0x6c, 0x53, 0x65, 0x71, 0x75, 0x65, 0x6e, 0x63, 0x65, 0x72,
	0x4f, 0x75, 0x74, 0x70, 0x75, 0x74, 0x44, 0x69, 0x72, 0x18, 0x09, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x17, 0x6c, 0x6f, 0x63, 0x61, 0x6c, 0x53, 0x65, 0x71, 0x75, 0x65, 0x6e, 0x63, 0x65, 0x72, 0x4f,
	0x75, 0x74, 0x70, 0x75, 0x74, 0x44, 0x69, 0x72, 0x12, 0x46, 0x0a, 0x0d, 0x6c, 0x69, 0x6e, 0x6b,
	0x65, 0x64, 0x53, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x73, 0x18, 0x0a, 0x20, 0x03, 0x28, 0x0b, 0x32,
	0x20, 0x2e, 0x73, 0x74, 0x61, 0x72, 0x6b, 0x2e, 0x52, 0x65, 0x63, 0x6f, 0x72, 0x64, 0x2e, 0x4c,
	0x69, 0x6e, 0x6b, 0x65, 0x64, 0x53, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x73, 0x45, 0x6e, 0x74, 0x72,
	0x79, 0x52, 0x0d, 0x6c, 0x69, 0x6e, 0x6b, 0x65, 0x64, 0x53, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x73,
	0x12, 0x4c, 0x0a, 0x0f, 0x6c, 0x69, 0x6e, 0x6b, 0x65, 0x64, 0x4c, 0x69, 0x62, 0x72, 0x61, 0x72,
	0x69, 0x65, 0x73, 0x18, 0x0b, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x22, 0x2e, 0x73, 0x74, 0x61, 0x72,
	0x6b, 0x2e, 0x52, 0x65, 0x63, 0x6f, 0x72, 0x64, 0x2e, 0x4c, 0x69, 0x6e, 0x6b, 0x65, 0x64, 0x4c,
	0x69, 0x62, 0x72, 0x61, 0x72, 0x69, 0x65, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x0f, 0x6c,
	0x69, 0x6e, 0x6b, 0x65, 0x64, 0x4c, 0x69, 0x62, 0x72, 0x61, 0x72, 0x69, 0x65, 0x73, 0x12, 0x37,
	0x0a, 0x08, 0x62, 0x61, 0x72, 0x63, 0x6f, 0x64, 0x65, 0x73, 0x18, 0x0c, 0x20, 0x03, 0x28, 0x0b,
	0x32, 0x1b, 0x2e, 0x73, 0x74, 0x61, 0x72, 0x6b, 0x2e, 0x52, 0x65, 0x63, 0x6f, 0x72, 0x64, 0x2e,
	0x42, 0x61, 0x72, 0x63, 0x6f, 0x64, 0x65, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x08, 0x62,
	0x61, 0x72, 0x63, 0x6f, 0x64, 0x65, 0x73, 0x1a, 0x40, 0x0a, 0x12, 0x4c, 0x69, 0x6e, 0x6b, 0x65,
	0x64, 0x53, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a,
	0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12,
	0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05,
	0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x1a, 0x42, 0x0a, 0x14, 0x4c, 0x69, 0x6e,
	0x6b, 0x65, 0x64, 0x4c, 0x69, 0x62, 0x72, 0x61, 0x72, 0x69, 0x65, 0x73, 0x45, 0x6e, 0x74, 0x72,
	0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03,
	0x6b, 0x65, 0x79, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x1a, 0x3b, 0x0a,
	0x0d, 0x42, 0x61, 0x72, 0x63, 0x6f, 0x64, 0x65, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10,
	0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79,
	0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x05, 0x52,
	0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x2a, 0x36, 0x0a, 0x06, 0x53, 0x74,
	0x61, 0x74, 0x75, 0x73, 0x12, 0x12, 0x0a, 0x0e, 0x55, 0x4e, 0x5f, 0x49, 0x4e, 0x49, 0x54, 0x49,
	0x41, 0x4c, 0x49, 0x5a, 0x45, 0x44, 0x10, 0x00, 0x12, 0x0c, 0x0a, 0x08, 0x75, 0x6e, 0x74, 0x61,
	0x67, 0x67, 0x65, 0x64, 0x10, 0x01, 0x12, 0x0a, 0x0a, 0x06, 0x74, 0x61, 0x67, 0x67, 0x65, 0x64,
	0x10, 0x02, 0x42, 0x09, 0x5a, 0x07, 0x2e, 0x3b, 0x73, 0x74, 0x61, 0x72, 0x6b, 0x62, 0x06, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_stark_proto_rawDescOnce sync.Once
	file_stark_proto_rawDescData = file_stark_proto_rawDesc
)

func file_stark_proto_rawDescGZIP() []byte {
	file_stark_proto_rawDescOnce.Do(func() {
		file_stark_proto_rawDescData = protoimpl.X.CompressGZIP(file_stark_proto_rawDescData)
	})
	return file_stark_proto_rawDescData
}

var file_stark_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_stark_proto_msgTypes = make([]protoimpl.MessageInfo, 8)
var file_stark_proto_goTypes = []interface{}{
	(Status)(0),                 // 0: stark.Status
	(*Comment)(nil),             // 1: stark.Comment
	(*Project)(nil),             // 2: stark.Project
	(*Record)(nil),              // 3: stark.Record
	nil,                         // 4: stark.Project.SamplesEntry
	nil,                         // 5: stark.Project.LibrariesEntry
	nil,                         // 6: stark.Record.LinkedSamplesEntry
	nil,                         // 7: stark.Record.LinkedLibrariesEntry
	nil,                         // 8: stark.Record.BarcodesEntry
	(*timestamp.Timestamp)(nil), // 9: google.protobuf.Timestamp
}
var file_stark_proto_depIdxs = []int32{
	9, // 0: stark.Comment.timestamp:type_name -> google.protobuf.Timestamp
	1, // 1: stark.Project.history:type_name -> stark.Comment
	4, // 2: stark.Project.samples:type_name -> stark.Project.SamplesEntry
	5, // 3: stark.Project.libraries:type_name -> stark.Project.LibrariesEntry
	1, // 4: stark.Record.history:type_name -> stark.Comment
	0, // 5: stark.Record.status:type_name -> stark.Status
	6, // 6: stark.Record.linkedSamples:type_name -> stark.Record.LinkedSamplesEntry
	7, // 7: stark.Record.linkedLibraries:type_name -> stark.Record.LinkedLibrariesEntry
	8, // 8: stark.Record.barcodes:type_name -> stark.Record.BarcodesEntry
	9, // [9:9] is the sub-list for method output_type
	9, // [9:9] is the sub-list for method input_type
	9, // [9:9] is the sub-list for extension type_name
	9, // [9:9] is the sub-list for extension extendee
	0, // [0:9] is the sub-list for field type_name
}

func init() { file_stark_proto_init() }
func file_stark_proto_init() {
	if File_stark_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_stark_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Comment); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_stark_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Project); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_stark_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Record); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_stark_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   8,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_stark_proto_goTypes,
		DependencyIndexes: file_stark_proto_depIdxs,
		EnumInfos:         file_stark_proto_enumTypes,
		MessageInfos:      file_stark_proto_msgTypes,
	}.Build()
	File_stark_proto = out.File
	file_stark_proto_rawDesc = nil
	file_stark_proto_goTypes = nil
	file_stark_proto_depIdxs = nil
}
