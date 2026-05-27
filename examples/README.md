# goconfgen examples

All generated examples in this directory use the same source files:

- `source/config.yml`
- `source/minimal.yml`
- `source/medium.yml`

The generated packages are intentionally checked in so output differences between generation modes are easy to inspect.

## Full complex example

Generates YAML, JSON, HJSON, CLI flags, validation, render methods, interfaces, and embedded full/minimal/medium presets.
This is the canonical "everything enabled" example; variants below only show reduced generation modes.

```bash
go run ./../cmd/goconfgen \
  -source ./source \
  -out ./complex/target \
  -pkg complexcfg \
  -force
```

## Variant examples

### YAML only

Generates only the YAML parser object. CLI, validation, render methods, and presets are disabled.

```bash
go run ./../cmd/goconfgen \
  -source ./source \
  -out ./variants/yaml_only/target \
  -pkg yamlonlycfg \
  -formats yaml \
  -with-cli=false \
  -with-render=false \
  -with-validate=false \
  -with-presets=false \
  -force
```

Expected shape: no `flags_gen.go`, no `validate_gen.go`, no `presets_gen.go`, no JSON/HJSON parser files.

### JSON + CLI

Generates a JSON file parser plus CLI flags. Presets are disabled.

```bash
go run ./../cmd/goconfgen \
  -source ./source \
  -out ./variants/json_cli/target \
  -pkg jsonclicfg \
  -formats json \
  -with-cli=true \
  -with-presets=false \
  -force
```

Expected shape: JSON parsing uses the standard `encoding/json` package. There are no YAML or HJSON parser files and no
`gopkg.in/yaml.v3` or `github.com/hjson/hjson-go/v4` dependency in this generated package.

### CLI only

Generates only CLI flag parsing and the parser registry entry for `cli`.

```bash
go run ./../cmd/goconfgen \
  -source ./source \
  -out ./variants/cli_only/target \
  -pkg clicfg \
  -formats cli \
  -force
```

Expected shape: no file parser objects, no render methods, no presets, no YAML/JSON/HJSON file parsing API.

### HJSON without presets

Generates only HJSON file parsing/rendering with validation. CLI and presets are disabled.

```bash
go run ./../cmd/goconfgen \
  -source ./source \
  -out ./variants/hjson_no_presets/target \
  -pkg hjsoncfg \
  -formats hjson \
  -with-cli=false \
  -with-presets=false \
  -force
```

Expected shape: HJSON output uses real HJSON syntax with bare object keys where safe. No preset byte blobs are emitted.
