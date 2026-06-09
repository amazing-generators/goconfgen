package emit

import (
	"strings"

	"github.com/amazing-generators/goconfgen/internal/semantic"
)

// // // // // // // // // //

// Генераторы тел рантайм-функций. Логика диспетчеризации по типу живёт здесь, в Go (под
// компилятором и тестами), а шаблоны лишь подставляют результат. go/format в emit.Build
// нормализует отступы, поэтому достаточно корректного набора инструкций.

func joinLines(lineArr ...string) string {
	return strings.Join(lineArr, "\n")
}

// renderFieldValue — тело renderValue<Path>: приводит значение поля к any для рендера.
func renderFieldValue(fieldObj *semantic.FieldObj) string {
	accessorText := "obj." + fieldObj.AccessorText

	switch fieldObj.KindObj {
	case semantic.FieldKindEnumScalar, semantic.FieldKindDuration, semantic.FieldKindSize:
		return "return " + accessorText + ".String(), nil"
	case semantic.FieldKindArrayEnum, semantic.FieldKindArrayDuration, semantic.FieldKindArraySize:
		return joinLines(
			"resultArr := make([]string, 0, len("+accessorText+"))",
			"for _, itemObj := range "+accessorText+" {",
			"resultArr = append(resultArr, itemObj.String())",
			"}",
			"return resultArr, nil",
		)
	case semantic.FieldKindMapStringAny:
		return "return normalizeAnyMap(" + accessorText + "), nil"
	default:
		return "return " + accessorText + ", nil"
	}
}

// renderFieldAssign — тело ветки applyMap для поля: приводит valueObj к типу поля и вызывает
// сеттер с originObj.
func renderFieldAssign(fieldObj *semantic.FieldObj) string {
	pathText := fieldObj.PathText
	setText := "obj." + setterName(fieldObj)
	goTypeText := fieldObj.GoTypeText
	bitText := bitSizeLiteral(fieldObj)

	unsupportedType := "return fmt.Errorf(\"field [" + pathText + "] unsupported type %T\", valueObj)"

	switch fieldObj.KindObj {
	case semantic.FieldKindEnumScalar:
		enumText := fieldObj.EnumObj.TypeName
		return joinLines(
			"switch typedValue := valueObj.(type) {",
			"case string:",
			"parsedValue, err := Parse"+enumText+"(strings.TrimSpace(typedValue))",
			"if err != nil {",
			"return fmt.Errorf(\"field ["+pathText+"]: %w\", err)",
			"}",
			setText+"(parsedValue, originObj)",
			"case "+enumText+":",
			setText+"(typedValue, originObj)",
			"default:",
			unsupportedType,
			"}",
		)
	case semantic.FieldKindArrayEnum:
		enumText := fieldObj.EnumObj.TypeName
		return joinLines(
			"typedValue, okTyped := valueObj.("+goTypeText+")",
			"if okTyped {",
			setText+"(typedValue, originObj)",
			"return nil",
			"}",
			"inputArr, ok := toAnySlice(valueObj)",
			"if !ok {",
			unsupportedType,
			"}",
			"resultArr := make("+goTypeText+", 0, len(inputArr))",
			"for itemIndex, itemValue := range inputArr {",
			"itemText, ok := itemValue.(string)",
			"if !ok {",
			"return fmt.Errorf(\"field ["+pathText+"][%d] must be enum string\", itemIndex)",
			"}",
			"parsedValue, err := Parse"+enumText+"(strings.TrimSpace(itemText))",
			"if err != nil {",
			"return fmt.Errorf(\"field ["+pathText+"][%d]: %w\", itemIndex, err)",
			"}",
			"resultArr = append(resultArr, parsedValue)",
			"}",
			setText+"(resultArr, originObj)",
		)
	case semantic.FieldKindDuration:
		return joinLines(
			"switch typedValue := valueObj.(type) {",
			"case time.Duration:",
			setText+"(typedValue, originObj)",
			"case string:",
			"parsedValue, err := time.ParseDuration(strings.TrimSpace(typedValue))",
			"if err != nil {",
			"return fmt.Errorf(\"field ["+pathText+"]: %w\", err)",
			"}",
			setText+"(parsedValue, originObj)",
			"default:",
			unsupportedType,
			"}",
		)
	case semantic.FieldKindSize:
		return joinLines(
			"switch typedValue := valueObj.(type) {",
			"case SizeObj:",
			setText+"(typedValue, originObj)",
			"case uint64:",
			setText+"(SizeObj(typedValue), originObj)",
			"case int:",
			"if typedValue < 0 {",
			"return fmt.Errorf(\"field ["+pathText+"] must be non-negative\")",
			"}",
			setText+"(SizeObj(typedValue), originObj)",
			"case int64:",
			"if typedValue < 0 {",
			"return fmt.Errorf(\"field ["+pathText+"] must be non-negative\")",
			"}",
			setText+"(SizeObj(typedValue), originObj)",
			"case float64:",
			"rawValue, ok := toUint64(typedValue)",
			"if !ok {",
			"return fmt.Errorf(\"field ["+pathText+"] must be non-negative integer\")",
			"}",
			setText+"(SizeObj(rawValue), originObj)",
			"case string:",
			"parsedValue, err := parseSize(strings.TrimSpace(typedValue))",
			"if err != nil {",
			"return fmt.Errorf(\"field ["+pathText+"]: %w\", err)",
			"}",
			setText+"(SizeObj(parsedValue), originObj)",
			"default:",
			unsupportedType,
			"}",
		)
	case semantic.FieldKindString:
		return joinLines(
			"typedValue, ok := valueObj.(string)",
			"if !ok {",
			unsupportedType,
			"}",
			setText+"(typedValue, originObj)",
		)
	case semantic.FieldKindBool:
		return joinLines(
			"typedValue, ok := valueObj.(bool)",
			"if !ok {",
			unsupportedType,
			"}",
			setText+"(typedValue, originObj)",
		)
	case semantic.FieldKindIntSigned:
		return joinLines(
			"rawValue, ok := toInt64(valueObj)",
			"if !ok || !validateSignedRange(rawValue, "+bitText+") {",
			"return fmt.Errorf(\"field ["+pathText+"] unsupported integer value %v\", valueObj)",
			"}",
			setText+"("+goTypeText+"(rawValue), originObj)",
		)
	case semantic.FieldKindIntUnsigned:
		return joinLines(
			"rawValue, ok := toUint64(valueObj)",
			"if !ok || !validateUnsignedRange(rawValue, "+bitText+") {",
			"return fmt.Errorf(\"field ["+pathText+"] unsupported unsigned integer value %v\", valueObj)",
			"}",
			setText+"("+goTypeText+"(rawValue), originObj)",
		)
	case semantic.FieldKindFloat:
		return joinLines(
			"rawValue, ok := toFloat64(valueObj)",
			"if !ok || !validateFloatRange(rawValue, "+bitText+") {",
			"return fmt.Errorf(\"field ["+pathText+"] unsupported float value %v\", valueObj)",
			"}",
			setText+"("+goTypeText+"(rawValue), originObj)",
		)
	case semantic.FieldKindArrayString, semantic.FieldKindArrayBool,
		semantic.FieldKindArrayIntSigned, semantic.FieldKindArrayIntUnsigned,
		semantic.FieldKindArrayFloat, semantic.FieldKindArrayDuration, semantic.FieldKindArraySize:
		return joinLines(
			"inputArr, ok := toAnySlice(valueObj)",
			"if !ok {",
			"typedValue, okTyped := valueObj.("+goTypeText+")",
			"if okTyped {",
			setText+"(typedValue, originObj)",
			"return nil",
			"}",
			unsupportedType,
			"}",
			renderFieldArrayDecode(fieldObj),
			setText+"(resultArr, originObj)",
		)
	case semantic.FieldKindMapStringString:
		return joinLines(
			"typedValue, ok := valueObj.(map[string]string)",
			"if ok {",
			setText+"(typedValue, originObj)",
			"return nil",
			"}",
			"anyMap, okAny := valueObj.(map[string]any)",
			"if !okAny {",
			unsupportedType,
			"}",
			"resultObj := make(map[string]string, len(anyMap))",
			"for keyText, itemValue := range anyMap {",
			"itemText, okItem := itemValue.(string)",
			"if !okItem {",
			"return fmt.Errorf(\"field ["+pathText+"][%s] must be string\", keyText)",
			"}",
			"resultObj[keyText] = itemText",
			"}",
			setText+"(resultObj, originObj)",
		)
	case semantic.FieldKindMapStringStringArr:
		return joinLines(
			"typedValue, ok := valueObj.(map[string][]string)",
			"if ok {",
			setText+"(typedValue, originObj)",
			"return nil",
			"}",
			"anyMap, okAny := valueObj.(map[string]any)",
			"if !okAny {",
			unsupportedType,
			"}",
			"resultObj := make(map[string][]string, len(anyMap))",
			"for keyText, itemValue := range anyMap {",
			"itemArr, okArr := toAnySlice(itemValue)",
			"if !okArr {",
			"return fmt.Errorf(\"field ["+pathText+"][%s] must be array\", keyText)",
			"}",
			"childArr := make([]string, 0, len(itemArr))",
			"for childIndex, childValue := range itemArr {",
			"childText, okChild := childValue.(string)",
			"if !okChild {",
			"return fmt.Errorf(\"field ["+pathText+"][%s][%d] must be string\", keyText, childIndex)",
			"}",
			"childArr = append(childArr, childText)",
			"}",
			"resultObj[keyText] = childArr",
			"}",
			setText+"(resultObj, originObj)",
		)
	case semantic.FieldKindMapStringAny:
		return joinLines(
			"typedValue, ok := valueObj.(map[string]any)",
			"if !ok {",
			unsupportedType,
			"}",
			setText+"(normalizeAnyMap(typedValue).(map[string]any), originObj)",
		)
	default:
		return "return fmt.Errorf(\"unsupported field type for [" + pathText + "]: " + fieldObj.TypeObj.SchemaText + "\")"
	}
}

// renderFieldArrayDecode — тело, определяющее resultArr из inputArr для массивных полей.
func renderFieldArrayDecode(fieldObj *semantic.FieldObj) string {
	pathText := fieldObj.PathText
	goTypeText := fieldObj.GoTypeText
	elemText := elemGoType(fieldObj)
	bitText := bitSizeLiteral(fieldObj)

	switch fieldObj.KindObj {
	case semantic.FieldKindArrayString:
		return joinLines(
			"resultArr := make([]string, 0, len(inputArr))",
			"for itemIndex, itemValue := range inputArr {",
			"itemText, okItem := itemValue.(string)",
			"if !okItem {",
			"return fmt.Errorf(\"field ["+pathText+"][%d] must be string\", itemIndex)",
			"}",
			"resultArr = append(resultArr, itemText)",
			"}",
		)
	case semantic.FieldKindArrayBool:
		return joinLines(
			"resultArr := make([]bool, 0, len(inputArr))",
			"for itemIndex, itemValue := range inputArr {",
			"itemFlag, okItem := itemValue.(bool)",
			"if !okItem {",
			"return fmt.Errorf(\"field ["+pathText+"][%d] must be bool\", itemIndex)",
			"}",
			"resultArr = append(resultArr, itemFlag)",
			"}",
		)
	case semantic.FieldKindArrayIntSigned:
		return joinLines(
			"resultArr := make("+goTypeText+", 0, len(inputArr))",
			"for itemIndex, itemValue := range inputArr {",
			"rawValue, okItem := toInt64(itemValue)",
			"if !okItem || !validateSignedRange(rawValue, "+bitText+") {",
			"return fmt.Errorf(\"field ["+pathText+"][%d] must be integer\", itemIndex)",
			"}",
			"resultArr = append(resultArr, "+elemText+"(rawValue))",
			"}",
		)
	case semantic.FieldKindArrayIntUnsigned:
		return joinLines(
			"resultArr := make("+goTypeText+", 0, len(inputArr))",
			"for itemIndex, itemValue := range inputArr {",
			"rawValue, okItem := toUint64(itemValue)",
			"if !okItem || !validateUnsignedRange(rawValue, "+bitText+") {",
			"return fmt.Errorf(\"field ["+pathText+"][%d] must be unsigned integer\", itemIndex)",
			"}",
			"resultArr = append(resultArr, "+elemText+"(rawValue))",
			"}",
		)
	case semantic.FieldKindArrayFloat:
		return joinLines(
			"resultArr := make("+goTypeText+", 0, len(inputArr))",
			"for itemIndex, itemValue := range inputArr {",
			"rawValue, okItem := toFloat64(itemValue)",
			"if !okItem || !validateFloatRange(rawValue, "+bitText+") {",
			"return fmt.Errorf(\"field ["+pathText+"][%d] must be float\", itemIndex)",
			"}",
			"resultArr = append(resultArr, "+elemText+"(rawValue))",
			"}",
		)
	case semantic.FieldKindArrayDuration:
		return joinLines(
			"resultArr := make([]time.Duration, 0, len(inputArr))",
			"for itemIndex, itemValue := range inputArr {",
			"switch typedItem := itemValue.(type) {",
			"case time.Duration:",
			"resultArr = append(resultArr, typedItem)",
			"case string:",
			"parsedValue, err := time.ParseDuration(strings.TrimSpace(typedItem))",
			"if err != nil {",
			"return fmt.Errorf(\"field ["+pathText+"][%d]: %w\", itemIndex, err)",
			"}",
			"resultArr = append(resultArr, parsedValue)",
			"default:",
			"return fmt.Errorf(\"field ["+pathText+"][%d] unsupported type %T\", itemIndex, itemValue)",
			"}",
			"}",
		)
	case semantic.FieldKindArraySize:
		return joinLines(
			"resultArr := make([]SizeObj, 0, len(inputArr))",
			"for itemIndex, itemValue := range inputArr {",
			"switch typedItem := itemValue.(type) {",
			"case SizeObj:",
			"resultArr = append(resultArr, typedItem)",
			"case uint64:",
			"resultArr = append(resultArr, SizeObj(typedItem))",
			"case int:",
			"if typedItem < 0 {",
			"return fmt.Errorf(\"field ["+pathText+"][%d] must be non-negative\", itemIndex)",
			"}",
			"resultArr = append(resultArr, SizeObj(typedItem))",
			"case float64:",
			"rawValue, okItem := toUint64(typedItem)",
			"if !okItem {",
			"return fmt.Errorf(\"field ["+pathText+"][%d] must be non-negative integer\", itemIndex)",
			"}",
			"resultArr = append(resultArr, SizeObj(rawValue))",
			"case string:",
			"parsedValue, err := parseSize(strings.TrimSpace(typedItem))",
			"if err != nil {",
			"return fmt.Errorf(\"field ["+pathText+"][%d]: %w\", itemIndex, err)",
			"}",
			"resultArr = append(resultArr, SizeObj(parsedValue))",
			"default:",
			"return fmt.Errorf(\"field ["+pathText+"][%d] unsupported type %T\", itemIndex, itemValue)",
			"}",
			"}",
		)
	default:
		return "return fmt.Errorf(\"unsupported array element type for [" + pathText + "]: " + fieldObj.TypeObj.ElemSchemaText + "\")"
	}
}
