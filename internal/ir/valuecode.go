package ir

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/amazing-generators/goconfgen/internal/schema"
	"gopkg.in/yaml.v3"
)

// // // // // // // // // //

func defaultCodeForLeaf(leafObj *schema.LeafObj) (string, error) {
	if leafObj.DefaultNodeObj == nil {
		return zeroCodeForType(leafObj.TypeObj, leafObj.EnumObj)
	}

	return nodeToCode(leafObj.DefaultNodeObj, leafObj.TypeObj, leafObj.EnumObj)
}

func zeroCodeForType(typeObj schema.TypeObj, enumObj *schema.EnumObj) (string, error) {
	switch typeObj.Family {
	case schema.TypeFamilyArray, schema.TypeFamilyMap:
		return "nil", nil
	}

	switch {
	case typeObj.IsEnum:
		return enumObj.TypeName + "(0)", nil
	case typeObj.IsDuration:
		return "time.Duration(0)", nil
	case typeObj.IsSize:
		return "SizeObj(0)", nil
	}

	switch typeObj.SchemaText {
	case "string":
		return `""`, nil
	case "bool":
		return "false", nil
	default:
		return "0", nil
	}
}

func nodeToCode(nodeObj *yaml.Node, typeObj schema.TypeObj, enumObj *schema.EnumObj) (string, error) {
	switch typeObj.Family {
	case schema.TypeFamilyScalar:
		return scalarNodeToCode(nodeObj, typeObj, enumObj)
	case schema.TypeFamilyArray:
		return arrayNodeToCode(nodeObj, typeObj, enumObj)
	case schema.TypeFamilyMap:
		return mapNodeToCode(nodeObj, typeObj)
	default:
		return "", fmt.Errorf("unsupported type family: %s", typeObj.Family)
	}
}

func scalarNodeToCode(nodeObj *yaml.Node, typeObj schema.TypeObj, enumObj *schema.EnumObj) (string, error) {
	switch {
	case typeObj.IsEnum:
		valueText := strings.TrimSpace(nodeObj.Value)
		for itemIndex, enumValueText := range enumObj.ValueTextArr {
			if enumValueText == valueText {
				return enumObj.ConstNameArr[itemIndex], nil
			}
		}

		return "", fmt.Errorf("unknown enum value: %s", valueText)
	case typeObj.IsDuration:
		durationValue, err := time.ParseDuration(strings.TrimSpace(nodeObj.Value))
		if err != nil {
			return "", err
		}

		return fmt.Sprintf("time.Duration(%d)", int64(durationValue)), nil
	case typeObj.IsSize:
		sizeValue, err := schema.ParseSize(nodeObj.Value)
		if err != nil {
			return "", err
		}

		return fmt.Sprintf("SizeObj(%d)", sizeValue), nil
	case typeObj.IsFloat:
		return strings.TrimSpace(nodeObj.Value), nil
	case typeObj.IsNumeric:
		return strings.TrimSpace(nodeObj.Value), nil
	case typeObj.SchemaText == "string":
		return strconv.Quote(nodeObj.Value), nil
	case typeObj.SchemaText == "bool":
		return strings.ToLower(strings.TrimSpace(nodeObj.Value)), nil
	default:
		return "", fmt.Errorf("unsupported scalar type: %s", typeObj.SchemaText)
	}
}

func arrayNodeToCode(nodeObj *yaml.Node, typeObj schema.TypeObj, enumObj *schema.EnumObj) (string, error) {
	elemTypeObj := schema.TypeObj{}
	if !typeObj.IsEnum {
		var err error
		elemTypeObj, err = schema.ParseType(typeObj.ElemSchemaText, nil)
		if err != nil {
			return "", err
		}
	}

	partTextArr := make([]string, 0, len(nodeObj.Content))

	for _, childNodeObj := range nodeObj.Content {
		targetTypeObj := elemTypeObj
		if typeObj.IsEnum {
			targetTypeObj = schema.TypeObj{SchemaText: "enum", GoText: enumObj.TypeName, Family: schema.TypeFamilyScalar, IsEnum: true}
		}

		itemCodeText, codeErr := scalarNodeToCode(childNodeObj, targetTypeObj, enumObj)
		if codeErr != nil {
			return "", codeErr
		}

		partTextArr = append(partTextArr, itemCodeText)
	}

	elemGoText := elemTypeObj.GoText
	if typeObj.IsEnum {
		elemGoText = enumObj.TypeName
	}

	return "[]" + elemGoText + "{" + strings.Join(partTextArr, ", ") + "}", nil
}

func mapNodeToCode(nodeObj *yaml.Node, typeObj schema.TypeObj) (string, error) {
	switch typeObj.SchemaText {
	case "map[string]string":
		partTextArr := make([]string, 0, len(nodeObj.Content)/2)

		for itemIndex := 0; itemIndex < len(nodeObj.Content); itemIndex += 2 {
			partTextArr = append(partTextArr, fmt.Sprintf("%q: %q", nodeObj.Content[itemIndex].Value, nodeObj.Content[itemIndex+1].Value))
		}

		return "map[string]string{" + strings.Join(partTextArr, ", ") + "}", nil
	case "map[string][]string":
		partTextArr := make([]string, 0, len(nodeObj.Content)/2)

		for itemIndex := 0; itemIndex < len(nodeObj.Content); itemIndex += 2 {
			valueNodeObj := nodeObj.Content[itemIndex+1]
			itemTextArr := make([]string, 0, len(valueNodeObj.Content))
			for _, childNodeObj := range valueNodeObj.Content {
				itemTextArr = append(itemTextArr, strconv.Quote(childNodeObj.Value))
			}

			partTextArr = append(partTextArr, fmt.Sprintf("%q: []string{%s}", nodeObj.Content[itemIndex].Value, strings.Join(itemTextArr, ", ")))
		}

		return "map[string][]string{" + strings.Join(partTextArr, ", ") + "}", nil
	case "map[string]any":
		var anyObj any
		if err := nodeObj.Decode(&anyObj); err != nil {
			return "", err
		}

		// // // // // // // // // //
		// Прямой Go-литерал: без runtime json.Unmarshal на каждый ApplyDefaults.
		return anyToGoLiteral(anyObj)
	default:
		return "", fmt.Errorf("unsupported map type: %s", typeObj.SchemaText)
	}
}

// // // // // // // // // //
// anyToGoLiteral переводит распакованный yaml.Decode результат в Go-литерал
// для подстановки в сгенерированный код. Карты сортируются для детерминизма.
func anyToGoLiteral(valueObj any) (string, error) {
	switch typedValue := valueObj.(type) {
	case nil:
		return "nil", nil
	case bool:
		if typedValue {
			return "true", nil
		}
		return "false", nil
	case string:
		return strconv.Quote(typedValue), nil
	case int:
		return strconv.FormatInt(int64(typedValue), 10), nil
	case int64:
		return strconv.FormatInt(typedValue, 10), nil
	case uint64:
		return strconv.FormatUint(typedValue, 10), nil
	case float64:
		return strconv.FormatFloat(typedValue, 'g', -1, 64), nil
	case map[string]any:
		keyTextArr := make([]string, 0, len(typedValue))
		for keyText := range typedValue {
			keyTextArr = append(keyTextArr, keyText)
		}
		sortStringArr(keyTextArr)

		partTextArr := make([]string, 0, len(keyTextArr))
		for _, keyText := range keyTextArr {
			childTextValue, err := anyToGoLiteral(typedValue[keyText])
			if err != nil {
				return "", err
			}
			partTextArr = append(partTextArr, strconv.Quote(keyText)+": "+childTextValue)
		}

		return "map[string]any{" + strings.Join(partTextArr, ", ") + "}", nil
	case []any:
		partTextArr := make([]string, 0, len(typedValue))
		for _, childValue := range typedValue {
			childTextValue, err := anyToGoLiteral(childValue)
			if err != nil {
				return "", err
			}
			partTextArr = append(partTextArr, childTextValue)
		}
		return "[]any{" + strings.Join(partTextArr, ", ") + "}", nil
	default:
		return "", fmt.Errorf("unsupported value in map[string]any default: %T", valueObj)
	}
}

func sortStringArr(textArr []string) {
	for outerIndex := 1; outerIndex < len(textArr); outerIndex++ {
		for innerIndex := outerIndex; innerIndex > 0 && textArr[innerIndex-1] > textArr[innerIndex]; innerIndex-- {
			textArr[innerIndex-1], textArr[innerIndex] = textArr[innerIndex], textArr[innerIndex-1]
		}
	}
}
