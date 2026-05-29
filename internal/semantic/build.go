package semantic

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/amazing-generators/goconfgen/internal/schema"
	"gopkg.in/yaml.v3"
)

// // // // // // // // // //

func Build(config ConfigObj) (*PackageObj, error) {
	schemaObj, err := schema.Parse(config.SchemaText)
	if err != nil {
		return nil, err
	}

	packageObj := &PackageObj{
		PackageName:   config.PackageName,
		SchemaPath:    config.SchemaPath,
		SchemaLabel:   config.SchemaLabel,
		MinimalPath:   config.MinimalPath,
		MediumPath:    config.MediumPath,
		Formats:       append([]string(nil), config.Formats...),
		HasYAML:       config.HasYAML,
		HasJSON:       config.HasJSON,
		HasHJSON:      config.HasHJSON,
		HasCLI:        config.HasCLI,
		HasValidate:   config.HasValidate,
		HasRender:     config.HasRender,
		HasPresets:    config.HasPresets,
		HasInterfaces: config.HasInterfaces,
	}

	rootObj, fieldObjArr, err := buildBranch(schemaObj.RootObj, "")
	if err != nil {
		return nil, err
	}

	if err = validateGeneratedNamespace(rootObj, fieldObjArr); err != nil {
		return nil, err
	}

	packageObj.RootObj = rootObj
	packageObj.FieldObjArr = fieldObjArr
	packageObj.EnumObjArr = collectEnum(fieldObjArr)
	packageObj.HasSize = hasSizeField(fieldObjArr)
	packageObj.HasDuration = hasDurationField(fieldObjArr)
	packageObj.HasMapAny = hasMapAnyField(fieldObjArr)
	packageObj.HasTextSlice = hasTextSliceField(fieldObjArr)
	packageObj.PrimaryFormat = primaryFormatTitle(packageObj)

	if err = validateFeatureCombination(packageObj); err != nil {
		return nil, err
	}

	if packageObj.HasPresets {
		fullPresetObj, presetErr := buildFullPreset(schemaObj.RootObj, packageObj)
		if presetErr != nil {
			return nil, presetErr
		}

		packageObj.FullPresetObj = fullPresetObj

		minimalPresetObj, presetErr := buildOptionalPreset("minimal", config.MinimalText, config.MinimalPath, config.MinimalPresent, schemaObj.RootObj, packageObj)
		if presetErr != nil {
			return nil, presetErr
		}

		packageObj.MinimalPresetObj = minimalPresetObj

		mediumPresetObj, presetErr := buildOptionalPreset("medium", config.MediumText, config.MediumPath, config.MediumPresent, schemaObj.RootObj, packageObj)
		if presetErr != nil {
			return nil, presetErr
		}

		packageObj.MediumPresetObj = mediumPresetObj

		presetObjArr := []PresetObj{fullPresetObj}
		if minimalPresetObj.PresentFlag {
			presetObjArr = append(presetObjArr, minimalPresetObj)
		}
		if mediumPresetObj.PresentFlag {
			presetObjArr = append(presetObjArr, mediumPresetObj)
		}

		nameTextArr := make([]string, 0, len(config.PresetTextMap))
		for nameText := range config.PresetTextMap {
			nameTextArr = append(nameTextArr, nameText)
		}
		sort.Strings(nameTextArr)
		for _, nameText := range nameTextArr {
			presetObj, presetErr := buildOptionalPreset(nameText, config.PresetTextMap[nameText], config.PresetPathMap[nameText], true, schemaObj.RootObj, packageObj)
			if presetErr != nil {
				return nil, presetErr
			}
			presetObjArr = append(presetObjArr, presetObj)
		}

		packageObj.PresetObjArr = presetObjArr
	}

	return packageObj, nil
}

func buildBranch(branchObj *schema.BranchObj, parentAccessorText string) (*BranchObj, []*FieldObj, error) {
	typeNameText := "ConfigObj"
	accessorText := ""

	if branchObj.PathText != "" {
		typeNameText = goTypeName(branchObj.PathText) + "Obj"
		accessorText = goName(branchObj.KeyText)
		if parentAccessorText != "" {
			accessorText = parentAccessorText + "." + goName(branchObj.KeyText)
		}
	}

	resultObj := &BranchObj{
		PathText:          branchObj.PathText,
		KeyText:           branchObj.KeyText,
		TypeNameText:      typeNameText,
		InterfaceTypeText: strings.TrimSuffix(typeNameText, "Obj") + "Interface",
		UsageText:         branchObj.UsageText,
		AccessorText:      accessorText,
		GenInterfaceFlag:  branchObj.GenInterfaceFlag,
		EntryObjArr:       make([]*EntryObj, 0, len(branchObj.EntryObjArr)),
		BranchObjArr:      make([]*BranchObj, 0, len(branchObj.BranchObjArr)),
		FieldObjArr:       make([]*FieldObj, 0, len(branchObj.LeafObjArr)),
	}

	fieldObjArr := make([]*FieldObj, 0, len(branchObj.LeafObjArr)+len(branchObj.BranchObjArr)*4)

	for _, entryObj := range branchObj.EntryObjArr {
		switch {
		case entryObj.LeafObj != nil:
			fieldObj, err := buildField(entryObj.LeafObj, accessorText)
			if err != nil {
				return nil, nil, err
			}

			resultObj.FieldObjArr = append(resultObj.FieldObjArr, fieldObj)
			resultObj.EntryObjArr = append(resultObj.EntryObjArr, &EntryObj{
				FieldObj: fieldObj,
			})
			fieldObjArr = append(fieldObjArr, fieldObj)
		case entryObj.BranchObj != nil:
			childResultObj, childFieldObjArr, err := buildBranch(entryObj.BranchObj, accessorText)
			if err != nil {
				return nil, nil, err
			}

			resultObj.BranchObjArr = append(resultObj.BranchObjArr, childResultObj)
			resultObj.EntryObjArr = append(resultObj.EntryObjArr, &EntryObj{
				BranchObj: childResultObj,
			})
			fieldObjArr = append(fieldObjArr, childFieldObjArr...)
		default:
			return nil, nil, fmt.Errorf("schema entry [%s] has neither branch nor leaf", branchObj.PathText)
		}
	}

	return resultObj, fieldObjArr, nil
}

func buildField(leafObj *schema.LeafObj, parentAccessorText string) (*FieldObj, error) {
	accessorText := goName(leafObj.KeyText)
	if parentAccessorText != "" {
		accessorText = parentAccessorText + "." + goName(leafObj.KeyText)
	}

	defaultCodeText, err := defaultCodeForLeaf(leafObj)
	if err != nil {
		return nil, err
	}

	enumObj := leafObj.EnumObj
	goTypeText := goTypeOfField(leafObj.TypeObj, enumObj)
	kindObj, bitSizeVal := ClassifyKind(leafObj.TypeObj)

	return &FieldObj{
		PathText:        leafObj.PathText,
		KeyText:         leafObj.KeyText,
		AccessorText:    accessorText,
		UsageText:       leafObj.UsageText,
		TypeObj:         leafObj.TypeObj,
		KindObj:         kindObj,
		BitSizeVal:      bitSizeVal,
		EnumObj:         enumObj,
		MinRangeObj:     leafObj.MinRangeObj,
		MaxRangeObj:     leafObj.MaxRangeObj,
		DefaultCodeText: defaultCodeText,
		GoTypeText:      goTypeText,
	}, nil
}

// goTypeOfField возвращает Go-имя типа в сгенерированном пакете для поля.
func goTypeOfField(typeObj schema.TypeObj, enumObj *schema.EnumObj) string {
	switch {
	case typeObj.IsEnum && typeObj.Family == schema.TypeFamilyScalar:
		return enumObj.TypeName
	case typeObj.IsEnum && typeObj.Family == schema.TypeFamilyArray:
		return "[]" + enumObj.TypeName
	case typeObj.IsSize && typeObj.Family == schema.TypeFamilyScalar:
		return "SizeObj"
	case typeObj.IsSize && typeObj.Family == schema.TypeFamilyArray:
		return "[]SizeObj"
	default:
		return typeObj.GoText
	}
}

func validateGeneratedNamespace(rootObj *BranchObj, fieldObjArr []*FieldObj) error {
	fieldByPathMap := make(map[string]string, len(fieldObjArr))
	fieldByAccessorMap := make(map[string]string, len(fieldObjArr))
	flagVarByNameMap := make(map[string]string, len(fieldObjArr))
	typeByNameMap := map[string]string{}
	interfaceByNameMap := map[string]string{}

	var walkBranch func(*BranchObj) error
	walkBranch = func(branchObj *BranchObj) error {
		if branchObj.PathText != "" {
			if err := claimName(typeByNameMap, branchObj.TypeNameText, branchObj.PathText, "generated type"); err != nil {
				return err
			}

			if branchObj.GenInterfaceFlag {
				if err := claimName(interfaceByNameMap, branchObj.InterfaceTypeText, branchObj.PathText, "generated interface"); err != nil {
					return err
				}
			}
		}

		memberByNameMap := make(map[string]string, len(branchObj.BranchObjArr)+len(branchObj.FieldObjArr))
		for _, childBranchObj := range branchObj.BranchObjArr {
			memberNameText := goName(childBranchObj.KeyText)
			if err := claimName(memberByNameMap, memberNameText, childBranchObj.PathText, "generated branch accessor"); err != nil {
				return err
			}
		}

		for _, fieldObj := range branchObj.FieldObjArr {
			memberNameText := goName(fieldObj.KeyText)
			if err := claimName(memberByNameMap, memberNameText, fieldObj.PathText, "generated field accessor"); err != nil {
				return err
			}
		}

		for _, childBranchObj := range branchObj.BranchObjArr {
			if err := walkBranch(childBranchObj); err != nil {
				return err
			}
		}

		return nil
	}

	for _, fieldObj := range fieldObjArr {
		if err := claimName(fieldByPathMap, fieldObj.PathText, fieldObj.PathText, "schema path"); err != nil {
			return err
		}

		if err := claimName(fieldByAccessorMap, fieldObj.AccessorText, fieldObj.PathText, "generated field path"); err != nil {
			return err
		}

		if fieldObj.EnumObj != nil {
			if err := claimName(typeByNameMap, fieldObj.EnumObj.TypeName, fieldObj.PathText, "generated enum type"); err != nil {
				return err
			}
			for _, constNameText := range fieldObj.EnumObj.ConstNameArr {
				if err := claimName(typeByNameMap, constNameText, fieldObj.PathText, "generated enum constant"); err != nil {
					return err
				}
			}
		}

		if err := claimName(flagVarByNameMap, fieldFlagVarName(fieldObj.PathText), fieldObj.PathText, "generated flag variable"); err != nil {
			return err
		}
	}

	return walkBranch(rootObj)
}

func validateFeatureCombination(packageObj *PackageObj) error {
	hasFileFormatFlag := packageObj.HasYAML || packageObj.HasJSON || packageObj.HasHJSON
	if packageObj.HasRender && !hasFileFormatFlag {
		return fmt.Errorf("render requires at least one file format")
	}
	if packageObj.HasPresets && !hasFileFormatFlag {
		return fmt.Errorf("presets require at least one file format")
	}
	if !packageObj.HasInterfaces && hasGeneratedInterface(packageObj.RootObj) {
		return fmt.Errorf("interfaces are disabled but schema requests generated interfaces")
	}
	return nil
}

func hasGeneratedInterface(branchObj *BranchObj) bool {
	if branchObj.GenInterfaceFlag {
		return true
	}
	for _, childObj := range branchObj.BranchObjArr {
		if hasGeneratedInterface(childObj) {
			return true
		}
	}
	return false
}

func fieldFlagVarName(pathText string) string {
	return "cFlag" + goName(strings.ReplaceAll(pathText, ".", "_")) + "Obj"
}

func claimName(ownerByNameMap map[string]string, nameText string, ownerText string, kindText string) error {
	if previousOwnerText, existsFlag := ownerByNameMap[nameText]; existsFlag {
		return fmt.Errorf("%s collision: [%s] conflicts with [%s] via [%s]", kindText, ownerText, previousOwnerText, nameText)
	}

	ownerByNameMap[nameText] = ownerText
	return nil
}

func collectEnum(fieldObjArr []*FieldObj) []*schema.EnumObj {
	resultArr := make([]*schema.EnumObj, 0, len(fieldObjArr))

	for _, fieldObj := range fieldObjArr {
		if fieldObj.EnumObj == nil {
			continue
		}

		resultArr = append(resultArr, fieldObj.EnumObj)
	}

	return resultArr
}

func buildFullPreset(rootObj *schema.BranchObj, packageObj *PackageObj) (PresetObj, error) {
	return buildPresetFormats("full", renderFullBranch(rootObj), true, true, packageObj)
}

func buildOptionalPreset(nameText string, textValue string, sourcePathText string, presentFlag bool, rootObj *schema.BranchObj, packageObj *PackageObj) (PresetObj, error) {
	if !presentFlag {
		return PresetObj{NameText: nameText}, nil
	}

	rootNodeObj, err := validateAndRenderPresetNode(textValue, rootObj, nameText)
	if err != nil {
		return PresetObj{}, err
	}

	return buildPresetFormats(nameText, rootNodeObj, true, false, packageObj, sourcePathText)
}

func buildPresetFormats(nameText string, rootNodeObj *yaml.Node, presentFlag bool, generatedFromDefaultsFlag bool, packageObj *PackageObj, sourcePathText ...string) (PresetObj, error) {
	yamlText := ""
	if packageObj.HasYAML {
		renderedText, err := renderYAMLNode(rootNodeObj)
		if err != nil {
			return PresetObj{}, fmt.Errorf("render preset [%s] yaml: %w", nameText, err)
		}

		yamlText = ensureTrailingNewline(renderedText)
	}

	jsonText := ""
	if packageObj.HasJSON {
		renderedText, err := renderJSONNode(rootNodeObj)
		if err != nil {
			return PresetObj{}, fmt.Errorf("render preset [%s] json: %w", nameText, err)
		}

		jsonText = ensureTrailingNewline(renderedText)
	}

	hjsonText := ""
	if packageObj.HasHJSON {
		renderedText, err := renderHJSONNode(rootNodeObj)
		if err != nil {
			return PresetObj{}, fmt.Errorf("render preset [%s] hjson: %w", nameText, err)
		}

		hjsonText = ensureTrailingNewline(renderedText)
	}

	return PresetObj{
		NameText:              nameText,
		SourcePathText:        firstOptionalText(sourcePathText...),
		YAMLText:              yamlText,
		JSONText:              jsonText,
		HJSONText:             hjsonText,
		PresentFlag:           presentFlag,
		GeneratedFromDefaults: generatedFromDefaultsFlag,
	}, nil
}

func ensureTrailingNewline(textValue string) string {
	if strings.HasSuffix(textValue, "\n") {
		return textValue
	}

	return textValue + "\n"
}

func firstOptionalText(textArr ...string) string {
	if len(textArr) == 0 {
		return ""
	}

	return textArr[0]
}

func goTypeName(pathText string) string {
	return schema.GoTypeName(pathText)
}

func goName(textValue string) string {
	return schema.GoName(textValue)
}

func hasSizeField(fieldObjArr []*FieldObj) bool {
	for _, fieldObj := range fieldObjArr {
		if fieldObj.TypeObj.IsSize {
			return true
		}
	}

	return false
}

func hasDurationField(fieldObjArr []*FieldObj) bool {
	for _, fieldObj := range fieldObjArr {
		if fieldObj.TypeObj.IsDuration {
			return true
		}
	}

	return false
}

func hasMapAnyField(fieldObjArr []*FieldObj) bool {
	for _, fieldObj := range fieldObjArr {
		if fieldObj.TypeObj.SchemaText == "map[string]any" {
			return true
		}
	}

	return false
}

func hasTextSliceField(fieldObjArr []*FieldObj) bool {
	for _, fieldObj := range fieldObjArr {
		if fieldObj.TypeObj.Family != schema.TypeFamilyArray {
			continue
		}

		if fieldObj.TypeObj.IsEnum || fieldObj.TypeObj.IsDuration || fieldObj.TypeObj.IsSize {
			return true
		}
	}

	return false
}

func primaryFormatTitle(packageObj *PackageObj) string {
	switch {
	case packageObj.HasYAML:
		return "YAML"
	case packageObj.HasJSON:
		return "JSON"
	case packageObj.HasHJSON:
		return "HJSON"
	default:
		return ""
	}
}

func renderYAMLNode(nodeObj *yaml.Node) (string, error) {
	var bufferObj bytes.Buffer
	encoderObj := yaml.NewEncoder(&bufferObj)
	encoderObj.SetIndent(2)

	if err := encoderObj.Encode(nodeObj); err != nil {
		return "", err
	}

	if err := encoderObj.Close(); err != nil {
		return "", err
	}

	return bufferObj.String(), nil
}
