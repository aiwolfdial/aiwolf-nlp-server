# AGENTS.md

人狼知能コンテスト（自然言語部門）のゲームサーバです。\
WebSocket でエージェント（クライアント）を待ち受け、マッチングしたエージェント同士で人狼ゲームを進行させます。

## 開発コマンド

```bash
go build ./...                    # ビルド
go test -race ./...               # 全テスト（test/ 配下は実際にサーバを起動して対戦する）
go test -race ./test/ -run TestTalkPhase1 -v  # 個別テスト
go vet ./...                      # 静的解析
gofmt -l .                        # 未整形ファイルの検出
go run . -c ./config/default_5.yml      # 5人村で起動
go run . -c ./config/freeform_5.yml     # 5人村（グループチャット方式）で起動
```

CI (`.github/workflows/test.yml`) は Go 1.25.x で `go build -race` と `go test -race` を実行します。\
`.vscode/launch.json` にデバッグ実行の設定があります。

### コマンドラインフラグ

- `-c`: 設定ファイルのパス（既定 `./default.yml`）
- `-a`: 解析モード（`matching.output_path` のマッチ履歴を集計する）
- `-r`: 縮約モード（`-s` のマッチ履歴を `-d` の設定へ縮約する）
- `-s` / `-d`: 縮約モードのソース/デスティネーション設定ファイル
- `-v` / `-h`: バージョン表示 / ヘルプ表示

## ディレクトリ構成

| ディレクトリ | 責務 |
| --- | --- |
| `main.go` | フラグ解析、`.env` 読み込み、設定読み込み、サーバ起動 |
| `transport/` | WebSocket 接続の受付、HTTP ルータ、REST API、認証 |
| `orchestrator/` | `GameManager`。マッチング成立からゲーム生成・破棄・シャットダウンまで |
| `matchmaking/` | 待機部屋、マッチオプティマイザ、マッチ履歴の解析 |
| `logic/` | ゲームロジック。日付進行、各フェーズ、発言の集約と制限 |
| `model/` | 設定・パケット・エージェント等のデータ構造。他パッケージに依存しない |
| `observer/` | ゲームイベントの通知インターフェースと配信の共通実装 |
| `service/` | observer の実装。JSON ログ、ゲームログ、リアルタイム配信、TTS |
| `store/` | マッチオプティマイザの永続化 |
| `util/` | 認証、文字数カウント、プロフィール生成などの補助関数 |
| `config/` | 配布用の設定ファイル (`default_*.yml` / `freeform_*.yml`) |
| `test/` | サーバを起動して実クライアントで対戦する結合テスト |
| `doc/` | 日本語 (`doc/ja`) と英語 (`doc/en`) のドキュメント |

依存の向きは `transport` → `orchestrator` → `logic` → `observer` → `model` です。\
`model` と `observer` は葉に近いパッケージなので、上位パッケージを import しないでください。

詳細は [アーキテクチャについて](/doc/ja/architecture.md) を参照してください。

## 設計上の約束

- **observer は logic の唯一の外部出力口**です。ロガーや配信を追加する場合は `observer.GameObserver` を実装し、`transport.Server.newObserver` で合成します。`logic` から直接ファイルや HTTP を触らないでください。
- **整形は sink 側で行います。** CSV 書式やブロードキャストパケットの組み立ては `service` / `observer` 側の責務で、`logic` はセマンティックなイベントを通知するだけです。
- **observer へ渡す値は読み取り専用のビュー型**（`model.AgentView` / `model.TalkView` / `model.GameSnapshot`）にします。`model.Agent` は `Connection` を含むため外部へ渡しません。
- **ゲームは設定を `RulesetView` / `SettingView` 経由で読みます。** getter のみのインターフェースなので、ゲーム開始後に設定が変わることはありません。
- 通信方式は設定の `talk.duration` / `whisper.duration` の有無で分岐します。ターンベース方式は `logic/communication_turn.go`、グループチャット方式は `logic/communication_freeform.go` にあり、発言の検証と文字数制限は `logic/communication_session.go` に共通化されています。

## コーディング規約

- コメントは日本語で、**なぜそうしているか**を1〜2行で書きます。自明な処理に説明を付けないでください。
- ログは `log/slog` を使い、メッセージは日本語、属性はキーバリューで渡します（例: `slog.Info("ゲームを開始します", "id", g.id)`）。
- エラーメッセージ・ユーザ向け文字列も日本語です。
- 標準の `gofmt` に従います。追加のリンタ設定はありません。

## テスト

`test/` 配下は実際に `transport.NewServer` でサーバを起動し、`TestClient` を接続させて1ゲーム分を通す結合テストです。\
設定は `test/config/*.yml` を使い、ポートは 49152〜65535 からランダムに選ばれます。\
フェーズやログの挙動を変えた場合は、`test/game_test.go` などの既存テストが通ることを確認してください。

## ドキュメント

`doc/ja` と `doc/en` は同じ内容の対訳です。**片方だけ更新しないでください。**

- [設定ファイルについて](/doc/ja/config.md): 設定ファイルの全キー
- [ゲームロジックの実装について](/doc/ja/logic.md): ルールとフェーズの進行
- [プロトコルの実装について](/doc/ja/protocol.md): サーバ・エージェント間のパケット
- [REST API について](/doc/ja/api.md): 監視・観戦用の HTTP API
- [アーキテクチャについて](/doc/ja/architecture.md): パッケージ構成と拡張点

設定キーを追加・改名した場合は `config/*.yml` すべてと `doc/*/config.md` を、\
パケットの構造を変えた場合は `doc/*/protocol.md` を合わせて更新します。

## コミット

Conventional Commits のプレフィックス（`feat:` `fix:` `refactor:` `chore:` `style:`）に日本語の説明を続けます。\
例: `fix: per_agentとbase_lengthが両方無効なとき発言本文が失われる問題を修正`
