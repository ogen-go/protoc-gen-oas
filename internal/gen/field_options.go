package gen

import (
	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// NewFieldOptions returns FieldOptions instance.
func NewFieldOptions(opts protoreflect.ProtoMessage) *FieldOptions {
	ext := proto.GetExtension(opts, annotations.E_FieldBehavior)
	fieldBehaviors, ok := ext.([]annotations.FieldBehavior)
	if !ok || fieldBehaviors == nil {
		return &FieldOptions{}
	}

	isRequired := false
	isOptional := false
	isOutputOnly := false
	isInputOnly := false
	isImmutable := false
	isUnorderedList := false
	isNonEmptyDefault := false

	for _, fieldBehavior := range fieldBehaviors {
		switch fieldBehavior {
		case annotations.FieldBehavior_OPTIONAL:
			isOptional = true

		case annotations.FieldBehavior_REQUIRED:
			isRequired = true

		case annotations.FieldBehavior_OUTPUT_ONLY:
			isOutputOnly = true

		case annotations.FieldBehavior_INPUT_ONLY:
			isInputOnly = true

		case annotations.FieldBehavior_IMMUTABLE:
			isImmutable = true

		case annotations.FieldBehavior_UNORDERED_LIST:
			isUnorderedList = true

		case annotations.FieldBehavior_NON_EMPTY_DEFAULT:
			isNonEmptyDefault = true
		}
	}

	return &FieldOptions{
		IsOptional:        isOptional,
		IsRequired:        isRequired,
		IsOutputOnly:      isOutputOnly,
		IsInputOnly:       isInputOnly,
		IsImmutable:       isImmutable,
		IsUnorderedList:   isUnorderedList,
		IsNonEmptyDefault: isNonEmptyDefault,
	}
}

// FieldOptions instance.
type FieldOptions struct {
	IsOptional        bool
	IsRequired        bool
	IsOutputOnly      bool
	IsInputOnly       bool
	IsImmutable       bool
	IsUnorderedList   bool
	IsNonEmptyDefault bool
}
