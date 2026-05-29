package goconfgen

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/amazing-generators/goconfgen/internal/semantic"
)

// // // // // // // // // //

const (
	cDefaultPackageName = "configgen"
	cDefaultSchemaFile  = "config.yml"
	cDefaultMinimalFile = "minimal.yml"
	cDefaultMediumFile  = "medium.yml"
)

// //

type ConfigObj struct {
	Schema      string
	SourceDir   string
	OutputDir   string
	PackageName string
	Formats     []string
	Presets     map[string]string
	Features    FeaturesObj
	Force       bool
}

type FeaturesObj struct {
	CLI        *bool
	Validate   *bool
	Render     *bool
	Presets    *bool
	Interfaces *bool
}

type ResultObj struct {
	OutputDir             string
	GeneratedFilePathArr  []string
	ResolvedSourcePath    string
	ResolvedPresetPathMap map[string]string
}

type normalizedConfigObj struct {
	SemanticConfig semantic.ConfigObj
	OutputDir      string
	Force          bool
}

// //

func normalizeConfig(config ConfigObj) (*normalizedConfigObj, error) {
	sourceDir, err := resolveSourceDir(config.SourceDir, config.Schema, config.Presets)
	if err != nil {
		return nil, err
	}

	schemaPath, err := resolveRequiredFile(config.Schema, sourceDir, cDefaultSchemaFile, "schema")
	if err != nil {
		return nil, err
	}
	schemaLabel := buildInputDisplayPath(config.Schema, config.SourceDir, schemaPath, cDefaultSchemaFile)

	outputDir, err := resolveOutputDir(config.OutputDir)
	if err != nil {
		return nil, err
	}

	formatTextArr, hasYAMLFlag, hasJSONFlag, hasHJSONFlag, err := normalizeFormats(config.Formats)
	if err != nil {
		return nil, err
	}
	hasCLIFlag := normalizeBool(config.Features.CLI, true)
	hasValidateFlag := normalizeBool(config.Features.Validate, true)
	hasRenderFlag := normalizeBool(config.Features.Render, true)
	hasPresetsFlag := normalizeBool(config.Features.Presets, true)
	hasInterfacesFlag := normalizeBool(config.Features.Interfaces, true)

	minimalPath := ""
	minimalFound := false
	mediumPath := ""
	mediumFound := false
	presetPathMap := map[string]string{}
	if hasPresetsFlag {
		if err = collectExplicitPresetPaths(config.Presets, presetPathMap); err != nil {
			return nil, err
		}

		if explicitPath, existsFlag := presetPathMap["minimal"]; existsFlag {
			minimalPath, err = absExistingFile(explicitPath, "minimal preset")
			if err != nil {
				return nil, err
			}
			minimalFound = true
			delete(presetPathMap, "minimal")
			if autoPath, autoFlag, autoErr := resolveOptionalFile("", sourceDir, cDefaultMinimalFile, "minimal preset"); autoErr != nil {
				return nil, autoErr
			} else if autoFlag && autoPath != "" {
				return nil, fmt.Errorf("preset name [minimal] is specified both explicitly and via source directory autodetection")
			}
		} else {
			minimalPath, minimalFound, err = resolveOptionalFile("", sourceDir, cDefaultMinimalFile, "minimal preset")
		}
		if err != nil {
			return nil, err
		}

		if explicitPath, existsFlag := presetPathMap["medium"]; existsFlag {
			mediumPath, err = absExistingFile(explicitPath, "medium preset")
			if err != nil {
				return nil, err
			}
			mediumFound = true
			delete(presetPathMap, "medium")
			if autoPath, autoFlag, autoErr := resolveOptionalFile("", sourceDir, cDefaultMediumFile, "medium preset"); autoErr != nil {
				return nil, autoErr
			} else if autoFlag && autoPath != "" {
				return nil, fmt.Errorf("preset name [medium] is specified both explicitly and via source directory autodetection")
			}
		} else {
			mediumPath, mediumFound, err = resolveOptionalFile("", sourceDir, cDefaultMediumFile, "medium preset")
		}
		if err != nil {
			return nil, err
		}
	} else if len(config.Presets) > 0 {
		return nil, fmt.Errorf("presets are disabled")
	}

	packageName := sanitizePackageName(config.PackageName)
	if packageName == "" {
		packageName = sanitizePackageName(filepath.Base(outputDir))
	}
	if packageName == "" {
		packageName = cDefaultPackageName
	}

	return &normalizedConfigObj{
		SemanticConfig: semantic.ConfigObj{
			SchemaPath:     schemaPath,
			SchemaLabel:    schemaLabel,
			MinimalPath:    minimalPath,
			MediumPath:     mediumPath,
			PresetPathMap:  presetPathMap,
			PackageName:    packageName,
			Formats:        formatTextArr,
			HasYAML:        hasYAMLFlag,
			HasJSON:        hasJSONFlag,
			HasHJSON:       hasHJSONFlag,
			HasCLI:         hasCLIFlag,
			HasValidate:    hasValidateFlag,
			HasRender:      hasRenderFlag,
			HasPresets:     hasPresetsFlag,
			HasInterfaces:  hasInterfacesFlag,
			MinimalPresent: minimalFound,
			MediumPresent:  mediumFound,
		},
		OutputDir: outputDir,
		Force:     config.Force,
	}, nil
}

func normalizeFormats(formatTextArr []string) ([]string, bool, bool, bool, error) {
	if len(formatTextArr) == 0 {
		return nil, false, false, false, fmt.Errorf("formats must not be empty")
	}

	seenFormatMap := make(map[string]struct{}, len(formatTextArr))
	resultArr := make([]string, 0, len(formatTextArr))

	for _, formatText := range formatTextArr {
		formatText = strings.TrimSpace(strings.ToLower(formatText))
		switch formatText {
		case "yaml", "json", "hjson":
		default:
			return nil, false, false, false, fmt.Errorf("unsupported format: %s", formatText)
		}

		if _, existsFlag := seenFormatMap[formatText]; existsFlag {
			continue
		}

		seenFormatMap[formatText] = struct{}{}
		resultArr = append(resultArr, formatText)
	}

	if len(resultArr) == 0 {
		return nil, false, false, false, fmt.Errorf("no formats are enabled")
	}

	return resultArr, containsTextObj(resultArr, "yaml"), containsTextObj(resultArr, "json"), containsTextObj(resultArr, "hjson"), nil
}

func normalizeBool(valueFlag *bool, defaultFlag bool) bool {
	if valueFlag == nil {
		return defaultFlag
	}

	return *valueFlag
}

func collectExplicitPresetPaths(presetPathByNameMap map[string]string, targetMap map[string]string) error {
	for nameText, pathText := range presetPathByNameMap {
		nameText = strings.TrimSpace(strings.ToLower(nameText))
		pathText = strings.TrimSpace(pathText)
		if nameText == "" {
			return fmt.Errorf("preset name must not be empty")
		}
		if nameText == "full" {
			return fmt.Errorf("preset name [full] is reserved for the schema-defaults preset")
		}
		if pathText == "" {
			return fmt.Errorf("preset [%s] path must not be empty", nameText)
		}
		if _, existsFlag := targetMap[nameText]; existsFlag {
			return fmt.Errorf("preset name [%s] is duplicated", nameText)
		}
		targetMap[nameText] = pathText
	}
	return nil
}

func resolveSourceDir(sourceDir string, schemaPath string, presetPathMap map[string]string) (string, error) {
	if strings.TrimSpace(sourceDir) != "" {
		return absDir(sourceDir, "source directory")
	}

	schemaPath = strings.TrimSpace(schemaPath)
	if schemaPath != "" {
		absPath, err := filepath.Abs(schemaPath)
		if err != nil {
			return "", fmt.Errorf("resolve input path: %w", err)
		}

		return filepath.Dir(absPath), nil
	}

	for _, pathValue := range presetPathMap {
		pathValue = strings.TrimSpace(pathValue)
		if pathValue == "" {
			continue
		}

		absPath, err := filepath.Abs(pathValue)
		if err != nil {
			return "", fmt.Errorf("resolve input path: %w", err)
		}

		return filepath.Dir(absPath), nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("read working directory: %w", err)
	}

	return cwd, nil
}

func resolveRequiredFile(explicitPath string, sourceDir string, defaultName string, label string) (string, error) {
	if strings.TrimSpace(explicitPath) != "" {
		return absExistingFile(explicitPath, label)
	}

	return absExistingFile(filepath.Join(sourceDir, defaultName), label)
}

func resolveOptionalFile(explicitPath string, sourceDir string, defaultName string, label string) (string, bool, error) {
	if strings.TrimSpace(explicitPath) != "" {
		pathValue, err := absExistingFile(explicitPath, label)
		if err != nil {
			return "", false, err
		}

		return pathValue, true, nil
	}

	pathValue := filepath.Join(sourceDir, defaultName)
	infoObj, err := os.Stat(pathValue)
	if err == nil {
		if infoObj.IsDir() {
			return "", false, fmt.Errorf("%s path is directory: %s", label, pathValue)
		}

		absPath, absErr := filepath.Abs(pathValue)
		if absErr != nil {
			return "", false, fmt.Errorf("resolve %s path: %w", label, absErr)
		}

		return absPath, true, nil
	}

	if errors.Is(err, os.ErrNotExist) {
		return "", false, nil
	}

	return "", false, fmt.Errorf("stat %s path: %w", label, err)
}

func resolveOutputDir(outputDir string) (string, error) {
	if strings.TrimSpace(outputDir) == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("read working directory: %w", err)
		}

		return cwd, nil
	}

	absPath, err := filepath.Abs(outputDir)
	if err != nil {
		return "", fmt.Errorf("resolve output directory: %w", err)
	}
	return absPath, nil
}

func buildInputDisplayPath(explicitPath string, sourceDir string, resolvedPath string, defaultName string) string {
	explicitPath = strings.TrimSpace(explicitPath)
	if explicitPath != "" {
		return cleanDisplayPath(explicitPath, resolvedPath)
	}

	sourceDir = strings.TrimSpace(sourceDir)
	if sourceDir != "" {
		return cleanDisplayPath(filepath.Join(sourceDir, defaultName), resolvedPath)
	}

	return cleanDisplayPath(resolvedPath, resolvedPath)
}

func cleanDisplayPath(pathValue string, resolvedPath string) string {
	pathValue = strings.TrimSpace(pathValue)
	if pathValue == "" {
		pathValue = resolvedPath
	}

	if filepath.IsAbs(pathValue) {
		if cwd, err := os.Getwd(); err == nil {
			if relPath, relErr := filepath.Rel(cwd, pathValue); relErr == nil && relPath != "." && !strings.HasPrefix(relPath, ".."+string(filepath.Separator)) && relPath != ".." {
				pathValue = relPath
			}
		}
	}

	return filepath.ToSlash(filepath.Clean(pathValue))
}

func absDir(pathValue string, label string) (string, error) {
	absPath, err := filepath.Abs(pathValue)
	if err != nil {
		return "", fmt.Errorf("resolve %s path: %w", label, err)
	}

	infoObj, err := os.Stat(absPath)
	if err != nil {
		return "", fmt.Errorf("stat %s path: %w", label, err)
	}

	if !infoObj.IsDir() {
		return "", fmt.Errorf("%s path is not directory: %s", label, absPath)
	}

	return absPath, nil
}

func absExistingFile(pathValue string, label string) (string, error) {
	absPath, err := filepath.Abs(pathValue)
	if err != nil {
		return "", fmt.Errorf("resolve %s path: %w", label, err)
	}

	infoObj, err := os.Stat(absPath)
	if err != nil {
		return "", fmt.Errorf("stat %s path: %w", label, err)
	}

	if infoObj.IsDir() {
		return "", fmt.Errorf("%s path is directory: %s", label, absPath)
	}

	return absPath, nil
}

func sanitizePackageName(packageName string) string {
	packageName = strings.TrimSpace(strings.ToLower(packageName))
	if packageName == "" {
		return ""
	}

	var builder strings.Builder

	for _, itemRune := range packageName {
		switch {
		case itemRune >= 'a' && itemRune <= 'z':
			builder.WriteRune(itemRune)
		case itemRune >= '0' && itemRune <= '9':
			if builder.Len() > 0 {
				builder.WriteRune(itemRune)
			}
		default:
			if builder.Len() > 0 && !strings.HasSuffix(builder.String(), "_") {
				builder.WriteByte('_')
			}
		}
	}

	resultValue := strings.Trim(builder.String(), "_")
	if resultValue == "" {
		return ""
	}

	return resultValue
}

func containsTextObj(textArr []string, textValue string) bool {
	for _, itemText := range textArr {
		if itemText == textValue {
			return true
		}
	}

	return false
}
