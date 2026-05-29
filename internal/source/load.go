package source

import (
	"fmt"
	"os"
)

// // // // // // // // // //

type ConfigObj struct {
	SchemaPath    string
	MinimalPath   string
	MediumPath    string
	PresetPathMap map[string]string
}

type ResultObj struct {
	SchemaText    string
	MinimalText   string
	MediumText    string
	PresetTextMap map[string]string
}

// //

func Load(config ConfigObj) (*ResultObj, error) {
	schemaArr, err := os.ReadFile(config.SchemaPath)
	if err != nil {
		return nil, fmt.Errorf("read schema file: %w", err)
	}

	resultObj := &ResultObj{
		SchemaText:    string(schemaArr),
		PresetTextMap: make(map[string]string, len(config.PresetPathMap)),
	}

	if config.MinimalPath != "" {
		dataArr, readErr := os.ReadFile(config.MinimalPath)
		if readErr != nil {
			return nil, fmt.Errorf("read minimal preset file: %w", readErr)
		}

		resultObj.MinimalText = string(dataArr)
	}

	if config.MediumPath != "" {
		dataArr, readErr := os.ReadFile(config.MediumPath)
		if readErr != nil {
			return nil, fmt.Errorf("read medium preset file: %w", readErr)
		}

		resultObj.MediumText = string(dataArr)
	}

	for nameText, pathText := range config.PresetPathMap {
		dataArr, readErr := os.ReadFile(pathText)
		if readErr != nil {
			return nil, fmt.Errorf("read preset [%s] file: %w", nameText, readErr)
		}

		resultObj.PresetTextMap[nameText] = string(dataArr)
	}

	return resultObj, nil
}
