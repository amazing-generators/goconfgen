package goconfgen

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// // // // // // // // // //

func TestRunComplexSourceDirObj(t *testing.T) {
	t.Helper()

	repoRootPath, err := os.Getwd()
	if err != nil {
		t.Fatalf("read working directory: %v", err)
	}

	sourcePath := writeComplexSourceDirObj(t)
	outputPath := filepath.Join(t.TempDir(), "generated")

	resultObj, err := Run(ConfigObj{
		SourceDir:   sourcePath,
		OutputDir:   outputPath,
		PackageName: "complexcfg",
		Formats:     []string{"yaml", "json", "hjson"},
		Force:       true,
	})
	if err != nil {
		t.Fatalf("run generator: %v", err)
	}

	if resultObj.OutputDir != outputPath {
		t.Fatalf("unexpected output dir: %s", resultObj.OutputDir)
	}

	assertDirLayoutObj(t, []string{
		"accessors_gen.go",
		"cli.go",
		"entrypoint.go",
		"enums_gen.go",
		"formats.go",
		"helpers_gen.go",
		"parse_hjson.go",
		"parse_json.go",
		"parse_yaml.go",
		"presets.go",
		"render_hjson.go",
		"render_json.go",
		"render_yaml.go",
		"types_gen.go",
		"validate.go",
	}, outputPath)
	assertGeneratedHeaderSchemaObj(t, outputPath, filepath.ToSlash(filepath.Join(sourcePath, "config.yml")))
	assertGeneratedPackageObj(t, repoRootPath, outputPath)
}

// //

func TestWriteFileAtomicallyForceObj(t *testing.T) {
	t.Helper()

	rootPath := t.TempDir()
	targetPath := filepath.Join(rootPath, "nested", "value.txt")
	dirPath := filepath.Dir(targetPath)

	if err := prepareOutputDir(dirPath, false); err == nil || strings.Contains(err.Error(), "output directory does not exist") == false {
		t.Fatalf("expected missing directory error without force, got: %v", err)
	}

	if err := prepareOutputDir(dirPath, true); err != nil {
		t.Fatalf("prepare output dir with force: %v", err)
	}

	if err := writeFileAtomically(targetPath, []byte("one"), false); err != nil {
		t.Fatalf("write file: %v", err)
	}

	dataArr, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("read written file: %v", err)
	}

	if string(dataArr) != "one" {
		t.Fatalf("unexpected written text: %q", string(dataArr))
	}

	if err = writeFileAtomically(targetPath, []byte("two"), false); err == nil || strings.Contains(err.Error(), "output file already exists") == false {
		t.Fatalf("expected existing file error without force, got: %v", err)
	}

	if err = prepareOutputDir(dirPath, true); err != nil {
		t.Fatalf("prepare output dir for overwrite: %v", err)
	}

	if err = writeFileAtomically(targetPath, []byte("two"), true); err != nil {
		t.Fatalf("overwrite file: %v", err)
	}

	dataArr, err = os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("read overwritten file: %v", err)
	}

	if string(dataArr) != "two" {
		t.Fatalf("unexpected overwritten text: %q", string(dataArr))
	}
}

func TestRunSimpleSchemaWithoutEnumsObj(t *testing.T) {
	t.Helper()

	repoRootPath, err := os.Getwd()
	if err != nil {
		t.Fatalf("read working directory: %v", err)
	}

	sourcePath := filepath.Join(t.TempDir(), "source")
	if err = os.MkdirAll(sourcePath, 0o755); err != nil {
		t.Fatalf("create source dir: %v", err)
	}

	schemaText := "name:\n  type: string\n  usage: Service name.\n  value: demo\n"
	if err = os.WriteFile(filepath.Join(sourcePath, "config.yml"), []byte(schemaText), 0o644); err != nil {
		t.Fatalf("write schema: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "generated")
	if _, err = Run(ConfigObj{
		SourceDir:   sourcePath,
		OutputDir:   outputPath,
		PackageName: "simplecfg",
		Formats:     []string{"yaml", "json", "hjson"},
		Force:       true,
	}); err != nil {
		t.Fatalf("run generator: %v", err)
	}

	smokeText := "package simplecfg\n\nimport \"testing\"\n\nfunc TestGeneratedSimpleSmokeObj(t *testing.T) {\n\tobj := New()\n\tif obj.Name != \"demo\" {\n\t\tt.Fatalf(\"unexpected default: %q\", obj.Name)\n\t}\n\tparsedObj, err := parseYAMLBytes([]byte(\"name: api\\n\"), true)\n\tif err != nil {\n\t\tt.Fatalf(\"parse yaml: %v\", err)\n\t}\n\tif parsedObj.Name != \"api\" {\n\t\tt.Fatalf(\"unexpected parsed value: %q\", parsedObj.Name)\n\t}\n}\n"
	assertGeneratedPackageSmokeObj(t, repoRootPath, outputPath, smokeText)
}

func TestGeneratedUnknownKeysAreRejectedObj(t *testing.T) {
	t.Helper()

	repoRootPath, err := os.Getwd()
	if err != nil {
		t.Fatalf("read working directory: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "generated")
	if _, err = Run(ConfigObj{
		Schema:      writeSchemaObj(t, "server:\n  port:\n    type: int\n    value: 8080\nextensions:\n  raw:\n    type: \"map[string]any\"\n    value: {}\n"),
		OutputDir:   outputPath,
		PackageName: "strictcfg",
		Formats:     []string{"yaml", "json", "hjson"},
		Force:       true,
	}); err != nil {
		t.Fatalf("run generator: %v", err)
	}

	smokeText := `package strictcfg

import (
	"strings"
	"testing"
)

func TestGeneratedUnknownKeysRejectedObj(t *testing.T) {
	testCaseArr := []struct {
		NameText   string
		ParserText string
		DataText   string
		WantText   string
	}{
		{NameText: "yaml root", ParserText: "yaml", DataText: "unknown_root: true\n", WantText: "unknown_root"},
		{NameText: "yaml nested", ParserText: "yaml", DataText: "server:\n  unknown: keep\n", WantText: "server.unknown"},
		{NameText: "json root", ParserText: "json", DataText: "{\"unknown_root\":true}", WantText: "unknown_root"},
		{NameText: "json nested", ParserText: "json", DataText: "{\"server\":{\"unknown\":true}}", WantText: "server.unknown"},
		{NameText: "hjson root", ParserText: "hjson", DataText: "{unknown_root:true}", WantText: "unknown_root"},
		{NameText: "hjson nested", ParserText: "hjson", DataText: "{server:{unknown:true}}", WantText: "server.unknown"},
	}

	for _, testCaseObj := range testCaseArr {
		t.Run(testCaseObj.NameText, func(t *testing.T) {
			_, err := parseByNameObj(testCaseObj.ParserText, []byte(testCaseObj.DataText))
			if err == nil || !strings.Contains(err.Error(), "unknown config key: "+testCaseObj.WantText) {
				t.Fatalf("expected unknown key error [%s], got: %v", testCaseObj.WantText, err)
			}
		})
	}
}

func TestGeneratedMapAnyKeysStayDynamicObj(t *testing.T) {
	obj, err := parseYAMLBytes([]byte("server:\n  port: 9090\nextensions:\n  raw:\n    plugin:\n      nested: true\n      count: 2\n"), true)
	if err != nil {
		t.Fatalf("map[string]any dynamic keys must not be rejected: %v", err)
	}
	if obj.Server.Port != 9090 {
		t.Fatalf("known key did not apply: %d", obj.Server.Port)
	}
	if obj.Extensions.Raw == nil || obj.Extensions.Raw["plugin"] == nil {
		t.Fatalf("map[string]any payload was not applied: %#v", obj.Extensions.Raw)
	}
}

func TestGeneratedCLIUnknownFlagRejectedObj(t *testing.T) {
	obj := New()
	if err := ApplyCLI(&obj, []string{"-server.unknown=1"}); err == nil {
		t.Fatalf("expected unknown cli flag error")
	}
}

func parseByNameObj(nameText string, dataArr []byte) (*ConfigObj, error) {
	switch nameText {
	case "yaml":
		return parseYAMLBytes(dataArr, true)
	case "json":
		return parseJSONBytes(dataArr, true)
	case "hjson":
		return parseHJSONBytes(dataArr, true)
	default:
		return nil, nil
	}
}
`
	assertGeneratedPackageSmokeObj(t, repoRootPath, outputPath, smokeText)
}

func TestRejectSchemaKeyCollisionsObj(t *testing.T) {
	t.Helper()

	outputPath := filepath.Join(t.TempDir(), "generated")

	_, err := Run(ConfigObj{
		Schema:      writeSchemaObj(t, "server:\n  port:\n    type: int\n    value: 80\n\"server.port\":\n  type: int\n  value: 81\n"),
		OutputDir:   outputPath,
		PackageName: "badcfg",
		Formats:     []string{"yaml", "json", "hjson"},
		Force:       true,
	})
	if err == nil || strings.Contains(err.Error(), "must not contain '.'") == false {
		t.Fatalf("expected dotted-key validation error, got: %v", err)
	}

	outputPath = filepath.Join(t.TempDir(), "generated")
	_, err = Run(ConfigObj{
		Schema:      writeSchemaObj(t, "foo-bar:\n  type: string\n  value: a\nfoo_bar:\n  type: string\n  value: b\n"),
		OutputDir:   outputPath,
		PackageName: "badcfg",
		Formats:     []string{"yaml", "json", "hjson"},
		Force:       true,
	})
	if err == nil || strings.Contains(err.Error(), "collision") == false {
		t.Fatalf("expected normalized-name collision error, got: %v", err)
	}
}

func TestRejectInvalidSchemaKeysObj(t *testing.T) {
	t.Helper()

	testCaseArr := []struct {
		NameText            string
		SchemaText          string
		ExpectedMessageText string
	}{
		{
			NameText:            "space",
			SchemaText:          "\"foo bar\":\n  type: string\n  value: demo\n",
			ExpectedMessageText: "must contain only ASCII letters, digits, '_' or '-'",
		},
		{
			NameText:            "leading digit",
			SchemaText:          "9server:\n  type: string\n  value: demo\n",
			ExpectedMessageText: "must start with ASCII letter",
		},
		{
			NameText:            "unsupported punctuation",
			SchemaText:          "\"foo:bar\":\n  type: string\n  value: demo\n",
			ExpectedMessageText: "must contain only ASCII letters, digits, '_' or '-'",
		},
	}

	for _, testCaseObj := range testCaseArr {
		t.Run(testCaseObj.NameText, func(t *testing.T) {
			t.Helper()

			_, err := Run(ConfigObj{
				Schema:      writeSchemaObj(t, testCaseObj.SchemaText),
				OutputDir:   filepath.Join(t.TempDir(), "generated"),
				PackageName: "badcfg",
				Formats:     []string{"yaml", "json", "hjson"},
				Force:       true,
			})
			if err == nil || strings.Contains(err.Error(), testCaseObj.ExpectedMessageText) == false {
				t.Fatalf("expected schema key validation error [%s], got: %v", testCaseObj.ExpectedMessageText, err)
			}
		})
	}
}

func TestEnumNameOverrideObj(t *testing.T) {
	t.Helper()

	repoRootPath, err := os.Getwd()
	if err != nil {
		t.Fatalf("read working directory: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "generated")
	if _, err = Run(ConfigObj{
		Schema:      writeSchemaObj(t, "level:\n  enum_name: global-log\n  enum: [ warn, error ]\n  value: warn\n"),
		OutputDir:   outputPath,
		PackageName: "enumcfg",
		Formats:     []string{"yaml", "json", "hjson"},
		Force:       true,
	}); err != nil {
		t.Fatalf("run generator: %v", err)
	}

	smokeText := "package enumcfg\n\nimport \"testing\"\n\nfunc TestGeneratedEnumNameOverrideObj(t *testing.T) {\n\tobj := New()\n\tif obj.Level != GlobalLogWarn {\n\t\tt.Fatalf(\"unexpected default enum: %v\", obj.Level)\n\t}\n\tparsedObj, err := parseYAMLBytes([]byte(\"level: error\\n\"), true)\n\tif err != nil {\n\t\tt.Fatalf(\"parse yaml: %v\", err)\n\t}\n\tif parsedObj.Level != GlobalLogError {\n\t\tt.Fatalf(\"unexpected parsed enum: %v\", parsedObj.Level)\n\t}\n\tif _, err = ParseGlobalLogEnum(\"warn\"); err != nil {\n\t\tt.Fatalf(\"parse enum helper: %v\", err)\n\t}\n}\n"
	assertGeneratedPackageSmokeObj(t, repoRootPath, outputPath, smokeText)
}

func TestRejectInvalidEnumNameObj(t *testing.T) {
	t.Helper()

	_, err := Run(ConfigObj{
		Schema:      writeSchemaObj(t, "level:\n  enum_name: \"--- 123-log\"\n  enum: [ warn ]\n  value: warn\n"),
		OutputDir:   filepath.Join(t.TempDir(), "generated"),
		PackageName: "badcfg",
		Formats:     []string{"yaml", "json", "hjson"},
		Force:       true,
	})
	if err == nil || strings.Contains(err.Error(), "name must start with ASCII letter") == false {
		t.Fatalf("expected enum_name leading digit error, got: %v", err)
	}
}

func TestRejectEnumValueNamedEnumObj(t *testing.T) {
	t.Helper()

	_, err := Run(ConfigObj{
		Schema:      writeSchemaObj(t, "level:\n  enum_name: global-log\n  enum: [ enum ]\n  value: enum\n"),
		OutputDir:   filepath.Join(t.TempDir(), "generated"),
		PackageName: "badcfg",
		Formats:     []string{"yaml", "json", "hjson"},
		Force:       true,
	})
	if err == nil || strings.Contains(err.Error(), "enum value [enum] is reserved") == false {
		t.Fatalf("expected reserved enum value error, got: %v", err)
	}
}

func TestRejectFloat32DefaultOverflowObj(t *testing.T) {
	t.Helper()

	_, err := Run(ConfigObj{
		Schema:      writeSchemaObj(t, "value:\n  type: float32\n  value: 1e100\n"),
		OutputDir:   filepath.Join(t.TempDir(), "generated"),
		PackageName: "floatcfg",
		Formats:     []string{"yaml", "json", "hjson"},
		Force:       true,
	})
	if err == nil || strings.Contains(err.Error(), "does not fit") == false {
		t.Fatalf("expected float32 overflow error, got: %v", err)
	}
}

func TestGeneratedParseSizeRejectsBadInputObj(t *testing.T) {
	t.Helper()

	repoRootPath, err := os.Getwd()
	if err != nil {
		t.Fatalf("read working directory: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "generated")
	if _, err = Run(ConfigObj{
		Schema:      writeSchemaObj(t, "limit:\n  type: size\n  value: 0\n"),
		OutputDir:   outputPath,
		PackageName: "sizecfg",
		Formats:     []string{"yaml", "json", "hjson"},
		Force:       true,
	}); err != nil {
		t.Fatalf("run generator: %v", err)
	}

	smokeText := "package sizecfg\n\nimport \"testing\"\n\nfunc TestGeneratedSizeRejectsBadInputObj(t *testing.T) {\n\tif _, err := ParseSize(\"-1.5kb\"); err == nil {\n\t\tt.Fatalf(\"expected negative decimal size error\")\n\t}\n\tif _, err := ParseSize(\"18446744073709551615tb\"); err == nil {\n\t\tt.Fatalf(\"expected overflow size error\")\n\t}\n}\n"
	assertGeneratedPackageSmokeObj(t, repoRootPath, outputPath, smokeText)
}

func TestGeneratedBoolFlagRejectsGarbageObj(t *testing.T) {
	t.Helper()

	repoRootPath, err := os.Getwd()
	if err != nil {
		t.Fatalf("read working directory: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "generated")
	if _, err = Run(ConfigObj{
		Schema:      writeSchemaObj(t, "enabled:\n  type: bool\n  value: false\n"),
		OutputDir:   outputPath,
		PackageName: "boolcfg",
		Formats:     []string{"yaml", "json", "hjson"},
		Force:       true,
	}); err != nil {
		t.Fatalf("run generator: %v", err)
	}

	smokeText := "package boolcfg\n\nimport \"testing\"\n\nfunc TestGeneratedBoolFlagRejectsGarbageObj(t *testing.T) {\n\tobj := New()\n\tif err := ApplyCLI(&obj, []string{\"-enabled=maybe\"}); err == nil {\n\t\tt.Fatalf(\"expected invalid bool error\")\n\t}\n}\n"
	assertGeneratedPackageSmokeObj(t, repoRootPath, outputPath, smokeText)
}

func TestGeneratedShorthandMapKeySymmetryObj(t *testing.T) {
	t.Helper()

	repoRootPath, err := os.Getwd()
	if err != nil {
		t.Fatalf("read working directory: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "generated")
	if _, err = Run(ConfigObj{
		Schema:      writeSchemaObj(t, "labels:\n  type: \"map[string]string\"\n  value:\n    env: dev\n"),
		OutputDir:   outputPath,
		PackageName: "mapcfg",
		Formats:     []string{"yaml", "json", "hjson"},
		Force:       true,
	}); err != nil {
		t.Fatalf("run generator: %v", err)
	}

	// Ключи с '-', '.' и длиной больше прежнего лимита 16 должны приниматься из CLI-shorthand,
	// как они принимаются из файла конфигурации.
	smokeText := "package mapcfg\n\nimport \"testing\"\n\nfunc TestGeneratedShorthandMapKeyObj(t *testing.T) {\n\tobj := New()\n\tif err := ApplyCLI(&obj, []string{\"-labels=feature-flag=on,very.long.key.name.exceeding.sixteen=yes\"}); err != nil {\n\t\tt.Fatalf(\"apply cli shorthand: %v\", err)\n\t}\n\tif obj.Labels[\"feature-flag\"] != \"on\" || obj.Labels[\"very.long.key.name.exceeding.sixteen\"] != \"yes\" {\n\t\tt.Fatalf(\"unexpected map: %#v\", obj.Labels)\n\t}\n}\n"
	assertGeneratedPackageSmokeObj(t, repoRootPath, outputPath, smokeText)
}

func TestGeneratedParseAutoValidatesObj(t *testing.T) {
	t.Helper()

	repoRootPath, err := os.Getwd()
	if err != nil {
		t.Fatalf("read working directory: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "generated")
	if _, err = Run(ConfigObj{
		Schema:      writeSchemaObj(t, "server:\n  port:\n    type: int\n    value: 8080\n    min: 1\n    max: 65535\n"),
		OutputDir:   outputPath,
		PackageName: "validcfg",
		Formats:     []string{"yaml", "json", "hjson"},
		Force:       true,
	}); err != nil {
		t.Fatalf("run generator: %v", err)
	}

	smokeText := `package validcfg

import (
	"strings"
	"testing"
)

func TestGeneratedParseAutoValidatesObj(t *testing.T) {
	testCaseArr := []struct {
		NameText   string
		ParserText string
		DataText   string
	}{
		{NameText: "yaml", ParserText: "yaml", DataText: "server:\n  port: 70000\n"},
		{NameText: "json", ParserText: "json", DataText: "{\"server\":{\"port\":70000}}"},
		{NameText: "hjson", ParserText: "hjson", DataText: "{server:{port:70000}}"},
	}

	for _, testCaseObj := range testCaseArr {
		t.Run(testCaseObj.NameText, func(t *testing.T) {
			_, err := parseByNameObj(testCaseObj.ParserText, []byte(testCaseObj.DataText))
			if err == nil || !strings.Contains(err.Error(), "field [server.port] must be <= 65535") {
				t.Fatalf("expected validation error, got: %v", err)
			}
		})
	}

	obj := New()
	err := ApplyCLI(&obj, []string{"-server.port=70000"})
	if err == nil || !strings.Contains(err.Error(), "field [server.port] must be <= 65535") {
		t.Fatalf("expected cli validation error, got: %v", err)
	}
}

func parseByNameObj(nameText string, dataArr []byte) (*ConfigObj, error) {
	switch nameText {
	case "yaml":
		return parseYAMLBytes(dataArr, true)
	case "json":
		return parseJSONBytes(dataArr, true)
	case "hjson":
		return parseHJSONBytes(dataArr, true)
	default:
		return nil, nil
	}
}
`
	assertGeneratedPackageSmokeObj(t, repoRootPath, outputPath, smokeText)
}

func TestGeneratedNumericFlagsRejectOverflowObj(t *testing.T) {
	t.Helper()

	repoRootPath, err := os.Getwd()
	if err != nil {
		t.Fatalf("read working directory: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "generated")
	if _, err = Run(ConfigObj{
		Schema:      writeSchemaObj(t, "small:\n  type: int8\n  value: 1\n"),
		OutputDir:   outputPath,
		PackageName: "smallcfg",
		Formats:     []string{"yaml", "json", "hjson"},
		Force:       true,
	}); err != nil {
		t.Fatalf("run generator: %v", err)
	}

	smokeText := "package smallcfg\n\nimport \"testing\"\n\nfunc TestGeneratedFlagOverflowObj(t *testing.T) {\n\tobj := New()\n\tif err := ApplyCLI(&obj, []string{\"-small=200\"}); err == nil {\n\t\tt.Fatalf(\"expected overflow error\")\n\t}\n}\n"
	assertGeneratedPackageSmokeObj(t, repoRootPath, outputPath, smokeText)
}

func TestPreserveMixedSchemaOrderObj(t *testing.T) {
	t.Helper()

	repoRootPath, err := os.Getwd()
	if err != nil {
		t.Fatalf("read working directory: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "generated")
	if _, err = Run(ConfigObj{
		Schema:      writeSchemaObj(t, "alpha:\n  type: string\n  value: A\nbeta:\n  child:\n    type: string\n    value: B\nzeta:\n  type: string\n  value: Z\n"),
		OutputDir:   outputPath,
		PackageName: "ordercfg",
		Formats:     []string{"yaml", "json", "hjson"},
		Force:       true,
	}); err != nil {
		t.Fatalf("run generator: %v", err)
	}

	smokeText := "package ordercfg\n\nimport \"testing\"\n\nfunc TestGeneratedOrderObj(t *testing.T) {\n\tconst expectedText = \"alpha: A\\nbeta:\\n  child: B\\nzeta: Z\\n\"\n\tif actualText := string(FullYAML()); actualText != expectedText {\n\t\tt.Fatalf(\"unexpected full preset order:\\n%s\", actualText)\n\t}\n}\n"
	assertGeneratedPackageSmokeObj(t, repoRootPath, outputPath, smokeText)
}

func TestGeneratedFullPresetConsistencyObj(t *testing.T) {
	t.Helper()

	repoRootPath, err := os.Getwd()
	if err != nil {
		t.Fatalf("read working directory: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "generated")
	if _, err = Run(ConfigObj{
		Schema:      writeSchemaObj(t, "items:\n  type: '[]string'\nlabels:\n  type: 'map[string]string'\n"),
		OutputDir:   outputPath,
		PackageName: "presetcfg",
		Formats:     []string{"yaml", "json", "hjson"},
		Force:       true,
	}); err != nil {
		t.Fatalf("run generator: %v", err)
	}

	smokeText := "package presetcfg\n\nimport (\n\t\"reflect\"\n\t\"testing\"\n)\n\nfunc TestGeneratedFullPresetConsistencyObj(t *testing.T) {\n\tfullObj := FullConfig()\n\tparsedObj, err := parseYAMLBytes(FullYAML(), true)\n\tif err != nil {\n\t\tt.Fatalf(\"parse full yaml: %v\", err)\n\t}\n\tif reflect.DeepEqual(*fullObj, *parsedObj) == false {\n\t\tt.Fatalf(\"full preset mismatch: %#v != %#v\", *fullObj, *parsedObj)\n\t}\n\tif fullObj.Items == nil || fullObj.Labels == nil {\n\t\tt.Fatalf(\"full preset must preserve explicit empty containers: %#v\", *fullObj)\n\t}\n}\n"
	assertGeneratedPackageSmokeObj(t, repoRootPath, outputPath, smokeText)
}

func TestSelectiveFormatsWithoutCLIObj(t *testing.T) {
	t.Helper()

	repoRootPath, err := os.Getwd()
	if err != nil {
		t.Fatalf("read working directory: %v", err)
	}

	withCLIFlag := false
	outputPath := filepath.Join(t.TempDir(), "generated")
	if _, err = Run(ConfigObj{
		Schema:      writeSchemaObj(t, "service:\n  host:\n    type: string\n    value: localhost\n"),
		OutputDir:   outputPath,
		PackageName: "yamlonlycfg",
		Formats:     []string{"yaml"},
		Features:    FeaturesObj{CLI: &withCLIFlag},
		Force:       true,
	}); err != nil {
		t.Fatalf("run generator: %v", err)
	}

	actualRelPathArr := collectRelPathArrObj(t, outputPath)
	for _, relPath := range actualRelPathArr {
		if relPath == "flags_gen.go" {
			t.Fatalf("flags_gen.go must not be generated when cli is disabled")
		}
	}

	assertFilesDoNotContainObj(t, outputPath, []string{
		"github.com/hjson/hjson-go/v4",
		"compress/flate",
		"\"flag\"",
	})

	smokeText := "package yamlonlycfg\n\nimport \"testing\"\n\nfunc TestGeneratedSelectiveFormatsObj(t *testing.T) {\n\tobj, err := parseYAMLBytes([]byte(\"service:\\n  host: api\\n\"), true)\n\tif err != nil {\n\t\tt.Fatalf(\"parse yaml: %v\", err)\n\t}\n\tif obj.Service.Host != \"api\" {\n\t\tt.Fatalf(\"unexpected parsed host: %q\", obj.Service.Host)\n\t}\n\tif FullConfig() == nil {\n\t\tt.Fatalf(\"expected full config\")\n\t}\n\tif string(FullYAML()) == \"\" {\n\t\tt.Fatalf(\"expected yaml preset\")\n\t}\n}\n"
	assertGeneratedPackageSmokeObj(t, repoRootPath, outputPath, smokeText)
}

func TestSelectiveGenerationRemovesStaleFilesObj(t *testing.T) {
	t.Helper()

	repoRootPath, err := os.Getwd()
	if err != nil {
		t.Fatalf("read working directory: %v", err)
	}

	sourcePath := writeComplexSourceDirObj(t)
	outputPath := filepath.Join(t.TempDir(), "generated")
	if _, err = Run(ConfigObj{
		SourceDir:   sourcePath,
		OutputDir:   outputPath,
		PackageName: "smallcfg",
		Formats:     []string{"yaml", "json", "hjson"},
		Force:       true,
	}); err != nil {
		t.Fatalf("run full generator: %v", err)
	}

	falseFlag := false
	if _, err = Run(ConfigObj{
		SourceDir:   sourcePath,
		OutputDir:   outputPath,
		PackageName: "smallcfg",
		Formats:     []string{"yaml"},
		Features: FeaturesObj{
			CLI:      &falseFlag,
			Validate: &falseFlag,
			Render:   &falseFlag,
			Presets:  &falseFlag,
		},
		Force: true,
	}); err != nil {
		t.Fatalf("run selective generator: %v", err)
	}

	for _, relPath := range []string{"cli.go", "parse_json.go", "parse_hjson.go", "render_yaml.go", "render_json.go", "render_hjson.go", "presets.go", "validate.go"} {
		if _, statErr := os.Stat(filepath.Join(outputPath, relPath)); !errors.Is(statErr, os.ErrNotExist) {
			t.Fatalf("expected stale file removed [%s], stat error: %v", relPath, statErr)
		}
	}

	assertGeneratedPackageSmokeObj(t, repoRootPath, outputPath, "package smallcfg\n\nimport \"testing\"\n\nfunc TestSelectivePackageObj(t *testing.T) {\n\tif _, err := parseYAMLBytes([]byte(\"server:\\n  port: 9090\\n\"), true); err != nil {\n\t\tt.Fatalf(\"parse yaml: %v\", err)\n\t}\n}\n")
}

func TestStaleCleanupHeaderScopedObj(t *testing.T) {
	t.Helper()

	sourcePath := writeComplexSourceDirObj(t)
	outputPath := filepath.Join(t.TempDir(), "generated")
	if _, err := Run(ConfigObj{
		SourceDir:   sourcePath,
		OutputDir:   outputPath,
		PackageName: "smallcfg",
		Formats:     []string{"yaml", "json", "hjson"},
		Force:       true,
	}); err != nil {
		t.Fatalf("run full generator: %v", err)
	}

	foreignPath := filepath.Join(outputPath, "handwritten.go")
	if err := os.WriteFile(foreignPath, []byte("package smallcfg\n\nfunc Foreign() {}\n"), 0o644); err != nil {
		t.Fatalf("write foreign file: %v", err)
	}
	legacyPath := filepath.Join(outputPath, "legacy_gen.go")
	if err := os.WriteFile(legacyPath, []byte(cGeneratedHeaderPrefix+"\n\npackage smallcfg\n"), 0o644); err != nil {
		t.Fatalf("write legacy generated file: %v", err)
	}

	falseFlag := false
	if _, err := Run(ConfigObj{
		SourceDir:   sourcePath,
		OutputDir:   outputPath,
		PackageName: "smallcfg",
		Formats:     []string{"yaml"},
		Features: FeaturesObj{
			CLI:      &falseFlag,
			Validate: &falseFlag,
			Render:   &falseFlag,
			Presets:  &falseFlag,
		},
		Force: true,
	}); err != nil {
		t.Fatalf("run selective generator: %v", err)
	}

	if _, err := os.Stat(foreignPath); err != nil {
		t.Fatalf("foreign file without header must be preserved: %v", err)
	}
	if _, err := os.Stat(legacyPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("arbitrary file with generated header must be removed, stat error: %v", err)
	}
}

func TestGeneratedFormatObjectsObj(t *testing.T) {
	t.Helper()

	repoRootPath, err := os.Getwd()
	if err != nil {
		t.Fatalf("read working directory: %v", err)
	}

	sourcePath := writeComplexSourceDirObj(t)
	outputPath := filepath.Join(t.TempDir(), "generated")
	if _, err = Run(ConfigObj{
		SourceDir:   sourcePath,
		OutputDir:   outputPath,
		PackageName: "complexcfg",
		Formats:     []string{"yaml", "json", "hjson"},
		Force:       true,
	}); err != nil {
		t.Fatalf("run generator: %v", err)
	}

	smokeText := "package complexcfg\n\nimport \"testing\"\n\nfunc TestGeneratedFormatObjectsObj(t *testing.T) {\n\tnameArr := FormatNames()\n\tif len(nameArr) != 3 || nameArr[0] != \"yaml\" || nameArr[1] != \"json\" || nameArr[2] != \"hjson\" {\n\t\tt.Fatalf(\"unexpected format names: %#v\", nameArr)\n\t}\n\tobj := New()\n\tfor _, formatObj := range Formats() {\n\t\tif !formatObj.CanRender() {\n\t\t\tt.Fatalf(\"format %q should render\", formatObj.Name())\n\t\t}\n\t\tdataArr, err := formatObj.Render(&obj, false)\n\t\tif err != nil {\n\t\t\tt.Fatalf(\"render %q: %v\", formatObj.Name(), err)\n\t\t}\n\t\tparsedObj, err := formatObj.Parse(dataArr)\n\t\tif err != nil {\n\t\t\tt.Fatalf(\"parse %q: %v\", formatObj.Name(), err)\n\t\t}\n\t\tif parsedObj.Server.Port != obj.Server.Port {\n\t\t\tt.Fatalf(\"roundtrip %q mismatch\", formatObj.Name())\n\t\t}\n\t}\n\tif _, ok := FormatByName(\"json\"); !ok {\n\t\tt.Fatalf(\"expected json format\")\n\t}\n\tif _, ok := FormatByName(\"toml\"); ok {\n\t\tt.Fatalf(\"unexpected toml format\")\n\t}\n}\n"
	assertGeneratedPackageSmokeObj(t, repoRootPath, outputPath, smokeText)
}

func TestGeneratedFormatObjectsRenderDisabledObj(t *testing.T) {
	t.Helper()

	repoRootPath, err := os.Getwd()
	if err != nil {
		t.Fatalf("read working directory: %v", err)
	}

	sourcePath := writeComplexSourceDirObj(t)
	outputPath := filepath.Join(t.TempDir(), "generated")
	falseFlag := false
	if _, err = Run(ConfigObj{
		SourceDir:   sourcePath,
		OutputDir:   outputPath,
		PackageName: "yamlonlycfg",
		Formats:     []string{"yaml"},
		Features: FeaturesObj{
			CLI:      &falseFlag,
			Validate: &falseFlag,
			Render:   &falseFlag,
			Presets:  &falseFlag,
		},
		Force: true,
	}); err != nil {
		t.Fatalf("run generator: %v", err)
	}

	smokeText := "package yamlonlycfg\n\nimport \"testing\"\n\nfunc TestGeneratedFormatRenderDisabledObj(t *testing.T) {\n\tformatObjArr := Formats()\n\tif len(formatObjArr) != 1 || formatObjArr[0].Name() != \"yaml\" {\n\t\tt.Fatalf(\"unexpected formats: %#v\", FormatNames())\n\t}\n\tif formatObjArr[0].CanRender() {\n\t\tt.Fatalf(\"render must be disabled\")\n\t}\n\tobj := New()\n\tif _, err := formatObjArr[0].Render(&obj, false); err == nil {\n\t\tt.Fatalf(\"expected render-unsupported error\")\n\t}\n}\n"
	assertGeneratedPackageSmokeObj(t, repoRootPath, outputPath, smokeText)
}

func assertDirLayoutObj(t *testing.T, expectedRelPathArr []string, actualPath string) {
	t.Helper()

	actualRelPathArr := collectRelPathArrObj(t, actualPath)

	if !equalStringArrObj(expectedRelPathArr, actualRelPathArr) {
		t.Fatalf("generated file list mismatch\nexpected: %v\nactual:   %v", expectedRelPathArr, actualRelPathArr)
	}
}

func assertGeneratedHeaderSchemaObj(t *testing.T, outputPath string, expectedSchemaText string) {
	t.Helper()

	dataArr, err := os.ReadFile(filepath.Join(outputPath, "types_gen.go"))
	if err != nil {
		t.Fatalf("read generated header: %v", err)
	}

	expectedLineText := "Source schema: " + expectedSchemaText
	if !strings.Contains(string(dataArr), expectedLineText) {
		t.Fatalf("generated header must contain [%s], got:\n%s", expectedLineText, string(dataArr[:minIntObj(len(dataArr), 180)]))
	}
}

func assertGeneratedPackageObj(t *testing.T, repoRootPath string, outputPath string) {
	t.Helper()

	smokeText := `package complexcfg

import (
	"bytes"
	"reflect"
	"testing"
)

func TestGeneratedSmokeObj(t *testing.T) {
	obj := New()
	if err := obj.Validate(); err != nil {
		t.Fatalf("validate defaults: %v", err)
	}
	if !HasMinimal() || !HasMedium() {
		t.Fatalf("expected optional presets")
	}
	presetObj := MediumConfig()
	if presetObj == nil || presetObj.Server.Port != 8443 {
		t.Fatalf("unexpected medium preset: %#v", presetObj)
	}
	fullObj := FullConfig()
	fullFromYAMLObj, err := parseYAMLBytes(FullYAML(), true)
	if err != nil {
		t.Fatalf("parse full yaml: %v", err)
	}
	if reflect.DeepEqual(*fullObj, *fullFromYAMLObj) == false {
		t.Fatalf("full config mismatch: %#v != %#v", *fullObj, *fullFromYAMLObj)
	}
	fullPresentArr, err := fullObj.RenderYAML(true)
	if err != nil {
		t.Fatalf("render full preset present: %v", err)
	}
	if bytes.Equal(fullPresentArr, []byte("{}\n")) {
		t.Fatalf("full preset object lost presence information")
	}
	if !bytes.Contains(FullHJSON(), []byte("# HTTP server settings.")) {
		t.Fatalf("expected comments inside HJSON preset")
	}
	fullFromHJSONObj, err := parseHJSONBytes(FullHJSON(), true)
	if err != nil {
		t.Fatalf("parse full hjson: %v", err)
	}
	if reflect.DeepEqual(*fullObj, *fullFromHJSONObj) == false {
		t.Fatalf("full hjson mismatch: %#v != %#v", *fullObj, *fullFromHJSONObj)
	}
	dataArr, err := obj.RenderJSON(false)
	if err != nil {
		t.Fatalf("render json: %v", err)
	}
	parsedObj, err := parseJSONBytes(dataArr, true)
	if err != nil {
		t.Fatalf("parse rendered json: %v", err)
	}
	if parsedObj.Server.Port != obj.Server.Port {
		t.Fatalf("roundtrip port mismatch: %d != %d", parsedObj.Server.Port, obj.Server.Port)
	}
	cliObj := New()
	if err = ApplyCLI(&cliObj, []string{"-server.port=9090", "-features.enabled=extra_a,extra_b"}); err != nil {
		t.Fatalf("apply cli: %v", err)
	}
	if cliObj.Server.Port != 9090 {
		t.Fatalf("cli override did not apply")
	}
	if !reflect.DeepEqual(cliObj.Features.Enabled, []string{"extra_a", "extra_b"}) {
		t.Fatalf("array replace semantics mismatch: %#v", cliObj.Features.Enabled)
	}
	secondObj := New()
	if err = ApplyCLI(&secondObj, []string{}); err != nil {
		t.Fatalf("second apply cli: %v", err)
	}
	if secondObj.Server.Port != 8080 {
		t.Fatalf("flag state leaked between applications: %d", secondObj.Server.Port)
	}
}
`
	assertGeneratedPackageSmokeObj(t, repoRootPath, outputPath, smokeText)
}

func assertGeneratedPackageSmokeObj(t *testing.T, repoRootPath string, outputPath string, smokeText string) {
	t.Helper()

	assertStandaloneGeneratedPackageObj(t, outputPath)

	goModText := "module generatedcfg\n\ngo 1.19\n"
	if err := os.WriteFile(filepath.Join(outputPath, "go.mod"), []byte(goModText), 0o644); err != nil {
		t.Fatalf("write generated go.mod: %v", err)
	}

	if err := os.WriteFile(filepath.Join(outputPath, "smoke_test.go"), []byte(smokeText), 0o644); err != nil {
		t.Fatalf("write generated smoke test: %v", err)
	}

	commandObj := exec.Command("go", "mod", "tidy")
	commandObj.Dir = outputPath
	commandObj.Env = os.Environ()
	if outputArr, err := commandObj.CombinedOutput(); err != nil {
		t.Fatalf("go mod tidy failed: %v\n%s", err, string(outputArr))
	}

	commandObj = exec.Command("go", "test", "./...")
	if canRunRaceObj(t, outputPath) {
		commandObj = exec.Command("go", "test", "-race", "./...")
	}
	commandObj.Dir = outputPath
	commandObj.Env = os.Environ()
	if outputArr, err := commandObj.CombinedOutput(); err != nil {
		t.Fatalf("generated package go test failed: %v\n%s", err, string(outputArr))
	}
}

func assertStandaloneGeneratedPackageObj(t *testing.T, outputPath string) {
	t.Helper()

	rtAliasPatternObj := regexp.MustCompile(`\brt\.`)
	runtimePkgPathText := "github.com/amazing-generators/goconfgen/" + "runtime"

	err := filepath.Walk(outputPath, func(pathText string, infoObj os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if infoObj.IsDir() || strings.HasSuffix(pathText, ".go") == false {
			return nil
		}

		dataArr, readErr := os.ReadFile(pathText)
		if readErr != nil {
			return readErr
		}

		dataText := string(dataArr)
		if strings.Contains(dataText, runtimePkgPathText) {
			return fmt.Errorf("generated file [%s] depends on goconfgen/runtime", pathText)
		}
		if rtAliasPatternObj.MatchString(dataText) {
			return fmt.Errorf("generated file [%s] still references rt alias", pathText)
		}

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func assertFilesDoNotContainObj(t *testing.T, rootPath string, needleTextArr []string) {
	t.Helper()

	err := filepath.Walk(rootPath, func(pathText string, infoObj os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if infoObj.IsDir() || !strings.HasSuffix(pathText, ".go") {
			return nil
		}

		dataArr, readErr := os.ReadFile(pathText)
		if readErr != nil {
			return readErr
		}

		dataText := string(dataArr)
		for _, needleText := range needleTextArr {
			if strings.Contains(dataText, needleText) {
				return fmt.Errorf("generated file [%s] unexpectedly contains [%s]", pathText, needleText)
			}
		}

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func canRunRaceObj(t *testing.T, workDir string) bool {
	t.Helper()

	commandObj := exec.Command("go", "env", "CGO_ENABLED")
	commandObj.Dir = workDir
	commandObj.Env = os.Environ()

	outputArr, err := commandObj.CombinedOutput()
	if err != nil {
		t.Fatalf("read go env CGO_ENABLED: %v\n%s", err, string(outputArr))
	}

	return strings.TrimSpace(string(outputArr)) == "1"
}

func writeSchemaObj(t *testing.T, schemaText string) string {
	t.Helper()

	sourcePath := filepath.Join(t.TempDir(), "source")
	if err := os.MkdirAll(sourcePath, 0o755); err != nil {
		t.Fatalf("create source dir: %v", err)
	}

	schemaPath := filepath.Join(sourcePath, "config.yml")
	if err := os.WriteFile(schemaPath, []byte(schemaText), 0o644); err != nil {
		t.Fatalf("write schema: %v", err)
	}

	return schemaPath
}

func writeComplexSourceDirObj(t *testing.T) string {
	t.Helper()

	sourcePath := filepath.Join(t.TempDir(), "source")
	if err := os.MkdirAll(sourcePath, 0o755); err != nil {
		t.Fatalf("create source dir: %v", err)
	}

	fileTextMap := map[string]string{
		"config.yml": `server:
  usage:
    - HTTP server settings.
    - Exercises generated comments and interfaces.
  gen_interface: true
  host:
    type: string
    usage: Listen host.
    value: 127.0.0.1
  port:
    type: int
    usage: Listen port.
    value: 8080
    min: 1
    max: 65535
  secure:
    type: bool
    usage: Enable TLS.
    value: false

logging:
  usage: Logging settings.
  level:
    enum_name: global-log
    enum: [ debug, info, warn, error ]
    usage: Minimum log level.
    value: info
  outputs:
    type: "[]enum"
    enum: [ console, json ]
    usage: Ordered logging sinks.
    value: [ console ]

features:
  usage: Feature toggles.
  enabled:
    type: "[]string"
    usage: Enabled feature keys.
    value: [ metrics, tracing ]
  rollout:
    type: "map[string][]string"
    usage: Per-environment rollout list.
    value:
      prod: [ metrics ]
      stage: [ metrics, tracing ]

labels:
  usage: Static metadata labels.
  common:
    type: "map[string]string"
    usage: Common labels.
    value:
      env: dev
      team: platform

extensions:
  usage: Arbitrary extension payload.
  raw:
    type: "map[string]any"
    usage: Extra unstructured payload.
    value:
      nested:
        retries: 3
        mode: safe
`,
		"minimal.yml": `server:
  host: 0.0.0.0
  port: 8080

logging:
  level: warn
`,
		"medium.yml": `server:
  host: 0.0.0.0
  port: 8443
  secure: true

logging:
  level: info
  outputs: [ console, json ]

labels:
  common:
    env: stage
    team: platform
`,
	}

	for nameText, fileText := range fileTextMap {
		if err := os.WriteFile(filepath.Join(sourcePath, nameText), []byte(fileText), 0o644); err != nil {
			t.Fatalf("write source file [%s]: %v", nameText, err)
		}
	}

	return sourcePath
}

func collectRelPathArrObj(t *testing.T, rootPath string) []string {
	t.Helper()

	resultArr := make([]string, 0, 16)

	err := filepath.Walk(rootPath, func(pathText string, infoObj os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if infoObj.IsDir() {
			return nil
		}

		relPath, relErr := filepath.Rel(rootPath, pathText)
		if relErr != nil {
			return relErr
		}

		if relPath == "go.mod" || relPath == "go.sum" || relPath == "smoke_test.go" {
			return nil
		}

		resultArr = append(resultArr, relPath)
		return nil
	})
	if err != nil {
		t.Fatalf("walk directory [%s]: %v", rootPath, err)
	}

	sort.Strings(resultArr)
	return resultArr
}

func equalStringArrObj(leftArr []string, rightArr []string) bool {
	if len(leftArr) != len(rightArr) {
		return false
	}

	for itemIndex := range leftArr {
		if leftArr[itemIndex] != rightArr[itemIndex] {
			return false
		}
	}

	return true
}

func minIntObj(leftVal int, rightVal int) int {
	if leftVal < rightVal {
		return leftVal
	}

	return rightVal
}
