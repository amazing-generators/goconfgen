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

func TestRunComplexExampleObj(t *testing.T) {
	t.Helper()

	repoRootPath, err := os.Getwd()
	if err != nil {
		t.Fatalf("read working directory: %v", err)
	}

	sourcePath := filepath.Join(repoRootPath, "examples", "source")
	expectedPath := filepath.Join(repoRootPath, "examples", "complex", "target")
	outputPath := filepath.Join(t.TempDir(), "generated")

	resultObj, err := Run(ConfigObj{
		SourceDir:   sourcePath,
		OutputDir:   outputPath,
		PackageName: "complexcfg",
		Force:       true,
	})
	if err != nil {
		t.Fatalf("run generator: %v", err)
	}

	if resultObj.OutputDir != outputPath {
		t.Fatalf("unexpected output dir: %s", resultObj.OutputDir)
	}

	assertDirLayoutObj(t, expectedPath, outputPath)
	assertGeneratedHeaderSchemaObj(t, outputPath, "examples/source/config.yml")
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
		Force:       true,
	}); err != nil {
		t.Fatalf("run generator: %v", err)
	}

	smokeText := "package simplecfg\n\nimport \"testing\"\n\nfunc TestGeneratedSimpleSmokeObj(t *testing.T) {\n\tobj := New()\n\tif obj.Name != \"demo\" {\n\t\tt.Fatalf(\"unexpected default: %q\", obj.Name)\n\t}\n\tparserObj, ok := ParserByName(\"yaml\")\n\tif !ok {\n\t\tt.Fatalf(\"yaml parser missing\")\n\t}\n\tparsedObj, err := parserObj.Parse([]byte(\"name: api\\n\"))\n\tif err != nil {\n\t\tt.Fatalf(\"parse yaml: %v\", err)\n\t}\n\tif parsedObj.Name != \"api\" {\n\t\tt.Fatalf(\"unexpected parsed value: %q\", parsedObj.Name)\n\t}\n}\n"
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
		SchemaPath:  writeSchemaObj(t, "server:\n  port:\n    type: int\n    value: 8080\nextensions:\n  raw:\n    type: \"map[string]any\"\n    value: {}\n"),
		OutputDir:   outputPath,
		PackageName: "strictcfg",
		Force:       true,
	}); err != nil {
		t.Fatalf("run generator: %v", err)
	}

	smokeText := `package strictcfg

import (
	"flag"
	"io"
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
			parserObj, ok := ParserByName(testCaseObj.ParserText)
			if !ok {
				t.Fatalf("parser missing: %s", testCaseObj.ParserText)
			}

			_, err := parserObj.Parse([]byte(testCaseObj.DataText))
			if err == nil || !strings.Contains(err.Error(), "unknown config key: "+testCaseObj.WantText) {
				t.Fatalf("expected unknown key error [%s], got: %v", testCaseObj.WantText, err)
			}
		})
	}
}

func TestGeneratedMapAnyKeysStayDynamicObj(t *testing.T) {
	parserObj, ok := ParserByName("yaml")
	if !ok {
		t.Fatalf("yaml parser missing")
	}

	obj, err := parserObj.Parse([]byte("server:\n  port: 9090\nextensions:\n  raw:\n    plugin:\n      nested: true\n      count: 2\n"))
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
	flagSet := flag.NewFlagSet("generated", flag.ContinueOnError)
	flagSet.SetOutput(io.Discard)
	RegisterFlags(flagSet)
	if err := flagSet.Parse([]string{"-server.unknown=1"}); err == nil {
		t.Fatalf("expected unknown cli flag error")
	}
}
`
	assertGeneratedPackageSmokeObj(t, repoRootPath, outputPath, smokeText)
}

func TestRejectSchemaKeyCollisionsObj(t *testing.T) {
	t.Helper()

	outputPath := filepath.Join(t.TempDir(), "generated")

	_, err := Run(ConfigObj{
		SchemaPath:  writeSchemaObj(t, "server:\n  port:\n    type: int\n    value: 80\n\"server.port\":\n  type: int\n  value: 81\n"),
		OutputDir:   outputPath,
		PackageName: "badcfg",
		Force:       true,
	})
	if err == nil || strings.Contains(err.Error(), "must not contain '.'") == false {
		t.Fatalf("expected dotted-key validation error, got: %v", err)
	}

	outputPath = filepath.Join(t.TempDir(), "generated")
	_, err = Run(ConfigObj{
		SchemaPath:  writeSchemaObj(t, "foo-bar:\n  type: string\n  value: a\nfoo_bar:\n  type: string\n  value: b\n"),
		OutputDir:   outputPath,
		PackageName: "badcfg",
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
				SchemaPath:  writeSchemaObj(t, testCaseObj.SchemaText),
				OutputDir:   filepath.Join(t.TempDir(), "generated"),
				PackageName: "badcfg",
				Force:       true,
			})
			if err == nil || strings.Contains(err.Error(), testCaseObj.ExpectedMessageText) == false {
				t.Fatalf("expected schema key validation error [%s], got: %v", testCaseObj.ExpectedMessageText, err)
			}
		})
	}
}

func TestRejectFloat32DefaultOverflowObj(t *testing.T) {
	t.Helper()

	_, err := Run(ConfigObj{
		SchemaPath:  writeSchemaObj(t, "value:\n  type: float32\n  value: 1e100\n"),
		OutputDir:   filepath.Join(t.TempDir(), "generated"),
		PackageName: "floatcfg",
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
		SchemaPath:  writeSchemaObj(t, "limit:\n  type: size\n  value: 0\n"),
		OutputDir:   outputPath,
		PackageName: "sizecfg",
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
		SchemaPath:  writeSchemaObj(t, "enabled:\n  type: bool\n  value: false\n"),
		OutputDir:   outputPath,
		PackageName: "boolcfg",
		Force:       true,
	}); err != nil {
		t.Fatalf("run generator: %v", err)
	}

	smokeText := "package boolcfg\n\nimport (\n\t\"flag\"\n\t\"testing\"\n)\n\nfunc TestGeneratedBoolFlagRejectsGarbageObj(t *testing.T) {\n\tflagSet := flag.NewFlagSet(\"generated\", flag.ContinueOnError)\n\tflagSet.SetOutput(testWriterObj{t})\n\tRegisterFlags(flagSet)\n\tif err := flagSet.Parse([]string{\"-enabled=maybe\"}); err == nil {\n\t\tt.Fatalf(\"expected invalid bool error\")\n\t}\n}\n\ntype testWriterObj struct{ t *testing.T }\n\nfunc (obj testWriterObj) Write(data []byte) (int, error) { return len(data), nil }\n"
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
		SchemaPath:  writeSchemaObj(t, "server:\n  port:\n    type: int\n    value: 8080\n    min: 1\n    max: 65535\n"),
		OutputDir:   outputPath,
		PackageName: "validcfg",
		Force:       true,
	}); err != nil {
		t.Fatalf("run generator: %v", err)
	}

	smokeText := `package validcfg

import (
	"flag"
	"io"
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
			parserObj, ok := ParserByName(testCaseObj.ParserText)
			if !ok {
				t.Fatalf("parser missing: %s", testCaseObj.ParserText)
			}

			_, err := parserObj.Parse([]byte(testCaseObj.DataText))
			if err == nil || !strings.Contains(err.Error(), "field [server.port] must be <= 65535") {
				t.Fatalf("expected validation error, got: %v", err)
			}
		})
	}

	flagSet := flag.NewFlagSet("generated", flag.ContinueOnError)
	flagSet.SetOutput(io.Discard)
	RegisterFlags(flagSet)
	if err := flagSet.Parse([]string{"-server.port=70000"}); err != nil {
		t.Fatalf("parse flags: %v", err)
	}

	obj := New()
	err := ApplyFlags(&obj)
	if err == nil || !strings.Contains(err.Error(), "field [server.port] must be <= 65535") {
		t.Fatalf("expected cli validation error, got: %v", err)
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
		SchemaPath:  writeSchemaObj(t, "small:\n  type: int8\n  value: 1\n"),
		OutputDir:   outputPath,
		PackageName: "smallcfg",
		Force:       true,
	}); err != nil {
		t.Fatalf("run generator: %v", err)
	}

	smokeText := "package smallcfg\n\nimport (\n\t\"flag\"\n\t\"testing\"\n)\n\nfunc TestGeneratedFlagOverflowObj(t *testing.T) {\n\tflagSet := flag.NewFlagSet(\"generated\", flag.ContinueOnError)\n\tRegisterFlags(flagSet)\n\tif err := flagSet.Parse([]string{\"-small=200\"}); err != nil {\n\t\tt.Fatalf(\"flag parse: %v\", err)\n\t}\n\tobj := New()\n\tif err := ApplyFlags(&obj); err == nil {\n\t\tt.Fatalf(\"expected overflow error\")\n\t}\n}\n"
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
		SchemaPath:  writeSchemaObj(t, "alpha:\n  type: string\n  value: A\nbeta:\n  child:\n    type: string\n    value: B\nzeta:\n  type: string\n  value: Z\n"),
		OutputDir:   outputPath,
		PackageName: "ordercfg",
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
		SchemaPath:  writeSchemaObj(t, "items:\n  type: '[]string'\nlabels:\n  type: 'map[string]string'\n"),
		OutputDir:   outputPath,
		PackageName: "presetcfg",
		Force:       true,
	}); err != nil {
		t.Fatalf("run generator: %v", err)
	}

	smokeText := "package presetcfg\n\nimport (\n\t\"reflect\"\n\t\"testing\"\n)\n\nfunc TestGeneratedFullPresetConsistencyObj(t *testing.T) {\n\tfullObj, err := FullConfig()\n\tif err != nil {\n\t\tt.Fatalf(\"full config: %v\", err)\n\t}\n\tparserObj, ok := ParserByName(\"yaml\")\n\tif !ok {\n\t\tt.Fatalf(\"yaml parser missing\")\n\t}\n\tparsedObj, err := parserObj.Parse(FullYAML())\n\tif err != nil {\n\t\tt.Fatalf(\"parse full yaml: %v\", err)\n\t}\n\tif reflect.DeepEqual(*fullObj, *parsedObj) == false {\n\t\tt.Fatalf(\"full preset mismatch: %#v != %#v\", *fullObj, *parsedObj)\n\t}\n\tif fullObj.Items == nil || fullObj.Labels == nil {\n\t\tt.Fatalf(\"full preset must preserve explicit empty containers: %#v\", *fullObj)\n\t}\n}\n"
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
		SchemaPath:  writeSchemaObj(t, "service:\n  host:\n    type: string\n    value: localhost\n"),
		OutputDir:   outputPath,
		PackageName: "yamlonlycfg",
		Formats:     []string{"yaml"},
		WithCLI:     &withCLIFlag,
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

	smokeText := "package yamlonlycfg\n\nimport \"testing\"\n\nfunc TestGeneratedSelectiveFormatsObj(t *testing.T) {\n\tparserObj, ok := ParserByName(\"yaml\")\n\tif !ok {\n\t\tt.Fatalf(\"yaml parser missing\")\n\t}\n\tobj, err := parserObj.Parse([]byte(\"service:\\n  host: api\\n\"))\n\tif err != nil {\n\t\tt.Fatalf(\"parse yaml: %v\", err)\n\t}\n\tif obj.Service.Host != \"api\" {\n\t\tt.Fatalf(\"unexpected parsed host: %q\", obj.Service.Host)\n\t}\n\tif _, err = FullConfig(); err != nil {\n\t\tt.Fatalf(\"full config: %v\", err)\n\t}\n\tif string(FullYAML()) == \"\" {\n\t\tt.Fatalf(\"expected yaml preset\")\n\t}\n}\n"
	assertGeneratedPackageSmokeObj(t, repoRootPath, outputPath, smokeText)
}

func TestSelectiveGenerationRemovesStaleFilesObj(t *testing.T) {
	t.Helper()

	repoRootPath, err := os.Getwd()
	if err != nil {
		t.Fatalf("read working directory: %v", err)
	}

	sourcePath := filepath.Join(repoRootPath, "examples", "source")
	outputPath := filepath.Join(t.TempDir(), "generated")
	if _, err = Run(ConfigObj{
		SourceDir:   sourcePath,
		OutputDir:   outputPath,
		PackageName: "smallcfg",
		Force:       true,
	}); err != nil {
		t.Fatalf("run full generator: %v", err)
	}

	falseFlag := false
	if _, err = Run(ConfigObj{
		SourceDir:      sourcePath,
		OutputDir:      outputPath,
		PackageName:    "smallcfg",
		Formats:        []string{"yaml"},
		WithCLI:        &falseFlag,
		WithValidate:   &falseFlag,
		WithRender:     &falseFlag,
		WithPresets:    &falseFlag,
		WithInterfaces: nil,
		Force:          true,
	}); err != nil {
		t.Fatalf("run selective generator: %v", err)
	}

	for _, relPath := range []string{"flags_gen.go", "parser_json_gen.go", "parser_hjson_gen.go", "parser_cli_gen.go", "presets_gen.go", "validate_gen.go"} {
		if _, statErr := os.Stat(filepath.Join(outputPath, relPath)); !errors.Is(statErr, os.ErrNotExist) {
			t.Fatalf("expected stale file removed [%s], stat error: %v", relPath, statErr)
		}
	}

	assertGeneratedPackageSmokeObj(t, repoRootPath, outputPath, "package smallcfg\n\nimport \"testing\"\n\nfunc TestSelectivePackageObj(t *testing.T) {\n\tif _, ok := ParserByName(\"yaml\"); !ok {\n\t\tt.Fatalf(\"yaml parser missing\")\n\t}\n\tif _, ok := ParserByName(\"json\"); ok {\n\t\tt.Fatalf(\"json parser must be absent\")\n\t}\n}\n")
}

func assertDirLayoutObj(t *testing.T, expectedPath string, actualPath string) {
	t.Helper()

	expectedRelPathArr := collectRelPathArrObj(t, expectedPath)
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

	smokeText := "package complexcfg\n\nimport (\n\t\"bytes\"\n\t\"flag\"\n\t\"reflect\"\n\t\"testing\"\n)\n\nfunc TestGeneratedSmokeObj(t *testing.T) {\n\tobj := New()\n\tif err := obj.Validate(); err != nil {\n\t\tt.Fatalf(\"validate defaults: %v\", err)\n\t}\n\tif !HasMinimal() || !HasMedium() {\n\t\tt.Fatalf(\"expected optional presets\")\n\t}\n\tpresetObj, err := MediumConfig()\n\tif err != nil {\n\t\tt.Fatalf(\"medium preset parse: %v\", err)\n\t}\n\tif presetObj == nil || presetObj.Server.Port != 8443 {\n\t\tt.Fatalf(\"unexpected medium preset: %#v\", presetObj)\n\t}\n\tfullObj, err := FullConfig()\n\tif err != nil {\n\t\tt.Fatalf(\"full preset parse: %v\", err)\n\t}\n\tyamlParserObj, ok := ParserByName(\"yaml\")\n\tif !ok {\n\t\tt.Fatalf(\"yaml parser missing\")\n\t}\n\tfullFromYAMLObj, err := yamlParserObj.Parse(FullYAML())\n\tif err != nil {\n\t\tt.Fatalf(\"parse full yaml: %v\", err)\n\t}\n\tif reflect.DeepEqual(*fullObj, *fullFromYAMLObj) == false {\n\t\tt.Fatalf(\"full config mismatch: %#v != %#v\", *fullObj, *fullFromYAMLObj)\n\t}\n\tfullPresentArr, err := yamlParserObj.RenderPresent(fullObj)\n\tif err != nil {\n\t\tt.Fatalf(\"render full preset present: %v\", err)\n\t}\n\tif bytes.Equal(fullPresentArr, []byte(\"{}\\n\")) {\n\t\tt.Fatalf(\"full preset object lost presence information\")\n\t}\n\tif !bytes.Contains(FullHJSON(), []byte(\"# HTTP server settings.\")) {\n\t\tt.Fatalf(\"expected comments inside HJSON preset\")\n\t}\n\thjsonParserObj, ok := ParserByName(\"hjson\")\n\tif !ok {\n\t\tt.Fatalf(\"hjson parser missing\")\n\t}\n\tfullFromHJSONObj, err := hjsonParserObj.Parse(FullHJSON())\n\tif err != nil {\n\t\tt.Fatalf(\"parse full hjson: %v\", err)\n\t}\n\tif reflect.DeepEqual(*fullObj, *fullFromHJSONObj) == false {\n\t\tt.Fatalf(\"full hjson mismatch: %#v != %#v\", *fullObj, *fullFromHJSONObj)\n\t}\n\tjsonParserObj, ok := ParserByName(\"json\")\n\tif !ok {\n\t\tt.Fatalf(\"json parser missing\")\n\t}\n\tdataArr, err := jsonParserObj.Render(&obj)\n\tif err != nil {\n\t\tt.Fatalf(\"render json: %v\", err)\n\t}\n\tparsedObj, err := jsonParserObj.Parse(dataArr)\n\tif err != nil {\n\t\tt.Fatalf(\"parse rendered json: %v\", err)\n\t}\n\tif parsedObj.Server.Port != obj.Server.Port {\n\t\tt.Fatalf(\"roundtrip port mismatch: %d != %d\", parsedObj.Server.Port, obj.Server.Port)\n\t}\n\tflagSet := flag.NewFlagSet(\"generated\", flag.ContinueOnError)\n\tRegisterFlags(flagSet)\n\tif err = flagSet.Parse([]string{\"-server.port=9090\", \"-features.enabled=extra_a,extra_b\"}); err != nil {\n\t\tt.Fatalf(\"flag parse: %v\", err)\n\t}\n\tcliObj := New()\n\tif err = ApplyFlags(&cliObj); err != nil {\n\t\tt.Fatalf(\"apply flags: %v\", err)\n\t}\n\tif cliObj.Server.Port != 9090 {\n\t\tt.Fatalf(\"cli override did not apply\")\n\t}\n\tif !reflect.DeepEqual(cliObj.Features.Enabled, []string{\"extra_a\", \"extra_b\"}) {\n\t\tt.Fatalf(\"array replace semantics mismatch: %#v\", cliObj.Features.Enabled)\n\t}\n\tsecondFlagSet := flag.NewFlagSet(\"second\", flag.ContinueOnError)\n\tRegisterFlags(secondFlagSet)\n\tif err = secondFlagSet.Parse([]string{}); err != nil {\n\t\tt.Fatalf(\"second flag parse: %v\", err)\n\t}\n\tsecondObj := New()\n\tif err = ApplyFlags(&secondObj); err != nil {\n\t\tt.Fatalf(\"second apply flags: %v\", err)\n\t}\n\tif secondObj.Server.Port != 8080 {\n\t\tt.Fatalf(\"flag state leaked between registrations: %d\", secondObj.Server.Port)\n\t}\n}\n"
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
