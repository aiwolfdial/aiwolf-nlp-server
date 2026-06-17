package store

// マッチオプティマイザの状態を永続化する。既定はファイル実装で、将来別の実装へ差し替えられる。
type MatchOptimizerStore interface {
	Load() ([]byte, error)
	Save(data []byte) error
}
