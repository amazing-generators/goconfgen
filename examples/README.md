# goconfgen examples

All generated examples in this directory use the same source files:

- `source/config.yml`
- `source/minimal.yml`
- `source/medium.yml`

The generated packages are intentionally checked in so output differences between generation modes are easy to inspect.
The source schema also demonstrates explicit enum naming with `enum_name: global-log`, which becomes `GlobalLogEnum`
and constants such as `GlobalLogWarn`.

`source/config.yml` is a full-coverage ("kitchen-sink") schema: it exercises every supported scalar, array, and map
type,
every integer/float width, the `duration` / `time.Duration` and `size` types, inline enums, and generated interfaces.
This makes the checked-in packages a complete golden reference, so regenerating after a generator change surfaces any
behavioral drift via `git diff`.

Regenerate every example at once with [`../_run/regen-examples.sh`](../_run/regen-examples.sh); it also fails if
regeneration produces any drift against the committed tree.

## Full complex example

Generates YAML, JSON, HJSON, CLI flags, validation, render methods, interfaces, and embedded full/minimal/medium presets.
This is the canonical "everything enabled" example; variants below only show reduced generation modes.

```bash
go run ./../cmd/goconfgen \
  -source ./source \
  -out ./variants/complex/target \
  -pkg complexcfg \
  -formats yaml,json,hjson \
  -force
```

## Variant examples

### YAML only

Generates only YAML parsing. CLI, validation, render methods, and presets are disabled.

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

Expected shape: no `cli.go`, no `validate.go`, no `presets.go`, no JSON/HJSON parser files.

### JSON + CLI

Generates JSON parsing/rendering plus CLI argument support. Presets are disabled.

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
