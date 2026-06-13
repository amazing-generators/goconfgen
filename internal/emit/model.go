package emit

import (
	"github.com/amazing-generators/goconfgen/internal/schema"
	"github.com/amazing-generators/goconfgen/internal/semantic"
)

// // // // // // // // // //

type FileObj struct {
	RelativePath string
	DataArr      []byte
}

type PresetTemplateObj struct {
	TitleText      string
	NameText       string
	SourcePathText string
	PresentFlag    bool
	FullFlag       bool
	YAMLText       string
	JSONText       string
	HJSONText      string
	YAMLStart      int
	YAMLLen        int
	JSONStart      int
	JSONLen        int
	HJSONStart     int
	HJSONLen       int
}

type TemplateDataObj struct {
	PackageName            string
	RootObj                *semantic.BranchObj
	FieldObjArr            []*semantic.FieldObj
	EnumObjArr             []*schema.EnumObj
	HasYAML                bool
	HasJSON                bool
	HasHJSON               bool
	HasCLI                 bool
	HasValidate            bool
	HasRender              bool
	HasPresets             bool
	HasInterfaces          bool
	HasSize                bool
	HasDuration            bool
	HasMapAny              bool
	HasTextSlice           bool
	PrimaryFormatTitle     string
	TypeImportTextArr      []string
	AccessorImportTextArr  []string
	FieldMetaImportTextArr []string
	RuntimeImportTextArr   []string
	FlagImportTextArr      []string
	ValidateImportTextArr  []string
	PresetObjArr           []PresetTemplateObj
	PresetDataArr          []byte
}
