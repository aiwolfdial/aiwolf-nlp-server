package logic

import (
	"log/slog"
	"sync/atomic"

	"github.com/aiwolfdial/aiwolf-nlp-server/model"
	"github.com/aiwolfdial/aiwolf-nlp-server/observer"
	"github.com/aiwolfdial/aiwolf-nlp-server/util"
	"github.com/oklog/ulid/v2"
)

type Game struct {
	id                string
	agents            []*model.Agent
	winSide           model.Team
	abortedByError    bool
	isFinished        atomic.Bool
	ruleset           model.RulesetView
	setting           model.SettingView
	currentDay        int
	isDaytime         bool
	gameStatuses      map[int]*model.GameStatus
	lastTalkIdxMap    map[*model.Agent]int
	lastWhisperIdxMap map[*model.Agent]int
	obs               observer.GameObserver
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
		ruleset:           model.NewRulesetView(*config),
		setting:           model.NewSettingView(settings),
		currentDay:        0,
		isDaytime:         true,
		gameStatuses:      gameStatuses,
		lastTalkIdxMap:    make(map[*model.Agent]int),
		lastWhisperIdxMap: make(map[*model.Agent]int),
		obs:               observer.NoopObserver{},
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
		ruleset:           model.NewRulesetView(*config),
		setting:           model.NewSettingView(settings),
		currentDay:        0,
		isDaytime:         true,
		gameStatuses:      gameStatuses,
		lastTalkIdxMap:    make(map[*model.Agent]int),
		lastWhisperIdxMap: make(map[*model.Agent]int),
		obs:               observer.NoopObserver{},
	}
}

func (g *Game) Start() model.Team {
	slog.Info("ゲームを開始します", "id", g.id)
	g.obs.OnGameStart(g.id, model.ViewsOf(g.agents), g.gameState())
	g.requestToEveryone(model.R_INITIALIZE)
	for {
		g.progressDay()
		g.progressNight()
		gameStatus := g.getCurrentGameStatus().NextDay()
		g.gameStatuses[g.currentDay+1] = &gameStatus
		g.currentDay++
		slog.Info("日付が進みました", "id", g.id, "day", g.currentDay)
		if g.ruleset.MaxDay() >= 0 && g.currentDay >= g.ruleset.MaxDay()+1 {
			slog.Info("最大日数に達したため、ゲームを終了します", "id", g.id, "day", g.currentDay)
			break
		}
		if g.shouldFinish() {
			break
		}
	}
	g.requestToEveryone(model.R_FINISH)
	g.obs.OnDayStatus(g.id, g.currentDay, g.agentStatuses())
	villagers, werewolves := util.CountAliveTeams(g.getCurrentGameStatus().StatusMap)
	g.obs.OnResult(g.id, g.currentDay, villagers, werewolves, g.winSide)
	g.closeAllAgents()
	g.obs.OnGameEnd(g.id, g.winSide, g.gameState())
	slog.Info("ゲームが終了しました", "id", g.id, "winSide", g.winSide)
	g.isFinished.Store(true)
	return g.winSide
}

func (g *Game) shouldFinish() bool {
	if util.CalcHasErrorAgents(g.agents) >= int(float64(len(g.agents))*g.ruleset.MaxContinueErrorRatio()) {
		slog.Warn("エラーが多発したため、ゲームを終了します", "id", g.id)
		g.abortedByError = true
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
	g.obs.OnDayStatus(g.id, g.currentDay, g.agentStatuses())

	for _, phase := range g.ruleset.DayPhases() {
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

	for _, phase := range g.ruleset.NightPhases() {
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

// nilのときはNoopに差し替え、Gameが常に非nilのobserverを保つ。
func (g *Game) SetObserver(o observer.GameObserver) {
	if o == nil {
		o = observer.NoopObserver{}
	}
	g.obs = o
}
