package core

import (
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/kano-lab/aiwolf-nlp-server/logic"
	"github.com/kano-lab/aiwolf-nlp-server/model"
	"github.com/kano-lab/aiwolf-nlp-server/service"
	"github.com/kano-lab/aiwolf-nlp-server/util"
)

type Server struct {
	config              model.Config
	upgrader            websocket.Upgrader
	waitingRoom         *WaitingRoom
	matchOptimizer      *MatchOptimizer
	gameSetting         *model.Setting
	games               []*logic.Game
	mu                  sync.RWMutex
	signaled            bool
	jsonLogger          *service.JSONLogger
	gameLogger          *service.GameLogger
	realtimeBroadcaster *service.RealtimeBroadcaster
}

func NewServer(config model.Config) *Server {
	server := &Server{
		config: config,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		waitingRoom: NewWaitingRoom(config),
		games:       make([]*logic.Game, 0),
		mu:          sync.RWMutex{},
		signaled:    false,
	}
	gameSettings, err := model.NewSetting(config)
	if err != nil {
		slog.Error("ゲーム設定の作成に失敗しました", "error", err)
		return nil
	}
	server.gameSetting = gameSettings
	if config.JSONLogger.Enable {
		server.jsonLogger = service.NewJSONLogger(config)
	}
	if config.GameLogger.Enable {
		server.gameLogger = service.NewGameLogger(config)
	}
	if config.RealtimeBroadcaster.Enable {
		server.realtimeBroadcaster = service.NewRealtimeBroadcaster(config)
	}
	if config.Matching.IsOptimize {
		matchOptimizer, err := NewMatchOptimizer(config)
		if err != nil {
			slog.Error("マッチオプティマイザの作成に失敗しました", "error", err)
			return nil
		}
		server.matchOptimizer = matchOptimizer
	}
	return server
}

func (s *Server) Run() {
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	router.Use(func(c *gin.Context) {
		c.Header("Server", "aiwolf-nlp-server/"+Version.Version+" "+runtime.Version()+" ("+runtime.GOOS+"; "+runtime.GOARCH+")")
	})

	router.GET("/ws", func(c *gin.Context) {
		s.handleConnections(c.Writer, c.Request)
	})

	if s.config.RealtimeBroadcaster.Enable {
		router.GET("/realtime", func(c *gin.Context) {
			s.realtimeBroadcaster.HandleConnections(c.Writer, c.Request)
		})
	}

	go func() {
		trap := make(chan os.Signal, 1)
		signal.Notify(trap, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGINT)
		sig := <-trap
		slog.Info("シグナルを受信しました", "signal", sig)
		s.signaled = true
		s.gracefullyShutdown()
		os.Exit(0)
	}()

	slog.Info("サーバを起動しました", "host", s.config.Server.WebSocket.Host, "port", s.config.Server.WebSocket.Port)
	err := router.Run(s.config.Server.WebSocket.Host + ":" + strconv.Itoa(s.config.Server.WebSocket.Port))
	if err != nil {
		slog.Error("サーバの起動に失敗しました", "error", err)
		return
	}
}

func (s *Server) gracefullyShutdown() {
	for {
		isFinished := true
		s.mu.RLock()
		for _, game := range s.games {
			if !game.IsFinished {
				isFinished = false
				break
			}
		}
		s.mu.RUnlock()
		if isFinished {
			break
		}
		time.Sleep(15 * time.Second)
	}
	slog.Info("全てのゲームが終了しました")
}

func (s *Server) handleConnections(w http.ResponseWriter, r *http.Request) {
	if s.signaled {
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
		token := strings.ReplaceAll(conn.Header.Get("Authorization"), "Bearer ", "")
		if !util.IsValidPlayerToken(s.config.Server.Authentication.Secret, token, conn.Team) {
			slog.Warn("トークンが無効です", "team", conn.Team)
			conn.Conn.Close()
			slog.Info("クライアントの接続を切断しました", "team", conn.Team)
			return
		}
	}
	s.waitingRoom.AddConnection(conn.Team, *conn)

	s.mu.Lock()
	var game *logic.Game
	if s.config.Matching.IsOptimize {
		for team := range s.waitingRoom.connections {
			s.matchOptimizer.updateTeam(team)
		}
		matches := s.matchOptimizer.getMatches()
		roleMapConns, err := s.waitingRoom.GetConnectionsWithMatchOptimizer(matches)
		if err != nil {
			slog.Error("待機部屋からの接続の取得に失敗しました", "error", err)
			s.mu.Unlock()
			return
		}
		game = logic.NewGameWithRole(&s.config, s.gameSetting, roleMapConns)
	} else {
		connections, err := s.waitingRoom.GetConnections()
		if err != nil {
			slog.Error("待機部屋からの接続の取得に失敗しました", "error", err)
			s.mu.Unlock()
			return
		}
		game = logic.NewGame(&s.config, s.gameSetting, connections)
	}
	if s.jsonLogger != nil {
		game.SetJSONLogger(s.jsonLogger)
	}
	if s.gameLogger != nil {
		game.SetGameLogger(s.gameLogger)
	}
	if s.realtimeBroadcaster != nil {
		game.SetRealtimeBroadcaster(s.realtimeBroadcaster)
	}
	s.games = append(s.games, game)
	s.mu.Unlock()

	go func() {
		winSide := game.Start()
		if s.config.Matching.IsOptimize {
			s.mu.Lock()
			defer s.mu.Unlock()
			if winSide != model.T_NONE {
				s.matchOptimizer.setMatchEnd(util.GetRoleTeamNamesMap(game.Agents))
			} else {
				s.matchOptimizer.setMatchWeight(util.GetRoleTeamNamesMap(game.Agents), 0)
			}
		}
	}()
}
