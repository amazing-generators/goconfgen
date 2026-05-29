package schema

import "gopkg.in/yaml.v3"

// // // // // // // // // //

type NodeKindObj string

const (
	NodeKindBranch NodeKindObj = "branch"
	NodeKindLeaf   NodeKindObj = "leaf"
)

type TypeFamilyObj string

const (
	TypeFamilyScalar TypeFamilyObj = "scalar"
	TypeFamilyArray  TypeFamilyObj = "array"
	TypeFamilyMap    TypeFamilyObj = "map"
)

type TypeObj struct {
	SchemaText     string
	GoText         string
	Family         TypeFamilyObj
	ElemSchemaText string
	IsDuration     bool
	IsSize         bool
	IsEnum         bool
	IsNumeric      bool
	IsSigned       bool
	IsFloat        bool
	MapValueText   string
}

type RangeObj struct {
	Text        string
	GoLiteral   string
	SignedVal   int64
	UnsignedVal uint64
	FloatVal    float64
	Kind        string
}

type EnumObj struct {
	PathText     string
	NameText     string
	TypeName     string
	UsageText    string
	ValueTextArr []string
	ConstNameArr []string
}

type LeafObj struct {
	PathText       string
	KeyText        string
	UsageText      string
	TypeObj        TypeObj
	DefaultNodeObj *yaml.Node
	EnumObj        *EnumObj
	MinRangeObj    *RangeObj
	MaxRangeObj    *RangeObj
}

// EntryObj хранит ребёнка ветки с сохранением порядка из YAML.
// Различаются по ненулевому BranchObj или LeafObj.
type EntryObj struct {
	BranchObj *BranchObj
	LeafObj   *LeafObj
}

type BranchObj struct {
	PathText               string
	KeyText                string
	UsageText              string
	GenInterfaceFlag       bool
	InheritedInterfaceFlag bool
	EntryObjArr            []*EntryObj
	BranchObjArr           []*BranchObj
	LeafObjArr             []*LeafObj
}

type ResultObj struct {
	RootObj *BranchObj
}
