package core

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

// registerAPI mounts the versioned REST API under /api/v1. Endpoints are
// read-only in this pass: liveness/readiness probes for orchestration and
// read-only views of running games backed by the GameManager. Mutating
// endpoints (start/stop a game, push config) are intentionally deferred.
func (s *Server) registerAPI(router *gin.Engine) {
	api := router.Group("/api/v1")

	// Liveness: always 200 while the process is up. Unauthenticated so container
	// orchestration can probe it.
	api.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "version": Version.Version})
	})

	// Readiness: 503 while draining so a load balancer stops sending new traffic.
	api.GET("/readyz", func(c *gin.Context) {
		if s.manager.IsShuttingDown() {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "draining"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})

	rulesets := api.Group("/rulesets")
	if s.config.Server.Authentication.Enable {
		rulesets.Use(receiverAuthMiddleware())
	}
	rulesets.GET("", func(c *gin.Context) {
		summaries, err := s.rulesets.List()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"rulesets": summaries})
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
		// Prefer the live in-memory snapshot (it carries day and per-agent
		// status); fall back to the manager's basic registry snapshot.
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
	// Server-Sent Events stream of a game's broadcast events, replacing
	// file-polling for live spectators. The existing /realtime static files
	// remain for the legacy viewer.
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
