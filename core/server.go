package core

import (
	"errors"
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

	"github.com/aiwolfdial/aiwolf-nlp-server/logic"
	"github.com/aiwolfdial/aiwolf-nlp-server/model"
	"github.com/aiwolfdial/aiwolf-nlp-server/service"
	"github.com/aiwolfdial/aiwolf-nlp-server/util"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type Server struct {
	config              model.Config
	upgrader            websocket.Upgrader
	waitingRoom         *WaitingRoom
	matchOptimizer      *MatchOptimizer
	gameSetting         *model.Setting
	games               sync.Map
	mu                  sync.RWMutex
	signaled            bool
	jsonLogger          *service.JSONLogger
	gameLogger          *service.GameLogger
	realtimeBroadcaster *service.RealtimeBroadcaster
	ttsBroadcaster      *service.TTSBroadcaster
	logger              *slog.Logger
}

func NewServer(config model.Config) (*Server, error) {
	logger := slog.Default().With(
		"component", "server",
		"host", config.Server.WebSocket.Host,
		"port", config.Server.WebSocket.Port,
	)

	server := &Server{
		config: config,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		waitingRoom: NewWaitingRoom(config),
		games:       sync.Map{},
		mu:          sync.RWMutex{},
		signaled:    false,
		logger:      logger,
	}
	gameSettings, err := model.NewSetting(config)
	if err != nil {
		return nil, errors.New("ゲーム設定の作成に失敗しました")
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
	if config.TTSBroadcaster.Enable {
		server.ttsBroadcaster = service.NewTTSBroadcaster(config)
	}
	if config.Matching.IsOptimize {
		matchOptimizer, err := NewMatchOptimizer(config)
		if err != nil {
			return nil, errors.New("マッチオプティマイザの作成に失敗しました")
		}
		server.matchOptimizer = matchOptimizer
	}
	return server, nil
}

func (s *Server) Run() {
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	router.Use(func(c *gin.Context) {
		c.Header("Server", "aiwolf-nlp-server/"+Version.Version+" "+runtime.Version()+" ("+runtime.GOOS+"; "+runtime.GOARCH+")")

		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Ngrok-Skip-Browser-Warning")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	router.GET("/ws", func(c *gin.Context) {
		s.handleConnections(c.Writer, c.Request)
	})

	if s.config.RealtimeBroadcaster.Enable {
		router.GET("/realtime", func(c *gin.Context) {
			s.realtimeBroadcaster.HandleConnections(c.Writer, c.Request)
		})
	}

	if s.config.TTSBroadcaster.Enable {
		router.Static("/tts", s.config.TTSBroadcaster.SegmentDir)
		go s.ttsBroadcaster.Start()
	}

	go func() {
		trap := make(chan os.Signal, 1)
		signal.Notify(trap, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGINT)
		sig := <-trap
		s.logger.Info("シグナルを受信しました", "signal", sig)
		s.signaled = true
		s.gracefullyShutdown()
		os.Exit(0)
	}()

	s.logger.Info("サーバを起動しました")
	err := router.Run(s.config.Server.WebSocket.Host + ":" + strconv.Itoa(s.config.Server.WebSocket.Port))
	if err != nil {
		s.logger.Error("サーバの起動に失敗しました", "error", err)
		return
	}
}

func (s *Server) gracefullyShutdown() {
	for {
		isFinished := true
		s.games.Range(func(key, value any) bool {
			game, ok := value.(*logic.Game)
			if !ok || !game.IsFinished() {
				isFinished = false
				return false
			}
			return true
		})
		if isFinished {
			break
		}
		time.Sleep(15 * time.Second)
	}
	s.logger.Info("全てのゲームが終了しました")
}

func (s *Server) handleConnections(w http.ResponseWriter, r *http.Request) {
	if s.signaled {
		s.logger.Warn("シグナルを受信したため、新しい接続を受け付けません")
		return
	}

	requestLogger := s.logger.With(
		"method", r.Method,
		"url", r.URL.String(),
		"remote_addr", r.RemoteAddr,
	)

	header := r.Header.Clone()
	ws, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		requestLogger.Error("クライアントのアップグレードに失敗しました", "error", err)
		return
	}
	conn, err := model.NewConnection(ws, &header)
	if err != nil {
		requestLogger.Error("クライアントの接続に失敗しました", "error", err)
		return
	}

	connLogger := requestLogger.With("team_name", conn.TeamName)
	if s.config.Server.Authentication.Enable {
		token := r.URL.Query().Get("token")
		if token != "" {
			if !util.IsValidPlayerToken(os.Getenv("SECRET_KEY"), token, conn.TeamName) {
				connLogger.Warn("トークンが無効です", "token_source", "query_param")
				conn.Conn.Close()
				connLogger.Info("クライアントの接続を切断しました")
				return
			}
		} else {
			token = strings.ReplaceAll(conn.Header.Get("Authorization"), "Bearer ", "")
			if !util.IsValidPlayerToken(os.Getenv("SECRET_KEY"), token, conn.TeamName) {
				connLogger.Warn("トークンが無効です", "token_source", "header")
				conn.Conn.Close()
				connLogger.Info("クライアントの接続を切断しました")
				return
			}
		}
	}
	s.waitingRoom.AddConnection(conn.TeamName, *conn)
	connLogger.Info("クライアントを待機室に追加しました")

	var game *logic.Game
	if s.config.Matching.IsOptimize {
		s.waitingRoom.connections.Range(func(key, value any) bool {
			team := key.(string)
			s.matchOptimizer.updateTeam(team)
			return true
		})
		matches := s.matchOptimizer.getMatches()
		roleMapConns, err := s.waitingRoom.GetConnectionsWithMatchOptimizer(matches)
		if err != nil {
			connLogger.Error("待機部屋からの接続の取得に失敗しました", "error", err, "match_mode", "optimized")
			return
		}
		game = logic.NewGameWithRole(&s.config, s.gameSetting, roleMapConns)
	} else {
		connections, err := s.waitingRoom.GetConnections()
		if err != nil {
			connLogger.Error("待機部屋からの接続の取得に失敗しました", "error", err, "match_mode", "standard")
			return
		}
		game = logic.NewGame(&s.config, s.gameSetting, connections)
	}
	if s.jsonLogger != nil {
		game.JsonLogger = s.jsonLogger
	}
	if s.gameLogger != nil {
		game.GameLogger = s.gameLogger
	}
	if s.realtimeBroadcaster != nil {
		game.RealtimeBroadcaster = s.realtimeBroadcaster
	}
	if s.ttsBroadcaster != nil {
		game.TTSBroadcaster = s.ttsBroadcaster
	}
	s.games.Store(game.ID, game)

	gameLogger := connLogger.With("game_id", game.ID)
	gameLogger.Info("新しいゲームを開始しました", "player_count", len(game.GetRoleTeamNamesMap()))

	go func() {
		winSide := game.Start()
		gameLogger.Info("ゲームが終了しました", "win_side", winSide)
		if s.config.Matching.IsOptimize {
			if winSide != model.T_NONE {
				s.matchOptimizer.setMatchEnd(game.GetRoleTeamNamesMap())
			} else {
				s.matchOptimizer.setMatchWeight(game.GetRoleTeamNamesMap(), 0)
			}
		}
	}()
}
