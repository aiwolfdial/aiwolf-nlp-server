package transport

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

// /api/v1 配下の読み取り専用API。ゲームの開始/停止や設定投入といった更新系は持たない。
func (s *Server) registerAPI(router *gin.Engine) {
	api := router.Group("/api/v1")

	// オーケストレーションのプローブ用に無認証で公開する。
	api.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "version": Version.Version})
	})

	api.GET("/readyz", func(c *gin.Context) {
		if s.manager.IsShuttingDown() {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "draining"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})

	api.GET("/ruleset", func(c *gin.Context) {
		c.JSON(http.StatusOK, s.config.RulesetInfo())
	})

	// チームごとの失敗率と隔離状況。マッチングの重みがなぜ下がったかを外から確認するために公開する。
	teams := api.Group("/teams")
	if s.config.Server.Authentication.Enable {
		teams.Use(receiverAuthMiddleware())
	}
	teams.GET("", func(c *gin.Context) {
		if s.teamHealth == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "team health unavailable"})
			return
		}
		body := gin.H{"teams": s.teamHealth.Snapshots()}
		if s.matchOptimizer != nil {
			done, total := s.matchOptimizer.Progress()
			body["progress"] = gin.H{"done": done, "total": total}
		}
		c.JSON(http.StatusOK, body)
	})
	teams.GET("/:name", func(c *gin.Context) {
		if s.teamHealth == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "team health unavailable"})
			return
		}
		snap, ok := s.teamHealth.Get(c.Param("name"))
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "team not found"})
			return
		}
		c.JSON(http.StatusOK, snap)
	})

	games := api.Group("/games")
	if s.config.Server.Authentication.Enable {
		games.Use(receiverAuthMiddleware())
	}
	games.GET("", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"games": s.manager.ListGames()})
	})
	games.GET("/:id", func(c *gin.Context) {
		id := c.Param("id")
		// 日付や生存状況を持つライブ状態を優先し、無ければ登録簿の基本情報にフォールバックする。
		if s.liveState != nil {
			if snap, ok := s.liveState.Snapshot(id); ok {
				c.JSON(http.StatusOK, snap)
				return
			}
		}
		snap, ok := s.manager.GetGame(id)
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "game not found"})
			return
		}
		c.JSON(http.StatusOK, snap)
	})
	// ファイルポーリングに代わるリアルタイム配信（SSE）。既存の /realtime 静的配信は残す。
	games.GET("/:id/events", func(c *gin.Context) {
		if s.liveState == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "live state unavailable"})
			return
		}
		ch, cancel, ok := s.liveState.Subscribe(c.Param("id"))
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "game not found"})
			return
		}
		defer cancel()
		c.Stream(func(w io.Writer) bool {
			select {
			case packet, ok := <-ch:
				if !ok {
					return false
				}
				c.SSEvent("broadcast", packet)
				return true
			case <-c.Request.Context().Done():
				return false
			}
		})
	})
}
