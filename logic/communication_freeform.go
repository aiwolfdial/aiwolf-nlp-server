package logic

import (
	"context"
	"log/slog"
	"strings"
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
	s.sendStart()

	talkChannel := make(chan *TalkSubmission, len(s.agents)*10)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*s.talkSetting.Duration)*time.Millisecond)
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

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case submission := <-talkChannel:
				mu.Lock()
				if s.validateSubmission(submission) {
					turn := turnMap[*submission.Agent]
					turnMap[*submission.Agent]++

					talk := s.buildTalk(submission.Agent, submission.Text, turn)
					s.appendTalk(talk)
					s.sendTalk(talk)
					s.logTalk(talk)
				}
				if s.allAgentsDone() {
					slog.Info("全エージェントの発言が終了したため、早期終了します", "id", s.game.id)
					cancel()
					mu.Unlock()
					return
				}
				mu.Unlock()

			case <-ctx.Done():
				return
			}
		}
	}()

	wg.Wait()
	slog.Info("グループチャット方式の通信を終了します", "id", s.game.id, "totalTalks", s.idx)

	// フェーズ終了後、チャネルに残った未処理メッセージを破棄
	for _, agent := range s.agents {
		agent.DrainMessages()
	}

	s.sendEnd()
}

func (s *CommunicationSession) sendStart() {
	request := model.R_TALK_PHASE_START
	if s.request == model.R_WHISPER {
		request = model.R_WHISPER_PHASE_START
	}
	s.send(request, nil)
}

func (s *CommunicationSession) sendEnd() {
	request := model.R_TALK_PHASE_END
	if s.request == model.R_WHISPER {
		request = model.R_WHISPER_PHASE_END
	}
	s.send(request, nil)
}

func (s *CommunicationSession) sendTalk(talk model.Talk) {
	request := model.R_TALK_BROADCAST
	if s.request == model.R_WHISPER {
		request = model.R_WHISPER_BROADCAST
	}
	s.send(request, &talk)
}

func (s *CommunicationSession) send(request model.Request, talk *model.Talk) {
	for _, agent := range s.agents {
		info := s.game.buildInfo(agent)
		packet := model.Packet{
			Request: &request,
			Info:    &info,
		}
		if talk != nil {
			if request == model.R_TALK_BROADCAST {
				packet.NewTalk = talk
			} else {
				packet.NewWhisper = talk
			}
		}
		if _, err := agent.SendPacket(packet, s.game.config.Server.Timeout.Action, s.game.config.Server.Timeout.Response, s.game.config.Server.Timeout.Acceptable); err != nil {
			slog.Error("パケットの送信に失敗しました", "id", s.game.id, "agent", agent.String(), "request", request.String(), "error", err)
		}
	}
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

func (s *CommunicationSession) listenForTalks(ctx context.Context, agent *model.Agent, talkChannel chan<- *TalkSubmission) {
	for {
		select {
		case msg := <-agent.ReadChannel():
			if msg.Err != nil {
				slog.Warn("エージェントの接続でエラーが発生したためリスンを終了します", "id", s.game.id, "agent", agent.String(), "error", msg.Err)
				agent.HasError = true
				return
			}

			text := strings.TrimSpace(string(msg.Data))
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

		case <-ctx.Done():
			return
		}
	}
}
