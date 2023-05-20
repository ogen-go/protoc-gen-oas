package gen

import (
	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func isFieldRequired(opts protoreflect.ProtoMessage) bool {
	return isFieldBehaviorIndicator(opts, annotations.FieldBehavior_REQUIRED)
}

func isFieldBehaviorIndicator(opts protoreflect.ProtoMessage, indicator annotations.FieldBehavior) bool {
	fieldBehaviors, ok := proto.GetExtension(opts, annotations.E_FieldBehavior).([]annotations.FieldBehavior)
	if !ok {
		return false
	}

	for _, fieldBehavior := range fieldBehaviors {
		if fieldBehavior == indicator {
			return true
		}
	}

	return false
}
