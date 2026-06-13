package schema

// // // // // // // // // //

// FieldKindObj — единственный дескрипторный тег поля для рантайм-диспетчеризации.
// Хранится прямо в TypeObj (см. ParseType), чтобы не было параллельной классификации.
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

// arrayKindFor отображает скалярный kind элемента на kind массива.
func arrayKindFor(elemKind FieldKindObj) FieldKindObj {
	switch elemKind {
	case FieldKindString:
		return FieldKindArrayString
	case FieldKindBool:
		return FieldKindArrayBool
	case FieldKindIntSigned:
		return FieldKindArrayIntSigned
	case FieldKindIntUnsigned:
		return FieldKindArrayIntUnsigned
	case FieldKindFloat:
		return FieldKindArrayFloat
	case FieldKindDuration:
		return FieldKindArrayDuration
	case FieldKindSize:
		return FieldKindArraySize
	case FieldKindEnumScalar:
		return FieldKindArrayEnum
	default:
		return FieldKindArrayString
	}
}
