package semantic

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/amazing-generators/goconfgen/internal/schema"
	"github.com/amazing-generators/goconfgen/internal/yamltool"
	"github.com/hjson/hjson-go/v4"
	"gopkg.in/yaml.v3"
)

// // // // // // // // // //

type presetRenderNodeObj struct {
	KeyText    string
	UsageText  string
	ValueObj   any
	NodeObjArr []*presetRenderNodeObj
	BranchFlag bool
}

func validateAndRenderPresetNode(textValue string, rootObj *schema.BranchObj, nameText string) (*yaml.Node, error) {
	var docNodeObj yaml.Node
	if err := yaml.Unmarshal([]byte(textValue), &docNodeObj); err != nil {
		return nil, fmt.Errorf("parse preset [%s] yaml: %w", nameText, err)
	}

	if docNodeObj.Kind != yaml.DocumentNode || len(docNodeObj.Content) == 0 {
		return nil, fmt.Errorf("preset [%s] is empty", nameText)
	}

	rootMapNodeObj := docNodeObj.Content[0]
	if rootMapNodeObj.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("preset [%s] root must be mapping", nameText)
	}

	if err := yamltool.ValidateDuplicateKey(rootMapNodeObj, ""); err != nil {
		return nil, fmt.Errorf("preset [%s]: %w", nameText, err)
	}

	return renderPresetBranch(rootMapNodeObj, rootObj, nameText)
}

func renderFullBranch(branchObj *schema.BranchObj) *yaml.Node {
	resultNodeObj := &yaml.Node{Kind: yaml.MappingNode}

	for _, entryObj := range branchObj.EntryObjArr {
		switch {
		case entryObj.BranchObj != nil:
			keyNodeObj := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: entryObj.BranchObj.KeyText}
			valueNodeObj := renderFullBranch(entryObj.BranchObj)
			attachUsage(keyNodeObj, valueNodeObj, entryObj.BranchObj.UsageText)
			resultNodeObj.Content = append(resultNodeObj.Content, keyNodeObj, valueNodeObj)
		case entryObj.LeafObj != nil:
			keyNodeObj := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: entryObj.LeafObj.KeyText}
			valueNodeObj := leafValueNode(entryObj.LeafObj)
			attachUsage(keyNodeObj, valueNodeObj, entryObj.LeafObj.UsageText)
			resultNodeObj.Content = append(resultNodeObj.Content, keyNodeObj, valueNodeObj)
		}
	}

	return resultNodeObj
}

func renderPresetBranch(userNodeObj *yaml.Node, branchObj *schema.BranchObj, nameText string) (*yaml.Node, error) {
	unknownPathTextArr := collectPresetUnknown(userNodeObj, branchObj)
	if len(unknownPathTextArr) > 0 {
		sort.Strings(unknownPathTextArr)
		return nil, fmt.Errorf("preset [%s]: unknown key [%s]", nameText, unknownPathTextArr[0])
	}

	resultNodeObj := &yaml.Node{Kind: yaml.MappingNode}

	for _, entryObj := range branchObj.EntryObjArr {
		switch {
		case entryObj.BranchObj != nil:
			childBranchObj := entryObj.BranchObj
			childUserNodeObj := lookupNode(userNodeObj, childBranchObj.KeyText)
			if childUserNodeObj == nil {
				continue
			}

			if childUserNodeObj.Kind != yaml.MappingNode {
				return nil, fmt.Errorf("preset [%s]: key [%s] must be mapping", nameText, childBranchObj.PathText)
			}

			valueNodeObj, err := renderPresetBranch(childUserNodeObj, childBranchObj, nameText)
			if err != nil {
				return nil, err
			}

			keyNodeObj := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: childBranchObj.KeyText}
			attachUsage(keyNodeObj, valueNodeObj, childBranchObj.UsageText)
			resultNodeObj.Content = append(resultNodeObj.Content, keyNodeObj, valueNodeObj)
		case entryObj.LeafObj != nil:
			leafObj := entryObj.LeafObj
			childUserNodeObj := lookupNode(userNodeObj, leafObj.KeyText)
			if childUserNodeObj == nil {
				continue
			}

			if err := schema.ValidateLeafNode(leafObj.PathText, leafObj, childUserNodeObj); err != nil {
				return nil, fmt.Errorf("preset [%s]: %w", nameText, err)
			}

			keyNodeObj := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: leafObj.KeyText}
			valueNodeObj := yamltool.CloneNode(childUserNodeObj)
			attachUsage(keyNodeObj, valueNodeObj, leafObj.UsageText)
			resultNodeObj.Content = append(resultNodeObj.Content, keyNodeObj, valueNodeObj)
		}
	}

	return resultNodeObj, nil
}

func collectPresetUnknown(nodeObj *yaml.Node, branchObj *schema.BranchObj) []string {
	resultArr := make([]string, 0)
	knownBranchMap := make(map[string]*schema.BranchObj, len(branchObj.BranchObjArr))
	knownLeafMap := make(map[string]*schema.LeafObj, len(branchObj.LeafObjArr))

	for _, childBranchObj := range branchObj.BranchObjArr {
		knownBranchMap[childBranchObj.KeyText] = childBranchObj
	}

	for _, leafObj := range branchObj.LeafObjArr {
		knownLeafMap[leafObj.KeyText] = leafObj
	}

	for itemIndex := 0; itemIndex < len(nodeObj.Content); itemIndex += 2 {
		keyNodeObj := nodeObj.Content[itemIndex]
		valueNodeObj := nodeObj.Content[itemIndex+1]

		if childBranchObj, existsFlag := knownBranchMap[keyNodeObj.Value]; existsFlag {
			if valueNodeObj.Kind == yaml.MappingNode {
				resultArr = append(resultArr, collectPresetUnknown(valueNodeObj, childBranchObj)...)
			}

			continue
		}

		if _, existsFlag := knownLeafMap[keyNodeObj.Value]; existsFlag {
			continue
		}

		pathText := keyNodeObj.Value
		if branchObj.PathText != "" {
			pathText = branchObj.PathText + "." + keyNodeObj.Value
		}

		resultArr = append(resultArr, pathText)
	}

	return resultArr
}

func lookupNode(nodeObj *yaml.Node, keyText string) *yaml.Node {
	for itemIndex := 0; itemIndex < len(nodeObj.Content); itemIndex += 2 {
		if nodeObj.Content[itemIndex].Value == keyText {
			return nodeObj.Content[itemIndex+1]
		}
	}

	return nil
}

func leafValueNode(leafObj *schema.LeafObj) *yaml.Node {
	if leafObj.DefaultNodeObj != nil {
		return yamltool.CloneNode(leafObj.DefaultNodeObj)
	}

	switch leafObj.TypeObj.Family {
	case schema.TypeFamilyArray:
		return &yaml.Node{Kind: yaml.SequenceNode}
	case schema.TypeFamilyMap:
		return &yaml.Node{Kind: yaml.MappingNode}
	}

	switch {
	case leafObj.TypeObj.IsEnum:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: leafObj.EnumObj.ValueTextArr[0]}
	case leafObj.TypeObj.IsDuration:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "0s"}
	case leafObj.TypeObj.IsSize:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "0"}
	}

	switch leafObj.TypeObj.SchemaText {
	case "string":
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: ""}
	case "bool":
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: "false"}
	default:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: "0"}
	}
}

func attachUsage(keyNodeObj *yaml.Node, valueNodeObj *yaml.Node, usageText string) {
	if usageText == "" {
		return
	}

	if valueNodeObj.Kind == yaml.ScalarNode && !bytes.ContainsRune([]byte(usageText), '\n') {
		valueNodeObj.LineComment = usageText
		return
	}

	keyNodeObj.HeadComment = usageText
}

func renderJSONNode(rootNodeObj *yaml.Node) (string, error) {
	renderNodeObj, err := buildRenderNode(rootNodeObj, "")
	if err != nil {
		return "", err
	}

	dataArr, err := renderPresetJSONBranch(renderNodeObj, "", true)
	if err != nil {
		return "", err
	}

	return string(dataArr), nil
}

func renderHJSONNode(rootNodeObj *yaml.Node) (string, error) {
	renderNodeObj, err := buildRenderNode(rootNodeObj, "")
	if err != nil {
		return "", err
	}

	dataArr, err := renderPresetHJSONBranch(renderNodeObj, "", true)
	if err != nil {
		return "", err
	}

	return string(dataArr), nil
}

func buildRenderNode(nodeObj *yaml.Node, keyText string) (*presetRenderNodeObj, error) {
	if nodeObj.Kind != yaml.MappingNode {
		var valueObj any
		if err := nodeObj.Decode(&valueObj); err != nil {
			return nil, err
		}

		return &presetRenderNodeObj{
			KeyText:    keyText,
			ValueObj:   valueObj,
			BranchFlag: false,
		}, nil
	}

	resultObj := &presetRenderNodeObj{
		KeyText:    keyText,
		BranchFlag: true,
		NodeObjArr: make([]*presetRenderNodeObj, 0, len(nodeObj.Content)/2),
	}

	for itemIndex := 0; itemIndex < len(nodeObj.Content); itemIndex += 2 {
		childKeyNodeObj := nodeObj.Content[itemIndex]
		childValueNodeObj := nodeObj.Content[itemIndex+1]
		childNodeObj, err := buildRenderNode(childValueNodeObj, childKeyNodeObj.Value)
		if err != nil {
			return nil, err
		}

		childNodeObj.UsageText = strings.TrimSpace(firstNonEmptyText(childKeyNodeObj.HeadComment, childValueNodeObj.LineComment))
		resultObj.NodeObjArr = append(resultObj.NodeObjArr, childNodeObj)
	}

	return resultObj, nil
}

func renderPresetJSONBranch(nodeObj *presetRenderNodeObj, indentText string, rootFlag bool) ([]byte, error) {
	var bufferObj bytes.Buffer

	if rootFlag {
		bufferObj.WriteString("{\n")
	} else {
		bufferObj.WriteString("{")
		if len(nodeObj.NodeObjArr) > 0 {
			bufferObj.WriteString("\n")
		}
	}

	for itemIndex, childNodeObj := range nodeObj.NodeObjArr {
		bufferObj.WriteString(indentText + "  " + strconv.Quote(childNodeObj.KeyText) + ": ")
		valueArr, err := renderPresetJSONValue(childNodeObj, indentText+"  ")
		if err != nil {
			return nil, err
		}

		bufferObj.Write(valueArr)
		if itemIndex+1 < len(nodeObj.NodeObjArr) {
			bufferObj.WriteString(",")
		}
		bufferObj.WriteString("\n")
	}

	bufferObj.WriteString(indentText + "}")
	if rootFlag {
		bufferObj.WriteString("\n")
	}

	return bufferObj.Bytes(), nil
}

func renderPresetJSONValue(nodeObj *presetRenderNodeObj, indentText string) ([]byte, error) {
	if nodeObj.BranchFlag {
		return renderPresetJSONBranch(nodeObj, indentText, false)
	}

	return json.Marshal(nodeObj.ValueObj)
}

func renderPresetHJSONBranch(nodeObj *presetRenderNodeObj, indentText string, rootFlag bool) ([]byte, error) {
	var bufferObj bytes.Buffer

	bufferObj.WriteString("{")
	if len(nodeObj.NodeObjArr) > 0 {
		bufferObj.WriteString("\n")
	}

	for _, childNodeObj := range nodeObj.NodeObjArr {
		if childNodeObj.UsageText != "" {
			for _, lineText := range strings.Split(childNodeObj.UsageText, "\n") {
				bufferObj.WriteString(indentText + "  # " + lineText + "\n")
			}
		}

		keyArr, err := renderPresetHJSONKeyText(childNodeObj.KeyText)
		if err != nil {
			return nil, err
		}

		bufferObj.WriteString(indentText + "  ")
		bufferObj.Write(keyArr)
		bufferObj.WriteString(": ")
		valueArr, err := renderPresetHJSONValue(childNodeObj, indentText+"  ")
		if err != nil {
			return nil, err
		}

		bufferObj.Write(valueArr)
		bufferObj.WriteString("\n")
	}

	bufferObj.WriteString(indentText + "}")
	if rootFlag {
		bufferObj.WriteString("\n")
	}

	return bufferObj.Bytes(), nil
}

func renderPresetHJSONValue(nodeObj *presetRenderNodeObj, indentText string) ([]byte, error) {
	if nodeObj.BranchFlag {
		return renderPresetHJSONBranch(nodeObj, indentText, false)
	}

	optsObj := hjson.DefaultOptions()
	optsObj.Comments = false
	optsObj.EmitRootBraces = false
	optsObj.IndentBy = "  "
	optsObj.BaseIndentation = indentText
	dataArr, err := hjson.MarshalWithOptions(nodeObj.ValueObj, optsObj)
	if err != nil {
		return nil, err
	}
	// Первая строка значения идёт сразу после "key: "; базовый отступ нужен лишь продолжениям.
	return bytes.TrimLeft(dataArr, " "), nil
}

func renderPresetHJSONKeyText(keyText string) ([]byte, error) {
	if isBarePresetHJSONKey(keyText) {
		return []byte(keyText), nil
	}

	return json.Marshal(keyText)
}

func isBarePresetHJSONKey(keyText string) bool {
	if keyText == "" {
		return false
	}

	for itemIndex, itemRune := range keyText {
		switch {
		case itemRune >= 'a' && itemRune <= 'z':
		case itemRune >= 'A' && itemRune <= 'Z':
		case itemRune >= '0' && itemRune <= '9' && itemIndex > 0:
		case itemRune == '_' || itemRune == '-':
		default:
			return false
		}
	}

	return true
}

func firstNonEmptyText(valueTextArr ...string) string {
	for _, valueText := range valueTextArr {
		if strings.TrimSpace(valueText) != "" {
			return valueText
		}
	}

	return ""
}
