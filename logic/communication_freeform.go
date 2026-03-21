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

func (s *CommunicationSession) runFreeform() {
	phaseStartPacket := model.Packet{
		Request: &model.R_TALK_PHASE_START,
	}
	if s.request == model.R_WHISPER {
		phaseStartPacket = model.Packet{
			Request: &model.R_WHISPER_PHASE_START,
		}
	}
	s.game.broadcastPacket(phaseStartPacket, s.agents)

	talkChannel := make(chan *TalkSubmission, len(s.agents)*10)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*s.talkSetting.Duration)*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, agent := range s.agents {
		wg.Add(1)
		go func(a *model.Agent) {
			defer wg.Done()
			s.listenForTalks(ctx, a, talkChannel)
		}(agent)
	}

	turnMap := make(map[model.Agent]int)
	for _, agent := range s.agents {
		turnMap[*agent] = 0
	}

	done := make(chan bool)

	go func() {
		for {
			select {
			case submission := <-talkChannel:
				mu.Lock()
				if s.validateSubmission(submission) {
					turn := turnMap[*submission.Agent]
					turnMap[*submission.Agent]++

					talk := s.buildTalk(submission.Agent, submission.Text, turn)
					s.appendTalk(talk)
					s.broadcastTalk(talk)
					s.logTalk(talk)
				}
				mu.Unlock()

			case <-ctx.Done():
				done <- true
				return
			}
		}
	}()

	<-done
	slog.Info("グループチャット方式の通信を終了します", "id", s.game.id, "totalTalks", s.idx)

	phaseEndPacket := model.Packet{
		Request: &model.R_TALK_PHASE_END,
	}
	if s.request == model.R_WHISPER {
		phaseEndPacket = model.Packet{
			Request: &model.R_WHISPER_PHASE_END,
		}
	}
	s.game.broadcastPacket(phaseEndPacket, s.agents)
}

func (s *CommunicationSession) validateSubmission(submission *TalkSubmission) bool {
	agent := submission.Agent
	text := submission.Text

	if text == model.T_OVER {
		return true
	}

	if !s.canAgentTalk(agent) {
		slog.Warn("残り発言回数または文字数が0のため拒否しました", "id", s.game.id, "agent", agent.String())
		return false
	}

	return true
}

func (s *CommunicationSession) broadcastTalk(talk model.Talk) {
	broadcastRequest := model.R_TALK_BROADCAST
	if s.request == model.R_WHISPER {
		broadcastRequest = model.R_WHISPER_BROADCAST
	}
	packet := model.Packet{
		Request: &broadcastRequest,
	}
	if s.request == model.R_TALK {
		packet.NewTalk = &talk
	} else {
		packet.NewWhisper = &talk
	}
	s.game.broadcastPacket(packet, s.agents)
}

func (s *CommunicationSession) listenForTalks(ctx context.Context, agent *model.Agent, talkChannel chan<- *TalkSubmission) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if !s.canAgentTalk(agent) {
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
				slog.Info("トークを受信しました", "id", s.game.id, "agent", agent.String(), "text", text)
			case <-ctx.Done():
				return
			}

			if text == model.T_OVER {
				slog.Info("エージェントがOverを送信しました", "id", s.game.id, "agent", agent.String())
				return
			}
		}
	}
}
