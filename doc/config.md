# 設定ファイルについて

## 環境変数ファイル (.env)

- `SECRET_KEY`: 設定ファイルの `server.authentication.enable` が `true` の場合にトークン検証時の秘密鍵
- `OPENAI_API_KEY`: 設定ファイルの `custom_profile.dynamic_profile.enable` が `true` の場合に使用するChatGPTのAPIキー

## server (サーバ設定)

### web_socket (WebSocketの設定)

- `host`: WebSocketサーバのホスト名
  同一マシン内で接続する場合は `127.0.0.1` を指定してください。
  ローカル内のマシンや外部から接続する場合は `0.0.0.0` を指定してください。
- `port`: WebSocketサーバのポート番号
  基本的に変更する必要はありません。

### authentication (認証の設定)

- `enable`: トークンによる接続認証を有効にするかどうか
  基本的には `false` で問題ありません。

### timeout (タイムアウトの設定)

- `action`: エージェントのアクションのタイムアウト時間
- `response`: エージェントのヘルスチェックのタイムアウト時間
- `acceptable`: サーバ側での猶予時間

- `max_continue_error_ratio`: ゲームを継続するエラーエージェントの最大割合

## game (ゲーム設定)

- `agent_count`: 1ゲームあたりのエージェント数
  5人ゲームの場合は `5`、13人ゲームの場合は `13` を指定してください。
- `max_day`: ゲーム内の最大日数 制限無しの場合は-1
- `vote_visibility`: 投票の結果を公開するかどうか

### talk (トークフェーズの設定)

#### max_count (発言回数の設定)

- `per_agent`: 1日あたりの1エージェントの最大発言回数
- `per_day`: 1日あたりの全体の発言回数

#### max_length (発言の文字数制限の設定)

- `count_in_word`: 単語数でカウントするかどうか
- `per_talk`: 1回のトークあたりの最大文字数 制限無しの場合は-1
- `mention_length`: 1回のトークあたりのメンションを含む場合の追加文字数
- `per_agent`: 1日あたりの1エージェントの最大文字数 制限無しの場合は-1
- `base_length`: 1日あたりの1エージェントの最大文字数に含まない最低文字数 制限無しの場合は-1

- `max_skip`: 1日あたりの1エージェントの最大スキップ回数

### whisper (囁きフェーズの設定)

[talk (トークフェーズの設定)](#talk-トークフェーズの設定)と同様です。

### vote (追放フェーズの設定)

- `max_count`: 1位タイの場合の最大再投票回数
- `allow_self_vote`: 自己投票を許可するか

### attack_vote (襲撃フェーズの設定)

- `max_count`: 1位タイの場合の最大再投票回数
- `allow_self_vote`: 自己投票を許可するか
- `allow_no_target`: 襲撃なしの日を許可するか

## logic (ロジックの設定)

### day_phases (昼セクションのフェーズの設定)

- `name`: 内部的なセクションの名前
- `actions`: 実行するフェーズ
- `only_day`: 特定の日のみに実行する場合の日付 なしの場合はキーごと削除
- `except_day`: 特定の日のみ実行しない場合の日付 なしの場合はキーごと削除

### night_phase (夜セクションのフェーズの設定)

[day_phases (昼セクションのフェーズの設定)](#day_phases-昼セクションのフェーズの設定)と同様です。

## matching (マッチングの設定)

- `self_match`: 同じチーム名のエージェント同士のみをマッチングさせるかどうか
  基本的には `true` で問題ありません。
- `is_optimize`: 最適化した組み合わせマッチングを有効にするかどうか (`self_match` が `false` の場合に限る)
  基本的には `false` で問題ありません。
- `team_count`: 参加するチーム数 (`is_optimize` が `true` の場合に限る)
- `game_count`: 全体のゲーム数 (`is_optimize` が `true` の場合に限る)
- `output_path`: マッチ履歴の出力ファイル (`is_optimize` が `true` の場合に限る)
- `infinite_loop`: 組み合わせマッチングがすべて終了した場合に全体のゲーム数分のゲームを追加するかどうか (`is_optimize` が `true` の場合に限る)
  基本的には `false` で問題ありません。

## custom_profile (カスタムプロフィールの設定)

- `enable`: カスタムプロフィールを有効にするかどうか
  基本的には `true` で問題ありません。

### profiles (各エージェントのカスタムプロフィール)

- `name`: エージェントの名前
- `avatar_url`: エージェントのアバター画像のURL
- `age`: エージェントの年齢
- `gender`: エージェントの性別
- `personality`: エージェントの性格

### dynamic_profile (動的プロフィールの設定)

- `enable`: 動的プロフィールを有効にするかどうか
  デバッグ目的の場合は `false` で問題ありません。
  本戦では事前に準備したカスタムプロフィール(`custom_profile`に記述されているもの)ではなく、ChatGPTを使用して動的にプロフィールを生成します。そのため、より本戦に近い環境で動作させるためには、`true` にしてください。
- `prompt`: プロフィール生成のためのプロンプト
- `attempts`: プロフィール生成の試行回数
- `avatars`: プロフィール生成に使用するアバター画像のURL

## json_logger (JSONロガーの設定)

- `enable`: JSONログの出力を有効にするかどうか
- `output_dir`: JSONログの出力先ディレクトリ
- `filename`: JSONログのファイル名
  拡張子は不要です。`{game_id}` でゲームIDが置換されます。`{timestamp}` でタイムスタンプが置換されます。`{teams}` でチーム名が置換されます。

## game_logger (ゲームロガーの設定)

- `enable`: ゲームログの出力を有効にするかどうか
- `output_dir`: ゲームログの出力先ディレクトリ
- `filename`: ゲームログのファイル名
  拡張子は不要です。`{game_id}` でゲームIDが置換されます。`{timestamp}` でタイムスタンプが置換されます。`{teams}` でチーム名が置換されます。

> [!NOTE]
> json_loggerはサーバと各エージェントの通信をJSON形式で記録するのに対して、game_loggerはゲームの進行を記録します。\
> game_loggerは従来のゲームサーバ([aiwolfdial/AIWolfNLPServer](https://github.com/aiwolfdial/AIWolfNLPServer))と互換性があります。予選時に提出する必要があるログはgame_loggerのログです。

## realtime_broadcaster (リアルタイムブロードキャスターの設定)

- `enable`: リアルタイムブロードキャスターを有効にするかどうか
- `delay`: パケット送信の遅延時間 (TTSブロードキャスターのラグ調整用)
- `output_dir`: リアルタイムブロードキャストログの出力先ディレクトリ
- `filename`: リアルタイムブロードキャストログのファイル名
  拡張子は不要です。`{game_id}` でゲームIDが置換されます。`{timestamp}` でタイムスタンプが置換されます。`{teams}` でチーム名が置換されます。

> [!NOTE]
> リアルタイムブロードキャスターは、ゲームの進行をリアルタイムで配信するための機能です。\
> [aiwolfdial.github.io/aiwolf-nlp-viewer/realtime](https://aiwolfdial.github.io/aiwolf-nlp-viewer/realtime) で確認できます。

## tts_broadcaster (TTSブロードキャスターの設定)

> [!NOTE]
> TTSブロードキャスターは、ゲーム内の発言を音声で再生するための機能です。\
> [VOICEVOX/voicevox_engine](https://github.com/VOICEVOX/voicevox_engine)を使用することで、音声合成を行います。

VOICEVOXが提供するDocker イメージを使用することを推奨します。\
`docker run --rm -p '127.0.0.1:50021:50021' voicevox/voicevox_engine:cpu-latest`

ゲームサーバ起動中はVOICEVOX のサーバが常に起動状態である必要があります。

- `enable`: TTSブロードキャスターを有効にするかどうか
- `async`: 非同期処理による生成を有効にするかどうか
- `target_duration`: 1セグメントの長さ
- `segment_dir`: セグメントの出力ディレクトリ
  このディレクトリ内のファイルはすべて公開されるため注意してください。
- `temp_dir`: 一時ファイルの出力ディレクトリ
  空白の場合はOS依存の一時ディレクトリを使用します。
- `host`: VOICEVOXサーバのホスト名
- `timeout`: VOICEVOXの生成タイムアウト時間
- `ffmpeg_path`: ffmpegのパス
- `ffprobe_path`: ffprobeのパス
- `convert_args`: 生成した音声がセグメント長を満たさない場合にセグメントを生成するための引数
- `duration_args`: 生成した音声の長さを取得するための引数
- `pre_convert_args`: 生成した音声がセグメント長を超える場合に事前変換を行うための引数
- `split_args`: 事前変換した音声をセグメントに分割するための引数
