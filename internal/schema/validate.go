package schema

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/amazing-generators/goconfgen/internal/yamltool"
	"gopkg.in/yaml.v3"
)

// // // // // // // // // //

func validateLeaf(pathText string, leafObj *LeafObj) error {
	if (leafObj.MinRangeObj != nil || leafObj.MaxRangeObj != nil) && leafObj.TypeObj.Family != TypeFamilyScalar {
		return fmt.Errorf("schema node [%s] min/max is supported only for scalar fields", pathText)
	}

	if leafObj.EnumObj != nil {
		if err := validateDefaultNode(pathText, leafObj); err != nil {
			return err
		}
	} else if leafObj.DefaultNodeObj != nil {
		if err := validateDefaultNode(pathText, leafObj); err != nil {
			return err
		}
	}

	if leafObj.MinRangeObj != nil && leafObj.MaxRangeObj != nil {
		switch leafObj.MinRangeObj.Kind {
		case "signed":
			if leafObj.MinRangeObj.SignedVal > leafObj.MaxRangeObj.SignedVal {
				return fmt.Errorf("schema node [%s] min is greater than max", pathText)
			}
		case "unsigned":
			if leafObj.MinRangeObj.UnsignedVal > leafObj.MaxRangeObj.UnsignedVal {
				return fmt.Errorf("schema node [%s] min is greater than max", pathText)
			}
		case "float":
			if leafObj.MinRangeObj.FloatVal > leafObj.MaxRangeObj.FloatVal {
				return fmt.Errorf("schema node [%s] min is greater than max", pathText)
			}
		}
	}

	return nil
}

func ValidateLeafNode(pathText string, leafObj *LeafObj, nodeObj *yaml.Node) error {
	return validateNodeAgainstLeaf(pathText, leafObj, nodeObj)
}

func validateDefaultNode(pathText string, leafObj *LeafObj) error {
	if leafObj.DefaultNodeObj == nil {
		return nil
	}

	return validateNodeAgainstLeaf(pathText, leafObj, leafObj.DefaultNodeObj)
}

func validateNodeAgainstLeaf(pathText string, leafObj *LeafObj, nodeObj *yaml.Node) error {
	switch leafObj.TypeObj.Family {
	case TypeFamilyScalar:
		return validateScalarNode(pathText, leafObj, nodeObj)
	case TypeFamilyArray:
		if nodeObj.Kind != yaml.SequenceNode {
			return fmt.Errorf("schema node [%s] default must be array", pathText)
		}

		elemTypeObj := TypeObj{}
		var err error
		if !leafObj.TypeObj.IsEnum {
			elemTypeObj, err = ParseType(leafObj.TypeObj.ElemSchemaText, nil)
			if err != nil {
				return err
			}
		}

		for itemIndex, childNodeObj := range nodeObj.Content {
			childLeafObj := *leafObj
			if leafObj.TypeObj.IsEnum {
				childLeafObj.TypeObj = TypeObj{
					SchemaText: "enum",
					GoText:     leafObj.EnumObj.TypeName,
					Family:     TypeFamilyScalar,
					IsEnum:     true,
				}
			} else {
				childLeafObj.TypeObj = elemTypeObj
			}
			childLeafObj.EnumObj = leafObj.EnumObj
			childLeafObj.MinRangeObj = nil
			childLeafObj.MaxRangeObj = nil

			if err = validateScalarNode(fmt.Sprintf("%s[%d]", pathText, itemIndex), &childLeafObj, childNodeObj); err != nil {
				return err
			}
		}

		return nil
	case TypeFamilyMap:
		if nodeObj.Kind != yaml.MappingNode {
			return fmt.Errorf("schema node [%s] default must be object", pathText)
		}

		return validateMapNode(pathText, leafObj, nodeObj)
	default:
		return fmt.Errorf("schema node [%s] unsupported type family", pathText)
	}
}

func validateMapNode(pathText string, leafObj *LeafObj, nodeObj *yaml.Node) error {
	if err := yamltool.ValidateDuplicateKey(nodeObj, pathText); err != nil {
		return fmt.Errorf("schema node [%s] default: %w", pathText, err)
	}

	for itemIndex := 0; itemIndex < len(nodeObj.Content); itemIndex += 2 {
		valueNodeObj := nodeObj.Content[itemIndex+1]
		childPathText := pathText + "." + nodeObj.Content[itemIndex].Value

		switch leafObj.TypeObj.SchemaText {
		case "map[string]string":
			if valueNodeObj.Kind != yaml.ScalarNode {
				return fmt.Errorf("schema node [%s] default must be string map", childPathText)
			}
		case "map[string][]string":
			if valueNodeObj.Kind != yaml.SequenceNode {
				return fmt.Errorf("schema node [%s] default must be string array", childPathText)
			}

			for seqIndex, itemNodeObj := range valueNodeObj.Content {
				if itemNodeObj.Kind != yaml.ScalarNode {
					return fmt.Errorf("schema node [%s][%d] default must be string", childPathText, seqIndex)
				}
			}
		case "map[string]any":
		default:
			return fmt.Errorf("schema node [%s] unsupported map type", pathText)
		}
	}

	return nil
}

func validateScalarNode(pathText string, leafObj *LeafObj, nodeObj *yaml.Node) error {
	switch {
	case leafObj.TypeObj.IsEnum:
		valueText := strings.TrimSpace(nodeObj.Value)
		if nodeObj.Kind != yaml.ScalarNode {
			return fmt.Errorf("schema node [%s] default must be enum string", pathText)
		}

		for _, enumValueText := range leafObj.EnumObj.ValueTextArr {
			if enumValueText == valueText {
				return nil
			}
		}

		return fmt.Errorf("schema node [%s] default enum value is unsupported", pathText)
	case leafObj.TypeObj.IsDuration:
		if nodeObj.Kind != yaml.ScalarNode {
			return fmt.Errorf("schema node [%s] default must be duration string", pathText)
		}

		durationValue, err := time.ParseDuration(strings.TrimSpace(nodeObj.Value))
		if err != nil {
			return fmt.Errorf("schema node [%s] default duration: %w", pathText, err)
		}

		return validateSignedRange(pathText, leafObj, int64(durationValue))
	case leafObj.TypeObj.IsSize:
		if nodeObj.Kind != yaml.ScalarNode {
			return fmt.Errorf("schema node [%s] default must be size string or int", pathText)
		}

		sizeValue, err := ParseSize(nodeObj.Value)
		if err != nil {
			return fmt.Errorf("schema node [%s] default size: %w", pathText, err)
		}

		return validateUnsignedRange(pathText, leafObj, sizeValue)
	case leafObj.TypeObj.IsFloat:
		if nodeObj.Kind != yaml.ScalarNode {
			return fmt.Errorf("schema node [%s] default must be float", pathText)
		}

		floatValue, err := strconv.ParseFloat(strings.TrimSpace(nodeObj.Value), 64)
		if err != nil {
			return fmt.Errorf("schema node [%s] default float: %w", pathText, err)
		}

		if err = validateFloatRange(pathText, leafObj, floatValue); err != nil {
			return err
		}

		return validateFloatFits(pathText, leafObj.TypeObj.SchemaText, floatValue)
	case leafObj.TypeObj.IsSigned:
		if nodeObj.Kind != yaml.ScalarNode {
			return fmt.Errorf("schema node [%s] default must be integer", pathText)
		}

		intValue, err := strconv.ParseInt(strings.TrimSpace(nodeObj.Value), 10, 64)
		if err != nil {
			return fmt.Errorf("schema node [%s] default integer: %w", pathText, err)
		}

		if err = validateSignedRange(pathText, leafObj, intValue); err != nil {
			return err
		}

		return validateScalarFits(pathText, leafObj.TypeObj.SchemaText, intValue, 0)
	case leafObj.TypeObj.IsNumeric:
		if nodeObj.Kind != yaml.ScalarNode {
			return fmt.Errorf("schema node [%s] default must be unsigned integer", pathText)
		}

		uintValue, err := strconv.ParseUint(strings.TrimSpace(nodeObj.Value), 10, 64)
		if err != nil {
			return fmt.Errorf("schema node [%s] default integer: %w", pathText, err)
		}

		if err = validateUnsignedRange(pathText, leafObj, uintValue); err != nil {
			return err
		}

		return validateScalarFits(pathText, leafObj.TypeObj.SchemaText, 0, uintValue)
	default:
		return validatePlainScalar(pathText, leafObj.TypeObj.SchemaText, nodeObj)
	}
}

func validatePlainScalar(pathText string, typeText string, nodeObj *yaml.Node) error {
	if nodeObj.Kind != yaml.ScalarNode {
		return fmt.Errorf("schema node [%s] default must be scalar", pathText)
	}

	switch typeText {
	case "string":
		return nil
	case "bool":
		if strings.EqualFold(nodeObj.Value, "true") || strings.EqualFold(nodeObj.Value, "false") {
			return nil
		}

		return fmt.Errorf("schema node [%s] default bool is invalid", pathText)
	default:
		return fmt.Errorf("schema node [%s] unsupported scalar type", pathText)
	}
}

func validateSignedRange(pathText string, leafObj *LeafObj, valueVal int64) error {
	if leafObj.MinRangeObj != nil && valueVal < leafObj.MinRangeObj.SignedVal {
		return fmt.Errorf("schema node [%s] default is below min", pathText)
	}

	if leafObj.MaxRangeObj != nil && valueVal > leafObj.MaxRangeObj.SignedVal {
		return fmt.Errorf("schema node [%s] default is above max", pathText)
	}

	return nil
}

func validateUnsignedRange(pathText string, leafObj *LeafObj, valueVal uint64) error {
	if leafObj.MinRangeObj != nil && valueVal < leafObj.MinRangeObj.UnsignedVal {
		return fmt.Errorf("schema node [%s] default is below min", pathText)
	}

	if leafObj.MaxRangeObj != nil && valueVal > leafObj.MaxRangeObj.UnsignedVal {
		return fmt.Errorf("schema node [%s] default is above max", pathText)
	}

	return nil
}

func validateFloatRange(pathText string, leafObj *LeafObj, valueVal float64) error {
	if leafObj.MinRangeObj != nil && valueVal < leafObj.MinRangeObj.FloatVal {
		return fmt.Errorf("schema node [%s] default is below min", pathText)
	}

	if leafObj.MaxRangeObj != nil && valueVal > leafObj.MaxRangeObj.FloatVal {
		return fmt.Errorf("schema node [%s] default is above max", pathText)
	}

	return nil
}

func validateScalarFits(pathText string, typeText string, intValue int64, uintValue uint64) error {
	switch typeText {
	case "int8":
		if intValue < math.MinInt8 || intValue > math.MaxInt8 {
			return fmt.Errorf("schema node [%s] default does not fit int8", pathText)
		}
	case "int16":
		if intValue < math.MinInt16 || intValue > math.MaxInt16 {
			return fmt.Errorf("schema node [%s] default does not fit int16", pathText)
		}
	case "int32":
		if intValue < math.MinInt32 || intValue > math.MaxInt32 {
			return fmt.Errorf("schema node [%s] default does not fit int32", pathText)
		}
	case "uint8":
		if uintValue > math.MaxUint8 {
			return fmt.Errorf("schema node [%s] default does not fit uint8", pathText)
		}
	case "uint16":
		if uintValue > math.MaxUint16 {
			return fmt.Errorf("schema node [%s] default does not fit uint16", pathText)
		}
	case "uint32":
		if uintValue > math.MaxUint32 {
			return fmt.Errorf("schema node [%s] default does not fit uint32", pathText)
		}
	}

	return nil
}

func validateFloatFits(pathText string, typeText string, floatValue float64) error {
	switch typeText {
	case "float", "float32":
		if math.IsInf(floatValue, 0) || floatValue > math.MaxFloat32 || floatValue < -math.MaxFloat32 {
			return fmt.Errorf("schema node [%s] default does not fit %s", pathText, typeText)
		}
	}

	return nil
}
