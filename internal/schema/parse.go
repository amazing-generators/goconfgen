package schema

import (
	"fmt"
	"strings"

	"github.com/amazing-generators/goconfgen/internal/yamltool"
	"gopkg.in/yaml.v3"
)

// // // // // // // // // //

type rawLeafObj struct {
	TypeText         string
	UsageText        string
	EnumNameText     string
	EnumValueTextArr []string
	DefaultNodeObj   *yaml.Node
	MinText          string
	MaxText          string
}

// //

func Parse(textValue string) (*ResultObj, error) {
	var docNodeObj yaml.Node
	if err := yaml.Unmarshal([]byte(textValue), &docNodeObj); err != nil {
		return nil, fmt.Errorf("parse schema yaml: %w", err)
	}

	if docNodeObj.Kind != yaml.DocumentNode || len(docNodeObj.Content) == 0 {
		return nil, fmt.Errorf("schema document is empty")
	}

	rootMapNodeObj := docNodeObj.Content[0]
	if rootMapNodeObj.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("schema root must be mapping")
	}

	if err := yamltool.ValidateDuplicateKey(rootMapNodeObj, ""); err != nil {
		return nil, fmt.Errorf("schema: %w", err)
	}

	rootObj, err := parseBranch("", "", rootMapNodeObj, false)
	if err != nil {
		return nil, err
	}

	return &ResultObj{
		RootObj: rootObj,
	}, nil
}

func parseBranch(pathText string, keyText string, nodeObj *yaml.Node, inheritedInterfaceFlag bool) (*BranchObj, error) {
	branchObj := &BranchObj{
		PathText:               pathText,
		KeyText:                keyText,
		InheritedInterfaceFlag: inheritedInterfaceFlag,
		EntryObjArr:            make([]*EntryObj, 0, len(nodeObj.Content)/2),
		BranchObjArr:           make([]*BranchObj, 0, len(nodeObj.Content)/2),
		LeafObjArr:             make([]*LeafObj, 0, len(nodeObj.Content)/2),
	}

	usageText, genInterfaceFlag, childKeyNodeArr, childValueNodeArr, err := extractBranchMeta(nodeObj, pathText)
	if err != nil {
		return nil, err
	}

	branchObj.UsageText = usageText
	branchObj.GenInterfaceFlag = genInterfaceFlag || inheritedInterfaceFlag

	for itemIndex := 0; itemIndex < len(childKeyNodeArr); itemIndex++ {
		childKeyNodeObj := childKeyNodeArr[itemIndex]
		childValueNodeObj := childValueNodeArr[itemIndex]
		if err = validateSchemaKey(pathText, childKeyNodeObj.Value); err != nil {
			return nil, err
		}

		childPathText := childKeyNodeObj.Value
		if pathText != "" {
			childPathText = pathText + "." + childKeyNodeObj.Value
		}

		rawLeafObj, leafFlag, extractErr := extractLeaf(childPathText, childValueNodeObj)
		if extractErr != nil {
			return nil, extractErr
		}

		if leafFlag {
			leafObj, leafErr := buildLeaf(childPathText, childKeyNodeObj.Value, rawLeafObj)
			if leafErr != nil {
				return nil, leafErr
			}

			branchObj.LeafObjArr = append(branchObj.LeafObjArr, leafObj)
			branchObj.EntryObjArr = append(branchObj.EntryObjArr, &EntryObj{
				LeafObj: leafObj,
			})
			continue
		}

		childBranchObj, branchErr := parseBranch(childPathText, childKeyNodeObj.Value, childValueNodeObj, branchObj.GenInterfaceFlag)
		if branchErr != nil {
			return nil, branchErr
		}

		branchObj.BranchObjArr = append(branchObj.BranchObjArr, childBranchObj)
		branchObj.EntryObjArr = append(branchObj.EntryObjArr, &EntryObj{
			BranchObj: childBranchObj,
		})
	}

	return branchObj, nil
}

func extractBranchMeta(nodeObj *yaml.Node, pathText string) (string, bool, []*yaml.Node, []*yaml.Node, error) {
	usageText := ""
	genInterfaceFlag := false
	childKeyNodeArr := make([]*yaml.Node, 0, len(nodeObj.Content)/2)
	childValueNodeArr := make([]*yaml.Node, 0, len(nodeObj.Content)/2)

	for itemIndex := 0; itemIndex < len(nodeObj.Content); itemIndex += 2 {
		keyNodeObj := nodeObj.Content[itemIndex]
		valueNodeObj := nodeObj.Content[itemIndex+1]

		switch keyNodeObj.Value {
		case "usage":
			usageValue, err := decodeUsage(valueNodeObj)
			if err != nil {
				return "", false, nil, nil, fmt.Errorf("schema node [%s] usage: %w", pathText, err)
			}

			usageText = usageValue
		case "gen_interface":
			if valueNodeObj.Kind != yaml.ScalarNode || valueNodeObj.Tag != "!!bool" {
				return "", false, nil, nil, fmt.Errorf("schema node [%s] gen_interface must be bool", pathText)
			}

			genInterfaceFlag = strings.EqualFold(valueNodeObj.Value, "true")
		default:
			childKeyNodeArr = append(childKeyNodeArr, keyNodeObj)
			childValueNodeArr = append(childValueNodeArr, valueNodeObj)
		}
	}

	return usageText, genInterfaceFlag, childKeyNodeArr, childValueNodeArr, nil
}

func extractLeaf(pathText string, nodeObj *yaml.Node) (*rawLeafObj, bool, error) {
	if nodeObj.Kind != yaml.MappingNode {
		return nil, false, fmt.Errorf("schema node [%s] must be mapping", pathText)
	}

	resultObj := &rawLeafObj{}
	childCountVal := 0
	hasLeafMarkerFlag := false
	hasUsageFlag := false

	for itemIndex := 0; itemIndex < len(nodeObj.Content); itemIndex += 2 {
		keyText := nodeObj.Content[itemIndex].Value
		valueNodeObj := nodeObj.Content[itemIndex+1]

		switch keyText {
		case "type":
			if valueNodeObj.Kind != yaml.ScalarNode {
				return nil, false, fmt.Errorf("schema node [%s] type must be scalar", pathText)
			}

			resultObj.TypeText = strings.TrimSpace(valueNodeObj.Value)
			hasLeafMarkerFlag = true
		case "value":
			resultObj.DefaultNodeObj = yamltool.CloneNode(valueNodeObj)
			hasLeafMarkerFlag = true
		case "min":
			resultObj.MinText = strings.TrimSpace(valueNodeObj.Value)
			hasLeafMarkerFlag = true
		case "max":
			resultObj.MaxText = strings.TrimSpace(valueNodeObj.Value)
			hasLeafMarkerFlag = true
		case "enum":
			enumValueTextArr, err := decodeEnum(valueNodeObj)
			if err != nil {
				return nil, false, fmt.Errorf("schema node [%s] enum: %w", pathText, err)
			}

			resultObj.EnumValueTextArr = enumValueTextArr
			hasLeafMarkerFlag = true
		case "enum_name":
			if valueNodeObj.Kind != yaml.ScalarNode {
				return nil, false, fmt.Errorf("schema node [%s] enum_name must be scalar", pathText)
			}

			resultObj.EnumNameText = strings.TrimSpace(valueNodeObj.Value)
			hasLeafMarkerFlag = true
		case "usage":
			// usage допустим и на ветке, и на листе; решение о виде узла не зависит от него.
			usageText, err := decodeUsage(valueNodeObj)
			if err != nil {
				return nil, false, fmt.Errorf("schema node [%s] usage: %w", pathText, err)
			}

			resultObj.UsageText = usageText
			hasUsageFlag = true
		case "gen_interface":
			// маркер только для веток; обрабатывается в parseBranch
			continue
		default:
			childCountVal++
		}
	}

	if hasLeafMarkerFlag && childCountVal > 0 {
		return nil, false, fmt.Errorf("schema node [%s] mixes leaf metadata and child branches", pathText)
	}

	if hasLeafMarkerFlag {
		return resultObj, true, nil
	}

	if childCountVal > 0 {
		return nil, false, nil
	}

	if hasUsageFlag {
		return nil, false, fmt.Errorf("schema node [%s] has usage but no leaf metadata or children", pathText)
	}

	return nil, false, fmt.Errorf("schema node [%s] is empty", pathText)
}

func buildLeaf(pathText string, keyText string, rawLeafObj *rawLeafObj) (*LeafObj, error) {
	if rawLeafObj.EnumNameText != "" && len(rawLeafObj.EnumValueTextArr) == 0 {
		return nil, fmt.Errorf("schema node [%s] enum_name requires enum", pathText)
	}

	typeObj, err := ParseType(rawLeafObj.TypeText, rawLeafObj.EnumValueTextArr)
	if err != nil {
		return nil, fmt.Errorf("schema node [%s] type: %w", pathText, err)
	}

	leafObj := &LeafObj{
		PathText:       pathText,
		KeyText:        keyText,
		UsageText:      rawLeafObj.UsageText,
		TypeObj:        typeObj,
		DefaultNodeObj: rawLeafObj.DefaultNodeObj,
	}

	if len(rawLeafObj.EnumValueTextArr) > 0 {
		enumObj, enumErr := BuildEnum(pathText, rawLeafObj.UsageText, rawLeafObj.EnumNameText, rawLeafObj.EnumValueTextArr)
		if enumErr != nil {
			return nil, enumErr
		}

		leafObj.EnumObj = enumObj
	}

	if rawLeafObj.MinText != "" {
		minRangeObj, rangeErr := ParseRange(typeObj, rawLeafObj.MinText)
		if rangeErr != nil {
			return nil, fmt.Errorf("schema node [%s] min: %w", pathText, rangeErr)
		}

		leafObj.MinRangeObj = minRangeObj
	}

	if rawLeafObj.MaxText != "" {
		maxRangeObj, rangeErr := ParseRange(typeObj, rawLeafObj.MaxText)
		if rangeErr != nil {
			return nil, fmt.Errorf("schema node [%s] max: %w", pathText, rangeErr)
		}

		leafObj.MaxRangeObj = maxRangeObj
	}

	if err = validateLeaf(pathText, leafObj); err != nil {
		return nil, err
	}

	return leafObj, nil
}

func validateSchemaKey(pathText string, keyText string) error {
	keyText = strings.TrimSpace(keyText)
	if keyText == "" {
		return fmt.Errorf("schema node [%s] key is empty", pathText)
	}

	if strings.Contains(keyText, ".") {
		return fmt.Errorf("schema node [%s] key [%s] must not contain '.'", pathText, keyText)
	}

	for itemIndex, itemRune := range keyText {
		switch {
		case itemRune >= 'a' && itemRune <= 'z':
		case itemRune >= 'A' && itemRune <= 'Z':
		case itemRune >= '0' && itemRune <= '9':
			if itemIndex == 0 {
				return fmt.Errorf("schema node [%s] key [%s] must start with ASCII letter", pathText, keyText)
			}
		case itemRune == '_' || itemRune == '-':
		default:
			return fmt.Errorf("schema node [%s] key [%s] must contain only ASCII letters, digits, '_' or '-'", pathText, keyText)
		}
	}

	return nil
}

func decodeUsage(nodeObj *yaml.Node) (string, error) {
	switch nodeObj.Kind {
	case yaml.ScalarNode:
		return strings.TrimSpace(nodeObj.Value), nil
	case yaml.SequenceNode:
		lineTextArr := make([]string, 0, len(nodeObj.Content))

		for _, itemNodeObj := range nodeObj.Content {
			if itemNodeObj.Kind != yaml.ScalarNode {
				return "", fmt.Errorf("usage list item must be scalar")
			}

			lineTextArr = append(lineTextArr, strings.TrimSpace(itemNodeObj.Value))
		}

		return strings.Join(lineTextArr, "\n"), nil
	default:
		return "", fmt.Errorf("usage must be scalar or list")
	}
}

func decodeEnum(nodeObj *yaml.Node) ([]string, error) {
	if nodeObj.Kind != yaml.SequenceNode {
		return nil, fmt.Errorf("enum must be sequence")
	}

	resultArr := make([]string, 0, len(nodeObj.Content))
	seenMap := make(map[string]struct{}, len(nodeObj.Content))

	for _, itemNodeObj := range nodeObj.Content {
		if itemNodeObj.Kind != yaml.ScalarNode {
			return nil, fmt.Errorf("enum value must be scalar")
		}

		valueText := strings.TrimSpace(itemNodeObj.Value)
		if valueText == "" {
			return nil, fmt.Errorf("enum value is empty")
		}

		if _, existsFlag := seenMap[valueText]; existsFlag {
			return nil, fmt.Errorf("duplicate enum value: %s", valueText)
		}

		seenMap[valueText] = struct{}{}
		resultArr = append(resultArr, valueText)
	}

	if len(resultArr) == 0 {
		return nil, fmt.Errorf("enum list is empty")
	}

	return resultArr, nil
}
