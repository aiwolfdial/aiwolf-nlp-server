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

For the type and description of each field, see [Protocol Schema](/doc/en/protocol-schema.md), which is generated from the schema.\
The definition itself lives in [schema/protocol.schema.json](/schema/protocol.schema.json).

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
