#!/usr/bin/env bash
# schema/protocol.schema.json から各言語の定義とドキュメントを生成します。
# 生成そのものは既製のツールが行い、このスクリプトは呼び出し方をまとめているだけです。
#
#   ./schema/generate.sh                     # サーバ側 (Goとドキュメント) を更新
#   ./schema/generate.sh --check             # 生成物がスキーマと一致するか検査
#   ./schema/generate.sh python <出力先>      # aiwolf-nlp-common のpacketを更新
set -euo pipefail

cd "$(dirname "$0")/.."

SCHEMA=schema/protocol.schema.json
ENUM_ATTRS=schema/enum_attributes.json
WIRE_OUT=model/wire/wire_gen.go
DOC_JA=doc/ja/protocol-schema.md
DOC_EN=doc/en/protocol-schema.md

CHECK=0
if [ "${1:-}" = "--check" ]; then
  CHECK=1
  shift
fi
TARGET="${1:-server}"

work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

# 生成結果を出力先へ反映する。--check のときは差分の有無だけを見る。
emit() {
  local src=$1 dst=$2
  if [ "$CHECK" = "1" ]; then
    if ! diff -q "$dst" "$src" >/dev/null 2>&1; then
      echo "スキーマと一致していません: $dst" >&2
      diff -u "$dst" "$src" | head -40 >&2 || true
      return 1
    fi
  else
    mkdir -p "$(dirname "$dst")"
    cp "$src" "$dst"
    echo "生成しました: $dst"
  fi
}

# 説明を英語へ差し替えたスキーマ。ドキュメントの英語版を生成するために使う。
english_schema() {
  python3 - "$SCHEMA" "$1" <<'PY'
import json, sys
def walk(o):
    if isinstance(o, dict):
        o = {k: walk(v) for k, v in o.items()}
        if "x-description-en" in o:
            o["description"] = o.pop("x-description-en")
        return o
    if isinstance(o, list):
        return [walk(v) for v in o]
    return o
json.dump(walk(json.load(open(sys.argv[1]))), open(sys.argv[2], "w"), ensure_ascii=False, indent=2)
PY
}

generate_go() {
  go tool github.com/atombender/go-jsonschema \
    --package wire \
    --only-models \
    --tags json \
    --disable-omitzero \
    --disable-custom-types-for-maps \
    --capitalization ID \
    --output "$work/wire_gen.go" \
    "$SCHEMA"
  gofmt -w "$work/wire_gen.go"
  emit "$work/wire_gen.go" "$WIRE_OUT"
}

generate_doc() {
  uvx --from jsonschema2md jsonschema2md "$SCHEMA" "$work/ja.md"
  english_schema "$work/en.schema.json"
  uvx --from jsonschema2md jsonschema2md "$work/en.schema.json" "$work/en.md"
  emit "$work/ja.md" "$DOC_JA"
  emit "$work/en.md" "$DOC_EN"
}

# 列挙値の付随属性 (役職の陣営など) は、カスタムテンプレートへ渡すために入れ子を1段深くする。
generate_python() {
  local out=$1
  [ -n "$out" ] || { echo "出力先を指定してください" >&2; exit 1; }
  python3 -c 'import json,sys; d=json.load(open(sys.argv[1])); json.dump({k:{"enum_attrs":v} for k,v in d.items()}, open(sys.argv[2],"w"))' \
    "$ENUM_ATTRS" "$work/extra.json"
  uvx --from datamodel-code-generator datamodel-codegen \
    --input "$SCHEMA" \
    --input-file-type jsonschema \
    --output-model-type dataclasses.dataclass \
    --target-python-version 3.11 \
    --formatters black \
    --custom-template-dir schema/python_templates \
    --extra-template-data "$work/extra.json" \
    --custom-file-header-path schema/python_templates/header.txt \
    --additional-imports "typing.Any,msgspec.convert" \
    --output "$work/models.py"
  emit "$work/models.py" "$out/_models.py"
}

case "$TARGET" in
  server) generate_go; generate_doc ;;
  go) generate_go ;;
  doc) generate_doc ;;
  python) generate_python "${2:-}" ;;
  *) echo "不明な対象: $TARGET" >&2; exit 1 ;;
esac

if [ "$CHECK" = "1" ]; then
  echo "生成物はスキーマと一致しています。"
fi
