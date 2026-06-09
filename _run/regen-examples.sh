#!/usr/bin/env bash
# Regenerates every checked-in example package from the single source schema.
# These packages are the golden reference: after a behavior-preserving change
# the final `git diff --exit-code examples/` must be empty.

set -Eeuo pipefail

run_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
root_dir="$(cd "${run_dir}/.." && pwd)"
cd "${root_dir}"

gen() {
  go run ./cmd/goconfgen -source ./examples/source "$@"
}

# Full "everything enabled" example: all formats, all features.
gen -out ./examples/variants/complex/target -pkg complexcfg \
  -formats yaml,json,hjson -force

# YAML only: no render, no CLI, no validate, no presets.
gen -out ./examples/variants/yaml_only/target -pkg yamlonlycfg \
  -formats yaml \
  -with-render=false -with-cli=false -with-validate=false -with-presets=false -force

# JSON + CLI: render and validate on, presets off.
gen -out ./examples/variants/json_cli/target -pkg jsonclicfg \
  -formats json -with-presets=false -force

# HJSON without presets: render and validate on, CLI and presets off.
gen -out ./examples/variants/hjson_no_presets/target -pkg hjsoncfg \
  -formats hjson -with-cli=false -with-presets=false -force

# Self-check: regeneration must be a no-op on a committed tree.
git diff --exit-code examples/ \
  && echo "regen-examples: clean (no drift)" \
  || { echo "regen-examples: drift detected (review the diff above)"; exit 1; }
