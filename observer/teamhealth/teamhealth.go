// チームごとの失敗率を直近の試合から集計し、マッチングの重みと隔離判断に使える形で公開する。
// livestate と同じく model と observer にしか依存しない葉パッケージとし、
// 重みをどう使うかは matchmaking 側、通知をどう出すかは service 側に任せる。
package teamhealth

import (
	"log/slog"
	"sort"
	"sync"
	"time"

	"github.com/aiwolfdial/aiwolf-nlp-server/model"
	"github.com/aiwolfdial/aiwolf-nlp-server/observer"
)

// 進行中のゲームで観測しているチーム単位の計測値。ゲーム確定時に record へ畳む。
type liveEntry struct {
	requests      int
	requestErrors int
	fatal         bool
}

// ウィンドウへ積む確定した1ゲーム分の記録。
type record struct {
	requests      int
	requestErrors int
	fatal         bool
	aborted       bool
}

type teamState struct {
	records          []record
	activeGames      int
	quarantinedUntil time.Time
	quarantineCount  int
	lastSeen         time.Time
}

// チーム名 -> 直近の成績。GameObserver として受け取ったイベントと、
// GameManager から報告されるゲームの確定結果の2系統で更新される。
type Tracker struct {
	observer.NoopObserver
	config model.TeamHealthConfig
	mu     sync.Mutex
	teams  map[string]*teamState
	games  map[string]map[string]*liveEntry
	now    func() time.Time
}

// APIとSlack通知が読む、あるチームの現在の健康状態。時刻はUnix秒。
type Snapshot struct {
	Team             string  `json:"team"`
	Games            int     `json:"games"`
	Requests         int     `json:"requests"`
	RequestErrors    int     `json:"request_errors"`
	FatalGames       int     `json:"fatal_games"`
	AbortedGames     int     `json:"aborted_games"`
	FailureRate      float64 `json:"failure_rate"`
	Weight           float64 `json:"weight"`
	Quarantined      bool    `json:"quarantined"`
	QuarantinedUntil *int64  `json:"quarantined_until,omitempty"`
	QuarantineCount  int     `json:"quarantine_count"`
	ActiveGames      int     `json:"active_games"`
	LastSeen         *int64  `json:"last_seen,omitempty"`
}

func New(config model.TeamHealthConfig) *Tracker {
	return &Tracker{
		config: withDefaults(config),
		teams:  make(map[string]*teamState),
		games:  make(map[string]map[string]*liveEntry),
		now:    time.Now,
	}
}

// 未設定のキーを既定値で埋める。設定ファイルに team_health を書かなくても
// 有効化しただけで意味のある挙動になるようにする。
func withDefaults(c model.TeamHealthConfig) model.TeamHealthConfig {
	if c.Window <= 0 {
		c.Window = 20
	}
	if c.MinGames <= 0 {
		c.MinGames = 3
	}
	if c.Scores.RequestError == 0 && c.Scores.Fatal == 0 && c.Scores.Abort == 0 {
		// 脱落は再送で回復しないため、応答エラーや異常終了より重く見る。
		c.Scores.RequestError = 1.0
		c.Scores.Fatal = 3.0
		c.Scores.Abort = 1.0
	}
	if c.WeightFloor <= 0 {
		c.WeightFloor = 0.1
	}
	if c.QuarantineRate <= 0 {
		c.QuarantineRate = 0.5
	}
	if c.QuarantineDuration <= 0 {
		c.QuarantineDuration = 30 * time.Minute
	}
	return c
}

// チーム名はクライアントが名乗った文字列で、サーバは検証していない。名前を変えながら
// 対戦を成立させ続けられると記録が際限なく増えるため、実運用のチーム数からかけ離れた
// 上限を置く。到達したら最終参加が古いものから捨てる。
const maxTrackedTeams = 1000

// 上限ちょうどまでしか削らないと以降は毎ゲーム破棄が走りログが埋まるため、
// ここまでまとめて落として次の破棄まで間隔を空ける。
const trackedTeamsLowWater = maxTrackedTeams * 9 / 10

func (t *Tracker) evictLocked() {
	if len(t.teams) <= maxTrackedTeams {
		return
	}
	now := t.now()
	type aged struct {
		team string
		seen time.Time
	}
	// 隔離中と対戦中のチームは判定に使うため残す。
	candidates := make([]aged, 0, len(t.teams))
	for team, st := range t.teams {
		if st.activeGames > 0 || st.quarantinedUntil.After(now) {
			continue
		}
		candidates = append(candidates, aged{team: team, seen: st.lastSeen})
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].seen.Before(candidates[j].seen) })
	dropped := 0
	for _, c := range candidates {
		if len(t.teams) <= trackedTeamsLowWater {
			break
		}
		delete(t.teams, c.team)
		dropped++
	}
	slog.Warn("チーム記録が上限に達したため古いものを破棄しました",
		"dropped", dropped, "teams", len(t.teams))
}

func (t *Tracker) state(team string) *teamState {
	st, ok := t.teams[team]
	if !ok {
		st = &teamState{}
		t.teams[team] = st
	}
	return st
}

func (t *Tracker) OnGameStart(id string, agents []model.AgentView, _ model.GameState) {
	t.mu.Lock()
	defer t.mu.Unlock()
	entries := make(map[string]*liveEntry)
	now := t.now()
	for _, a := range agents {
		if _, ok := entries[a.TeamName]; ok {
			// 自己対戦では同じチームが複数席を占めるので、チーム単位でまとめて数える。
			continue
		}
		entries[a.TeamName] = &liveEntry{}
		st := t.state(a.TeamName)
		st.activeGames++
		st.lastSeen = now
	}
	t.games[id] = entries
}

func (t *Tracker) OnResponse(id string, agent model.AgentView, _ string, err error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	entry := t.entry(id, agent.TeamName)
	if entry == nil {
		return
	}
	entry.requests++
	if err != nil {
		entry.requestErrors++
	}
}

func (t *Tracker) OnAgentFatal(id string, agent model.AgentView, _ error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if entry := t.entry(id, agent.TeamName); entry != nil {
		entry.fatal = true
	}
}

func (t *Tracker) entry(id string, team string) *liveEntry {
	entries, ok := t.games[id]
	if !ok {
		return nil
	}
	return entries[team]
}

// FinishGame が返すゲーム1件分の確定結果。通知の本文を組み立てるのに使う。
type GameOutcome struct {
	Teams       []string
	FatalTeams  []string
	Quarantined []Snapshot
}

// FinishGame はゲームの確定結果をウィンドウへ積む。abortedByError はエラー多発による
// 打ち切りのときだけ true で、max_day 到達による引き分けとは区別する。
// 新たに隔離へ入ったチームも返し、通知はロックの外で呼び出し側が行う。
func (t *Tracker) FinishGame(id string, abortedByError bool) GameOutcome {
	t.mu.Lock()
	defer t.mu.Unlock()
	entries, ok := t.games[id]
	if !ok {
		return GameOutcome{}
	}
	delete(t.games, id)

	now := t.now()
	outcome := GameOutcome{}
	for team, entry := range entries {
		outcome.Teams = append(outcome.Teams, team)
		if entry.fatal {
			outcome.FatalTeams = append(outcome.FatalTeams, team)
		}
		st := t.state(team)
		if st.activeGames > 0 {
			st.activeGames--
		}
		st.lastSeen = now
		st.records = append(st.records, record{
			requests:      entry.requests,
			requestErrors: entry.requestErrors,
			fatal:         entry.fatal,
			aborted:       abortedByError,
		})
		if len(st.records) > t.config.Window {
			st.records = st.records[len(st.records)-t.config.Window:]
		}
		// 隔離中の失敗は記録するだけで期限を延ばさない。延ばすと失敗が失敗を呼んで
		// 復帰できなくなるため、必ず一度は復帰させて直近の成績で再評価する。
		if st.quarantinedUntil.After(now) {
			continue
		}
		if len(st.records) < t.config.MinGames {
			continue
		}
		if t.failureRate(st) <= t.config.QuarantineRate {
			continue
		}
		st.quarantinedUntil = now.Add(t.config.QuarantineDuration)
		st.quarantineCount++
		outcome.Quarantined = append(outcome.Quarantined, t.snapshot(team, st, now))
	}
	t.evictLocked()
	// map の走査順は不定なので、通知やログの出力が毎回ぶれないよう並べておく。
	sort.Strings(outcome.Teams)
	sort.Strings(outcome.FatalTeams)
	sort.Slice(outcome.Quarantined, func(i, j int) bool {
		return outcome.Quarantined[i].Team < outcome.Quarantined[j].Team
	})
	return outcome
}

// 直近ウィンドウの失敗率を 0..1 で返す。3指標の加重平均で、重みの比だけが意味を持つ。
func (t *Tracker) failureRate(st *teamState) float64 {
	games := len(st.records)
	if games == 0 {
		return 0
	}
	var requests, requestErrors, fatal, aborted int
	for _, r := range st.records {
		requests += r.requests
		requestErrors += r.requestErrors
		if r.fatal {
			fatal++
		}
		if r.aborted {
			aborted++
		}
	}
	sum, total := 0.0, 0.0
	// リクエストが1件も無いゲームしか無いときは応答エラー率を評価できないので項ごと落とす。
	if requests > 0 {
		sum += t.config.Scores.RequestError * float64(requestErrors) / float64(requests)
		total += t.config.Scores.RequestError
	}
	sum += t.config.Scores.Fatal * float64(fatal) / float64(games)
	total += t.config.Scores.Fatal
	sum += t.config.Scores.Abort * float64(aborted) / float64(games)
	total += t.config.Scores.Abort
	if total == 0 {
		return 0
	}
	return sum / total
}

// TeamWeight はマッチの重みに掛ける係数を返す。実績が足りないチームは減点しない。
func (t *Tracker) TeamWeight(team string) float64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	st, ok := t.teams[team]
	if !ok || len(st.records) < t.config.MinGames {
		return 1.0
	}
	weight := 1.0 - t.failureRate(st)
	if weight < t.config.WeightFloor {
		return t.config.WeightFloor
	}
	return weight
}

// IsQuarantined は隔離期間中かどうかを返す。期限を過ぎれば自動で復帰する。
func (t *Tracker) IsQuarantined(team string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	st, ok := t.teams[team]
	return ok && st.quarantinedUntil.After(t.now())
}

func (t *Tracker) snapshot(team string, st *teamState, now time.Time) Snapshot {
	snap := Snapshot{
		Team:            team,
		Games:           len(st.records),
		FailureRate:     t.failureRate(st),
		Weight:          1.0,
		QuarantineCount: st.quarantineCount,
		ActiveGames:     st.activeGames,
	}
	for _, r := range st.records {
		snap.Requests += r.requests
		snap.RequestErrors += r.requestErrors
		if r.fatal {
			snap.FatalGames++
		}
		if r.aborted {
			snap.AbortedGames++
		}
	}
	if len(st.records) >= t.config.MinGames {
		snap.Weight = max(t.config.WeightFloor, 1.0-snap.FailureRate)
	}
	if st.quarantinedUntil.After(now) {
		snap.Quarantined = true
		until := st.quarantinedUntil.Unix()
		snap.QuarantinedUntil = &until
	}
	if !st.lastSeen.IsZero() {
		seen := st.lastSeen.Unix()
		snap.LastSeen = &seen
	}
	return snap
}

// Snapshots は全チームの現在の健康状態をチーム名順で返す。REST API が読む。
func (t *Tracker) Snapshots() []Snapshot {
	t.mu.Lock()
	defer t.mu.Unlock()
	now := t.now()
	snaps := make([]Snapshot, 0, len(t.teams))
	for team, st := range t.teams {
		snaps = append(snaps, t.snapshot(team, st, now))
	}
	sort.Slice(snaps, func(i, j int) bool { return snaps[i].Team < snaps[j].Team })
	return snaps
}

func (t *Tracker) Get(team string) (Snapshot, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	st, ok := t.teams[team]
	if !ok {
		return Snapshot{}, false
	}
	return t.snapshot(team, st, t.now()), true
}
