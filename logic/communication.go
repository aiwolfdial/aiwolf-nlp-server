package logic

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/aiwolfdial/aiwolf-nlp-server/model"
	"github.com/aiwolfdial/aiwolf-nlp-server/util"
)

func (g *Game) doWhisper() {
	slog.Info("囁きフェーズを開始します", "id", g.id, "day", g.currentDay)
	g.conductCommunication(model.R_WHISPER)
}

func (g *Game) doTalk() {
	slog.Info("トークフェーズを開始します", "id", g.id, "day", g.currentDay)
	g.conductCommunication(model.R_TALK)
}

func (g *Game) conductCommunication(request model.Request) {
	var agents []*model.Agent
	var talkConfig *model.TalkConfig

	switch request {
	case model.R_TALK:
		agents = g.getAliveAgents()
		talkConfig = &g.config.Game.Talk
	case model.R_WHISPER:
		agents = g.getAliveWerewolves()
		talkConfig = &g.config.Game.Whisper
	default:
		return
	}
	if len(agents) < 2 {
		slog.Warn("エージェント数が2未満のため、通信を行いません", "id", g.id, "agentNum", len(agents))
		return
	}

	// モードによって処理を分岐
	if talkConfig.Mode == "freeform" {
		g.conductFreeformCommunication(request, agents)
	} else {
		g.conductTurnBasedCommunication(request, agents)
	}
}

func (g *Game) conductTurnBasedCommunication(request model.Request, agents []*model.Agent) {
	var talkSetting *model.TalkSetting
	var talkList *[]model.Talk

	switch request {
	case model.R_TALK:
		talkSetting = &g.setting.Talk.TalkSetting
		talkList = &g.getCurrentGameStatus().Talks
	case model.R_WHISPER:
		talkSetting = &g.setting.Whisper.TalkSetting
		talkList = &g.getCurrentGameStatus().Whispers
	default:
		return
	}

	remainCountMap := make(map[model.Agent]int)
	remainLengthMap := make(map[model.Agent]int)
	remainSkipMap := make(map[model.Agent]int)
	for _, agent := range agents {
		remainCountMap[*agent] = talkSetting.MaxCount.PerAgent
		if talkSetting.MaxLength.PerAgent != nil {
			remainLengthMap[*agent] = *talkSetting.MaxLength.PerAgent
		}
		remainSkipMap[*agent] = talkSetting.MaxSkip
	}
	g.getCurrentGameStatus().RemainCountMap = &remainCountMap
	g.getCurrentGameStatus().RemainLengthMap = &remainLengthMap
	g.getCurrentGameStatus().RemainSkipMap = &remainSkipMap

	rand.Shuffle(len(agents), func(i, j int) {
		agents[i], agents[j] = agents[j], agents[i]
	})

	idx := 0
	for i := range talkSetting.MaxCount.PerDay {
		cnt := false
		for _, agent := range agents {
			if remainCountMap[*agent] <= 0 {
				continue
			}
			if value, exists := remainLengthMap[*agent]; exists {
				if value <= 0 {
					continue
				}
			}
			remainCountMap[*agent]--
			text := g.getTalkWhisperText(agent, request)
			switch text {
			case model.T_SKIP:
				if remainSkipMap[*agent] <= 0 {
					text = model.T_OVER
					slog.Warn("スキップ回数が上限に達したため、発言をオーバーに置換しました", "id", g.id, "agent", agent.String())
				} else {
					remainSkipMap[*agent]--
					slog.Info("発言をスキップしました", "id", g.id, "agent", agent.String())
				}
			case model.T_FORCE_SKIP:
				text = model.T_SKIP
				slog.Warn("強制スキップが指定されたため、発言をスキップに置換しました", "id", g.id, "agent", agent.String())
			}
			if text != model.T_OVER && text != model.T_SKIP {
				remainSkipMap[*agent] = talkSetting.MaxSkip
				slog.Info("発言がオーバーもしくはスキップではないため、スキップ回数をリセットしました", "id", g.id, "agent", agent.String())
			}

			if text != model.T_OVER && text != model.T_SKIP && text != model.T_FORCE_SKIP {
				mention := ""
				commonText := ""
				mentionText := ""
				
				if talkSetting.MaxLength.PerAgent != nil || talkSetting.MaxLength.BaseLength != nil {
					baseLength := 0
					if talkSetting.MaxLength.BaseLength != nil {
						baseLength = *talkSetting.MaxLength.BaseLength
					}

					mentionIdx := -1
					if talkSetting.MaxLength.MentionLength != nil {
						for _, a := range g.agents {
							if a != agent {
								if strings.Contains(text, "@"+a.String()) {
									if mentionIdx == -1 {
										mention = "@" + a.String()
										mentionIdx = strings.Index(text, mention)
									}
									if strings.Index(text, mention) < mentionIdx {
										mention = "@" + a.String()
										mentionIdx = strings.Index(text, mention)
									}
								}
							}
						}
					}

					if mentionIdx != -1 {
						remainLength := baseLength
						if value, exists := remainLengthMap[*agent]; exists {
							remainLength += value
						}
						mentionBefore := text[:mentionIdx]
						mentionAfter := text[mentionIdx+len(mention):]

						mention = " " + mention + " "

						commonText = util.TrimLength(mentionBefore, remainLength, *talkSetting.MaxLength.CountInWord, *talkSetting.MaxLength.CountSpaces)
						cost := util.CountLength(mentionBefore, *talkSetting.MaxLength.CountInWord, *talkSetting.MaxLength.CountSpaces) - baseLength
						if cost > 0 {
							if _, exists := remainLengthMap[*agent]; exists {
								remainLengthMap[*agent] -= cost
							}
						}

						remainLength = *talkSetting.MaxLength.MentionLength
						if value, exists := remainLengthMap[*agent]; exists {
							remainLength += value
						}
						mentionText = util.TrimLength(mentionAfter, remainLength, *talkSetting.MaxLength.CountInWord, *talkSetting.MaxLength.CountSpaces)
						mentionCost := util.CountLength(mentionText, *talkSetting.MaxLength.CountInWord, *talkSetting.MaxLength.CountSpaces) - *talkSetting.MaxLength.MentionLength
						if mentionCost > 0 {
							if _, exists := remainLengthMap[*agent]; exists {
								remainLengthMap[*agent] -= mentionCost
							}
						}
					} else {
						remainLength := baseLength
						if value, exists := remainLengthMap[*agent]; exists {
							remainLength += value
						}
						commonText = util.TrimLength(text, remainLength, *talkSetting.MaxLength.CountInWord, *talkSetting.MaxLength.CountSpaces)
						cost := util.CountLength(text, *talkSetting.MaxLength.CountInWord, *talkSetting.MaxLength.CountSpaces) - baseLength
						if cost > 0 {
							if _, exists := remainLengthMap[*agent]; exists {
								remainLengthMap[*agent] -= cost
							}
						}
					}
				}
				if talkSetting.MaxLength.PerTalk != nil {
					commonLength := util.CountLength(commonText, *talkSetting.MaxLength.CountInWord, *talkSetting.MaxLength.CountSpaces)
					mentionLength := util.CountLength(mentionText, *talkSetting.MaxLength.CountInWord, *talkSetting.MaxLength.CountSpaces)
					totalLength := commonLength + mentionLength

					if totalLength > *talkSetting.MaxLength.PerTalk {
						if commonLength > *talkSetting.MaxLength.PerTalk{
							commonText = util.TrimLength(commonText, *talkSetting.MaxLength.PerTalk, *talkSetting.MaxLength.CountInWord, *talkSetting.MaxLength.CountSpaces)
							mention = ""
							mentionText = ""
						} else {
							mentionText = util.TrimLength(mentionText, *talkSetting.MaxLength.PerTalk - commonLength, *talkSetting.MaxLength.CountInWord, *talkSetting.MaxLength.CountSpaces)
						}
						slog.Warn("発言が最大文字数を超えたため、切り捨てました", "id", g.id, "agent", agent.String())
					}
				}
				text = commonText + mention + mentionText
				if utf8.RuneCountInString(text) == 0 {
					text = model.T_OVER
					slog.Warn("文字数が0のため、発言をオーバーに置換しました", "id", g.id, "agent", agent.String())
				}
			}

			talk := model.Talk{
				Idx:   idx,
				Day:   g.getCurrentGameStatus().Day,
				Turn:  i,
				Agent: *agent,
				Text:  text,
			}
			idx++
			*talkList = append(*talkList, talk)
			if text != model.T_OVER {
				cnt = true
			} else {
				remainCountMap[*agent] = 0
				slog.Info("発言がオーバーであるため、残り発言回数を0にしました", "id", g.id, "agent", agent.String())
			}
			if g.gameLogger != nil {
				if request == model.R_TALK {
					g.gameLogger.AppendLog(g.id, fmt.Sprintf("%d,talk,%d,%d,%d,%s", g.currentDay, talk.Idx, talk.Turn, talk.Agent.Idx, talk.Text))
				} else {
					g.gameLogger.AppendLog(g.id, fmt.Sprintf("%d,whisper,%d,%d,%d,%s", g.currentDay, talk.Idx, talk.Turn, talk.Agent.Idx, talk.Text))
				}
			}
			if g.realtimeBroadcaster != nil {
				if request == model.R_TALK {
					packet := g.getRealtimeBroadcastPacket()
					packet.Event = "トーク"
					packet.Message = &talk.Text
					packet.BubbleIdx = &agent.Idx
					g.realtimeBroadcaster.Broadcast(packet)
				} else {
					packet := g.getRealtimeBroadcastPacket()
					packet.Event = "囁き"
					packet.Message = &talk.Text
					packet.BubbleIdx = &agent.Idx
					g.realtimeBroadcaster.Broadcast(packet)
				}
			}
			if g.ttsBroadcaster != nil {
				g.ttsBroadcaster.BroadcastText(g.id, talk.Text, agent.Profile.VoiceID)
			}
			slog.Info("発言を受信しました", "id", g.id, "agent", agent.String(), "text", text, "count", remainCountMap[*agent], "length", remainLengthMap[*agent], "skip", remainSkipMap[*agent])
		}
		if !cnt {
			break
		}
	}

	g.getCurrentGameStatus().RemainCountMap = nil
	g.getCurrentGameStatus().RemainLengthMap = nil
	g.getCurrentGameStatus().RemainSkipMap = nil
}

func (g *Game) getTalkWhisperText(agent *model.Agent, request model.Request) string {
	text, err := g.requestToAgent(agent, request)
	if text == model.T_FORCE_SKIP {
		text = model.T_SKIP
		slog.Warn("クライアントから強制スキップが指定されたため、発言をスキップに置換しました", "id", g.id, "agent", agent.String())
	}
	if err != nil {
		text = model.T_FORCE_SKIP
		slog.Warn("リクエストの送受信に失敗したため、発言をスキップに置換しました", "id", g.id, "agent", agent.String())
	}
	return text
}

func (g *Game) conductFreeformCommunication(request model.Request, agents []*model.Agent) {
	var talkSetting *model.TalkSetting
	var talkConfig *model.TalkConfig

	switch request {
	case model.R_TALK:
		talkSetting = &g.setting.Talk.TalkSetting
		talkConfig = &g.config.Game.Talk
	case model.R_WHISPER:
		talkSetting = &g.setting.Whisper.TalkSetting
		talkConfig = &g.config.Game.Whisper
	default:
		return
	}

	slog.Info("グループチャット方式の通信を開始します", "id", g.id, "request", request.Type, "timeLimit", talkConfig.TimeLimit)

	// 残り回数マップを初期化
	remainCountMap := make(map[model.Agent]int)
	remainLengthMap := make(map[model.Agent]int)
	remainSkipMap := make(map[model.Agent]int)
	for _, agent := range agents {
		remainCountMap[*agent] = talkSetting.MaxCount.PerAgent
		if talkSetting.MaxLength.PerAgent != nil {
			remainLengthMap[*agent] = *talkSetting.MaxLength.PerAgent
		}
		remainSkipMap[*agent] = talkSetting.MaxSkip
	}
	g.getCurrentGameStatus().RemainCountMap = &remainCountMap
	g.getCurrentGameStatus().RemainLengthMap = &remainLengthMap
	g.getCurrentGameStatus().RemainSkipMap = &remainSkipMap

	// エージェントをフェーズ中状態に設定
	for _, agent := range agents {
		if request == model.R_TALK {
			agent.InTalkPhase = true
		} else {
			agent.InWhisperPhase = true
		}
	}

	// フェーズ開始を通知
	phaseStartRequest := model.R_TALK_PHASE_START
	if request == model.R_WHISPER {
		phaseStartRequest = model.R_WHISPER_PHASE_START
	}
	g.broadcastPhaseStart(phaseStartRequest, agents)

	// 並行処理でトーク受付
	var talkList *[]model.Talk
	switch request {
	case model.R_TALK:
		talkList = &g.getCurrentGameStatus().Talks
	case model.R_WHISPER:
		talkList = &g.getCurrentGameStatus().Whispers
	}

	talkChannel := make(chan *TalkSubmission, len(agents)*10)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(talkConfig.TimeLimit)*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	var mu sync.Mutex

	// 各エージェントからのトーク受信ゴルーチンを起動
	for _, agent := range agents {
		wg.Add(1)
		go func(a *model.Agent) {
			defer wg.Done()
			g.listenForTalks(ctx, a, request, talkChannel, &remainCountMap)
		}(agent)
	}

	// トーク受付と配信のメインループ
	idx := len(*talkList)
	done := make(chan bool)

	go func() {
		for {
			select {
			case submission := <-talkChannel:
				mu.Lock()
				if g.validateTalkSubmission(submission, &remainCountMap, &remainLengthMap, talkSetting) {
					talk := g.processTalkSubmission(submission, request, idx, talkSetting, &remainCountMap, &remainLengthMap, &remainSkipMap)
					idx++
					*talkList = append(*talkList, talk)
					g.broadcastTalk(talk, agents, request)
					g.logTalk(talk, request)
				}
				mu.Unlock()

			case <-ctx.Done():
				done <- true
				return
			}
		}
	}()

	// タイムアウトまたは全員Over待機
	<-done
	slog.Info("グループチャット方式の通信を終了します", "id", g.id, "totalTalks", idx)

	// フェーズ終了を通知
	for _, agent := range agents {
		agent.InTalkPhase = false
		agent.InWhisperPhase = false
	}
	phaseEndRequest := model.R_TALK_PHASE_END
	if request == model.R_WHISPER {
		phaseEndRequest = model.R_WHISPER_PHASE_END
	}
	g.broadcastPhaseEnd(phaseEndRequest, agents)

	g.getCurrentGameStatus().RemainCountMap = nil
	g.getCurrentGameStatus().RemainLengthMap = nil
	g.getCurrentGameStatus().RemainSkipMap = nil
}

func (g *Game) broadcastPhaseStart(request model.Request, agents []*model.Agent) {
	for _, agent := range agents {
		info := g.buildInfo(agent)
		packet := model.Packet{
			Request: &request,
			Info:    &info,
		}
		err := agent.SendPacketNoResponse(packet)
		if err != nil {
			slog.Error("フェーズ開始通知の送信に失敗しました", "id", g.id, "agent", agent.String(), "error", err)
		}
	}
}

func (g *Game) broadcastPhaseEnd(request model.Request, agents []*model.Agent) {
	for _, agent := range agents {
		packet := model.Packet{
			Request: &request,
		}
		err := agent.SendPacketNoResponse(packet)
		if err != nil {
			slog.Error("フェーズ終了通知の送信に失敗しました", "id", g.id, "agent", agent.String(), "error", err)
		}
	}
}

func (g *Game) broadcastTalk(talk model.Talk, agents []*model.Agent, request model.Request) {
	broadcastRequest := model.R_TALK_BROADCAST
	if request == model.R_WHISPER {
		broadcastRequest = model.R_WHISPER_BROADCAST
	}

	for _, agent := range agents {
		packet := model.Packet{
			Request: &broadcastRequest,
		}

		if request == model.R_TALK {
			packet.NewTalk = &talk
		} else {
			packet.NewWhisper = &talk
		}

		err := agent.SendPacketNoResponse(packet)
		if err != nil {
			slog.Error("トーク配信の送信に失敗しました", "id", g.id, "agent", agent.String(), "error", err)
		}
	}
}

type TalkSubmission struct {
	Agent *model.Agent
	Text  string
	Time  time.Time
}

func (g *Game) listenForTalks(ctx context.Context, agent *model.Agent, request model.Request, talkChannel chan<- *TalkSubmission, remainCountMap *map[model.Agent]int) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// エージェントが既にOverを送っている、または残り回数が0の場合はスキップ
			if (*remainCountMap)[*agent] <= 0 {
				time.Sleep(100 * time.Millisecond)
				continue
			}

			// タイムアウト付きで受信（ノンブロッキング）
			text, err := agent.ReceiveWithTimeout(100 * time.Millisecond)
			if err != nil {
				// タイムアウトまたはエラー（まだトークが来ていない）
				continue
			}

			// 空文字列は無視
			if text == "" {
				continue
			}

			// トーク送信
			submission := &TalkSubmission{
				Agent: agent,
				Text:  text,
				Time:  time.Now(),
			}

			select {
			case talkChannel <- submission:
				slog.Info("トークを受信しました", "id", g.id, "agent", agent.String(), "text", text)
			case <-ctx.Done():
				return
			}

			// Overを送った場合、このエージェントは終了
			if text == model.T_OVER {
				slog.Info("エージェントがOverを送信しました", "id", g.id, "agent", agent.String())
				return
			}
		}
	}
}

func (g *Game) validateTalkSubmission(submission *TalkSubmission, remainCountMap *map[model.Agent]int, remainLengthMap *map[model.Agent]int, talkSetting *model.TalkSetting) bool {
	agent := submission.Agent
	text := submission.Text

	// 残り回数チェック
	remainCount := (*remainCountMap)[*agent]
	if remainCount <= 0 && text != model.T_OVER {
		slog.Warn("残り回数が0のため発言拒否", "id", g.id, "agent", agent.String())
		return false
	}

	// 文字数チェック
	if talkSetting.MaxLength.PerTalk != nil && *talkSetting.MaxLength.PerTalk > 0 {
		textLength := util.CountLength(text, *talkSetting.MaxLength.CountInWord, *talkSetting.MaxLength.CountSpaces)
		if textLength > *talkSetting.MaxLength.PerTalk && text != model.T_OVER && text != model.T_SKIP {
			slog.Warn("文字数超過のため発言拒否", "id", g.id, "agent", agent.String(), "length", textLength, "max", *talkSetting.MaxLength.PerTalk)
			return false
		}
	}

	return true
}

func (g *Game) processTalkSubmission(submission *TalkSubmission, request model.Request, idx int, talkSetting *model.TalkSetting, remainCountMap *map[model.Agent]int, remainLengthMap *map[model.Agent]int, remainSkipMap *map[model.Agent]int) model.Talk {
	agent := submission.Agent
	text := submission.Text

	// 残り回数を減らす
	(*remainCountMap)[*agent]--

	// Skip/Overの処理
	switch text {
	case model.T_SKIP:
		if (*remainSkipMap)[*agent] <= 0 {
			text = model.T_OVER
			slog.Warn("スキップ回数が上限に達したため、発言をオーバーに置換しました", "id", g.id, "agent", agent.String())
		} else {
			(*remainSkipMap)[*agent]--
			slog.Info("発言をスキップしました", "id", g.id, "agent", agent.String())
		}
	case model.T_FORCE_SKIP:
		text = model.T_SKIP
		slog.Warn("強制スキップが指定されたため、発言をスキップに置換しました", "id", g.id, "agent", agent.String())
	}

	// Skip/Over以外の場合、スキップ回数をリセット
	if text != model.T_OVER && text != model.T_SKIP && text != model.T_FORCE_SKIP {
		(*remainSkipMap)[*agent] = talkSetting.MaxSkip
	}

	// Overの場合、残り回数を0にする
	if text == model.T_OVER {
		(*remainCountMap)[*agent] = 0
		slog.Info("発言がオーバーであるため、残り発言回数を0にしました", "id", g.id, "agent", agent.String())
	}

	// 文字数制限の処理（既存のロジックを簡略化）
	if text != model.T_OVER && text != model.T_SKIP && text != model.T_FORCE_SKIP {
		if talkSetting.MaxLength.PerTalk != nil && *talkSetting.MaxLength.PerTalk > 0 {
			text = util.TrimLength(text, *talkSetting.MaxLength.PerTalk, *talkSetting.MaxLength.CountInWord, *talkSetting.MaxLength.CountSpaces)
			if utf8.RuneCountInString(text) == 0 {
				text = model.T_OVER
				slog.Warn("文字数が0のため、発言をオーバーに置換しました", "id", g.id, "agent", agent.String())
			}
		}
	}

	talk := model.Talk{
		Idx:   idx,
		Day:   g.getCurrentGameStatus().Day,
		Turn:  0, // グループチャット方式ではターン概念なし
		Agent: *agent,
		Text:  text,
	}

	return talk
}

func (g *Game) logTalk(talk model.Talk, request model.Request) {
	if g.gameLogger != nil {
		if request == model.R_TALK {
			g.gameLogger.AppendLog(g.id, fmt.Sprintf("%d,talk,%d,%d,%d,%s", g.currentDay, talk.Idx, talk.Turn, talk.Agent.Idx, talk.Text))
		} else {
			g.gameLogger.AppendLog(g.id, fmt.Sprintf("%d,whisper,%d,%d,%d,%s", g.currentDay, talk.Idx, talk.Turn, talk.Agent.Idx, talk.Text))
		}
	}
	if g.realtimeBroadcaster != nil {
		packet := g.getRealtimeBroadcastPacket()
		if request == model.R_TALK {
			packet.Event = "トーク"
		} else {
			packet.Event = "囁き"
		}
		packet.Message = &talk.Text
		packet.BubbleIdx = &talk.Agent.Idx
		g.realtimeBroadcaster.Broadcast(packet)
	}
	if g.ttsBroadcaster != nil && talk.Agent.Profile != nil {
		g.ttsBroadcaster.BroadcastText(g.id, talk.Text, talk.Agent.Profile.VoiceID)
	}
}
