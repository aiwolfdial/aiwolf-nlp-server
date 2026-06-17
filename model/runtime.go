package model

import (
	"log/slog"
	"os"
	"strconv"
)

// ApplyEnvOverrides applies server/runtime-level overrides from environment
// variables on top of a loaded config. It only touches deployment-level fields
// (host/port, auth, output directories, external service hosts) and never the
// per-game ruleset, so existing config files keep working unchanged. When no
// environment variables are set this is a no-op, preserving current behaviour.
//
// This is the primary mechanism for configuring the server in a container,
// where host/port and output directories must come from the environment rather
// than being baked into a YAML file. The JWT SECRET_KEY is read directly from
// the environment elsewhere and is intentionally not duplicated here.
func (c *Config) ApplyEnvOverrides() {
	if v := os.Getenv("AIWOLF_HOST"); v != "" {
		c.Server.WebSocket.Host = v
	}
	if v := os.Getenv("AIWOLF_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			c.Server.WebSocket.Port = p
		} else {
			slog.Warn("AIWOLF_PORTの解析に失敗しました", "value", v, "error", err)
		}
	}
	if v, ok := lookupBool("AIWOLF_AUTH_ENABLE"); ok {
		c.Server.Authentication.Enable = v
	}
	if v := os.Getenv("AIWOLF_JSON_LOG_DIR"); v != "" {
		c.JSONLogger.OutputDir = v
	}
	if v := os.Getenv("AIWOLF_GAME_LOG_DIR"); v != "" {
		c.GameLogger.OutputDir = v
	}
	if v := os.Getenv("AIWOLF_REALTIME_DIR"); v != "" {
		c.RealtimeBroadcaster.OutputDir = v
	}
	if v := os.Getenv("AIWOLF_TTS_HOST"); v != "" {
		c.TTSBroadcaster.Host = v
	}
	if v := os.Getenv("AIWOLF_MATCH_OUTPUT"); v != "" {
		c.Matching.OutputPath = v
	}
}

// lookupBool reads a boolean-ish env var, returning (value, true) when the
// variable is set to a recognised value and (false, false) when it is unset.
func lookupBool(key string) (bool, bool) {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return false, false
	}
	switch v {
	case "1", "true", "TRUE", "True":
		return true, true
	case "0", "false", "FALSE", "False":
		return false, true
	default:
		slog.Warn("真偽値の環境変数の解析に失敗しました", "key", key, "value", v)
		return false, false
	}
}
