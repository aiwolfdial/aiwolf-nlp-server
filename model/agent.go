package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

type Agent struct {
	Idx          int
	TeamName     string
	OriginalName string
	GameName     string
	Role         Role
	Connection   *websocket.Conn
	HasError     bool
}

func NewAgent(idx int, role Role, conn Connection) (*Agent, error) {
	agent := &Agent{
		Idx:          idx,
		TeamName:     conn.TeamName,
		OriginalName: conn.OriginalName,
		GameName:     "Agent[" + fmt.Sprintf("%02d", idx) + "]",
		Role:         role,
		Connection:   conn.Conn,
		HasError:     false,
	}
	slog.Info("エージェントを作成しました", "idx", agent.Idx, "agent", agent.String(), "role", agent.Role, "connection", agent.Connection.RemoteAddr())
	return agent, nil
}

func (a *Agent) SendPacket(packet Packet, actionTimeout, responseTimeout, acceptableTimeout time.Duration) (string, error) {
	if a.HasError {
		slog.Error("エージェントにエラーが発生しているため、リクエストを送信できません", "agent", a.String())
		return "", errors.New("エージェントにエラーが発生しているため、リクエストを送信できません")
	}
	req, err := json.Marshal(packet)
	if err != nil {
		slog.Error("パケットの作成に失敗しました", "error", err)
		a.HasError = true
		return "", err
	}
	err = a.Connection.WriteMessage(websocket.TextMessage, req)
	if err != nil {
		slog.Error("パケットの送信に失敗しました", "error", err)
		a.HasError = true
		return "", err
	}
	slog.Info("パケットを送信しました", "agent", a.String(), "packet", packet)
	if packet.Request.RequireResponse {
		responseChan := make(chan []byte)
		errChan := make(chan error)
		go func() {
			_, res, err := a.Connection.ReadMessage()
			if err != nil {
				errChan <- err
				return
			}
			responseChan <- res
		}()
		select {
		case res := <-responseChan:
			slog.Info("レスポンスを受信しました", "agent", a.String(), "response", string(res))
			return strings.TrimRight(string(res), "\n"), nil
		case err := <-errChan:
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				slog.Error("接続が閉じられました", "error", err)
				a.HasError = true
				return "", err
			}
			slog.Warn("レスポンスの受信に失敗したため、NAMEリクエストを送信します", "agent", a.String(), "error", err)
		case <-time.After(actionTimeout + time.Second*5):
			slog.Warn("レスポンスの受信がタイムアウトしたため、NAMEリクエストを送信します", "agent", a.String())
		}
		nameReq, err := json.Marshal(Packet{Request: &R_NAME})
		if err != nil {
			slog.Error("NAMEパケットの作成に失敗しました", "error", err)
			a.HasError = true
			return "", err
		}
		err = a.Connection.WriteMessage(websocket.TextMessage, nameReq)
		if err != nil {
			slog.Error("NAMEパケットの送信に失敗しました", "error", err)
			a.HasError = true
			return "", err
		}
		slog.Info("NAMEパケットを送信しました", "agent", a.String())
		select {
		case res := <-responseChan:
			if strings.TrimRight(string(res), "\n") == a.OriginalName {
				slog.Info("NAMEリクエストのレスポンスを受信しました", "agent", a.String(), "response", string(res))
				return "", errors.New("リクエストのレスポンス受信がタイムアウトしました")
			} else {
				slog.Error("不正なNAMEリクエストのレスポンスを受信しました", "agent", a.String(), "response", string(res))
				a.HasError = true
				return "", errors.New("不正なNAMEリクエストのレスポンスを受信しました")
			}
		case err := <-errChan:
			slog.Error("NAMEリクエストのレスポンス受信に失敗しました", "agent", a.String(), "error", err)
			a.HasError = true
			return "", err
		case <-time.After(responseTimeout):
			slog.Error("NAMEリクエストのレスポンス受信がタイムアウトしました", "agent", a.String())
			a.HasError = true
			return "", errors.New("NAMEリクエストのレスポンス受信がタイムアウトしました")
		}
	}
	return "", nil
}

func (a *Agent) Close() {
	a.Connection.Close()
	slog.Info("エージェントをクローズしました", "agent", a.String())
}

func (a Agent) String() string {
	return a.GameName
}

func (a Agent) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.String())
}
