package emit

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/amazing-generators/goconfgen/internal/schema"
	"github.com/amazing-generators/goconfgen/internal/semantic"
)

// // // // // // // // // //

func collectTypeImports(packageObj *semantic.PackageObj) []string {
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

func collectAccessorImports(packageObj *semantic.PackageObj) []string {
	importTextMap := map[string]bool{}
	if packageObj.HasDuration {
		importTextMap["time"] = true
	}
	return sortedImportArr(importTextMap)
}

func collectFieldMetaImports(packageObj *semantic.PackageObj) []string {
	importTextMap := map[string]bool{
		"fmt": true,
	}

	for _, fieldObj := range packageObj.FieldObjArr {
		switch fieldObj.KindObj {
		case semantic.FieldKindIntSigned, semantic.FieldKindIntUnsigned, semantic.FieldKindFloat, semantic.FieldKindArrayIntSigned, semantic.FieldKindArrayIntUnsigned, semantic.FieldKindArrayFloat:
			importTextMap["strconv"] = true
		case semantic.FieldKindDuration, semantic.FieldKindArrayDuration:
			importTextMap["time"] = true
			importTextMap["strings"] = true
		case semantic.FieldKindSize, semantic.FieldKindArraySize, semantic.FieldKindEnumScalar, semantic.FieldKindArrayEnum:
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
			case semantic.FieldKindBool, semantic.FieldKindArrayBool:
				importTextMap["strconv"] = true
				importTextMap["strings"] = true
			case semantic.FieldKindIntSigned, semantic.FieldKindIntUnsigned, semantic.FieldKindFloat, semantic.FieldKindArrayIntSigned, semantic.FieldKindArrayIntUnsigned, semantic.FieldKindArrayFloat:
				importTextMap["strings"] = true
			}
		}
	}

	return sortedImportArr(importTextMap)
}

func collectValidateImports(packageObj *semantic.PackageObj) []string {
	importTextMap := map[string]bool{
		"fmt": true,
	}

	return sortedImportArr(importTextMap)
}

func collectFlagImports(packageObj *semantic.PackageObj) []string {
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

func collectRuntimeImports(packageObj *semantic.PackageObj) []string {
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

func sortedImportArr(importTextMap map[string]bool) []string {
	resultArr := make([]string, 0, len(importTextMap))
	for importText := range importTextMap {
		resultArr = append(resultArr, importText)
	}
	sort.Strings(resultArr)
	return resultArr
}

func interfaceReturnType(branchObj *semantic.BranchObj) string {
	if branchObj.GenInterfaceFlag {
		return branchObj.InterfaceTypeText
	}

	return branchObj.TypeNameText
}

func elemGoType(fieldObj *semantic.FieldObj) string {
	return strings.TrimPrefix(fieldObj.GoTypeText, "[]")
}

func goName(textValue string) string {
	return schema.GoName(textValue)
}

func quoteJoin(textArr []string) string {
	partArr := make([]string, 0, len(textArr))
	for _, itemText := range textArr {
		partArr = append(partArr, strconv.Quote(itemText))
	}
	return strings.Join(partArr, ", ")
}

func enumValues(fieldObj *semantic.FieldObj) []string {
	if fieldObj.EnumObj == nil {
		return nil
	}
	return fieldObj.EnumObj.ValueTextArr
}

func boolLiteral(flag bool) string {
	if flag {
		return "true"
	}
	return "false"
}

func flagVarName(fieldObj *semantic.FieldObj) string {
	return "cFlag" + goName(strings.ReplaceAll(fieldObj.PathText, ".", "_")) + "Obj"
}

func fieldMetaVarName(fieldObj *semantic.FieldObj) string {
	return "cField" + goName(strings.ReplaceAll(fieldObj.PathText, ".", "_")) + "MetaObj"
}

func setterName(fieldObj *semantic.FieldObj) string {
	return "set" + goName(strings.ReplaceAll(fieldObj.PathText, ".", "_"))
}

func kindLiteral(kindObj semantic.FieldKindObj) string {
	switch kindObj {
	case semantic.FieldKindString:
		return "fieldKindString"
	case semantic.FieldKindBool:
		return "fieldKindBool"
	case semantic.FieldKindIntSigned:
		return "fieldKindIntSigned"
	case semantic.FieldKindIntUnsigned:
		return "fieldKindIntUnsigned"
	case semantic.FieldKindFloat:
		return "fieldKindFloat"
	case semantic.FieldKindDuration:
		return "fieldKindDuration"
	case semantic.FieldKindSize:
		return "fieldKindSize"
	case semantic.FieldKindEnumScalar:
		return "fieldKindEnumScalar"
	case semantic.FieldKindArrayString:
		return "fieldKindArrayString"
	case semantic.FieldKindArrayBool:
		return "fieldKindArrayBool"
	case semantic.FieldKindArrayIntSigned:
		return "fieldKindArrayIntSigned"
	case semantic.FieldKindArrayIntUnsigned:
		return "fieldKindArrayIntUnsigned"
	case semantic.FieldKindArrayFloat:
		return "fieldKindArrayFloat"
	case semantic.FieldKindArrayDuration:
		return "fieldKindArrayDuration"
	case semantic.FieldKindArraySize:
		return "fieldKindArraySize"
	case semantic.FieldKindArrayEnum:
		return "fieldKindArrayEnum"
	case semantic.FieldKindMapStringString:
		return "fieldKindMapStringString"
	case semantic.FieldKindMapStringStringArr:
		return "fieldKindMapStringStringArr"
	case semantic.FieldKindMapStringAny:
		return "fieldKindMapStringAny"
	default:
		return "fieldKindString"
	}
}

func bitSizeLiteral(fieldObj *semantic.FieldObj) string {
	if fieldObj.BitSizeVal != 0 {
		return strconv.Itoa(fieldObj.BitSizeVal)
	}

	switch fieldObj.KindObj {
	case semantic.FieldKindIntSigned, semantic.FieldKindIntUnsigned, semantic.FieldKindArrayIntSigned, semantic.FieldKindArrayIntUnsigned:
		if fieldObj.GoTypeText == "int" || fieldObj.GoTypeText == "uint" || fieldObj.GoTypeText == "[]int" || fieldObj.GoTypeText == "[]uint" {
			return "strconv.IntSize"
		}
	}

	return "0"
}

func flagCtorExpr(fieldObj *semantic.FieldObj) string {
	if fieldObj.TypeObj.SchemaText == "bool" && fieldObj.TypeObj.Family == "scalar" {
		return "&runtimeBoolFlagObj{}"
	}

	return "&runtimeStringFlagObj{}"
}

func singleLineUsage(textValue string) string {
	return strings.TrimSpace(strings.ReplaceAll(textValue, "\n", " "))
}

func compareExpr(fieldObj *semantic.FieldObj, fieldExprText string, literalText string, operatorText string) string {
	leftText := fieldExprText
	rightText := literalText
	if fieldObj.TypeObj.IsSize {
		leftText = "uint64(" + fieldExprText + ")"
		rightText = "uint64(" + literalText + ")"
	}
	return leftText + " " + operatorText + " " + rightText
}

func rangeCompare(fieldObj *semantic.FieldObj, operatorText string, literalText string) string {
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
