#!/bin/sh
set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$root"

config=./examples/c-kit.json
generated=./generated-demo
build=./build/verify
mkdir -p "$build"

gcc_flags='-std=c99 -pedantic-errors -Wall -Wextra -Werror -Wconversion -Wsign-conversion -Wshadow -Wstrict-prototypes -Wmissing-prototypes -Wvla -Wformat=2 -Wundef -Wcast-qual -Wwrite-strings'
header_c_flags='-std=c99 -pedantic-errors -Wall -Wextra -Werror'
header_cxx_flags='-std=c++17 -pedantic-errors -Wall -Wextra -Werror'

printf '%s\n' '[1/9] Go unit tests'
go test ./...

printf '%s\n' '[2/9] PDF to Markdown completeness'
uv run --with pdfplumber==0.11.7 python tools/pdf-reference/verify_reference.py \
  --pdf references/huawei-c-language-programming-standard-dkba-2826-2011-5.pdf \
  --markdown references/huawei-c-language-programming-standard-dkba-2826-2011-5.md

printf '%s\n' '[3/9] 139-clause deterministic catalog'
python3 tools/rules/build_rule_catalog.py --check

printf '%s\n' '[4/9] c-kit, snippets, and rule-catalog JSON schemas'
python3 tools/json/validate_json_contracts.py

printf '%s\n' '[5/9] simple demo and public advanced examples generation'
go run ./cmd/cstylekit demo --config "$config" --out "$generated"

printf '%s\n' '[6/9] golden demo lint'
go run ./cmd/cstylekit lint --config "$config" --scope files \
  --file "$generated/demo.c" \
  --file "$generated/demo.h" \
  --file "$generated/advanced/state_machine.c" \
  --file "$generated/advanced/state_machine.h" \
  --file "$generated/advanced/protocol.c" \
  --file "$generated/advanced/protocol.h" \
  --file "$generated/advanced/fixed_pool.c" \
  --file "$generated/advanced/fixed_pool.h" \
  --file "$generated/advanced/tests/advanced_test.c"

sources="$generated/demo.c $generated/advanced/state_machine.c $generated/advanced/protocol.c"
sources="$sources $generated/advanced/fixed_pool.c $generated/advanced/tests/advanced_test.c"

printf '%s\n' '[7/9] GCC strict C99 and behavior tests'
# shellcheck disable=SC2086
gcc $gcc_flags -I "$generated" -I "$generated/advanced" -o "$build/demo_gcc" $sources
"$build/demo_gcc"

printf '%s\n' '[8/9] Clang strict C99'
# shellcheck disable=SC2086
clang $gcc_flags -fsyntax-only -I "$generated" -I "$generated/advanced" $sources

printf '%s\n' '[9/9] independent C99 and C++17 headers'
for header in demo.h; do
  # shellcheck disable=SC2086
  gcc $header_c_flags -I "$generated" -x c -fsyntax-only -include "$header" /dev/null
  # shellcheck disable=SC2086
  g++ $header_cxx_flags -I "$generated" -x c++ -fsyntax-only -include "$header" /dev/null
  # shellcheck disable=SC2086
  clang $header_c_flags -I "$generated" -x c -fsyntax-only -include "$header" /dev/null
  # shellcheck disable=SC2086
  clang++ $header_cxx_flags -I "$generated" -x c++ -fsyntax-only -include "$header" /dev/null
done

for header in state_machine.h protocol.h fixed_pool.h; do
  # shellcheck disable=SC2086
  gcc $header_c_flags -I "$generated/advanced" -x c -fsyntax-only -include "$header" /dev/null
  # shellcheck disable=SC2086
  g++ $header_cxx_flags -I "$generated/advanced" -x c++ -fsyntax-only -include "$header" /dev/null
  # shellcheck disable=SC2086
  clang $header_c_flags -I "$generated/advanced" -x c -fsyntax-only -include "$header" /dev/null
  # shellcheck disable=SC2086
  clang++ $header_cxx_flags -I "$generated/advanced" -x c++ -fsyntax-only -include "$header" /dev/null
done

printf '%s\n' 'verification passed: PDF, 139 clauses, snippets, lint, compilers, headers, tests'
