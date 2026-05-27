package codegen

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/amazing-generators/goconfgen/internal/ir"
	"github.com/amazing-generators/goconfgen/internal/naming"
)

// // // // // // // // // //

func collectTypeImports(packageObj *ir.PackageObj) []string {
	if !packageObj.HasSize && !packageObj.HasDuration {
		return nil
	}

	importTextMap := map[string]bool{}
	if packageObj.HasSize {
		importTextMap["encoding/json"] = true
	}
	if packageObj.HasDuration {
		importTextMap["time"] = true
	}

	return sortedImportArr(importTextMap)
}

func collectAccessorImports(packageObj *ir.PackageObj) []string {
	importTextMap := map[string]bool{}
	if packageObj.HasDuration {
		importTextMap["time"] = true
	}
	return sortedImportArr(importTextMap)
}

func collectFieldMetaImports(packageObj *ir.PackageObj) []string {
	importTextMap := map[string]bool{
		"fmt": true,
	}

	for _, fieldObj := range packageObj.FieldObjArr {
		switch fieldObj.KindObj {
		case ir.FieldKindIntSigned, ir.FieldKindIntUnsigned, ir.FieldKindFloat, ir.FieldKindArrayIntSigned, ir.FieldKindArrayIntUnsigned, ir.FieldKindArrayFloat:
			importTextMap["strconv"] = true
		case ir.FieldKindDuration, ir.FieldKindArrayDuration:
			importTextMap["time"] = true
			importTextMap["strings"] = true
		case ir.FieldKindSize, ir.FieldKindArraySize, ir.FieldKindEnumScalar, ir.FieldKindArrayEnum:
			importTextMap["strings"] = true
		}

		if fieldObj.MinRangeObj != nil || fieldObj.MaxRangeObj != nil {
			if fieldObj.TypeObj.IsDuration {
				importTextMap["time"] = true
			}
		}
	}

	if packageObj.HasCLI {
		for _, fieldObj := range packageObj.FieldObjArr {
			switch fieldObj.KindObj {
			case ir.FieldKindBool, ir.FieldKindArrayBool:
				importTextMap["strconv"] = true
				importTextMap["strings"] = true
			case ir.FieldKindIntSigned, ir.FieldKindIntUnsigned, ir.FieldKindFloat, ir.FieldKindArrayIntSigned, ir.FieldKindArrayIntUnsigned, ir.FieldKindArrayFloat:
				importTextMap["strings"] = true
			}
		}
	}

	return sortedImportArr(importTextMap)
}

func collectValidateImports(packageObj *ir.PackageObj) []string {
	importTextMap := map[string]bool{
		"fmt": true,
	}

	return sortedImportArr(importTextMap)
}

func collectFlagImports(packageObj *ir.PackageObj) []string {
	if !packageObj.HasCLI {
		return nil
	}

	importTextMap := map[string]bool{
		"flag":    true,
		"fmt":     true,
		"strings": true,
	}

	return sortedImportArr(importTextMap)
}

func collectRuntimeImports(packageObj *ir.PackageObj) []string {
	importTextMap := map[string]bool{
		"fmt":           true,
		"math":          true,
		"os":            true,
		"path/filepath": true,
		"strings":       true,
	}

	if packageObj.HasYAML || packageObj.HasJSON || packageObj.HasHJSON {
		importTextMap["bytes"] = true
	}
	if packageObj.HasYAML {
		importTextMap["gopkg.in/yaml.v3"] = true
	}
	if packageObj.HasJSON || packageObj.HasHJSON || packageObj.HasCLI {
		importTextMap["encoding/json"] = true
	}
	if packageObj.HasJSON {
		importTextMap["io"] = true
	}
	if packageObj.HasHJSON {
		importTextMap["github.com/hjson/hjson-go/v4"] = true
	}
	if packageObj.HasJSON || packageObj.HasSize || packageObj.HasCLI {
		importTextMap["strconv"] = true
	}
	if packageObj.HasMapAny {
		importTextMap["sort"] = true
	}
	if packageObj.HasTextSlice {
		importTextMap["fmt"] = true
	}

	return sortedImportArr(importTextMap)
}

func collectPresetImports(packageObj *ir.PackageObj) []string {
	return nil
}

func sortedImportArr(importTextMap map[string]bool) []string {
	resultArr := make([]string, 0, len(importTextMap))
	for importText := range importTextMap {
		resultArr = append(resultArr, importText)
	}
	sort.Strings(resultArr)
	return resultArr
}

func interfaceReturnType(branchObj *ir.BranchObj) string {
	if branchObj.GenInterfaceFlag {
		return branchObj.InterfaceTypeText
	}

	return branchObj.TypeNameText
}

func fieldGoType(fieldObj *ir.FieldObj) string {
	switch {
	case fieldObj.TypeObj.IsEnum && fieldObj.TypeObj.Family == "scalar":
		return fieldObj.EnumObj.TypeName
	case fieldObj.TypeObj.IsEnum && fieldObj.TypeObj.Family == "array":
		return "[]" + fieldObj.EnumObj.TypeName
	case fieldObj.TypeObj.IsSize && fieldObj.TypeObj.Family == "scalar":
		return "SizeObj"
	case fieldObj.TypeObj.IsSize && fieldObj.TypeObj.Family == "array":
		return "[]SizeObj"
	default:
		return fieldObj.TypeObj.GoText
	}
}

func elemGoType(fieldObj *ir.FieldObj) string {
	return strings.TrimPrefix(fieldObj.GoTypeText, "[]")
}

func goName(textValue string) string {
	return naming.GoName(textValue)
}

func quoteJoin(textArr []string) string {
	partArr := make([]string, 0, len(textArr))
	for _, itemText := range textArr {
		partArr = append(partArr, strconv.Quote(itemText))
	}
	return strings.Join(partArr, ", ")
}

func enumValueArr(fieldObj *ir.FieldObj) []string {
	if fieldObj.EnumObj == nil {
		return nil
	}
	return fieldObj.EnumObj.ValueTextArr
}

func enumValues(fieldObj *ir.FieldObj) []string {
	return enumValueArr(fieldObj)
}

func boolLiteral(flag bool) string {
	if flag {
		return "true"
	}
	return "false"
}

func flagVarName(fieldObj *ir.FieldObj) string {
	return "cFlag" + goName(strings.ReplaceAll(fieldObj.PathText, ".", "_")) + "Obj"
}

func fieldMetaVarName(fieldObj *ir.FieldObj) string {
	return "cField" + goName(strings.ReplaceAll(fieldObj.PathText, ".", "_")) + "MetaObj"
}

func setterName(fieldObj *ir.FieldObj) string {
	return "set" + goName(strings.ReplaceAll(fieldObj.PathText, ".", "_"))
}

func kindLiteral(kindObj ir.FieldKindObj) string {
	switch kindObj {
	case ir.FieldKindString:
		return "fieldKindString"
	case ir.FieldKindBool:
		return "fieldKindBool"
	case ir.FieldKindIntSigned:
		return "fieldKindIntSigned"
	case ir.FieldKindIntUnsigned:
		return "fieldKindIntUnsigned"
	case ir.FieldKindFloat:
		return "fieldKindFloat"
	case ir.FieldKindDuration:
		return "fieldKindDuration"
	case ir.FieldKindSize:
		return "fieldKindSize"
	case ir.FieldKindEnumScalar:
		return "fieldKindEnumScalar"
	case ir.FieldKindArrayString:
		return "fieldKindArrayString"
	case ir.FieldKindArrayBool:
		return "fieldKindArrayBool"
	case ir.FieldKindArrayIntSigned:
		return "fieldKindArrayIntSigned"
	case ir.FieldKindArrayIntUnsigned:
		return "fieldKindArrayIntUnsigned"
	case ir.FieldKindArrayFloat:
		return "fieldKindArrayFloat"
	case ir.FieldKindArrayDuration:
		return "fieldKindArrayDuration"
	case ir.FieldKindArraySize:
		return "fieldKindArraySize"
	case ir.FieldKindArrayEnum:
		return "fieldKindArrayEnum"
	case ir.FieldKindMapStringString:
		return "fieldKindMapStringString"
	case ir.FieldKindMapStringStringArr:
		return "fieldKindMapStringStringArr"
	case ir.FieldKindMapStringAny:
		return "fieldKindMapStringAny"
	default:
		return "fieldKindString"
	}
}

func bitSizeLiteral(fieldObj *ir.FieldObj) string {
	if fieldObj.BitSizeVal != 0 {
		return strconv.Itoa(fieldObj.BitSizeVal)
	}

	switch fieldObj.KindObj {
	case ir.FieldKindIntSigned, ir.FieldKindIntUnsigned, ir.FieldKindArrayIntSigned, ir.FieldKindArrayIntUnsigned:
		if fieldObj.GoTypeText == "int" || fieldObj.GoTypeText == "uint" || fieldObj.GoTypeText == "[]int" || fieldObj.GoTypeText == "[]uint" {
			return "strconv.IntSize"
		}
	}

	return "0"
}

func flagCtorExpr(fieldObj *ir.FieldObj) string {
	if fieldObj.TypeObj.SchemaText == "bool" && fieldObj.TypeObj.Family == "scalar" {
		return "&runtimeBoolFlagObj{}"
	}

	return "&runtimeStringFlagObj{}"
}

func singleLineUsage(textValue string) string {
	return strings.TrimSpace(strings.ReplaceAll(textValue, "\n", " "))
}

func compareExpr(fieldObj *ir.FieldObj, fieldExprText string, literalText string, operatorText string) string {
	leftText := fieldExprText
	rightText := literalText
	if fieldObj.TypeObj.IsSize {
		leftText = "uint64(" + fieldExprText + ")"
		rightText = "uint64(" + literalText + ")"
	}
	return leftText + " " + operatorText + " " + rightText
}

func rangeCompare(fieldObj *ir.FieldObj, operatorText string, literalText string) string {
	return compareExpr(fieldObj, "obj."+fieldObj.AccessorText, literalText, operatorText)
}

func renderHexBytes(dataArr []byte) string {
	if len(dataArr) == 0 {
		return ""
	}

	var builder strings.Builder
	for itemIndex, itemByte := range dataArr {
		if itemIndex > 0 {
			builder.WriteString(", ")
		}
		if itemIndex > 0 && itemIndex%24 == 0 {
			builder.WriteString("\n\t")
		}
		builder.WriteString(fmt.Sprintf("0x%02x", itemByte))
	}
	return builder.String()
}
