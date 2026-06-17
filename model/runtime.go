package model

import (
	"log/slog"
	"os"
	"strconv"
)

// 環境変数でサーバ/ランタイム設定を上書きする。コンテナ実行で設定ファイルを編集せずに
// host/portや出力先を差し替えるための仕組み。未設定の変数は無視するため、未設定時は
// 設定ファイルの値がそのまま使われる。ゲームのルールは対象外。
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

// 未設定なら(false, false)を返し、設定済みのときのみ第2戻り値をtrueにする。
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
