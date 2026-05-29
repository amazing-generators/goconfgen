package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/amazing-generators/goconfgen"
)

// // // // // // // // // //

func main() {
	if err := runCLI(os.Args[1:]); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func runCLI(argsArr []string) error {
	if len(argsArr) > 0 && (argsArr[0] == "help" || argsArr[0] == "-h" || argsArr[0] == "--help") {
		flagSet := flag.NewFlagSet("goconfgen", flag.ContinueOnError)
		flagSet.SetOutput(os.Stderr)
		printUsage(flagSet)
		return nil
	}

	flagSet := flag.NewFlagSet("goconfgen", flag.ContinueOnError)
	flagSet.SetOutput(os.Stderr)

	config := goconfgen.ConfigObj{}
	formatText := "yaml,json,hjson"
	presetFlagObj := presetFlagObj{}
	withCLIFlag := true
	withValidateFlag := true
	withRenderFlag := true
	withPresetsFlag := true
	withInterfacesFlag := true

	flagSet.StringVar(&config.SourceDir, "source", "", "source directory that contains config.yml and optional presets")
	flagSet.StringVar(&config.Schema, "schema", "", "explicit schema file path")
	flagSet.StringVar(&config.OutputDir, "out", "", "output directory for generated files")
	flagSet.StringVar(&config.PackageName, "pkg", "", "package name for generated runtime code")
	flagSet.StringVar(&formatText, "formats", formatText, "comma-separated subset of yaml,json,hjson")
	flagSet.Var(&presetFlagObj, "preset", "preset in NAME=PATH form; may be repeated")
	flagSet.BoolVar(&withCLIFlag, "with-cli", true, "generate CLI flag helpers")
	flagSet.BoolVar(&withValidateFlag, "with-validate", true, "generate validation helpers")
	flagSet.BoolVar(&withRenderFlag, "with-render", true, "generate render methods")
	flagSet.BoolVar(&withPresetsFlag, "with-presets", true, "generate embedded preset helpers")
	flagSet.BoolVar(&withInterfacesFlag, "with-interfaces", true, "generate schema-requested interfaces")
	flagSet.BoolVar(&config.Force, "force", false, "create missing output directories and overwrite existing generated files")
	flagSet.Usage = func() {
		printUsage(flagSet)
	}

	if err := flagSet.Parse(argsArr); err != nil {
		return err
	}

	config.Formats = parseFormatsArg(formatText)
	config.Presets = presetFlagObj.ValueMap
	config.Features.CLI = &withCLIFlag
	config.Features.Validate = &withValidateFlag
	config.Features.Render = &withRenderFlag
	config.Features.Presets = &withPresetsFlag
	config.Features.Interfaces = &withInterfacesFlag

	resultObj, err := goconfgen.Run(config)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintln(os.Stdout, "Generated runtime directory:", resultObj.OutputDir)
	for _, pathText := range resultObj.GeneratedFilePathArr {
		_, _ = fmt.Fprintln(os.Stdout, "Generated file:", pathText)
	}

	return nil
}

func printUsage(flagSet *flag.FlagSet) {
	_, _ = fmt.Fprintln(os.Stderr, "Usage: goconfgen [flags]")
	_, _ = fmt.Fprintln(os.Stderr, "")
	_, _ = fmt.Fprintln(os.Stderr, "Flags:")
	flagSet.PrintDefaults()
}

func parseFormatsArg(textValue string) []string {
	textValue = strings.TrimSpace(textValue)
	if textValue == "" {
		return nil
	}

	partTextArr := strings.Split(textValue, ",")
	resultArr := make([]string, 0, len(partTextArr))
	for _, partText := range partTextArr {
		partText = strings.TrimSpace(partText)
		if partText == "" {
			continue
		}

		resultArr = append(resultArr, partText)
	}

	return resultArr
}

type presetFlagObj struct {
	ValueMap map[string]string
}

func (obj *presetFlagObj) String() string {
	if obj == nil || len(obj.ValueMap) == 0 {
		return ""
	}
	partTextArr := make([]string, 0, len(obj.ValueMap))
	for nameText, pathText := range obj.ValueMap {
		partTextArr = append(partTextArr, nameText+"="+pathText)
	}
	return strings.Join(partTextArr, ",")
}

func (obj *presetFlagObj) Set(valueText string) error {
	partTextArr := strings.SplitN(valueText, "=", 2)
	if len(partTextArr) != 2 {
		return fmt.Errorf("preset must use NAME=PATH form")
	}
	nameText := strings.TrimSpace(partTextArr[0])
	pathText := strings.TrimSpace(partTextArr[1])
	if nameText == "" || pathText == "" {
		return fmt.Errorf("preset name and path must not be empty")
	}
	if obj.ValueMap == nil {
		obj.ValueMap = map[string]string{}
	}
	if _, existsFlag := obj.ValueMap[nameText]; existsFlag {
		return fmt.Errorf("preset [%s] is specified more than once", nameText)
	}
	obj.ValueMap[nameText] = pathText
	return nil
}
