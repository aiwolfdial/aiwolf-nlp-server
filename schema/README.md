# スキーマについて

`protocol.schema.json` は、サーバがエージェントへ送信するパケットの構造の唯一の定義です（JSON Schema 2020-12）。\
Go・Python・ドキュメントの3か所に同じ構造を手で書き写していた状態を解消するために置かれています。

生成そのものは既製のツールが行い、`generate.sh` は呼び出し方をまとめているだけです。

```bash
./schema/generate.sh                    # Goとドキュメントを更新
./schema/generate.sh --check            # 生成物がスキーマと一致するか検査（CIが実行）
./schema/generate.sh python <出力先>     # aiwolf-nlp-common のpacketを更新
```

| 出力先 | ツール |
| --- | --- |
| `model/wire/wire_gen.go` | [go-jsonschema](https://github.com/omissis/go-jsonschema)（`go.mod` の `tool` でバージョン固定） |
| `doc/ja/protocol-schema.md`, `doc/en/protocol-schema.md` | [jsonschema2md](https://github.com/sbrunner/jsonschema2md) |
| `aiwolf-nlp-common` の `packet/_models.py` | [datamodel-code-generator](https://github.com/koxudaxi/datamodel-code-generator) |

`aiwolf-nlp-common` 側は、参照するサーバのリビジョンを同リポジトリの `.schema-ref` に固定しており、\
CI (`schema-check.yml`) がそのリビジョンの `generate.sh` で `--check` を実行します。\
スキーマを変更したら、サーバ側をマージしてから `.schema-ref` を更新し、common 側で生成を回してください。

## スキーマを書くときの約束

素の JSON Schema です。独自キーワードは `x-description-en`（英語ドキュメント用の説明）だけで、\
`description` には日本語を書きます。ただし、生成物の形を決めるために2点だけ約束があります。

- **`talk_history` / `whisper_history` は `"type": ["array", "null"]`**。\
  こう書くと go-jsonschema がポインタ型を生成し、「空配列を送った」と「フィールドを送らなかった」を区別できます。\
  ただの `"array"` にすると空配列がキーごと省略され、エージェント側の挙動が変わります。
- **囁きの設定は `SettingTalk` を参照する**。\
  別定義にすると Go で相互に変換できない別の型になり、`model.TalkSetting` をトークと囁きで共用できなくなります。

## スキーマに入れないもの

役職と陣営・種族の対応、リクエストごとの応答要否は `enum_attributes.json` にあります。\
JSON Schema では列挙値に付随する属性を表現できないためです。このファイルは2か所で使われます。

- Python: `datamodel-code-generator` のカスタムテンプレートへ渡され、`Role.team` などのプロパティになる
- Go: `model/schema_test.go` が読み、手書きの `model/role.go` `model/request.go` と一致するか検査する

Go 側は生成していないので、値を足すときは `enum_attributes.json` と `model/role.go` の両方を更新してください。\
片方だけ直すとテストが落ちます。

## 既製ツールを使っている理由

当初は独自のジェネレータ（約1,400行）を書いていましたが、実際に既製ツールを試したところ\
オプションで必要な出力が得られたため置き換えました。判断の根拠は次のとおりです。

- **go-jsonschema**: `--only-models --tags json --disable-omitzero --disable-custom-types-for-maps --capitalization ID` を\
  付けると、余分な検証コードやタグのない構造体が得られます。`GameID` の大文字化も `--capitalization` で解決します。
- **datamodel-code-generator**: `--custom-template-dir` で列挙にプロパティを足せます。\
  `from_dict` は msgspec の `convert` を呼ぶだけのテンプレートにしてあり、未知の列挙値や必須項目の欠落を\
  該当箇所のパス付きで弾きます（手書きの `from_dict` にはこの検証がありませんでした）。
- **jsonschema2md**: `$defs` 単位で相互リンクする簡潔な Markdown を出します。\
  英語版は `description` を `x-description-en` へ差し替えたスキーマから生成しています。
- **quicktype** は不採用です。`$defs` の名前を使わず利用箇所から型名を捏造するため\
  （`Role` → `RoleMapValue`、`Talk` → `NewTalk`）、公開APIには使えません。
- **Protobuf / protojson** も不採用です。protojson はゼロ値のフィールドを省略するため、\
  `vote_visibility: false` や `day: 0` が JSON から消え、wire format が変わってしまいます。
