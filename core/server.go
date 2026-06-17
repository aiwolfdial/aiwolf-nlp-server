package core

import (
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/aiwolfdial/aiwolf-nlp-server/model"
	"github.com/aiwolfdial/aiwolf-nlp-server/observer"
	"github.com/aiwolfdial/aiwolf-nlp-server/observer/livestate"
	"github.com/aiwolfdial/aiwolf-nlp-server/service"
	"github.com/aiwolfdial/aiwolf-nlp-server/util"
	"github.com/gorilla/websocket"
)

type Server struct {
	config              model.Config
	upgrader            websocket.Upgrader
	manager             *GameManager
	liveState           *livestate.LiveState
	jsonLogger          *service.JSONLogger
	gameLogger          *service.GameLogger
	realtimeBroadcaster *service.RealtimeBroadcaster
	ttsBroadcaster      *service.TTSBroadcaster
}

func NewServer(config model.Config) (*Server, error) {
	server := &Server{
		config: config,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		liveState: livestate.New(),
	}
	gameSettings, err := model.NewSetting(config)
	if err != nil {
		return nil, errors.New("ゲーム設定の作成に失敗しました")
	}
	if config.JSONLogger.Enable {
		server.jsonLogger = service.NewJSONLogger(config)
	}
	if config.GameLogger.Enable {
		server.gameLogger = service.NewGameLogger(config)
	}
	if config.TTSBroadcaster.Enable {
		server.ttsBroadcaster = service.NewTTSBroadcaster(config)
	}
	if config.RealtimeBroadcaster.Enable {
		server.realtimeBroadcaster = service.NewRealtimeBroadcaster(config)
	}
	var matchOptimizer *MatchOptimizer
	if config.Matching.IsOptimize {
		matchOptimizer, err = NewMatchOptimizer(config)
		if err != nil {
			return nil, errors.New("マッチオプティマイザの作成に失敗しました")
		}
	}
	server.manager = NewGameManager(config, gameSettings, NewWaitingRoom(config), matchOptimizer, server.newObserver)
	return server, nil
}

func (s *Server) newObserver() observer.GameObserver {
	var observers []observer.GameObserver
	if s.jsonLogger != nil {
		observers = append(observers, s.jsonLogger.AsObserver())
	}
	if s.gameLogger != nil {
		observers = append(observers, s.gameLogger.AsObserver())
	}
	if s.realtimeBroadcaster != nil {
		observers = append(observers, s.realtimeBroadcaster.AsObserver())
	}
	if s.ttsBroadcaster != nil {
		observers = append(observers, s.ttsBroadcaster.AsObserver())
	}
	if s.liveState != nil {
		observers = append(observers, s.liveState)
	}
	return observer.NewComposite(observers...)
}

func (s *Server) Run() {
	router := s.buildRouter()

	go func() {
		trap := make(chan os.Signal, 1)
		signal.Notify(trap, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGINT)
		sig := <-trap
		slog.Info("シグナルを受信しました", "signal", sig)
		s.manager.BeginShutdown()
		s.manager.WaitAllFinished()
		os.Exit(0)
	}()

	slog.Info("サーバを起動しました", "host", s.config.Server.WebSocket.Host, "port", s.config.Server.WebSocket.Port)
	err := router.Run(s.config.Server.WebSocket.Host + ":" + strconv.Itoa(s.config.Server.WebSocket.Port))
	if err != nil {
		slog.Error("サーバの起動に失敗しました", "error", err)
		return
	}
}

func (s *Server) handleConnections(w http.ResponseWriter, r *http.Request) {
	if s.manager.IsShuttingDown() {
		slog.Warn("シグナルを受信したため、新しい接続を受け付けません")
		return
	}
	header := r.Header.Clone()
	ws, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("クライアントのアップグレードに失敗しました", "error", err)
		return
	}
	conn, err := model.NewConnection(ws, &header)
	if err != nil {
		slog.Error("クライアントの接続に失敗しました", "error", err)
		return
	}
	if s.config.Server.Authentication.Enable {
		token := r.URL.Query().Get("token")
		if token != "" {
			if !util.IsValidPlayerToken(os.Getenv("SECRET_KEY"), token, conn.TeamName) {
				slog.Warn("トークンが無効です", "team_name", conn.TeamName)
				conn.Conn.Close()
				slog.Info("クライアントの接続を切断しました", "team_name", conn.TeamName)
				return
			}
		} else {
			token = strings.ReplaceAll(conn.Header.Get("Authorization"), "Bearer ", "")
			if !util.IsValidPlayerToken(os.Getenv("SECRET_KEY"), token, conn.TeamName) {
				slog.Warn("トークンが無効です", "team_name", conn.TeamName)
				conn.Conn.Close()
				slog.Info("クライアントの接続を切断しました", "team_name", conn.TeamName)
				return
			}
		}
	}

	s.manager.TryStartGame(*conn)
}
