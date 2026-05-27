package ir

import "github.com/amazing-generators/goconfgen/internal/schema"

// // // // // // // // // //

type ConfigObj struct {
	SchemaText     string
	MinimalText    string
	MediumText     string
	PackageName    string
	SchemaPath     string
	SchemaLabel    string
	MinimalPath    string
	MediumPath     string
	Formats        []string
	HasYAML        bool
	HasJSON        bool
	HasHJSON       bool
	HasCLI         bool
	HasValidate    bool
	HasRender      bool
	HasPresets     bool
	HasInterfaces  bool
	MinimalPresent bool
	MediumPresent  bool
}

type FieldObj struct {
	PathText        string
	KeyText         string
	AccessorText    string
	UsageText       string
	TypeObj         schema.TypeObj
	KindObj         FieldKindObj
	BitSizeVal      int
	EnumObj         *schema.EnumObj
	MinRangeObj     *schema.RangeObj
	MaxRangeObj     *schema.RangeObj
	DefaultCodeText string
	GoTypeText      string
}

type EntryObj struct {
	BranchObj *BranchObj
	FieldObj  *FieldObj
}

type BranchObj struct {
	PathText          string
	KeyText           string
	TypeNameText      string
	InterfaceTypeText string
	UsageText         string
	AccessorText      string
	GenInterfaceFlag  bool
	EntryObjArr       []*EntryObj
	BranchObjArr      []*BranchObj
	FieldObjArr       []*FieldObj
}

type PresetObj struct {
	NameText              string
	SourcePathText        string
	YAMLText              string
	JSONText              string
	HJSONText             string
	PresentFlag           bool
	GeneratedFromDefaults bool
}

type PackageObj struct {
	PackageName      string
	SchemaPath       string
	SchemaLabel      string
	MinimalPath      string
	MediumPath       string
	Formats          []string
	HasYAML          bool
	HasJSON          bool
	HasHJSON         bool
	HasCLI           bool
	HasValidate      bool
	HasRender        bool
	HasPresets       bool
	HasInterfaces    bool
	HasSize          bool
	HasDuration      bool
	HasMapAny        bool
	HasTextSlice     bool
	PrimaryFormat    string
	RootObj          *BranchObj
	FieldObjArr      []*FieldObj
	EnumObjArr       []*schema.EnumObj
	FullPresetObj    PresetObj
	MinimalPresetObj PresetObj
	MediumPresetObj  PresetObj
}
