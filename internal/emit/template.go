package emit

import (
	"bytes"
	"embed"
	"fmt"
	"strconv"
	"strings"
	"text/template"
)

// // // // // // // // // //

//go:embed templates/*.tmpl
var cTemplateFS embed.FS

var cTemplateObj = template.Must(template.New("goconfgen").Funcs(template.FuncMap{
	"importLine":          importLine,
	"quote":               strconv.Quote,
	"quoteJoin":           quoteJoin,
	"hexBytes":            renderHexBytes,
	"goName":              goName,
	"fieldGoType":         fieldGoType,
	"fieldMetaVarName":    fieldMetaVarName,
	"kindLiteral":         kindLiteral,
	"bitSizeLiteral":      bitSizeLiteral,
	"setterName":          setterName,
	"commentLines":        commentLines,
	"elemGoType":          elemGoType,
	"enumValues":          enumValues,
	"interfaceReturnType": interfaceReturnType,
	"rangeCompare":        rangeCompare,
	"flagVarName":         flagVarName,
	"flagCtorExpr":        flagCtorExpr,
	"singleLineUsage":     singleLineUsage,
	"boolLiteral":         boolLiteral,
	"rawBytesLiteral":     rawBytesLiteral,
}).ParseFS(cTemplateFS, "templates/*.tmpl"))

// //

func renderTemplateFile(fileNameText string, dataObj *TemplateDataObj) ([]byte, error) {
	var bufferObj bytes.Buffer
	templateNameText := strings.TrimSuffix(fileNameText, ".go") + ".tmpl"
	if err := cTemplateObj.ExecuteTemplate(&bufferObj, templateNameText, dataObj); err != nil {
		return nil, fmt.Errorf("execute template [%s]: %w", fileNameText, err)
	}

	return bufferObj.Bytes(), nil
}

func importLine(importText string) string {
	if strings.Contains(importText, "\"") {
		return importText
	}

	return strconv.Quote(importText)
}

func commentLines(textValue string) []string {
	textValue = strings.TrimSpace(textValue)
	if textValue == "" {
		return nil
	}

	return strings.Split(textValue, "\n")
}

func rawBytesLiteral(textValue string) string {
	if !strings.Contains(textValue, "`") {
		return "[]byte(`" + textValue + "`)"
	}

	partArr := strings.Split(textValue, "`")
	quotedPartArr := make([]string, 0, len(partArr)*2-1)
	for indexVal, partText := range partArr {
		if partText != "" {
			quotedPartArr = append(quotedPartArr, "`"+partText+"`")
		}
		if indexVal+1 < len(partArr) {
			quotedPartArr = append(quotedPartArr, strconv.Quote("`"))
		}
	}

	if len(quotedPartArr) == 0 {
		return "[]byte(\"\")"
	}

	return "[]byte(" + strings.Join(quotedPartArr, " + ") + ")"
}
