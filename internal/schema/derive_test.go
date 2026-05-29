package schema

import "testing"

// // // // // // // // // //

func TestGoNameObj(t *testing.T) {
	t.Helper()

	caseArr := []struct {
		InputText    string
		ExpectedText string
	}{
		{"server", "Server"},
		{"body_max", "BodyMax"},
		{"foo-bar", "FooBar"},
		// Цифра начинает новый сегмент имени.
		{"a1b", "A1B"},
		{"abc", "Abc"},
		{"abcD", "Abcd"},
		{"userJSON", "Userjson"},
		{"", "Value"},
		{"---", "Value"},
	}

	for _, caseObj := range caseArr {
		actualText := GoName(caseObj.InputText)
		if actualText != caseObj.ExpectedText {
			t.Fatalf("GoName(%q) = %q, want %q", caseObj.InputText, actualText, caseObj.ExpectedText)
		}
	}
}

func TestGoTypeNameObj(t *testing.T) {
	t.Helper()

	if got := GoTypeName("server.host"); got != "ServerHost" {
		t.Fatalf("GoTypeName(server.host) = %q", got)
	}

	if got := GoTypeName("logging.outputs.deep"); got != "LoggingOutputsDeep" {
		t.Fatalf("GoTypeName(logging.outputs.deep) = %q", got)
	}
}

func TestGoNameCollisionCaseObj(t *testing.T) {
	t.Helper()

	if GoName("a1b") != GoName("A1B") {
		t.Fatalf("expected collision normalization for a1b and A1B")
	}
}

func TestGoExportedNameStrictObj(t *testing.T) {
	t.Helper()

	caseArr := []struct {
		InputText    string
		ExpectedText string
	}{
		{"global-log", "GlobalLog"},
		{"global log", "GlobalLog"},
		{"global::log", "GlobalLog"},
		{"global2-log", "Global2Log"},
	}

	for _, caseObj := range caseArr {
		actualText, err := goExportedNameStrict(caseObj.InputText)
		if err != nil {
			t.Fatalf("goExportedNameStrict(%q) unexpected error: %v", caseObj.InputText, err)
		}
		if actualText != caseObj.ExpectedText {
			t.Fatalf("goExportedNameStrict(%q) = %q, want %q", caseObj.InputText, actualText, caseObj.ExpectedText)
		}
	}
}

func TestGoExportedNameStrictRejectsDigitStartObj(t *testing.T) {
	t.Helper()

	if _, err := goExportedNameStrict("  --- 123-log"); err == nil {
		t.Fatalf("expected leading digit error")
	}
}
