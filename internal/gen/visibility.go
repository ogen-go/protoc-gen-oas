package gen

import (
	"slices"
	"strings"

	"google.golang.org/genproto/googleapis/api/visibility"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func isInternalField(opts protoreflect.ProtoMessage) bool {
	return isFieldVisibilityIndicator(opts, "INTERNAL")
}

func isPreviewField(opts protoreflect.ProtoMessage) bool {
	return isFieldVisibilityIndicator(opts, "PREVIEW")
}

func isFieldVisibilityIndicator(opts protoreflect.ProtoMessage, restriction string) bool {
	return isVisibilityIndicator(opts, visibility.E_FieldVisibility, restriction)
}

func isInternalMessage(opts protoreflect.ProtoMessage) bool {
	return isMessageVisibilityIndicator(opts, "INTERNAL")
}

func isMessageVisibilityIndicator(opts protoreflect.ProtoMessage, restriction string) bool {
	return isVisibilityIndicator(opts, visibility.E_MessageVisibility, restriction)
}

func isVisibilityIndicator(opts protoreflect.ProtoMessage, ext protoreflect.ExtensionType, restriction string) bool {
	fieldInfo, ok := proto.GetExtension(opts, ext).(*visibility.VisibilityRule)
	if !ok || fieldInfo == nil {
		return false
	}

	restrictions := strings.Split(fieldInfo.Restriction, ",")
	for i := range restrictions {
		restrictions[i] = strings.TrimSpace(restrictions[i])
	}
	return slices.Contains(restrictions, restriction)
}
