package yamltool

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// // // // // // // // // //

// CloneNode возвращает рекурсивную копию yaml.Node вместе с детьми.
func CloneNode(nodeObj *yaml.Node) *yaml.Node {
	if nodeObj == nil {
		return nil
	}

	clonedNodeObj := *nodeObj
	clonedNodeObj.Content = make([]*yaml.Node, 0, len(nodeObj.Content))

	for _, childNodeObj := range nodeObj.Content {
		clonedNodeObj.Content = append(clonedNodeObj.Content, CloneNode(childNodeObj))
	}

	return &clonedNodeObj
}

// ValidateDuplicateKey рекурсивно проверяет уникальность ключей внутри одного mapping-узла.
// При нарушении возвращает ошибку с полным dot-путём.
func ValidateDuplicateKey(nodeObj *yaml.Node, prefixText string) error {
	if nodeObj.Kind != yaml.MappingNode {
		for _, childNodeObj := range nodeObj.Content {
			if err := ValidateDuplicateKey(childNodeObj, prefixText); err != nil {
				return err
			}
		}

		return nil
	}

	seenMap := make(map[string]struct{}, len(nodeObj.Content)/2)

	for itemIndex := 0; itemIndex < len(nodeObj.Content); itemIndex += 2 {
		keyNodeObj := nodeObj.Content[itemIndex]
		valueNodeObj := nodeObj.Content[itemIndex+1]
		keyText := keyNodeObj.Value
		pathText := keyText
		if prefixText != "" {
			pathText = prefixText + "." + keyText
		}

		if _, existsFlag := seenMap[keyText]; existsFlag {
			return fmt.Errorf("duplicate key: %s", pathText)
		}

		seenMap[keyText] = struct{}{}

		if err := ValidateDuplicateKey(valueNodeObj, pathText); err != nil {
			return err
		}
	}

	return nil
}
