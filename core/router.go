package core

import (
	"net/http"
	"os"
	"runtime"
	"strings"

	"github.com/aiwolfdial/aiwolf-nlp-server/util"
	"github.com/gin-gonic/gin"
)

func (s *Server) buildRouter() *gin.Engine {
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
		realtimeGroup := router.Group("/realtime")
		if s.config.Server.Authentication.Enable {
			realtimeGroup.Use(receiverAuthMiddleware())
		}
		realtimeGroup.Static("/", s.config.RealtimeBroadcaster.OutputDir)
	}

	if s.config.TTSBroadcaster.Enable {
		router.Static("/tts", s.config.TTSBroadcaster.SegmentDir)
		go s.ttsBroadcaster.Start()
	}

	s.registerAPI(router)
	return router
}

// /realtime と REST API で共用する RECEIVER トークン検証。
func receiverAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.Query("token")
		if token == "" {
			token = strings.ReplaceAll(c.GetHeader("Authorization"), "Bearer ", "")
		}
		if token == "" {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		if !util.IsValidReceiver(os.Getenv("SECRET_KEY"), token) {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		c.Next()
	}
}
