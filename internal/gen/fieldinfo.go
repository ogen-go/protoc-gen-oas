package gen

import (
	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func isFieldUUID4Format(opts protoreflect.ProtoMessage) bool {
	return isFieldInfoIndicator(opts, annotations.FieldInfo_UUID4)
}

func isFieldIPV4Format(opts protoreflect.ProtoMessage) bool {
	return isFieldInfoIndicator(opts, annotations.FieldInfo_IPV4)
}

func isFieldIPV6Format(opts protoreflect.ProtoMessage) bool {
	return isFieldInfoIndicator(opts, annotations.FieldInfo_IPV6)
}

func isFieldIPFormat(opts protoreflect.ProtoMessage) bool {
	return isFieldInfoIndicator(opts, annotations.FieldInfo_IPV4_OR_IPV6)
}

func isFieldInfoIndicator(opts protoreflect.ProtoMessage, indicator annotations.FieldInfo_Format) bool {
	fieldInfo, ok := proto.GetExtension(opts, annotations.E_FieldInfo).(*annotations.FieldInfo)
	if !ok || fieldInfo == nil {
		return false
	}

	return fieldInfo.Format == indicator
}
