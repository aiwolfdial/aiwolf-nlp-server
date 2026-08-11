# About the REST API

[api in Japanese](/doc/ja/api.md)

In addition to the WebSocket endpoint for agents (`/ws`), the server provides a read-only HTTP API for health monitoring and spectating games.
There are no endpoints for updates such as starting/stopping games or submitting configurations.

All responses include `Access-Control-Allow-Origin: *`.

## Endpoints

| Endpoint | Authentication | Description |
| --- | --- | --- |
| `GET /api/v1/healthz` | Not required | Liveness check |
| `GET /api/v1/readyz` | Not required | Whether new connections are accepted |
| `GET /api/v1/ruleset` | Not required | The ruleset this server is running |
| `GET /api/v1/games` | Required | List of games in progress |
| `GET /api/v1/games/{id}` | Required | Current state of the specified game |
| `GET /api/v1/games/{id}/events` | Required | Event stream for the specified game (SSE) |
| `GET /realtime/...` | Required | Static delivery of realtime broadcaster logs |
| `GET /tts/...` | Not required | Static delivery of TTS segments |

Endpoints marked as "Required" demand a token only when `server.authentication.enable` is `true` in the configuration file.
`/realtime` is exposed only when `realtime_broadcaster.enable` is `true`, and `/tts` only when `tts_broadcaster.enable` is `true`.

## Authentication

When `server.authentication.enable` is `true`, the spectator endpoints require a RECEIVER token.
The token is an HMAC-signed JWT using the `SECRET_KEY` environment variable as the secret key, and its claims must include `role: RECEIVER`.
(The token used when an agent connects to `/ws` is a different one, containing `role: PLAYER` and a `team` claim indicating the team name.)

Pass it in either of the following ways.

```bash
curl "http://127.0.0.1:8080/api/v1/games?token=<TOKEN>"
curl -H "Authorization: Bearer <TOKEN>" http://127.0.0.1:8080/api/v1/games
```

If the token is missing or invalid, `401 Unauthorized` is returned.

## Endpoint Details

### GET /api/v1/healthz

Indicates that the process is able to respond. Always returns `200 OK`.

```json
{ "status": "ok", "version": "v0.0.0" }
```

### GET /api/v1/readyz

Indicates whether new connections can be accepted.
While shutting down after receiving `SIGTERM` or similar, it returns `503 Service Unavailable` with `{"status": "draining"}`.

```json
{ "status": "ready" }
```

### GET /api/v1/ruleset

Returns the ruleset this process is running. Since the server runs with one configuration per process, it returns a single ruleset rather than a list.

- `agent_count` (int): The number of agents per game.
- `max_day` (int): The maximum number of days in the game. -1 if there is no limit.
- `vote_visibility` (bool): Whether the results of votes are revealed.
- `is_optimize` (bool): Whether optimized combination matching is enabled.
- `self_match` (bool): Whether self-play mode is enabled.
- `roles` (dict[str, int]): The number of agents for each role.

### GET /api/v1/games

Returns the list of games in progress. Finished games are removed from the registry and are therefore not included.

```json
{ "games": [ { "id": "...", "agents": [], "finished": false } ] }
```

Each element has the same structure as [GET /api/v1/games/{id}](#get-apiv1gamesid), except that `day` and `status_by_agent` are not populated.
Use the individual endpoint if you need them.

### GET /api/v1/games/{id}

Returns the current state of the specified game. If it does not exist, `404 Not Found` is returned.

- `id` (str): The identifier of the game.
- `day` (int): The current day.
- `finished` (bool): Whether the game has finished.
- `win_side` (str): The winning team. Normally an empty string, since finished games are removed from the registry.
- `agents` (list[Agent]): The list of participating agents, as of the start of the game.
  - `idx` (int): The index of the agent.
  - `team_name` (str): The team name.
  - `original_name` (str): The agent name given at connection time.
  - `game_name` (str): The name of the agent in the game.
  - `role` (str): The role.
  - `alive` (bool): Whether the agent is alive. Always `true`, since it is the value at the start of the game.
- `status_by_agent` (dict[int, str]): The status (`ALIVE` / `DEAD`) for each agent index.

Refer to `status_by_agent` rather than `agents[].alive` for the current status.

### GET /api/v1/games/{id}/events

Streams the events of the specified game via Server-Sent Events. If the game does not exist, `404 Not Found` is returned.
The event name is `broadcast`, and the data is the JSON of a [broadcast packet](#broadcast-packet).

The most recent packet is delivered immediately upon subscribing, so a late subscriber can still obtain the current state.
The stream ends when the game finishes.

```bash
curl -N http://127.0.0.1:8080/api/v1/games/<GAME_ID>/events
```

```text
event:broadcast
data:{"id":"...","idx":1,"day":0,"is_day":true,"agents":[...],"event":"開始","message":"ゲームが開始されました","timestamp":1750000000}
```

> [!NOTE]
> The same content is also recorded to a JSONL file by `realtime_broadcaster` and served statically from `/realtime`.
> SSE can be used instead of polling that file.

## Broadcast Packet

- `id` (str): The identifier of the game.
- `idx` (int): The packet index, numbered sequentially from 1 within a game.
- `day` (int): The current day.
- `is_day` (bool): Whether it is the day section.
- `agents` (list[Agent]): The current information of each agent.
  - `idx` (int): The index of the agent.
  - `team` (str): The team name.
  - `name` (str): The name of the agent in the game.
  - `profile` (str | None): The profile.
  - `avatar` (str | None): The URL of the avatar image.
  - `role` (str): The role.
  - `is_alive` (bool): Whether the agent is alive.
- `event` (str): The type of the event.
- `message` (str | None): The text associated with the event, or the content of a speech.
- `from_idx` (int | None): The index of the agent that acted.
- `to_idx` (int | None): The index of the agent that was targeted.
- `bubble_idx` (int | None): The index of the agent whose speech bubble should be displayed.
- `timestamp` (int): The time the event occurred (Unix seconds).

The event types are as follows. Note that the `event` values are Japanese strings.

| event | message | from_idx | to_idx | bubble_idx |
| --- | --- | --- | --- | --- |
| `開始` (start) | Fixed text | - | - | - |
| `終了` (end) | Winning team | - | - | - |
| `トーク` (talk) | Content of the speech | - | - | Speaker |
| `囁き` (whisper) | Content of the speech | - | - | Speaker |
| `投票` (vote) | - | Voter | Vote target | - |
| `襲撃投票` (attack vote) | - | Voter | Vote target | - |
| `追放` (execution) | - | - | Executed agent (omitted if none) | - |
| `占い` (divine) | - | Seer | Divine target | - |
| `護衛` (guard) | - | Bodyguard | Guard target | - |
| `襲撃` (attack) | - | -1 if guarded | Attack target (omitted if none) | - |
