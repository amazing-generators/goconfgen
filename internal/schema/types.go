package schema

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// // // // // // // // // //

var cScalarTypeMap = map[string]TypeObj{
	"string":        {SchemaText: "string", GoText: "string", Family: TypeFamilyScalar},
	"bool":          {SchemaText: "bool", GoText: "bool", Family: TypeFamilyScalar},
	"int":           {SchemaText: "int", GoText: "int", Family: TypeFamilyScalar, IsNumeric: true, IsSigned: true},
	"int8":          {SchemaText: "int8", GoText: "int8", Family: TypeFamilyScalar, IsNumeric: true, IsSigned: true},
	"int16":         {SchemaText: "int16", GoText: "int16", Family: TypeFamilyScalar, IsNumeric: true, IsSigned: true},
	"int32":         {SchemaText: "int32", GoText: "int32", Family: TypeFamilyScalar, IsNumeric: true, IsSigned: true},
	"int64":         {SchemaText: "int64", GoText: "int64", Family: TypeFamilyScalar, IsNumeric: true, IsSigned: true},
	"uint":          {SchemaText: "uint", GoText: "uint", Family: TypeFamilyScalar, IsNumeric: true},
	"uint8":         {SchemaText: "uint8", GoText: "uint8", Family: TypeFamilyScalar, IsNumeric: true},
	"uint16":        {SchemaText: "uint16", GoText: "uint16", Family: TypeFamilyScalar, IsNumeric: true},
	"uint32":        {SchemaText: "uint32", GoText: "uint32", Family: TypeFamilyScalar, IsNumeric: true},
	"uint64":        {SchemaText: "uint64", GoText: "uint64", Family: TypeFamilyScalar, IsNumeric: true},
	"float":         {SchemaText: "float", GoText: "float32", Family: TypeFamilyScalar, IsNumeric: true, IsSigned: true, IsFloat: true},
	"float32":       {SchemaText: "float32", GoText: "float32", Family: TypeFamilyScalar, IsNumeric: true, IsSigned: true, IsFloat: true},
	"float64":       {SchemaText: "float64", GoText: "float64", Family: TypeFamilyScalar, IsNumeric: true, IsSigned: true, IsFloat: true},
	"duration":      {SchemaText: "duration", GoText: "time.Duration", Family: TypeFamilyScalar, IsDuration: true, IsNumeric: true, IsSigned: true},
	"time.Duration": {SchemaText: "time.Duration", GoText: "time.Duration", Family: TypeFamilyScalar, IsDuration: true, IsNumeric: true, IsSigned: true},
	"size":          {SchemaText: "size", GoText: "SizeObj", Family: TypeFamilyScalar, IsSize: true, IsNumeric: true},
}

var cMapTypeMap = map[string]TypeObj{
	"map[string]string":   {SchemaText: "map[string]string", GoText: "map[string]string", Family: TypeFamilyMap, MapValueText: "string"},
	"map[string]any":      {SchemaText: "map[string]any", GoText: "map[string]any", Family: TypeFamilyMap, MapValueText: "any"},
	"map[string][]string": {SchemaText: "map[string][]string", GoText: "map[string][]string", Family: TypeFamilyMap, MapValueText: "[]string"},
}

// //

func ParseType(typeText string, enumValueTextArr []string) (TypeObj, error) {
	typeText = strings.TrimSpace(typeText)
	if len(enumValueTextArr) > 0 {
		if typeText == "" || typeText == "enum" {
			return TypeObj{
				SchemaText: "enum",
				GoText:     "uint16",
				Family:     TypeFamilyScalar,
				IsEnum:     true,
			}, nil
		}

		if typeText == "[]enum" {
			return TypeObj{
				SchemaText:     "[]enum",
				GoText:         "[]uint16",
				Family:         TypeFamilyArray,
				ElemSchemaText: "enum",
				IsEnum:         true,
			}, nil
		}

		return TypeObj{}, fmt.Errorf("enum leaf has unsupported explicit type: %s", typeText)
	}

	if typeObj, existsFlag := cScalarTypeMap[typeText]; existsFlag {
		return typeObj, nil
	}

	if typeObj, existsFlag := cMapTypeMap[typeText]; existsFlag {
		return typeObj, nil
	}

	if strings.HasPrefix(typeText, "[]") {
		elemTypeText := strings.TrimPrefix(typeText, "[]")
		elemTypeObj, err := ParseType(elemTypeText, nil)
		if err != nil {
			return TypeObj{}, err
		}

		if elemTypeObj.Family != TypeFamilyScalar {
			return TypeObj{}, fmt.Errorf("unsupported array element type: %s", elemTypeText)
		}

		return TypeObj{
			SchemaText:     typeText,
			GoText:         "[]" + elemTypeObj.GoText,
			Family:         TypeFamilyArray,
			ElemSchemaText: elemTypeObj.SchemaText,
			IsDuration:     elemTypeObj.IsDuration,
			IsSize:         elemTypeObj.IsSize,
			IsEnum:         elemTypeObj.IsEnum,
			IsNumeric:      elemTypeObj.IsNumeric,
			IsSigned:       elemTypeObj.IsSigned,
			IsFloat:        elemTypeObj.IsFloat,
		}, nil
	}

	return TypeObj{}, fmt.Errorf("unsupported type: %s", typeText)
}

func BuildEnum(pathText string, usageText string, nameText string, valueTextArr []string) (*EnumObj, error) {
	baseNameText, err := buildEnumBaseName(pathText, nameText)
	if err != nil {
		return nil, fmt.Errorf("schema node [%s] enum_name: %w", pathText, err)
	}

	constNameArr, err := buildEnumConstNameArr(pathText, baseNameText, valueTextArr)
	if err != nil {
		return nil, err
	}

	return &EnumObj{
		PathText:     pathText,
		NameText:     baseNameText,
		TypeName:     baseNameText + "Enum",
		UsageText:    strings.TrimSpace(usageText),
		ValueTextArr: append([]string(nil), valueTextArr...),
		ConstNameArr: constNameArr,
	}, nil
}

func buildEnumBaseName(pathText string, nameText string) (string, error) {
	nameText = strings.TrimSpace(nameText)
	if nameText == "" {
		return GoName(pathText), nil
	}

	return goExportedNameStrict(nameText)
}

func buildEnumConstNameArr(pathText string, prefixText string, valueTextArr []string) ([]string, error) {
	resultArr := make([]string, 0, len(valueTextArr))
	seenMap := make(map[string]string, len(valueTextArr))

	for _, valueText := range valueTextArr {
		valueNameText := GoName(valueText)
		if valueNameText == "Enum" {
			return nil, fmt.Errorf("schema node [%s] enum value [%s] is reserved", pathText, valueText)
		}

		constNameText := prefixText + valueNameText
		if previousValueText, existsFlag := seenMap[constNameText]; existsFlag {
			return nil, fmt.Errorf("schema node [%s] enum constant collision: [%s] conflicts with [%s] via [%s]", pathText, valueText, previousValueText, constNameText)
		}

		seenMap[constNameText] = valueText
		resultArr = append(resultArr, constNameText)
	}

	return resultArr, nil
}

func ParseRange(typeObj TypeObj, textValue string) (*RangeObj, error) {
	textValue = strings.TrimSpace(textValue)
	if textValue == "" {
		return nil, fmt.Errorf("range is empty")
	}

	switch {
	case typeObj.IsDuration:
		durationValue, err := time.ParseDuration(textValue)
		if err != nil {
			return nil, err
		}

		return &RangeObj{
			Text:      textValue,
			GoLiteral: fmt.Sprintf("time.Duration(%d)", int64(durationValue)),
			SignedVal: int64(durationValue),
			Kind:      "signed",
		}, nil
	case typeObj.IsSize:
		sizeValue, err := ParseSize(textValue)
		if err != nil {
			return nil, err
		}

		return &RangeObj{
			Text:        textValue,
			GoLiteral:   fmt.Sprintf("SizeObj(%d)", uint64(sizeValue)),
			UnsignedVal: uint64(sizeValue),
			Kind:        "unsigned",
		}, nil
	case typeObj.IsFloat:
		floatValue, err := strconv.ParseFloat(textValue, 64)
		if err != nil {
			return nil, err
		}

		return &RangeObj{
			Text:      textValue,
			GoLiteral: textValue,
			FloatVal:  floatValue,
			Kind:      "float",
		}, nil
	case typeObj.IsSigned:
		intValue, err := strconv.ParseInt(textValue, 10, 64)
		if err != nil {
			return nil, err
		}

		return &RangeObj{
			Text:      textValue,
			GoLiteral: textValue,
			SignedVal: intValue,
			Kind:      "signed",
		}, nil
	case typeObj.IsNumeric:
		uintValue, err := strconv.ParseUint(textValue, 10, 64)
		if err != nil {
			return nil, err
		}

		return &RangeObj{
			Text:        textValue,
			GoLiteral:   textValue,
			UnsignedVal: uintValue,
			Kind:        "unsigned",
		}, nil
	default:
		return nil, fmt.Errorf("range is not supported for type: %s", typeObj.SchemaText)
	}
}

func ParseSize(textValue string) (uint64, error) {
	textValue = strings.TrimSpace(strings.ToLower(textValue))
	if textValue == "" {
		return 0, fmt.Errorf("size is empty")
	}

	unitTextArr := []struct {
		SuffixText string
		FactorVal  float64
	}{
		{"kib", 1024},
		{"kb", 1000},
		{"mib", math.Pow(1024, 2)},
		{"mb", math.Pow(1000, 2)},
		{"gib", math.Pow(1024, 3)},
		{"gb", math.Pow(1000, 3)},
		{"tib", math.Pow(1024, 4)},
		{"tb", math.Pow(1000, 4)},
		{"b", 1},
	}

	factorVal := float64(1)

	for _, itemObj := range unitTextArr {
		if strings.HasSuffix(textValue, itemObj.SuffixText) {
			textValue = strings.TrimSpace(strings.TrimSuffix(textValue, itemObj.SuffixText))
			factorVal = itemObj.FactorVal
			break
		}
	}

	if textValue == "" {
		return 0, fmt.Errorf("size numeric part is empty")
	}

	if strings.HasPrefix(textValue, "-") {
		return 0, fmt.Errorf("size must be non-negative")
	}

	numberVal, err := strconv.ParseFloat(textValue, 64)
	if err != nil {
		return 0, err
	}

	if math.IsNaN(numberVal) || math.IsInf(numberVal, 0) || numberVal < 0 {
		return 0, fmt.Errorf("size must be non-negative")
	}

	scaledVal := numberVal * factorVal
	if math.IsNaN(scaledVal) || math.IsInf(scaledVal, 0) || scaledVal > float64(math.MaxUint64) {
		return 0, fmt.Errorf("size is too large")
	}

	return uint64(scaledVal), nil
}
