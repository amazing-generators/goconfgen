package goconfgen

import (
	"fmt"
	"path/filepath"
	"sort"

	"github.com/amazing-generators/goconfgen/internal/codegen"
	"github.com/amazing-generators/goconfgen/internal/ir"
	"github.com/amazing-generators/goconfgen/internal/source"
)

// // // // // // // // // //

func Run(config ConfigObj) (*ResultObj, error) {
	normalizedConfig, err := normalizeConfig(config)
	if err != nil {
		return nil, err
	}

	sourceObj, err := source.Load(source.ConfigObj{
		SchemaPath:  normalizedConfig.IRConfig.SchemaPath,
		MinimalPath: normalizedConfig.IRConfig.MinimalPath,
		MediumPath:  normalizedConfig.IRConfig.MediumPath,
	})
	if err != nil {
		return nil, err
	}

	irConfig := normalizedConfig.IRConfig
	irConfig.SchemaText = sourceObj.SchemaText
	irConfig.MinimalText = sourceObj.MinimalText
	irConfig.MediumText = sourceObj.MediumText

	packageObj, err := ir.Build(irConfig)
	if err != nil {
		return nil, err
	}

	fileObjArr, err := codegen.Build(packageObj)
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
	if err = cleanupStaleGeneratedFiles(normalizedConfig.OutputDir, keepNameMap, codegen.KnownGeneratedFileNameArr(), normalizedConfig.Force); err != nil {
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

	return &ResultObj{
		OutputDir:            normalizedConfig.OutputDir,
		GeneratedFilePathArr: generatedFilePathArr,
		ResolvedSchemaPath:   normalizedConfig.IRConfig.SchemaPath,
		ResolvedMinimalPath:  normalizedConfig.IRConfig.MinimalPath,
		ResolvedMediumPath:   normalizedConfig.IRConfig.MediumPath,
	}, nil
}
