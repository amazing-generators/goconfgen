package semantic

import "github.com/amazing-generators/goconfgen/internal/schema"

// // // // // // // // // //

// FieldKindObj и его константы — алиасы на единый источник в schema, чтобы у пакета был
// один набор тегов поля без параллельной классификации.
type FieldKindObj = schema.FieldKindObj

const (
	FieldKindString             = schema.FieldKindString
	FieldKindBool               = schema.FieldKindBool
	FieldKindIntSigned          = schema.FieldKindIntSigned
	FieldKindIntUnsigned        = schema.FieldKindIntUnsigned
	FieldKindFloat              = schema.FieldKindFloat
	FieldKindDuration           = schema.FieldKindDuration
	FieldKindSize               = schema.FieldKindSize
	FieldKindEnumScalar         = schema.FieldKindEnumScalar
	FieldKindArrayString        = schema.FieldKindArrayString
	FieldKindArrayBool          = schema.FieldKindArrayBool
	FieldKindArrayIntSigned     = schema.FieldKindArrayIntSigned
	FieldKindArrayIntUnsigned   = schema.FieldKindArrayIntUnsigned
	FieldKindArrayFloat         = schema.FieldKindArrayFloat
	FieldKindArrayDuration      = schema.FieldKindArrayDuration
	FieldKindArraySize          = schema.FieldKindArraySize
	FieldKindArrayEnum          = schema.FieldKindArrayEnum
	FieldKindMapStringString    = schema.FieldKindMapStringString
	FieldKindMapStringStringArr = schema.FieldKindMapStringStringArr
	FieldKindMapStringAny       = schema.FieldKindMapStringAny
)

// //

// ClassifyKind читает kind и разрядность поля прямо из дескриптора типа.
func ClassifyKind(typeObj schema.TypeObj) (FieldKindObj, int) {
	return typeObj.Kind, typeObj.BitSize
}
