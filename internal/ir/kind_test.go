package ir

import (
	"testing"

	"github.com/amazing-generators/goconfgen/internal/schema"
)

// // // // // // // // // //

func TestClassifyKindObj(t *testing.T) {
	t.Helper()

	testCaseArr := []struct {
		TypeText   string
		EnumArr    []string
		KindObj    FieldKindObj
		BitSizeVal int
	}{
		{"string", nil, FieldKindString, 0},
		{"bool", nil, FieldKindBool, 0},
		{"int", nil, FieldKindIntSigned, 0},
		{"int8", nil, FieldKindIntSigned, 8},
		{"int16", nil, FieldKindIntSigned, 16},
		{"int32", nil, FieldKindIntSigned, 32},
		{"int64", nil, FieldKindIntSigned, 64},
		{"uint", nil, FieldKindIntUnsigned, 0},
		{"uint8", nil, FieldKindIntUnsigned, 8},
		{"uint16", nil, FieldKindIntUnsigned, 16},
		{"uint32", nil, FieldKindIntUnsigned, 32},
		{"uint64", nil, FieldKindIntUnsigned, 64},
		{"float", nil, FieldKindFloat, 32},
		{"float32", nil, FieldKindFloat, 32},
		{"float64", nil, FieldKindFloat, 64},
		{"duration", nil, FieldKindDuration, 64},
		{"time.Duration", nil, FieldKindDuration, 64},
		{"size", nil, FieldKindSize, 64},
		{"enum", []string{"a"}, FieldKindEnumScalar, 0},
		{"[]string", nil, FieldKindArrayString, 0},
		{"[]bool", nil, FieldKindArrayBool, 0},
		{"[]int", nil, FieldKindArrayIntSigned, 0},
		{"[]int16", nil, FieldKindArrayIntSigned, 16},
		{"[]uint", nil, FieldKindArrayIntUnsigned, 0},
		{"[]uint32", nil, FieldKindArrayIntUnsigned, 32},
		{"[]float32", nil, FieldKindArrayFloat, 32},
		{"[]float64", nil, FieldKindArrayFloat, 64},
		{"[]duration", nil, FieldKindArrayDuration, 64},
		{"[]size", nil, FieldKindArraySize, 64},
		{"[]enum", []string{"a"}, FieldKindArrayEnum, 0},
		{"map[string]string", nil, FieldKindMapStringString, 0},
		{"map[string][]string", nil, FieldKindMapStringStringArr, 0},
		{"map[string]any", nil, FieldKindMapStringAny, 0},
	}

	for _, testCaseObj := range testCaseArr {
		t.Run(testCaseObj.TypeText, func(t *testing.T) {
			t.Helper()

			typeObj, err := schema.ParseType(testCaseObj.TypeText, testCaseObj.EnumArr)
			if err != nil {
				t.Fatalf("parse type: %v", err)
			}

			kindObj, bitSizeVal := ClassifyKind(typeObj)
			if kindObj != testCaseObj.KindObj || bitSizeVal != testCaseObj.BitSizeVal {
				t.Fatalf("unexpected classification: (%d, %d)", kindObj, bitSizeVal)
			}
		})
	}
}
