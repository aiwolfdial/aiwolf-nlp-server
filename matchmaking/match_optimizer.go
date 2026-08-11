package matchmaking

import (
	"encoding/json"
	"errors"
	"log/slog"
	"math"
	"sort"
	"sync"

	"github.com/aiwolfdial/aiwolf-nlp-server/model"
	"github.com/aiwolfdial/aiwolf-nlp-server/store"
	"github.com/aiwolfdial/aiwolf-nlp-server/util"
)

// チームごとの信頼度をマッチの重みへ持ち込む入口。observer/teamhealth.Tracker が実装する。
// マッチ単位の失敗（Weight）とチーム単位の失敗（この係数）を同じ実効重みの上で扱うために挟む。
type TeamScorer interface {
	TeamWeight(team string) float64
	IsQuarantined(team string) bool
}

type MatchOptimizer struct {
	mu                sync.RWMutex              `json:"-"`
	store             store.MatchOptimizerStore `json:"-"`
	scorer            TeamScorer                `json:"-"`
	abortWeightFactor float64                   `json:"-"`
	InfiniteLoop      bool                      `json:"infinite_loop"`
	TeamCount         int                       `json:"team_count"`
	GameCount         int                       `json:"game_count"`
	RoleNumMap        map[model.Role]int        `json:"role_num_map"`
	IdxTeamMap        map[int]string            `json:"idx_team_map"`
	ScheduledMatches  []model.MatchWeight       `json:"scheduled_matches"`
	EndedMatches      []map[model.Role][]int    `json:"ended_matches"`
}

func (mo *MatchOptimizer) MarshalJSON() ([]byte, error) {
	roleNumMap := make(map[string]int)
	for k, v := range mo.RoleNumMap {
		roleNumMap[k.String()] = v
	}
	endedMatches := make([]map[string][]int, len(mo.EndedMatches))
	for i, match := range mo.EndedMatches {
		endedMatches[i] = make(map[string][]int)
		for role, idxs := range match {
			endedMatches[i][role.String()] = idxs
		}
	}
	scheduledMatches := make([]model.MatchWeight, len(mo.ScheduledMatches))
	copy(scheduledMatches, mo.ScheduledMatches)
	type Alias MatchOptimizer
	return json.Marshal(&struct {
		*Alias
		RoleNumMap       map[string]int      `json:"role_num_map"`
		EndedMatches     []map[string][]int  `json:"ended_matches"`
		ScheduledMatches []model.MatchWeight `json:"scheduled_matches"`
	}{
		Alias:            (*Alias)(mo),
		RoleNumMap:       roleNumMap,
		EndedMatches:     endedMatches,
		ScheduledMatches: scheduledMatches,
	})
}

func (mo *MatchOptimizer) UnmarshalJSON(data []byte) error {
	type Alias MatchOptimizer
	aux := &struct {
		*Alias
		RoleNumMap       map[string]int     `json:"role_num_map"`
		EndedMatches     []map[string][]int `json:"ended_matches"`
		ScheduledMatches []struct {
			RoleIdxs map[string][]int `json:"role_idxs"`
			Weight   float64          `json:"weight"`
		} `json:"scheduled_matches"`
	}{
		Alias: (*Alias)(mo),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	mo.RoleNumMap = make(map[model.Role]int)
	for role, num := range aux.RoleNumMap {
		mo.RoleNumMap[model.RoleFromString(role)] = num
	}
	mo.EndedMatches = make([]map[model.Role][]int, len(aux.EndedMatches))
	for i, match := range aux.EndedMatches {
		mo.EndedMatches[i] = make(map[model.Role][]int)
		for role, idxs := range match {
			mo.EndedMatches[i][model.RoleFromString(role)] = idxs
		}
	}
	mo.ScheduledMatches = make([]model.MatchWeight, len(aux.ScheduledMatches))
	for i, scheduledMatch := range aux.ScheduledMatches {
		mo.ScheduledMatches[i] = model.MatchWeight{
			RoleIdxs: make(map[model.Role][]int),
			Weight:   scheduledMatch.Weight,
		}
		for role, idxs := range scheduledMatch.RoleIdxs {
			mo.ScheduledMatches[i].RoleIdxs[model.RoleFromString(role)] = idxs
		}
	}
	return nil
}

func NewMatchOptimizer(config model.Config) (*MatchOptimizer, error) {
	st := store.NewFileMatchOptimizerStore(config.Matching.OutputPath)
	data, err := st.Load()
	if err != nil {
		slog.Warn("マッチオプティマイザの読み込みに失敗しました", "error", err)
		return NewMatchOptimizerFromConfig(config)
	}
	var mo MatchOptimizer
	if err := json.Unmarshal(data, &mo); err != nil {
		slog.Error("マッチオプティマイザのパースに失敗しました", "error", err)
		return nil, err
	}
	mo.store = st
	mo.abortWeightFactor = config.Matching.AbortWeightFactor
	mo.save()
	return &mo, nil
}

func NewMatchOptimizerFromConfig(config model.Config) (*MatchOptimizer, error) {
	slog.Info("マッチオプティマイザを作成します")
	roles, err := model.RolesFromConfig(config)
	if err != nil {
		return nil, err
	}
	mo := &MatchOptimizer{
		store:             store.NewFileMatchOptimizerStore(config.Matching.OutputPath),
		abortWeightFactor: config.Matching.AbortWeightFactor,
		InfiniteLoop:      config.Matching.InfiniteLoop,
		TeamCount:         config.Matching.TeamCount,
		GameCount:         config.Matching.GameCount,
		RoleNumMap:        roles,
		IdxTeamMap:        map[int]string{},
	}
	mo.initialize()
	return mo, nil
}

// SetTeamScorer はチーム単位の信頼度の供給元を差し込む。nil のままなら
// 従来どおりマッチ単位の Weight だけで優先度が決まる。
func (mo *MatchOptimizer) SetTeamScorer(scorer TeamScorer) {
	mo.mu.Lock()
	defer mo.mu.Unlock()
	mo.scorer = scorer
}

// スケジュール済みマッチを優先度の高い順に返す。優先度は保存された Weight に参加チームの
// 信頼度を掛けた実効重みで、マッチ単位の失敗もチーム単位の失敗も同じ重みの上で表現する。
// 隔離中のチームを含むマッチは候補から外す。
func (mo *MatchOptimizer) GetMatches() []map[model.Role][]string {
	mo.mu.Lock()
	defer mo.mu.Unlock()
	count := 0
	for _, match := range mo.ScheduledMatches {
		if match.Weight > 0.0 {
			count++
		}
	}
	if count == 0 && mo.InfiniteLoop {
		slog.Info("スケジュールされたマッチがないため、新たに追加します")
		mo.appendLocked()
	}

	// 保存された重みの降順で格納スライス自体も並べ替える。RoleIdxs が同一のマッチは
	// Equal で区別できず、SetMatchEnd も updateWeight も先頭の一致要素を選ぶため、
	// この順序が「消化されるのは重みの高い方、下げられた方は残る」という対応を決める。
	sort.SliceStable(mo.ScheduledMatches, func(i, j int) bool {
		return mo.ScheduledMatches[i].Weight > mo.ScheduledMatches[j].Weight
	})

	type candidate struct {
		teams  map[model.Role][]string
		weight float64
	}
	candidates := make([]candidate, 0, len(mo.ScheduledMatches))
	for _, match := range mo.ScheduledMatches {
		teams := util.IdxMatchToTeamNameMatch(mo.IdxTeamMap, match.RoleIdxs)
		candidates = append(candidates, candidate{teams: teams, weight: mo.effectiveWeight(match.Weight, teams)})
	}
	// 返却は実効重みの降順。元の実装は組み立てた後に並べ替えていたため、重みが次回の
	// 呼び出しまで反映されなかった。チームの信頼度は変動するので格納順には持ち込まない。
	sort.SliceStable(candidates, func(i, j int) bool { return candidates[i].weight > candidates[j].weight })

	matches := []map[model.Role][]string{}
	excluded := 0
	for _, c := range candidates {
		if mo.hasQuarantinedTeam(c.teams) {
			excluded++
			continue
		}
		matches = append(matches, c.teams)
	}
	// 隔離で候補が全部消えるとゲームが二度と成立しないため、そのときだけ隔離を無視する。
	if len(matches) == 0 && excluded > 0 {
		slog.Warn("隔離により候補が無くなったため、隔離を無視して全マッチを対象にします", "excluded", excluded)
		for _, c := range candidates {
			matches = append(matches, c.teams)
		}
	}
	return matches
}

// マッチ単位の重みに参加チームの信頼度を掛ける。1チームでも不安定なら全体の優先度が下がる。
func (mo *MatchOptimizer) effectiveWeight(weight float64, teams map[model.Role][]string) float64 {
	if mo.scorer == nil {
		return weight
	}
	for _, names := range teams {
		for _, team := range names {
			if team == "" {
				// 未接続でまだ idx_team_map に載っていないチーム。実績が無いので減点しない。
				continue
			}
			weight *= mo.scorer.TeamWeight(team)
		}
	}
	return weight
}

func (mo *MatchOptimizer) hasQuarantinedTeam(teams map[model.Role][]string) bool {
	if mo.scorer == nil {
		return false
	}
	for _, names := range teams {
		for _, team := range names {
			if team != "" && mo.scorer.IsQuarantined(team) {
				return true
			}
		}
	}
	return false
}

// Teams は対戦表に登録済みのチーム名を返す。接続していないチームを割り出すのに使う。
func (mo *MatchOptimizer) Teams() []string {
	mo.mu.RLock()
	defer mo.mu.RUnlock()
	teams := make([]string, 0, len(mo.IdxTeamMap))
	for _, team := range mo.IdxTeamMap {
		teams = append(teams, team)
	}
	sort.Strings(teams)
	return teams
}

// Progress は消化済みと予定の試合数を返す。進捗通知と REST API が読む。
func (mo *MatchOptimizer) Progress() (done int, total int) {
	mo.mu.RLock()
	defer mo.mu.RUnlock()
	total = mo.GameCount
	if total <= 0 {
		total = len(mo.EndedMatches) + len(mo.ScheduledMatches)
	}
	return len(mo.EndedMatches), total
}

func (mo *MatchOptimizer) UpdateTeam(team string) {
	mo.mu.Lock()
	defer mo.mu.Unlock()
	for _, t := range mo.IdxTeamMap {
		if t == team {
			slog.Info("チームが既に登録されています", "team", team)
			return
		}
	}
	idx := len(mo.IdxTeamMap)
	if idx >= mo.TeamCount {
		slog.Warn("チーム数が上限に達しているため追加できません", "team", team)
		return
	}
	mo.IdxTeamMap[idx] = team
	slog.Info("チームを追加しました", "team", team, "idx", idx)
	mo.save()
}

func (mo *MatchOptimizer) initialize() error {
	mo.mu.Lock()
	defer mo.mu.Unlock()
	slog.Info("マッチオプティマイザを初期化します")
	mo.EndedMatches = []map[model.Role][]int{}
	mo.ScheduledMatches = []model.MatchWeight{}
	return mo.appendLocked()
}

// 呼び出し側が mo.mu を保持している前提。GetMatches から呼ぶため、
// ここで再度ロックを取ると自己デッドロックになる。
func (mo *MatchOptimizer) appendLocked() error {
	theoretical, roles := util.CalcTheoretical(mo.RoleNumMap, mo.GameCount, mo.TeamCount)
	slog.Info("各役職の理論値を計算しました", "theoretical", theoretical)

	maxAttempts := mo.GameCount * mo.TeamCount * 5
	var bestMatches []map[model.Role][]int
	bestDeviation := math.MaxFloat64
	slog.Info("マッチング最適化を開始します", "attempts", maxAttempts)

	for attempt := range maxAttempts {
		matches, deviation := util.GenerateMatches(mo.GameCount, mo.TeamCount, roles, theoretical)
		if bestMatches == nil || deviation < bestDeviation {
			slog.Info("より良い解が見つかりました", "deviation", deviation, "attempt", attempt)
			bestMatches = matches
			bestDeviation = deviation
		}
	}

	if bestMatches != nil {
		for _, match := range bestMatches {
			mw := model.MatchWeight{
				RoleIdxs: match,
				Weight:   1.0,
			}
			mo.ScheduledMatches = append(mo.ScheduledMatches, mw)
		}
		mo.save()
		slog.Info("最良の解を採用します", "bestDeviation", bestDeviation)
		return nil
	}
	return errors.New("最適なマッチングが見つかりませんでした")
}

func (mo *MatchOptimizer) SetMatchEnd(match map[model.Role][]string) {
	mo.mu.Lock()
	defer mo.mu.Unlock()
	idxMatch := util.TeamNameMatchToIdxMatch(mo.IdxTeamMap, match)

	for i, scheduledMatch := range mo.ScheduledMatches {
		if scheduledMatch.Equal(model.MatchWeight{RoleIdxs: idxMatch}) {
			mo.ScheduledMatches = append(mo.ScheduledMatches[:i], mo.ScheduledMatches[i+1:]...)
			slog.Info("スケジュールされたマッチから削除しました", "length", len(mo.ScheduledMatches))

			mo.EndedMatches = append(mo.EndedMatches, idxMatch)
			slog.Info("マッチ履歴を追加しました", "length", len(mo.EndedMatches))
			mo.save()
			return
		}
	}
	slog.Warn("スケジュールされたマッチが見つかりませんでした")
}

func (mo *MatchOptimizer) SetMatchWeight(match map[model.Role][]string, weight float64) {
	mo.updateWeight(match, func(float64) float64 { return weight })
}

// PenalizeMatch は異常終了したマッチの重みを下げる。abort_weight_factor が 0 なら
// 従来どおり一度で最下位まで落ち、0 より大きければ失敗のたびに段階的に下がる。
func (mo *MatchOptimizer) PenalizeMatch(match map[model.Role][]string) {
	mo.updateWeight(match, func(current float64) float64 { return current * mo.abortWeightFactor })
}

func (mo *MatchOptimizer) updateWeight(match map[model.Role][]string, next func(float64) float64) {
	mo.mu.Lock()
	defer mo.mu.Unlock()
	idxMatch := util.TeamNameMatchToIdxMatch(mo.IdxTeamMap, match)

	for i, scheduledMatch := range mo.ScheduledMatches {
		if scheduledMatch.Equal(model.MatchWeight{RoleIdxs: idxMatch}) {
			mo.ScheduledMatches[i].Weight = next(mo.ScheduledMatches[i].Weight)
			slog.Info("スケジュールされたマッチの重みを更新しました", "weight", mo.ScheduledMatches[i].Weight)
			mo.save()
			return
		}
	}
	slog.Warn("スケジュールされたマッチが見つかりませんでした")
}

func (mo *MatchOptimizer) save() error {
	if mo.store == nil {
		return errors.New("マッチオプティマイザのストアが設定されていません")
	}
	jsonData, err := json.Marshal(mo)
	if err != nil {
		return err
	}
	return mo.store.Save(jsonData)
}
