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
	formatText := ""
	withCLIFlag := true
	withValidateFlag := true
	withRenderFlag := true
	withPresetsFlag := true
	withInterfacesFlag := true

	flagSet.StringVar(&config.SourceDir, "source", "", "source directory that contains config.yml and optional presets")
	flagSet.StringVar(&config.SchemaPath, "schema", "", "explicit schema file path")
	flagSet.StringVar(&config.MinimalPath, "minimal", "", "explicit minimal preset file path")
	flagSet.StringVar(&config.MediumPath, "medium", "", "explicit medium preset file path")
	flagSet.StringVar(&config.OutputDir, "out", "", "output directory for generated files")
	flagSet.StringVar(&config.PackageName, "pkg", "", "package name for generated runtime code")
	flagSet.StringVar(&formatText, "formats", "", "comma-separated subset of yaml,json,hjson; empty keeps all formats")
	flagSet.BoolVar(&withCLIFlag, "with-cli", true, "generate CLI flag helpers")
	flagSet.BoolVar(&withValidateFlag, "with-validate", true, "generate validation helpers")
	flagSet.BoolVar(&withRenderFlag, "with-render", true, "generate render methods on parser objects")
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
	config.WithCLI = &withCLIFlag
	config.WithValidate = &withValidateFlag
	config.WithRender = &withRenderFlag
	config.WithPresets = &withPresetsFlag
	config.WithInterfaces = &withInterfacesFlag

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
