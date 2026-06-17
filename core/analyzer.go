package core

import (
	"bufio"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/aiwolfdial/aiwolf-nlp-server/model"
	"github.com/aiwolfdial/aiwolf-nlp-server/store"
)

func Analyzer(config model.Config) {
	data, err := os.ReadFile(config.Matching.OutputPath)
	if err != nil {
		slog.Warn("マッチオプティマイザの読み込みに失敗しました", "error", err)
		return
	}
	var mo MatchOptimizer
	if err := json.Unmarshal(data, &mo); err != nil {
		slog.Error("マッチオプティマイザのパースに失敗しました", "error", err)
		return
	}

	slog.Info("マッチオプティマイザの統計データを分析します")
	for idx, team := range mo.IdxTeamMap {
		slog.Info("登録済みチームを取得しました", "idx", idx, "team", team)

		scheduledRoles := make(map[model.Role]int)
		for _, match := range mo.ScheduledMatches {
			for role, idxs := range match.RoleIdxs {
				for _, i := range idxs {
					if idx == i {
						scheduledRoles[role]++
					}
				}
			}
		}
		sum := 0
		for _, count := range scheduledRoles {
			sum += count
		}
		slog.Info("スケジュールされた役職を取得しました", "idx", idx, "roles", scheduledRoles, "sum", sum)

		endedRoles := make(map[model.Role]int)
		for _, match := range mo.EndedMatches {
			for role, idxs := range match {
				for _, i := range idxs {
					if idx == i {
						endedRoles[role]++
					}
				}
			}
		}
		sum = 0
		for _, count := range endedRoles {
			sum += count
		}
		slog.Info("終了した役職を取得しました", "idx", idx, "roles", endedRoles, "sum", sum)
	}

	if config.GameLogger.Enable {
		slog.Info("ログサービスの統計データを分析します")

		filePaths, err := filepath.Glob(filepath.Join(config.GameLogger.OutputDir, "*.log"))
		if err != nil {
			slog.Warn("ファイルの取得に失敗しました", "error", err)
		}

		counts := make(map[string]map[model.Role]*Count)

		for _, filePath := range filePaths {
			file, err := os.Open(filePath)
			if err != nil {
				slog.Warn("ファイルの読み込みに失敗しました", "error", err)
			}
			defer file.Close()

			teamsRole := make(map[string]model.Role)
			errorTeams := []string{}
			var winSide *model.Team

			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				line := scanner.Text()
				values := strings.Split(line, ",")
				if (len(values) == 6 || len(values) == 7) && values[1] == "status" {
					if values[0] == "0" {
						team := strings.TrimRight(values[5], "1234567890")
						role := model.RoleFromString(values[3])
						teamsRole[team] = role
					} else {
						team := strings.TrimRight(values[5], "1234567890")
						status := values[4]
						if status == "" {
							errorTeams = append(errorTeams, team)
						}
					}
				}
				if len(values) == 5 && values[1] == "result" {
					side := model.TeamFromString(values[4])
					winSide = &side
				}
			}

			if len(teamsRole) == 0 {
				slog.Warn("役職が取得できませんでした", "file", filePath)
				continue
			}

			if winSide == nil {
				slog.Warn("結果が取得できませんでした", "file", filePath)
				continue
			}

			for team, role := range teamsRole {
				if _, exists := counts[team]; !exists {
					counts[team] = make(map[model.Role]*Count)
				}
				if _, exists := counts[team][role]; !exists {
					counts[team][role] = &Count{}
				}

				if slices.Contains(errorTeams, team) {
					counts[team][role].Error++
				}

				if *winSide == model.T_NONE {
					counts[team][role].None++
				} else {
					counts[team][role].Succeed++

					if *winSide == model.T_VILLAGER && role != model.R_WEREWOLF && role != model.R_POSSESSED {
						counts[team][role].Win++
					} else if *winSide == model.T_WEREWOLF && (role == model.R_WEREWOLF || role == model.R_POSSESSED) {
						counts[team][role].Win++
					} else {
						counts[team][role].Lose++
					}
				}
			}
		}

		for team, roles := range counts {
			global := &Count{}
			for role, count := range roles {
				slog.Info("統計データを取得しました", "team", team, "role", role, "win", count.Win, "lose", count.Lose, "error", count.Error, "none", count.None, "succeed", count.Succeed)
				global.Win += count.Win
				global.Lose += count.Lose
				global.Error += count.Error
				global.None += count.None
				global.Succeed += count.Succeed
			}
			slog.Info("統計データを取得しました", "team", team, "win", global.Win, "lose", global.Lose, "error", global.Error, "none", global.None, "succeed", global.Succeed)
		}
	}
}

func Reduction(src model.Config, dst model.Config) {
	Analyzer(dst)

	srcData, err := os.ReadFile(src.Matching.OutputPath)
	if err != nil {
		slog.Warn("マッチオプティマイザの読み込みに失敗しました", "error", err)
		return
	}
	var srcMo MatchOptimizer
	if err := json.Unmarshal(srcData, &srcMo); err != nil {
		slog.Error("マッチオプティマイザのパースに失敗しました", "error", err)
		return
	}

	dstData, err := os.ReadFile(dst.Matching.OutputPath)
	if err != nil {
		slog.Warn("マッチオプティマイザの読み込みに失敗しました", "error", err)
		return
	}
	var dstMo MatchOptimizer
	if err := json.Unmarshal(dstData, &dstMo); err != nil {
		slog.Error("マッチオプティマイザのパースに失敗しました", "error", err)
		return
	}
	dstMo.store = store.NewFileMatchOptimizerStore(dst.Matching.OutputPath)

	for _, srcMatch := range srcMo.EndedMatches {
		for i, dstMatch := range dstMo.ScheduledMatches {
			if dstMatch.Equal(model.MatchWeight{RoleIdxs: srcMatch}) {
				dstMo.ScheduledMatches = append(dstMo.ScheduledMatches[:i], dstMo.ScheduledMatches[i+1:]...)
				slog.Info("重複したマッチを削除しました", "match", srcMatch)
			}
		}
	}

	dstMo.save()
	Analyzer(dst)
}

type Count struct {
	Succeed int
	None    int
	Win     int
	Lose    int
	Error   int
}
