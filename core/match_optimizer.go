package core

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

type MatchOptimizer struct {
	mu               sync.RWMutex              `json:"-"`
	store            store.MatchOptimizerStore `json:"-"`
	InfiniteLoop     bool                      `json:"infinite_loop"`
	TeamCount        int                       `json:"team_count"`
	GameCount        int                       `json:"game_count"`
	RoleNumMap       map[model.Role]int        `json:"role_num_map"`
	IdxTeamMap       map[int]string            `json:"idx_team_map"`
	ScheduledMatches []model.MatchWeight       `json:"scheduled_matches"`
	EndedMatches     []map[model.Role][]int    `json:"ended_matches"`
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
		store:        store.NewFileMatchOptimizerStore(config.Matching.OutputPath),
		InfiniteLoop: config.Matching.InfiniteLoop,
		TeamCount:    config.Matching.TeamCount,
		GameCount:    config.Matching.GameCount,
		RoleNumMap:   roles,
		IdxTeamMap:   map[int]string{},
	}
	mo.initialize()
	return mo, nil
}

func (mo *MatchOptimizer) getMatches() []map[model.Role][]string {
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
		mo.append()
	}
	matches := []map[model.Role][]string{}
	for _, match := range mo.ScheduledMatches {
		matches = append(matches, util.IdxMatchToTeamNameMatch(mo.IdxTeamMap, match.RoleIdxs))
	}
	sort.Slice(mo.ScheduledMatches, func(i, j int) bool {
		return mo.ScheduledMatches[i].Weight > mo.ScheduledMatches[j].Weight
	})
	return matches
}

func (mo *MatchOptimizer) updateTeam(team string) {
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
	slog.Info("マッチオプティマイザを初期化します")
	mo.EndedMatches = []map[model.Role][]int{}
	mo.ScheduledMatches = []model.MatchWeight{}
	mo.mu.Unlock()
	return mo.append()
}

func (mo *MatchOptimizer) append() error {
	mo.mu.Lock()
	defer mo.mu.Unlock()

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

func (mo *MatchOptimizer) setMatchEnd(match map[model.Role][]string) {
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

func (mo *MatchOptimizer) setMatchWeight(match map[model.Role][]string, weight float64) {
	mo.mu.Lock()
	defer mo.mu.Unlock()
	idxMatch := util.TeamNameMatchToIdxMatch(mo.IdxTeamMap, match)

	for i, scheduledMatch := range mo.ScheduledMatches {
		if scheduledMatch.Equal(model.MatchWeight{RoleIdxs: idxMatch}) {
			mo.ScheduledMatches[i].Weight = weight
			slog.Info("スケジュールされたマッチの重みを設定しました", "weight", weight)
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
