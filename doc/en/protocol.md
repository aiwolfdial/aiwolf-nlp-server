# About the Implementation of the Protocol

[protocol in Japanese](/doc/ja/protocol.md)

This document explains the implementation of the protocol.\
The protocol here refers not to a technical layer protocol but to the string-based protocol used during the interaction between the Werewolf AI agents and the server.\
Additionally, in the traditional Werewolf AI competition connection system, the agent side listens as a server, and the game master side was referred to as the competition connection system. However, in this system using WebSocket, the game master side listens as a server, and the agent side is referred to as the client.

## Overview of the Protocol

Messages sent from the server to the agents are all in JSON string format.\
In contrast, messages sent from the agents to the server are raw strings.\
In this document, messages sent from the server to the agents are referred to as requests (packets), and messages sent from the agents to the server are referred to as responses.

### Overview of Requests

- [Name Request](#name-request-name) `NAME`
- [Game Start Request](#game-start-request-initialize) `INITIALIZE`
- [Day Start Request](#day-start-request-daily_initialize) `DAILY_INITIALIZE`
- [Whisper Request](#whisper-request-whisper--talk-request-talk) `WHISPER`
- [Talk Request](#whisper-request-whisper--talk-request-talk) `TALK`
- [Day End Request](#day-end-request-daily_finish) `DAILY_FINISH`
- [Divine Request](#divine-request-divine) `DIVINE`
- [Guard Request](#guard-request-guard) `GUARD`
- [Vote Request](#vote-request-vote) `VOTE`
- [Attack Request](#attack-request-attack) `ATTACK`
- [Game End Request](#game-end-request-finish) `FINISH`
- [Talk Phase Start Request](#talk-phase-start-request-talk_phase_start) `TALK_PHASE_START` (Group Chat mode only)
- [Talk Phase End Request](#talk-phase-end-request-talk_phase_end) `TALK_PHASE_END` (Group Chat mode only)
- [Talk Broadcast Request](#talk-broadcast-request-talk_broadcast) `TALK_BROADCAST` (Group Chat mode only)
- [Whisper Phase Start Request](#whisper-phase-start-request-whisper_phase_start) `WHISPER_PHASE_START` (Group Chat mode only)
- [Whisper Phase End Request](#whisper-phase-end-request-whisper_phase_end) `WHISPER_PHASE_END` (Group Chat mode only)
- [Whisper Broadcast Request](#whisper-broadcast-request-whisper_broadcast) `WHISPER_BROADCAST` (Group Chat mode only)

Depending on the type of request, the information contained in the request and whether a response is required differs.\
For detailed implementation, refer to [request.go](../model/request.go) and [packet.go](../model/packet.go).

### Overview of Responses

Responses can either return natural language strings from the agents in response to Talk and Whisper requests (e.g., `Hello`) or return the name of the target agent (e.g., `Agent[01]`) for requests like Voting or Divining.

## Structure of Requests

Packet structure.

- request ([Request](#request)): Type of request.
- info ([Info](#info) | None): Information indicating the current game settings.
- setting ([Setting](#setting) | None): Game setting information.
- talk_history (list[[Talk](#talk)] | None): History of talks.
- whisper_history (list[[Talk](#talk)] | None): History of whispers.
- new_talk ([Talk](#talk) | None): Newly broadcast talk in group chat mode. (Only for `TALK_BROADCAST` requests).
- new_whisper ([Talk](#talk) | None): Newly broadcast whisper in group chat mode. (Only for `WHISPER_BROADCAST` requests).

### Request

Detailed descriptions for each type of request are provided below.

#### Name Request (NAME)

The Name Request is sent when an agent connects to the server.\
The agent must return its own name upon receiving this request.\
When multiple agents connect, a unique number should be appended to the name.\
For example, if the agent returns the name `kanolab`, it should be returned as `kanolab1`, `kanolab2`, etc.\
The part of the name before the number is treated as the agent's team name.

> [!IMPORTANT]
> The name referred to here is used for server-side matching and differs from the agent's name within the game.

#### Game Start Request (INITIALIZE)

The Game Start Request is sent when the game begins.\
The agent does not need to return anything upon receiving this request.

#### Day Start Request (DAILY_INITIALIZE)

The Day Start Request is sent when the day begins, i.e., when the next day starts.\
The agent does not need to return anything upon receiving this request.

#### Whisper Request (WHISPER) / Talk Request (TALK)

The Whisper and Talk Requests are sent when either a whisper or talk is requested.\
The Whisper Request is sent to werewolves only when two or more werewolves are still alive.\
The agent must respond to this request with a natural language string for either whispering or talking.\
The server only sends the differential from the previous agent's request, not the entire history.

#### Day End Request (DAILY_FINISH)

The Day End Request is sent when the day ends, i.e., when the night begins.\
The agent does not need to return anything upon receiving this request.\
The conversation history up until that point is sent.\
Even if there are fewer than two werewolves alive and the whisper phase does not exist, whisper history is still sent to werewolves.

#### Divine Request (DIVINE)

The Divine Request is sent when a divination is requested.\
It is sent only to the seers.\
The agent must respond to this request with the name of the agent to be divined.

#### Guard Request (GUARD)

The Guard Request is sent when a guard action is requested.\
It is sent only to bodyguards.\
The agent must respond with the name of the agent to be guarded.

#### Vote Request (VOTE)

The Vote Request is sent when voting to exile an agent.\
The agent must respond to this request with the name of the agent to be voted on.

#### Attack Request (ATTACK)

The Attack Request is sent when voting to attack an agent.\
It is sent only to werewolves.\
The agent must respond with the name of the agent to be attacked.\
The conversation history up until that point is sent.\
Even if there are fewer than two werewolves alive and no whisper phase exists, whisper history is still sent to werewolves.

#### Game End Request (FINISH)

The Game End Request is sent when the game ends.\
The agent does not need to return anything upon receiving this request.\
The keys for this request are the same as the Game Start Request, except that [Setting](#setting) is not sent.\
Unlike the Game Start Request, the [Info](#info) contains the role_map, which includes the roles of all agents, including those other than the agent.

#### Talk Phase Start Request (TALK_PHASE_START)

Sent when the talk phase starts in group chat mode.\
The agent does not need to return anything upon receiving this request.\
After receiving this request, the agent can freely send talks without waiting for requests from the server.

#### Talk Phase End Request (TALK_PHASE_END)

Sent when the talk phase ends in group chat mode.\
The agent does not need to return anything upon receiving this request.\
After receiving this request, the agent must stop sending talks.

#### Talk Broadcast Request (TALK_BROADCAST)

Broadcast to all participating agents when an agent sends a talk in group chat mode.\
The agent does not need to return anything upon receiving this request.\
The `new_talk` field in the packet contains the newly sent talk.

#### Whisper Phase Start Request (WHISPER_PHASE_START)

Sent when the whisper phase starts in group chat mode.\
Behaves the same as the Talk Phase Start Request. Only sent to werewolf agents.

#### Whisper Phase End Request (WHISPER_PHASE_END)

Sent when the whisper phase ends in group chat mode.\
Behaves the same as the Talk Phase End Request. Only sent to werewolf agents.

#### Whisper Broadcast Request (WHISPER_BROADCAST)

Broadcast to werewolf agents when an agent sends a whisper in group chat mode.\
The agent does not need to return anything upon receiving this request.\
The `new_whisper` field in the packet contains the newly sent whisper.

### Info

The structure that contains information about the current state of the game within the packet.

- game_id (str): Game identifier.
- day (int): Current day.
- agent (str): The name of the agent.
- profile (str | None): The agent's profile. (Only for `INITIALIZE` request). If not set, it is None.
- medium_result ([Judge](#judge) | None): The result of the medium (only if the agent's role is Medium and the result is set).
- divine_result ([Judge](#judge) | None): The result of the divination (only if the agent's role is Seer and the result is set).
- executed_agent (str | None): The result of the previous night's exile (only if an agent was exiled).
- attacked_agent (str | None): The result of the previous night's attack (only if an agent was attacked).
- vote_list (list[[Vote](#vote)] | None): The results of the votes (only if vote results are public).
- attack_vote_list (list[[Vote](#vote)] | None): The results of the attack votes (only if the agent's role is Werewolf and the attack vote results are public).
- status_map (dict[str, [Status](#status)]): A map showing the survival status of each agent.
- role_map (dict[str, [Role](#role)]): A map showing the roles of each agent (roles of agents other than oneself are not visible).
- remain_count (int | None): The maximum number of remaining possible talk or whisper requests (only for `TALK` or `WHISPER` requests).
- remain_length (int | None): The maximum number of characters that can be consumed by remaining talk or whisper requests, excluding the minimum character count. If no limit, set to None.
- remain_skip (int | None): The number of remaining skips allowed for talk or whisper requests (only for `TALK` or `WHISPER` requests).

### Judge

The structure that contains the results of divinations or medium results.

- day (int): The day the judgment was made.
- agent (str): The agent who made the judgment.
- target (str): The agent who was judged.
- result ([Species](#species)): The result of the judgment.

### Species

An enumeration that represents species.

- HUMAN (str): Human.
- WEREWOLF (str): Werewolf.

### Vote

The structure that contains the content of a vote.

- day (int): The day the vote took place.
- agent (str): The agent who cast the vote.
- target (str): The agent who was voted on.

### Status

An enumeration representing the survival status of an agent.

- ALIVE (str): Alive.
- DEAD (str): Dead.

### Role

An enumeration representing the role of an agent.

- WEREWOLF (str): Werewolf.
- POSSESSED (str): Madman.
- SEER (str): Seer.
- BODYGUARD (str): Bodyguard.
- VILLAGER (str): Villager.
- MEDIUM (str): Medium.

### Setting

The structure that contains the game settings.

- agent_count (int): Number of players in the game.
- max_day (int | None): Maximum number of days in the game. If no limit, set to None.
- role_num_map (dict[[Role](#role), int]): A map showing the number of each role.
- vote_visibility (bool): Whether to reveal the results of the votes.
- talk.max.count.per_agent (int): Maximum number of speeches per agent per day.
- talk.max.count.per_day (int): Maximum number of speeches for all agents per day.
- talk.max.length.count_in_word (bool | None): Whether to count by word count. If not set, it is None.
- talk.max_length.count_spaces (bool | None): Whether to include spaces when counting characters. None if not set.
- talk.max.length.per_talk (int | None): Maximum number of characters per talk. If no limit, set to None.
- talk.max.length.mention_length (int | None): Additional character count when mentioning another agent in a talk. If no limit, set to None.
- talk.max.length.per_agent (int | None): Maximum number of characters per agent per day. If no limit, set to None.
- talk.max.length.base_length (int | None): Minimum number of characters not included in the daily character limit per agent. If no limit, set to None.
- talk.duration (int | None): Total time limit for the group chat phase (in seconds). None if not set.
- talk.max.skip (int): Maximum number of skips per agent per day.
- whisper.max.count.per_agent (int): Maximum number of whispers per agent per day.
- whisper.max.count.per_day (int): Maximum number of whispers for all agents per day.
- whisper.max.length.count_in_word (bool | None): Whether to count by word count. If not set, it is None.
- whisper.max_length.count_spaces (bool | None): Whether to include spaces when counting characters. None if not set.
- whisper.max.length.per_talk (int | None): Maximum number of characters per whisper. If no limit, set to None.
- whisper.max.length.mention_length (int | None): Additional character count when mentioning another agent in a whisper. If no limit, set to None.
- whisper.max.length.per_agent (int | None): Maximum number of characters per agent per day in whispers. If no limit, set to None.
- whisper.max.length.base_length (int | None): Minimum number of characters not included in the daily whisper character limit per agent. If no limit, set to None.
- whisper.duration (int | None): Total time limit for the group chat phase (in seconds). None if not set.
- whisper.max.skip (int): Maximum number of skips per agent per day in whispers.
- vote.max.count (int): Maximum number of re-votes allowed in case of a tie for first place.
- vote.allow_self_vote (bool): Whether self-voting is allowed.
- attack_vote.max.count (int): Maximum number of re-votes allowed for attacks in case of a tie for first place.
- attack_vote.allow_self_vote (bool): Whether self-voting is allowed for attacks.
- attack_vote.allow_no_target (bool): Whether to allow a day with no target for an attack.
- timeout.action (int): Timeout duration for agent actions (in milliseconds).
- timeout.response (int): Timeout duration for agent survival checks (in milliseconds).

### Talk

The structure that contains the content of the conversation.

- idx (int): Index of the conversation.
- day (int): The day the conversation took place.
- turn (int): The turn number when the conversation took place.
- agent (str): The name of the agent who spoke.
- text (str): The content of the conversation.
- skip (bool): Whether the conversation was skipped.
- over (bool): Whether the conversation was over.
