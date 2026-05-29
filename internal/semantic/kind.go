package semantic

import "github.com/amazing-generators/goconfgen/internal/schema"

// // // // // // // // // //

type FieldKindObj uint8

const (
	FieldKindString FieldKindObj = iota
	FieldKindBool
	FieldKindIntSigned
	FieldKindIntUnsigned
	FieldKindFloat
	FieldKindDuration
	FieldKindSize
	FieldKindEnumScalar
	FieldKindArrayString
	FieldKindArrayBool
	FieldKindArrayIntSigned
	FieldKindArrayIntUnsigned
	FieldKindArrayFloat
	FieldKindArrayDuration
	FieldKindArraySize
	FieldKindArrayEnum
	FieldKindMapStringString
	FieldKindMapStringStringArr
	FieldKindMapStringAny
)

// //

func ClassifyKind(typeObj schema.TypeObj) (FieldKindObj, int) {
	switch typeObj.Family {
	case schema.TypeFamilyScalar:
		return classifyScalarKind(typeObj.SchemaText, typeObj)
	case schema.TypeFamilyArray:
		kindObj, bitSizeVal := classifyScalarKind(typeObj.ElemSchemaText, typeObj)
		switch kindObj {
		case FieldKindString:
			return FieldKindArrayString, bitSizeVal
		case FieldKindBool:
			return FieldKindArrayBool, bitSizeVal
		case FieldKindIntSigned:
			return FieldKindArrayIntSigned, bitSizeVal
		case FieldKindIntUnsigned:
			return FieldKindArrayIntUnsigned, bitSizeVal
		case FieldKindFloat:
			return FieldKindArrayFloat, bitSizeVal
		case FieldKindDuration:
			return FieldKindArrayDuration, bitSizeVal
		case FieldKindSize:
			return FieldKindArraySize, bitSizeVal
		case FieldKindEnumScalar:
			return FieldKindArrayEnum, bitSizeVal
		}
	case schema.TypeFamilyMap:
		switch typeObj.SchemaText {
		case "map[string]string":
			return FieldKindMapStringString, 0
		case "map[string][]string":
			return FieldKindMapStringStringArr, 0
		case "map[string]any":
			return FieldKindMapStringAny, 0
		}
	}

	return FieldKindString, 0
}

func classifyScalarKind(schemaText string, typeObj schema.TypeObj) (FieldKindObj, int) {
	if typeObj.IsEnum || schemaText == "enum" {
		return FieldKindEnumScalar, 0
	}
	if typeObj.IsDuration || schemaText == "duration" || schemaText == "time.Duration" {
		return FieldKindDuration, 64
	}
	if typeObj.IsSize || schemaText == "size" {
		return FieldKindSize, 64
	}

	switch schemaText {
	case "string":
		return FieldKindString, 0
	case "bool":
		return FieldKindBool, 0
	case "int":
		return FieldKindIntSigned, 0
	case "int8":
		return FieldKindIntSigned, 8
	case "int16":
		return FieldKindIntSigned, 16
	case "int32":
		return FieldKindIntSigned, 32
	case "int64":
		return FieldKindIntSigned, 64
	case "uint":
		return FieldKindIntUnsigned, 0
	case "uint8":
		return FieldKindIntUnsigned, 8
	case "uint16":
		return FieldKindIntUnsigned, 16
	case "uint32":
		return FieldKindIntUnsigned, 32
	case "uint64":
		return FieldKindIntUnsigned, 64
	case "float", "float32":
		return FieldKindFloat, 32
	case "float64":
		return FieldKindFloat, 64
	default:
		return FieldKindString, 0
	}
}
