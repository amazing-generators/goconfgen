package goconfgen

import (
	"fmt"
	"path/filepath"
	"sort"

	"github.com/amazing-generators/goconfgen/internal/emit"
	"github.com/amazing-generators/goconfgen/internal/semantic"
	"github.com/amazing-generators/goconfgen/internal/source"
)

// // // // // // // // // //

func Run(config ConfigObj) (*ResultObj, error) {
	normalizedConfig, err := normalizeConfig(config)
	if err != nil {
		return nil, err
	}

	sourceObj, err := source.Load(source.ConfigObj{
		SchemaPath:    normalizedConfig.SemanticConfig.SchemaPath,
		MinimalPath:   normalizedConfig.SemanticConfig.MinimalPath,
		MediumPath:    normalizedConfig.SemanticConfig.MediumPath,
		PresetPathMap: normalizedConfig.SemanticConfig.PresetPathMap,
	})
	if err != nil {
		return nil, err
	}

	semanticConfig := normalizedConfig.SemanticConfig
	semanticConfig.SchemaText = sourceObj.SchemaText
	semanticConfig.MinimalText = sourceObj.MinimalText
	semanticConfig.MediumText = sourceObj.MediumText
	semanticConfig.PresetTextMap = sourceObj.PresetTextMap

	packageObj, err := semantic.Build(semanticConfig)
	if err != nil {
		return nil, err
	}

	fileObjArr, err := emit.Build(packageObj)
	if err != nil {
		return nil, err
	}

	if err = prepareOutputDir(normalizedConfig.OutputDir, normalizedConfig.Force); err != nil {
		return nil, err
	}

	keepNameMap := make(map[string]struct{}, len(fileObjArr))
	for _, fileObj := range fileObjArr {
		keepNameMap[fileObj.RelativePath] = struct{}{}
	}
	if err = cleanupStaleGeneratedFiles(normalizedConfig.OutputDir, keepNameMap, normalizedConfig.Force); err != nil {
		return nil, err
	}

	generatedFilePathArr := make([]string, 0, len(fileObjArr))

	for _, fileObj := range fileObjArr {
		targetPath := filepath.Join(normalizedConfig.OutputDir, fileObj.RelativePath)
		if err = writeFileAtomically(targetPath, fileObj.DataArr, normalizedConfig.Force); err != nil {
			return nil, fmt.Errorf("write generated file [%s]: %w", fileObj.RelativePath, err)
		}

		generatedFilePathArr = append(generatedFilePathArr, targetPath)
	}

	sort.Strings(generatedFilePathArr)
	presetPathMap := make(map[string]string, len(normalizedConfig.SemanticConfig.PresetPathMap)+2)
	if normalizedConfig.SemanticConfig.MinimalPath != "" {
		presetPathMap["minimal"] = normalizedConfig.SemanticConfig.MinimalPath
	}
	if normalizedConfig.SemanticConfig.MediumPath != "" {
		presetPathMap["medium"] = normalizedConfig.SemanticConfig.MediumPath
	}
	for nameText, pathText := range normalizedConfig.SemanticConfig.PresetPathMap {
		presetPathMap[nameText] = pathText
	}

	return &ResultObj{
		OutputDir:             normalizedConfig.OutputDir,
		GeneratedFilePathArr:  generatedFilePathArr,
		ResolvedSourcePath:    normalizedConfig.SemanticConfig.SchemaPath,
		ResolvedPresetPathMap: presetPathMap,
	}, nil
}
