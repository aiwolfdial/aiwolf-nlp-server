# About the Implementation of Game Logic

[logic in Japanese](/doc/ja/logic.md)

This document explains the implementation of the game logic.\
The game logic here refers to the rules of the werewolf game and the processes involved in carrying them out.

## Werewolf Game Rules

### Roles, Factions, and Species

The werewolf game has the following roles:

| Role      | English Name | Faction          | Species  | Special Ability                                               |
| --------- | ------------ | ---------------- | -------- | ------------------------------------------------------------- |
| WEREWOLF  | WEREWOLF     | Werewolf Faction | Werewolf | Votes to attack one agent during the attack phase             |
| POSSESSED | POSSESSED    | Werewolf Faction | Human    | Wins if the Werewolf Faction wins                             |
| SEER      | SEER         | Villager Faction | Human    | Selects one agent during the divination phase                 |
| BODYGUARD | BODYGUARD    | Villager Faction | Human    | Protects one agent during the guard phase                     |
| VILLAGER  | VILLAGER     | Villager Faction | Human    | None                                                          |
| MEDIUM    | MEDIUM       | Villager Faction | Human    | Can learn the species of agents exiled during the exile phase |

The English name for the Villager faction is `VILLAGER`, and the English name for the Werewolf faction is `WEREWOLF`.\
The English name for the Human species is `HUMAN`, and the English name for the Werewolf species is `WEREWOLF`.

For more detailed implementation, please refer to [role.go](../model/role.go).

### Number of Players

#### 5-Player Game

| Role      | Number |
| --------- | ------ |
| WEREWOLF  | 1      |
| POSSESSED | 1      |
| SEER      | 1      |
| BODYGUARD | 0      |
| VILLAGER  | 2      |
| MEDIUM    | 0      |

#### 9-Player Game

| Role      | Number |
| --------- | ------ |
| WEREWOLF  | 2      |
| POSSESSED | 1      |
| SEER      | 1      |
| BODYGUARD | 1      |
| VILLAGER  | 3      |
| MEDIUM    | 1      |

#### 13-Player Game

| Role      | Number |
| --------- | ------ |
| WEREWOLF  | 3      |
| POSSESSED | 1      |
| SEER      | 1      |
| BODYGUARD | 1      |
| VILLAGER  | 6      |
| MEDIUM    | 1      |

### Game Flow

At the start of the game, an `INITIALIZE` request is sent to all agents.

The game is divided into two sections: the [day section](#day-section) and the [night section](#night-section).\
The game starts with the day section on day 0, followed by the night section on day 0, the day section on day 1, and so on.

The game ends when one of the following conditions is met at the end of the [night section](#night-section), [exile phase](#exile-phase), or [attack phase](#attack-phase):

- The number of surviving agents of the Werewolf species is equal to or greater than the number of surviving agents of the Human species: Victory for the Werewolf Faction
- The number of surviving agents of the Werewolf species is 0: Victory for the Villager Faction
- The number of agents in an error state exceeds the maximum allowable error ratio for continuing the game

When the game ends, a `FINISH` request is sent to all agents.

#### Day Section

1. At the start of the day section, a `DAILY_INITIALIZE` request is sent to all agents.
2. If `setting.talk_on_first_day` is `true` and it is the first day (day 0), the [whisper phase](#whisper-phase) begins.
3. The [talk phase](#talk-phase) begins.

#### Night Section

1. At the start of the night section, a `DAILY_FINISH` request is sent to all agents.
2. If `setting.talk_on_first_day` is `true` and it is the first night (night 0), the [whisper phase](#whisper-phase) begins.
3. If it is not day 0, the [exile phase](#exile-phase) begins.
4. The [divination phase](#divination-phase) begins.
5. If it is not day 0, the [whisper phase](#whisper-phase) begins.
6. If it is not day 0, the [guard phase](#guard-phase) begins.
7. If it is not day 0, the [attack phase](#attack-phase) begins.

### About Phases

#### Whisper Phase

If the number of surviving werewolf agents is fewer than 2, this phase is skipped.\
If the number of surviving werewolf agents is 2 or more, the following process occurs.

For information about turn handling, see [turn handling for speeches](#turn-handling-for-speeches).

#### Talk Phase

If the number of surviving agents is fewer than 2, this phase is skipped.\
If the number of surviving agents is 2 or more, the following process occurs.

For information about turn handling, see [turn handling for speeches](#turn-handling-for-speeches).

#### Exile Phase

A `VOTE` request is sent to the surviving agents.\
The responses from the agents are received.\
The valid votes for the most-voted agent are counted, and if there is exactly one agent with the most votes, that agent is exiled.\
If multiple agents have the most votes, the vote will be repeated up to `setting.vote.max_count` times.\
If the vote is repeated and multiple agents still have the most votes, one agent is randomly chosen from the last vote to be exiled.\
If there are no valid votes, no agent is exiled.\
If an agent is exiled, this result is recorded as the exile result and the medium result.

#### Divination Phase

A `DIVINE` request is sent to the surviving seers.\
The responses from the agents are received.\
The species of the target agent is recorded as the divination result.\
If the target is not surviving, no result is recorded.

#### Guard Phase

A `GUARD` request is sent to the surviving bodyguards.\
The responses from the agents are received.\
The target agent is recorded as the guard target.\
If the target is not surviving, no result is recorded.\
If the target is the agent themselves, no result is recorded.

#### Attack Phase

A `ATTACK` request is sent to the surviving werewolves.\
The responses from the agents are received.\
The valid votes for the most-voted agent are counted, and if there is exactly one agent with the most votes, that agent is attacked.\
If multiple agents have the most votes, the vote will be repeated up to `setting.attack_vote.max_count` times.\
If the vote is repeated and multiple agents still have the most votes and `setting.attack_vote.allow_no_target` is `false`, one agent is randomly chosen from the last vote to be attacked.\
The agent is attacked only if the target agent is not guarded.\
The guard is only effective if a bodyguard is surviving at this point.\
If there are no valid votes, no agent is attacked.\
If an agent is attacked, the result is recorded as the attack result.

### About Communication Modes

In the talk phase and whisper phase, one of the following two communication modes is used depending on the configuration:

- **Turn-based mode**: Used when `duration` is not set. This is the traditional mode where the server sends requests to agents in order and receives responses.
- **Group chat (freeform) mode**: Used when `duration` is set. Agents can freely send messages within the time limit without waiting for individual requests from the server.

#### Group Chat Mode Processing

1. At the start of the phase, a `TALK_PHASE_START` (or `WHISPER_PHASE_START` for the whisper phase) request is sent to the participating agents.
2. Agents can freely send text messages to the server within the time limit specified by `duration`.
3. When the server receives a message from an agent, it validates the speech count and character length limits.
4. Valid messages are broadcast to all participating agents as a `TALK_BROADCAST` (or `WHISPER_BROADCAST` for the whisper phase) request.
5. If an agent sends `Over`, that agent can no longer send messages.
6. When the time limit expires, a `TALK_PHASE_END` (or `WHISPER_PHASE_END` for the whisper phase) request is sent to the participating agents, ending the phase.

> [!NOTE]
> In group chat mode, the `max_count.per_day` setting is not used.\
> The `max_count.per_agent` limit on speeches per agent and character length limits are applied in the same way as in the traditional turn-based mode.

### Turn Handling for Speeches

During the whisper phase, the limit `setting.whisper.max_count` is used.\
During the talk phase, the limit `setting.talk.max_count` is used.

If there is a limit on `max_length.base_length`, that value is used; otherwise, 0 is used as `base_length`.\
The remaining characters initialized by `max_length.per_agent` are referred to as `remain_length`.

A random permutation of surviving werewolf agents (or surviving agents in the talk phase) is created.

#### Repeat the following process up to `max_count.per_day` times

Starting from the front of the permutation, the following process is repeated for each agent:

If the agent's `max_count.per_agent` remaining count is 0, skip.\
If the agent's `remain_length` is 0 or less, skip. (If `remaining_length` is not set, skip)\
Send a `WHISPER` request to the agent.\
Wait for a response to the `WHISPER` request.\
If an error occurs, replace the speech with a skip (without increasing the skip count).\
If the skip count exceeds `game.skip.max_count`, replace the speech with an over-speech.\
If the speech is neither over nor skipped, reset the skip count.\
Perform the process for [speech length limits](#speech-length-limits).\
If the speech is over, set the remaining count to 0.

If all agents' speeches are over, end the talk phase.

### Speech Length Limits

During the whisper phase, the limit `setting.whisper.max_length` is used.\
During the talk phase, the limit `setting.talk.max_length` is used.

If the speech is neither over, skipped, nor forced-skip, the following processing is done in order:

#### 1. If there is a limit on `max_length.per_agent` or `max_length.base_length`

**1.a. If a mention (`@AgentName`) is included:**

The portion before the first mention `@` is considered `mention_before`.\
If a mention (`@AgentName`) is included, the part after the first mention is considered `mention_after`.\
Limit `mention_before` to `base_length` + `remain_length` characters.\
If the length of `mention_before` - `base_length` is positive, subtract that from `remain_length`.\
Limit `mention_after` to `max_length.mention_length` + `remain_length` characters.\
If the length of `mention_after` - `max_length.mention_length` is positive, subtract that from `remain_length`.\
Combine `mention_before`, mention, and `mention_after` to form the speech.

**1.b. If no mention (`@AgentName`) is included:**

Limit the speech to `base_length` + `remain_length` characters.\
If the length of the speech - `base_length` is positive, subtract that from `remain_length`.

#### 2. If there is a limit on `max_length.per_talk`

Limit the speech to `max_length.per_talk` characters.

#### 3. If the speech length is 0, replace the speech with an over-speech
