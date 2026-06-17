package model

import (
	"log/slog"
	"os"
	"strconv"
)

// 環境変数でhost/portを上書きする。コンテナ実行で設定ファイルを編集せずに差し替えるための
// 仕組み。未設定時は設定ファイルの値が使われる。
func (c *Config) ApplyEnvOverrides() {
	if v := os.Getenv("HOST"); v != "" {
		c.Server.WebSocket.Host = v
	}
	if v := os.Getenv("PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			c.Server.WebSocket.Port = p
		} else {
			slog.Warn("PORTの解析に失敗しました", "value", v, "error", err)
		}
	}
}
