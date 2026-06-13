package schema

import "testing"

// // // // // // // // // //

func TestParseSizeIntegerExactObj(t *testing.T) {
	okCaseArr := []struct {
		TextValue string
		Expected  uint64
	}{
		{"8MiB", 8 * 1024 * 1024},
		{"1KiB", 1024},
		{"1kb", 1000},
		{"1.5kb", 1500},
		{"0", 0},
		{"255b", 255},
		// 2^53+1 is not representable as float64; the integer path must keep it exact.
		{"9007199254740993b", 9007199254740993},
		{"18446744073709551615b", 18446744073709551615},
	}

	for _, caseObj := range okCaseArr {
		gotVal, err := ParseSize(caseObj.TextValue)
		if err != nil {
			t.Fatalf("ParseSize(%q) unexpected error: %v", caseObj.TextValue, err)
		}
		if gotVal != caseObj.Expected {
			t.Fatalf("ParseSize(%q) = %d, want %d", caseObj.TextValue, gotVal, caseObj.Expected)
		}
	}

	errCaseArr := []string{"-1.5kb", "18446744073709551615tb", "", "kb", "abc"}
	for _, textValue := range errCaseArr {
		if _, err := ParseSize(textValue); err == nil {
			t.Fatalf("ParseSize(%q) expected error", textValue)
		}
	}
}
