package logic

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/aiwolfdial/aiwolf-nlp-server/model"
)

type TalkSubmission struct {
	Agent *model.Agent
	Text  string
	Time  time.Time
}

func (g *Game) conductFreeformCommunication(request model.Request, agents []*model.Agent) {
	talkSetting, talkList := g.getTalkContext(request)
	if talkSetting == nil {
		return
	}

	remainCountMap, remainLengthMap, remainSkipMap := g.initRemainMaps(agents, talkSetting)
	defer g.clearRemainMaps()

	phaseStartPacket := model.Packet{
		Request: &model.R_TALK_PHASE_START,
	}
	if request == model.R_WHISPER {
		phaseStartPacket = model.Packet{
			Request: &model.R_WHISPER_PHASE_START,
		}
	}
	g.broadcastPacket(phaseStartPacket, agents)

	talkChannel := make(chan *TalkSubmission, len(agents)*10)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*talkSetting.Duration)*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, agent := range agents {
		wg.Add(1)
		go func(a *model.Agent) {
			defer wg.Done()
			g.listenForTalks(ctx, a, talkChannel, &remainCountMap, &remainLengthMap)
		}(agent)
	}

	turnMap := make(map[model.Agent]int)
	for _, agent := range agents {
		turnMap[*agent] = 0
	}

	idx := len(*talkList)
	done := make(chan bool)

	go func() {
		for {
			select {
			case submission := <-talkChannel:
				mu.Lock()
				if g.validateFreeformSubmission(submission, &remainCountMap, &remainLengthMap) {
					turn := turnMap[*submission.Agent]
					turnMap[*submission.Agent]++

					talk := g.processAndCreateTalk(submission.Agent, submission.Text, idx, turn, talkSetting, &remainCountMap, &remainLengthMap, &remainSkipMap)
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

	<-done

	phaseEndPacket := model.Packet{
		Request: &model.R_TALK_PHASE_END,
	}
	if request == model.R_WHISPER {
		phaseEndPacket = model.Packet{
			Request: &model.R_WHISPER_PHASE_END,
		}
	}
	g.broadcastPacket(phaseEndPacket, agents)
}

func (g *Game) validateFreeformSubmission(submission *TalkSubmission, remainCountMap *map[model.Agent]int, remainLengthMap *map[model.Agent]int) bool {
	agent := submission.Agent
	text := submission.Text

	if text == model.T_OVER {
		return true
	}

	if !canAgentTalk(agent, remainCountMap, remainLengthMap) {
		slog.Warn("残り発言回数または文字数が0のため拒否しました", "id", g.id, "agent", agent.String())
		return false
	}

	return true
}

func (g *Game) broadcastTalk(talk model.Talk, agents []*model.Agent, request model.Request) {
	broadcastRequest := model.R_TALK_BROADCAST
	if request == model.R_WHISPER {
		broadcastRequest = model.R_WHISPER_BROADCAST
	}
	packet := model.Packet{
		Request: &broadcastRequest,
	}
	if request == model.R_TALK {
		packet.NewTalk = &talk
	} else {
		packet.NewWhisper = &talk
	}
	g.broadcastPacket(packet, agents)
}

func (g *Game) listenForTalks(ctx context.Context, agent *model.Agent, talkChannel chan<- *TalkSubmission, remainCountMap *map[model.Agent]int, remainLengthMap *map[model.Agent]int) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if !canAgentTalk(agent, remainCountMap, remainLengthMap) {
				time.Sleep(100 * time.Millisecond)
				continue
			}

			text, err := agent.ReceiveWithTimeout(100 * time.Millisecond)
			if err != nil {
				continue
			}

			if text == "" {
				continue
			}

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

			if text == model.T_OVER {
				slog.Info("エージェントがOverを送信しました", "id", g.id, "agent", agent.String())
				return
			}
		}
	}
}
