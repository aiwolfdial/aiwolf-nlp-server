# About the Configuration File

[config in Japanese](/doc/ja/config.md)

## Environment Variable File (.env)

- `SECRET_KEY`: The secret key used for token verification when `server.authentication.enable` is set to `true` in the configuration file.
- `OPENAI_API_KEY`: The API key for ChatGPT used when `custom_profile.dynamic_profile.enable` is set to `true` in the configuration file.

## server (Server Settings)

### web_socket (WebSocket Settings)

- `host`: The hostname of the WebSocket server.
  For connecting within the same machine, set it to `127.0.0.1`.
  For connecting from a local or external machine, set it to `0.0.0.0`.
- `port`: The port number for the WebSocket server.
  It generally does not need to be changed.

### authentication (Authentication Settings)

- `enable`: Whether to enable connection authentication via tokens.
  Typically, it should be set to `false`.

### timeout (Timeout Settings)

- `action`: Timeout duration for agent actions.
- `response`: Timeout duration for agent health checks.
- `acceptable`: Grace period on the server side.

- `max_continue_error_ratio`: The maximum ratio of error agents that can continue in the game.

## game (Game Settings)

- `agent_count`: The number of agents per game.
  For a 5-player game, set it to `5`, for a 9-player game, set it to `9`, and for a 13-player game, set it to `13`.
- `max_day`: The maximum number of days in the game. If there is no limit, set it to `-1`.
- `vote_visibility`: Whether to reveal the results of votes.

### talk (Talk Phase Settings)

- `duration`: The total time limit for the group chat (freeform) communication phase.
  When this is set, communication uses the group chat mode instead of the turn-based mode.
  When not set, the traditional turn-based mode is used.

#### max_count (Max Speaking Count Settings)

- `per_agent`: The maximum number of times a single agent can speak per day.
- `per_day`: The maximum total number of speeches for all agents per day.

#### max_length (Speech Length Limit Settings)

- `count_in_word`: Whether to count by the number of words.
- `count_spaces`: Whether to include spaces when counting characters.
- `per_talk`: The maximum number of characters per speech. If there is no limit, set it to `-1`.
- `mention_length`: Additional characters when including mentions in a speech.
- `per_agent`: The maximum number of characters a single agent can speak in a day. If there is no limit, set it to `-1`.
- `base_length`: The minimum number of characters not included in the daily character limit for a single agent. If there is no limit, set it to `-1`.

- `max_skip`: The maximum number of skips a single agent can have per day.

### whisper (Whisper Phase Settings)

Same as the [talk (Talk Phase Settings)](#talk-talk-phase-settings).

### vote (Voting Phase Settings)

- `max_count`: The maximum number of re-votes allowed when there is a tie for 1st place.
- `allow_self_vote`: Whether to allow self-voting.

### attack_vote (Attack Phase Settings)

- `max_count`: The maximum number of re-votes allowed when there is a tie for 1st place.
- `allow_self_vote`: Whether to allow self-voting.
- `allow_no_target`: Whether to allow a day without an attack.

## logic (Logic Settings)

### day_phases (Day Phase Settings)

- `name`: The internal name of the section.
- `actions`: The phases to be executed.
- `only_day`: The specific days on which to execute the phase. If there are none, delete the key.
- `except_day`: The specific days on which not to execute the phase. If there are none, delete the key.

### night_phase (Night Phase Settings)

Same as [day_phases (Day Phase Settings)](#day_phases-day-phase-settings).

### roles (Role Count Settings)

This is a structure with counts as keys.
The total number of roles should match the sum of all the keys.

- `WEREWOLF`: The number of werewolves.
- `POSSESSED`: The number of possessed (madmen).
- `SEER`: The number of seers.
- `BODYGUARD`: The number of bodyguards.
- `VILLAGER`: The number of villagers.
- `MEDIUM`: The number of mediums.

## matching (Matching Settings)

- `self_match`: Whether to match agents with the same team name only.
  Generally, it should be set to `true`.
- `is_optimize`: Whether to enable optimized matching when `self_match` is `false`.
  Generally, it should be set to `false`.
- `team_count`: The number of participating teams. (Only applies when `is_optimize` is `true`).
- `game_count`: The total number of games. (Only applies when `is_optimize` is `true`).
- `output_path`: The output file path for the match history. (Only applies when `is_optimize` is `true`).
- `infinite_loop`: Whether to add more games after all combinations of matching have been completed. (Only applies when `is_optimize` is `true`).
  Generally, it should be set to `false`.

## custom_profile (Custom Profile Settings)

- `enable`: Whether to enable custom profiles.
  Generally, it should be set to `true`.
- `profile_encoding`: Items to be encoded in custom profiles or dynamic profiles.

### profiles (Custom Profiles for Each Agent)

- `name`: The name of the agent.
- `avatar_url`: The URL of the agent's avatar image.
- `age`: The age of the agent (optional).
- `gender`: The gender of the agent (optional).
- `personality`: The personality of the agent (optional).

### dynamic_profile (Dynamic Profile Settings)

- `enable`: Whether to enable dynamic profiles.
  For debugging purposes, it can be set to `false`.
  In actual use, to simulate a more realistic environment, it should be set to `true`, where dynamic profiles are generated using ChatGPT instead of custom profiles prepared in advance.
- `prompt`: The prompt used for generating the profile.
- `attempts`: The number of attempts to generate the profile.
- `model`: The model used for profile generation.
- `max_tokens`: The maximum number of tokens for profile generation.
- `avatars`: The URLs of the avatars used for generating the profile.

## json_logger (JSON Logger Settings)

- `enable`: Whether to enable output of JSON logs.
- `output_dir`: The directory to output JSON logs.
- `filename`: The filename for the JSON logs.
  No extension is needed. `{game_id}` will be replaced with the game ID, `{timestamp}` with the timestamp, and `{teams}` with the team names.

## game_logger (Game Logger Settings)

- `enable`: Whether to enable output of game logs.
- `output_dir`: The directory to output game logs.
- `filename`: The filename for the game logs.
  No extension is needed. `{game_id}` will be replaced with the game ID, `{timestamp}` with the timestamp, and `{teams}` with the team names.

> [!NOTE]
> The json_logger records communication between the server and agents in JSON format, while the game_logger records the progress of the game.\
> The game_logger is compatible with the traditional game server ([aiwolfdial/AIWolfNLPServer](https://github.com/aiwolfdial/AIWolfNLPServer)). The logs to be submitted during the preliminaries are the game_logger logs.

## realtime_broadcaster (Real-Time Broadcaster Settings)

- `enable`: Whether to enable the real-time broadcaster.
- `delay`: The delay time for packet transmission (used for adjusting TTS broadcaster lag).
- `output_dir`: The directory for real-time broadcast logs.
  Please be aware that all files in this directory will be made public.
- `filename`: The filename for the real-time broadcast logs.
  No extension is needed. `{game_id}` will be replaced with the game ID, `{timestamp}` with the timestamp, and `{teams}` with the team names.

> [!NOTE]
> The real-time broadcaster is a feature for broadcasting the progress of the game in real-time.\
> It can be checked at [aiwolfdial.github.io/aiwolf-nlp-viewer/realtime](https://aiwolfdial.github.io/aiwolf-nlp-viewer/realtime).

## tts_broadcaster (TTS Broadcaster Settings)

> [!NOTE]
> The TTS broadcaster is a feature for playing in-game speech as audio.\
> Voice synthesis is done using [VOICEVOX/voicevox_engine](https://github.com/VOICEVOX/voicevox_engine).

We recommend using the Docker image provided by VOICEVOX.\
`docker run --rm -p '127.0.0.1:50021:50021' voicevox/voicevox_engine:cpu-latest`

During the game server's operation, the VOICEVOX server must always be running.

- `enable`: Whether to enable the TTS broadcaster.
- `async`: Whether to enable asynchronous processing for generation.
- `target_duration`: The length of one segment.
- `segment_dir`: The directory for outputting segments.
  Please be aware that all files in this directory will be made public.
- `temp_dir`: The directory for temporary files.
  If left blank, the OS-dependent temporary directory will be used.
- `host`: The hostname of the VOICEVOX server.
- `timeout`: The timeout duration for VOICEVOX generation.
- `ffmpeg_path`: The path to ffmpeg.
- `ffprobe_path`: The path to ffprobe.
- `convert_args`: Arguments for generating a segment if the generated audio doesn't meet the segment length.
- `duration_args`: Arguments to retrieve the length of the generated audio.
- `pre_convert_args`: Arguments for pre-conversion if the generated audio exceeds the segment length.
- `split_args`: Arguments for splitting pre-converted audio into segments.
