# aiwolf-nlp-server

[README in English](/README.en.md)

人狼知能コンテスト（自然言語部門） のゲームサーバです。

サンプルエージェントについては、[aiwolfdial/aiwolf-nlp-agent](https://github.com/aiwolfdial/aiwolf-nlp-agent) を参考にしてください。

## ドキュメント

- [設定ファイルについて](/doc/ja/config.md)
- [ゲームロジックの実装について](/doc/ja/logic.md)
- [プロトコルの実装について](/doc/ja/protocol.md)

## 実行方法

デフォルトのサーバアドレスは `ws://127.0.0.1:8080/ws` です。エージェントプログラムの接続先には、このアドレスを指定してください。\
同じチーム名のエージェント同士のみをマッチングさせる自己対戦モードは、デフォルトで有効になっています。そのため、異なるチーム名のエージェント同士をマッチングさせる場合は、設定ファイルを変更してください。\
設定ファイルの変更方法については、[設定ファイルについて](/doc/ja/config.md)を参照してください。

### Linux

```bash
curl -LO https://github.com/aiwolfdial/aiwolf-nlp-server/releases/latest/download/aiwolf-nlp-server-linux-amd64
curl -LO https://github.com/aiwolfdial/aiwolf-nlp-server/releases/latest/download/default_5.yml
curl -LO https://github.com/aiwolfdial/aiwolf-nlp-server/releases/latest/download/default_9.yml
curl -LO https://github.com/aiwolfdial/aiwolf-nlp-server/releases/latest/download/default_13.yml
curl -Lo .env https://github.com/aiwolfdial/aiwolf-nlp-server/releases/latest/download/example.env
chmod u+x ./aiwolf-nlp-server-linux-amd64
./aiwolf-nlp-server-linux-amd64 -c ./default_5.yml # 5人ゲームの場合
# ./aiwolf-nlp-server-linux-amd64 -c ./default_9.yml # 9人ゲームの場合
# ./aiwolf-nlp-server-linux-amd64 -c ./default_13.yml # 13人ゲームの場合
```

### Windows

```bash
curl -LO https://github.com/aiwolfdial/aiwolf-nlp-server/releases/latest/download/aiwolf-nlp-server-windows-amd64.exe
curl -LO https://github.com/aiwolfdial/aiwolf-nlp-server/releases/latest/download/default_5.yml
curl -LO https://github.com/aiwolfdial/aiwolf-nlp-server/releases/latest/download/default_9.yml
curl -LO https://github.com/aiwolfdial/aiwolf-nlp-server/releases/latest/download/default_13.yml
curl -Lo .env https://github.com/aiwolfdial/aiwolf-nlp-server/releases/latest/download/example.env
.\aiwolf-nlp-server-windows-amd64.exe -c .\default_5.yml # 5人ゲームの場合
# .\aiwolf-nlp-server-windows-amd64.exe -c .\default_9.yml # 9人ゲームの場合
# .\aiwolf-nlp-server-windows-amd64.exe -c .\default_13.yml # 13人ゲームの場合
```

### macOS (Intel)

> [!NOTE]
> 開発元が不明なアプリケーションとしてブロックされる場合があります。\
> 下記サイトを参考に、実行許可を与えてください。  
> <https://support.apple.com/ja-jp/guide/mac-help/mh40616/mac>

```bash
curl -LO https://github.com/aiwolfdial/aiwolf-nlp-server/releases/latest/download/aiwolf-nlp-server-darwin-amd64
curl -LO https://github.com/aiwolfdial/aiwolf-nlp-server/releases/latest/download/default_5.yml
curl -LO https://github.com/aiwolfdial/aiwolf-nlp-server/releases/latest/download/default_9.yml
curl -LO https://github.com/aiwolfdial/aiwolf-nlp-server/releases/latest/download/default_13.yml
curl -Lo .env https://github.com/aiwolfdial/aiwolf-nlp-server/releases/latest/download/example.env
chmod u+x ./aiwolf-nlp-server-darwin-amd64
./aiwolf-nlp-server-darwin-amd64 -c ./default_5.yml # 5人ゲームの場合
# ./aiwolf-nlp-server-darwin-amd64 -c ./default_9.yml # 9人ゲームの場合
# ./aiwolf-nlp-server-darwin-amd64 -c ./default_13.yml # 13人ゲームの場合
```

### macOS (Apple Silicon)

> [!NOTE]
> 開発元が不明なアプリケーションとしてブロックされる場合があります。\
> 下記サイトを参考に、実行許可を与えてください。  
> <https://support.apple.com/ja-jp/guide/mac-help/mh40616/mac>

```bash
curl -LO https://github.com/aiwolfdial/aiwolf-nlp-server/releases/latest/download/aiwolf-nlp-server-darwin-arm64
curl -LO https://github.com/aiwolfdial/aiwolf-nlp-server/releases/latest/download/default_5.yml
curl -LO https://github.com/aiwolfdial/aiwolf-nlp-server/releases/latest/download/default_9.yml
curl -LO https://github.com/aiwolfdial/aiwolf-nlp-server/releases/latest/download/default_13.yml
curl -Lo .env https://github.com/aiwolfdial/aiwolf-nlp-server/releases/latest/download/example.env
chmod u+x ./aiwolf-nlp-server-darwin-arm64
./aiwolf-nlp-server-darwin-arm64 -c ./default_5.yml # 5人ゲームの場合
# ./aiwolf-nlp-server-darwin-arm64 -c ./default_9.yml # 9人ゲームの場合
# ./aiwolf-nlp-server-darwin-arm64 -c ./default_13.yml # 13人ゲームの場合
```

### Docker

```bash
docker compose up --build # default_5.yml で起動（ポート 8080 を公開）
```

設定ファイルは `./config` を、ログは `./log` をマウントして利用します。\
TTS（VOICEVOX）を利用する場合は `docker-compose.yml` の `voicevox` サービスのコメントを外し、`AIWOLF_TTS_HOST` を設定してください（スリムなサーバイメージには ffmpeg は含まれません）。

## 環境変数による上書き

コンテナ実行など、設定ファイルを編集せずにサーバ設定を上書きしたい場合は、以下の環境変数が利用できます（未設定時は設定ファイルの値が使われます）。

| 環境変数 | 上書き対象 |
| --- | --- |
| `AIWOLF_HOST` | `server.web_socket.host` |
| `AIWOLF_PORT` | `server.web_socket.port` |
| `AIWOLF_AUTH_ENABLE` | `server.authentication.enable` |
| `AIWOLF_JSON_LOG_DIR` | `json_logger.output_dir` |
| `AIWOLF_GAME_LOG_DIR` | `game_logger.output_dir` |
| `AIWOLF_REALTIME_DIR` | `realtime_broadcaster.output_dir` |
| `AIWOLF_TTS_HOST` | `tts_broadcaster.host` |
| `AIWOLF_MATCH_OUTPUT` | `matching.output_path` |
| `AIWOLF_RULESETS_DIR` | `/api/v1/rulesets` が走査する設定ディレクトリ |
| `SECRET_KEY` | 認証有効時の JWT 検証鍵 |

## REST API

エージェント用の WebSocket（`/ws`）に加えて、ゲームの進捗・設定を参照するための読み取り専用 REST API を提供します（`server.authentication.enable` が有効な場合、`/api/v1/games` と `/api/v1/rulesets` は RECEIVER トークンが必要です）。

| エンドポイント | 説明 |
| --- | --- |
| `GET /api/v1/healthz` | 死活監視（常に 200） |
| `GET /api/v1/readyz` | 受付可否（シャットダウン中は 503） |
| `GET /api/v1/games` | 進行中ゲームの一覧 |
| `GET /api/v1/games/:id` | 指定ゲームの現在状態 |
| `GET /api/v1/games/:id/events` | ゲームのリアルタイムイベント（SSE） |
| `GET /api/v1/rulesets` | 利用可能な設定ファイルの一覧 |
