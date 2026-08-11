# REST API について

[api in English](/doc/en/api.md)

エージェント用の WebSocket (`/ws`) に加えて、サーバの死活監視とゲームの観戦のための読み取り専用 HTTP API を提供します。\
ゲームの開始・停止や設定の投入といった更新系のエンドポイントはありません。

すべてのレスポンスに `Access-Control-Allow-Origin: *` が付与されます。

## エンドポイント一覧

| エンドポイント | 認証 | 説明 |
| --- | --- | --- |
| `GET /api/v1/healthz` | 不要 | 死活監視 |
| `GET /api/v1/readyz` | 不要 | 接続の受付可否 |
| `GET /api/v1/ruleset` | 不要 | このサーバが実行中のルール |
| `GET /api/v1/teams` | 必要 | チームごとの失敗率と隔離状況 |
| `GET /api/v1/teams/{name}` | 必要 | 指定したチームの失敗率と隔離状況 |
| `GET /api/v1/games` | 必要 | 進行中のゲーム一覧 |
| `GET /api/v1/games/{id}` | 必要 | 指定したゲームの現在状態 |
| `GET /api/v1/games/{id}/events` | 必要 | 指定したゲームのイベント配信 (SSE) |
| `GET /realtime/...` | 必要 | リアルタイムブロードキャストログの静的配信 |
| `GET /tts/...` | 不要 | TTS セグメントの静的配信 |

認証欄が「必要」のエンドポイントは、設定ファイルの `server.authentication.enable` が `true` の場合に限りトークンを要求します。\
`/realtime` は `realtime_broadcaster.enable` が、`/tts` は `tts_broadcaster.enable` が `true` の場合にのみ公開されます。

## 認証

`server.authentication.enable` が `true` の場合、閲覧者向けのエンドポイントは RECEIVER トークンを要求します。\
トークンは環境変数 `SECRET_KEY` を秘密鍵とする HMAC 署名の JWT で、クレームに `role: RECEIVER` を含む必要があります。\
（エージェントが `/ws` へ接続する際のトークンは、`role: PLAYER` とチーム名を示す `team` クレームを含む別のトークンです。）

以下のいずれかの方法で渡します。

```bash
curl "http://127.0.0.1:8080/api/v1/games?token=<TOKEN>"
curl -H "Authorization: Bearer <TOKEN>" http://127.0.0.1:8080/api/v1/games
```

トークンが無い、もしくは無効な場合は `401 Unauthorized` を返します。

## 各エンドポイント

### GET /api/v1/healthz

プロセスが応答可能であることを示します。常に `200 OK` を返します。

```json
{ "status": "ok", "version": "v0.0.0" }
```

### GET /api/v1/readyz

新しい接続を受け付けられるかどうかを示します。\
`SIGTERM` などを受信してシャットダウン中の場合は `503 Service Unavailable` と `{"status": "draining"}` を返します。

```json
{ "status": "ready" }
```

### GET /api/v1/ruleset

このプロセスが実行中のルールを返します。サーバは1プロセス1設定で動作するため、一覧ではなく単一のルールを返します。

- `agent_count` (int): 1ゲームあたりのエージェント数.
- `max_day` (int): ゲーム内の最大日数. 制限がない場合は -1.
- `vote_visibility` (bool): 投票の結果を公開するか.
- `is_optimize` (bool): 最適化した組み合わせマッチングが有効か.
- `self_match` (bool): 自己対戦モードが有効か.
- `roles` (dict[str, int]): 役職ごとの人数.

### GET /api/v1/teams

チームごとの失敗率と隔離状況を返します。マッチングの重みがなぜ下がったかを外から確認するために使用します。\
`team_health.enable` が `false` の場合は `404 Not Found` を返します。

集計は直近 `team_health.window` 試合が対象で、それより古い記録は評価から外れます。\
各項目の意味は [team_health の設定](/doc/ja/config.md#team_health-チーム健全性の設定) を参照してください。

```json
{
  "teams": [
    {
      "team": "kanolab",
      "games": 12,
      "requests": 480,
      "request_errors": 3,
      "fatal_games": 1,
      "aborted_games": 1,
      "failure_rate": 0.08,
      "weight": 0.92,
      "quarantined": false,
      "quarantine_count": 0,
      "active_games": 1,
      "last_seen": 1750000000
    }
  ],
  "progress": { "done": 40, "total": 100 }
}
```

- `teams` (list[Team]): チームごとの状態. チーム名の昇順.
  - `team` (str): チーム名.
  - `games` (int): 評価対象となっている試合数.
  - `requests` (int): 評価対象の試合で送信したリクエスト数.
  - `request_errors` (int): そのうちタイムアウトやエラーで終わった数.
  - `fatal_games` (int): エージェントが脱落した試合数.
  - `aborted_games` (int): エラー多発で打ち切られた試合数.
  - `failure_rate` (float): 失敗率. 0から1.
  - `weight` (float): マッチの重みに掛かる係数. 実績が `min_games` に満たない場合は 1.0.
  - `quarantined` (bool): 隔離中か.
  - `quarantined_until` (int | None): 隔離が解除される時刻 (Unix 秒). 隔離中のみ.
  - `quarantine_count` (int): 隔離された回数の累計.
  - `active_games` (int): 現在進行中の試合数.
  - `last_seen` (int | None): 最後にゲームへ参加した時刻 (Unix 秒).
- `progress` (dict | None): 消化済みと予定の試合数. `matching.is_optimize` が `true` の場合のみ.

### GET /api/v1/teams/{name}

指定したチームの状態を返します。構造は `teams` の各要素と同じです。\
一度もゲームへ参加していないチームの場合は `404 Not Found` を返します。

### GET /api/v1/games

進行中のゲームの一覧を返します。終了したゲームは登録簿から取り除かれるため、含まれません。

```json
{ "games": [ { "id": "...", "agents": [], "finished": false } ] }
```

各要素の構造は [GET /api/v1/games/{id}](#get-apiv1gamesid) と同じですが、`day` と `status_by_agent` は設定されません。\
これらが必要な場合は個別のエンドポイントを参照してください。

### GET /api/v1/games/{id}

指定したゲームの現在状態を返します。存在しない場合は `404 Not Found` を返します。

- `id` (str): ゲームの識別子.
- `day` (int): 現在の日数.
- `finished` (bool): ゲームが終了しているか.
- `win_side` (str): 勝利陣営. 終了したゲームは登録簿から取り除かれるため、通常は空文字列.
- `agents` (list[Agent]): 参加エージェントの一覧. ゲーム開始時点の情報.
  - `idx` (int): エージェントのインデックス.
  - `team_name` (str): チーム名.
  - `original_name` (str): 接続時のエージェント名.
  - `game_name` (str): ゲーム内のエージェント名.
  - `role` (str): 役職.
  - `alive` (bool): 生存しているか. 開始時点の値のため常に `true`.
- `status_by_agent` (dict[int, str]): エージェントのインデックスごとの生存状態 (`ALIVE` / `DEAD`).

現在の生存状態は `agents[].alive` ではなく `status_by_agent` を参照してください。

### GET /api/v1/games/{id}/events

指定したゲームのイベントを Server-Sent Events で配信します。存在しない場合は `404 Not Found` を返します。\
イベント名は `broadcast` で、データは [ブロードキャストパケット](#ブロードキャストパケット) の JSON です。

購読を開始した時点で直近のパケットが1件配信されるため、途中から接続しても現在状態を取得できます。\
ゲームが終了すると配信は終了します。

```bash
curl -N http://127.0.0.1:8080/api/v1/games/<GAME_ID>/events
```

```text
event:broadcast
data:{"id":"...","idx":1,"day":0,"is_day":true,"agents":[...],"event":"開始","message":"ゲームが開始されました","timestamp":1750000000}
```

> [!NOTE]
> 同じ内容は `realtime_broadcaster` によって JSONL ファイルにも記録され、`/realtime` から静的配信されます。\
> SSE はそのファイルをポーリングする代わりに使用できます。

## ブロードキャストパケット

- `id` (str): ゲームの識別子.
- `idx` (int): ゲーム内で1から連番となるパケットのインデックス.
- `day` (int): 現在の日数.
- `is_day` (bool): 昼セクションであるか.
- `agents` (list[Agent]): 各エージェントの現在の情報.
  - `idx` (int): エージェントのインデックス.
  - `team` (str): チーム名.
  - `name` (str): ゲーム内のエージェント名.
  - `profile` (str | None): プロフィール.
  - `avatar` (str | None): アバター画像の URL.
  - `role` (str): 役職.
  - `is_alive` (bool): 生存しているか.
- `event` (str): イベントの種類.
- `message` (str | None): イベントに紐づく文言や発言の内容.
- `from_idx` (int | None): 行動したエージェントのインデックス.
- `to_idx` (int | None): 対象となったエージェントのインデックス.
- `bubble_idx` (int | None): 発言を表示するエージェントのインデックス.
- `timestamp` (int): イベントの発生時刻 (Unix 秒).

`event` の種類は以下の通りです。

| event | message | from_idx | to_idx | bubble_idx |
| --- | --- | --- | --- | --- |
| `開始` | 固定文言 | - | - | - |
| `終了` | 勝利陣営 | - | - | - |
| `トーク` | 発言の内容 | - | - | 発言者 |
| `囁き` | 発言の内容 | - | - | 発言者 |
| `投票` | - | 投票者 | 投票先 | - |
| `襲撃投票` | - | 投票者 | 投票先 | - |
| `追放` | - | - | 追放者 (いない場合は省略) | - |
| `占い` | - | 占い師 | 占い先 | - |
| `護衛` | - | 騎士 | 護衛先 | - |
| `襲撃` | - | 護衛された場合は -1 | 襲撃対象 (いない場合は省略) | - |
