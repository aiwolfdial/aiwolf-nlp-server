package logic

import (
	"fmt"
	"log/slog"

	"github.com/aiwolfdial/aiwolf-nlp-server/model"
	"github.com/aiwolfdial/aiwolf-nlp-server/service"
	"github.com/aiwolfdial/aiwolf-nlp-server/util"
	"github.com/oklog/ulid/v2"
)

type Game struct {
	id                           string
	agents                       []*model.Agent
	winSide                      model.Team
	isFinished                   bool
	config                       *model.Config
	setting                      *model.Setting
	currentDay                   int
	isDaytime                    bool
	gameStatuses                 map[int]*model.GameStatus
	lastTalkIdxMap               map[*model.Agent]int
	lastWhisperIdxMap            map[*model.Agent]int
	jsonLogger                   *service.JSONLogger
	gameLogger                   *service.GameLogger
	realtimeBroadcaster          *service.RealtimeBroadcaster
	ttsBroadcaster               *service.TTSBroadcaster
	realtimeBroadcasterPacketIdx int
}

func NewGame(config *model.Config, settings *model.Setting, conns []model.Connection) *Game {
	id := ulid.Make().String()
	var agents []*model.Agent
	if config.CustomProfile.Enable {
		if config.CustomProfile.DynamicProfile.Enable {
			profiles, err := util.GenerateProfiles(config.CustomProfile.DynamicProfile, config.CustomProfile.ProfileEncoding, config.Game.AgentCount)
			if err != nil {
				slog.Error("プロフィールの生成に失敗したため、カスタムプロフィールを使用します", "error", err)
				agents = util.CreateAgentsWithProfiles(conns, settings.RoleNumMap, config.CustomProfile.Profiles, config.CustomProfile.ProfileEncoding)
			} else {
				agents = util.CreateAgentsWithProfiles(conns, settings.RoleNumMap, profiles, config.CustomProfile.ProfileEncoding)
			}
		} else {
			agents = util.CreateAgentsWithProfiles(conns, settings.RoleNumMap, config.CustomProfile.Profiles, config.CustomProfile.ProfileEncoding)
		}
	} else {
		agents = util.CreateAgents(conns, settings.RoleNumMap)
	}
	gameStatus := model.NewInitializeGameStatus(agents)
	gameStatuses := make(map[int]*model.GameStatus)
	gameStatuses[0] = &gameStatus
	slog.Info("ゲームを作成しました", "id", id)
	return &Game{
		id:                id,
		agents:            agents,
		winSide:           model.T_NONE,
		isFinished:        false,
		config:            config,
		setting:           settings,
		currentDay:        0,
		isDaytime:         true,
		gameStatuses:      gameStatuses,
		lastTalkIdxMap:    make(map[*model.Agent]int),
		lastWhisperIdxMap: make(map[*model.Agent]int),
	}
}

func NewGameWithRole(config *model.Config, settings *model.Setting, roleMapConns map[model.Role][]model.Connection) *Game {
	id := ulid.Make().String()
	var agents []*model.Agent
	if config.CustomProfile.Enable {
		if config.CustomProfile.DynamicProfile.Enable {
			profiles, err := util.GenerateProfiles(config.CustomProfile.DynamicProfile, config.CustomProfile.ProfileEncoding, config.Game.AgentCount)
			if err != nil {
				slog.Error("プロフィールの生成に失敗したため、カスタムプロフィールを使用します", "error", err)
				agents = util.CreateAgentsWithRoleAndProfile(roleMapConns, config.CustomProfile.Profiles, config.CustomProfile.ProfileEncoding)
			} else {
				agents = util.CreateAgentsWithRoleAndProfile(roleMapConns, profiles, config.CustomProfile.ProfileEncoding)
			}
		} else {
			agents = util.CreateAgentsWithRoleAndProfile(roleMapConns, config.CustomProfile.Profiles, config.CustomProfile.ProfileEncoding)
		}
	} else {
		agents = util.CreateAgentsWithRole(roleMapConns)
	}
	gameStatus := model.NewInitializeGameStatus(agents)
	gameStatuses := make(map[int]*model.GameStatus)
	gameStatuses[0] = &gameStatus
	slog.Info("ゲームを作成しました", "id", id)
	return &Game{
		id:                id,
		agents:            agents,
		winSide:           model.T_NONE,
		isFinished:        false,
		config:            config,
		setting:           settings,
		currentDay:        0,
		isDaytime:         true,
		gameStatuses:      gameStatuses,
		lastTalkIdxMap:    make(map[*model.Agent]int),
		lastWhisperIdxMap: make(map[*model.Agent]int),
	}
}

func (g *Game) Start() model.Team {
	slog.Info("ゲームを開始します", "id", g.id)
	if g.jsonLogger != nil {
		g.jsonLogger.TrackStartGame(g.id, g.agents)
	}
	if g.gameLogger != nil {
		g.gameLogger.TrackStartGame(g.id, g.agents)
	}
	if g.realtimeBroadcaster != nil {
		g.realtimeBroadcaster.TrackStartGame(g.id, g.agents)
	}
	if g.ttsBroadcaster != nil {
		g.ttsBroadcaster.CreateStream(g.id)
	}
	if g.realtimeBroadcaster != nil {
		packet := g.getRealtimeBroadcastPacket()
		packet.Event = "開始"
		message := "ゲームが開始されました"
		packet.Message = &message
		g.realtimeBroadcaster.Broadcast(packet)
	}
	if g.ttsBroadcaster != nil {
		g.ttsBroadcaster.BroadcastText(g.id, "ゲームが開始されました", 23)
	}
	g.requestToEveryone(model.R_INITIALIZE)
	for {
		g.progressDay()
		g.progressNight()
		gameStatus := g.getCurrentGameStatus().NextDay()
		g.gameStatuses[g.currentDay+1] = &gameStatus
		g.currentDay++
		slog.Info("日付が進みました", "id", g.id, "day", g.currentDay)
		if g.config.Game.MaxDay >= 0 && g.currentDay >= g.config.Game.MaxDay+1 {
			slog.Info("最大日数に達したため、ゲームを終了します", "id", g.id, "day", g.currentDay)
			break
		}
		if g.shouldFinish() {
			break
		}
	}
	g.requestToEveryone(model.R_FINISH)
	if g.gameLogger != nil {
		for _, agent := range g.agents {
			g.gameLogger.AppendLog(g.id, fmt.Sprintf("%d,status,%d,%s,%s,%s,%s", g.currentDay, agent.Idx, agent.Role.Name, g.getCurrentGameStatus().StatusMap[*agent].String(), agent.OriginalName, agent.GameName))
		}
		villagers, werewolves := util.CountAliveTeams(g.getCurrentGameStatus().StatusMap)
		g.gameLogger.AppendLog(g.id, fmt.Sprintf("%d,result,%d,%d,%s", g.currentDay, villagers, werewolves, g.winSide))
	}
	if g.realtimeBroadcaster != nil {
		packet := g.getRealtimeBroadcastPacket()
		packet.Event = "終了"
		message := string(g.winSide)
		packet.Message = &message
		g.realtimeBroadcaster.Broadcast(packet)
	}
	if g.ttsBroadcaster != nil {
		g.ttsBroadcaster.BroadcastText(g.id, "ゲームが終了しました", 23)
	}
	g.closeAllAgents()
	if g.jsonLogger != nil {
		g.jsonLogger.TrackEndGame(g.id, g.winSide)
	}
	if g.gameLogger != nil {
		g.gameLogger.TrackEndGame(g.id)
	}
	if g.realtimeBroadcaster != nil {
		g.realtimeBroadcaster.TrackEndGame(g.id)
	}
	slog.Info("ゲームが終了しました", "id", g.id, "winSide", g.winSide)
	g.isFinished = true
	return g.winSide
}

func (g *Game) shouldFinish() bool {
	if util.CalcHasErrorAgents(g.agents) >= int(float64(len(g.agents))*g.config.Server.MaxContinueErrorRatio) {
		slog.Warn("エラーが多発したため、ゲームを終了します", "id", g.id)
		return true
	}
	g.winSide = util.CalcWinSideTeam(g.getCurrentGameStatus().StatusMap)
	if g.winSide != model.T_NONE {
		slog.Info("勝利チームが決定したため、ゲームを終了します", "id", g.id)
		return true
	}
	return false
}

func (g *Game) progressDay() {
	slog.Info("昼セクションを開始します", "id", g.id, "day", g.currentDay)
	g.isDaytime = true
	g.requestToEveryone(model.R_DAILY_INITIALIZE)
	if g.gameLogger != nil {
		for _, agent := range g.agents {
			g.gameLogger.AppendLog(g.id, fmt.Sprintf("%d,status,%d,%s,%s,%s,%s", g.currentDay, agent.Idx, agent.Role.Name, g.getCurrentGameStatus().StatusMap[*agent].String(), agent.OriginalName, agent.GameName))
		}
	}

	for _, phase := range g.config.Logic.DayPhases {
		if phase.OnlyDay != nil && *phase.OnlyDay != g.currentDay {
			slog.Info("実行対象の日ではないため、フェーズをスキップします", "id", g.id, "day", g.currentDay, "phase", phase.Name)
			continue
		}
		if phase.ExceptDay != nil && *phase.ExceptDay == g.currentDay {
			slog.Info("除外対象の日であるため、フェーズをスキップします", "id", g.id, "day", g.currentDay, "phase", phase.Name)
			continue
		}
		slog.Info("昼セクションのフェーズを開始します", "id", g.id, "day", g.currentDay, "phase", phase.Name)
		g.executePhase(phase.Actions)
		if g.shouldFinish() {
			return
		}
	}

	slog.Info("昼セクションを終了します", "id", g.id, "day", g.currentDay)
}

func (g *Game) progressNight() {
	slog.Info("夜セクションを開始します", "id", g.id, "day", g.currentDay)
	g.isDaytime = false
	g.requestToEveryone(model.R_DAILY_FINISH)

	for _, phase := range g.config.Logic.NightPhases {
		if phase.OnlyDay != nil && *phase.OnlyDay != g.currentDay {
			slog.Info("実行対象の日ではないため、フェーズをスキップします", "id", g.id, "day", g.currentDay, "phase", phase.Name)
			continue
		}
		if phase.ExceptDay != nil && *phase.ExceptDay == g.currentDay {
			slog.Info("除外対象の日であるため、フェーズをスキップします", "id", g.id, "day", g.currentDay, "phase", phase.Name)
			continue
		}
		slog.Info("夜セクションのフェーズを実行します", "id", g.id, "day", g.currentDay, "phase", phase.Name)
		g.executePhase(phase.Actions)
		if g.shouldFinish() {
			return
		}
	}

	slog.Info("夜セクションを終了します", "id", g.id, "day", g.currentDay)
}

func (g *Game) executePhase(actions []string) {
	for _, action := range actions {
		switch action {
		case "talk":
			g.doTalk()
		case "whisper":
			g.doWhisper()
		case "execution":
			g.doExecution()
		case "divine":
			g.doDivine()
		case "guard":
			g.doGuard()
		case "attack":
			g.doAttack()
		default:
			slog.Warn("不明なアクションです", "action", action)
		}
	}
}

func (g *Game) GetID() string {
	return g.id
}

func (g *Game) SetJSONLogger(logger *service.JSONLogger) {
	g.jsonLogger = logger
}

func (g *Game) SetGameLogger(logger *service.GameLogger) {
	g.gameLogger = logger
}

func (g *Game) SetRealtimeBroadcaster(broadcaster *service.RealtimeBroadcaster) {
	g.realtimeBroadcaster = broadcaster
}

func (g *Game) SetTTSBroadcaster(broadcaster *service.TTSBroadcaster) {
	g.ttsBroadcaster = broadcaster
}
