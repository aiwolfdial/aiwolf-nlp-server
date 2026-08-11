package model

import (
	"log/slog"
	"os"
	"time"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Server              ServerConfig              `yaml:"server"`
	Game                GameConfig                `yaml:"game"`
	Logic               LogicConfig               `yaml:"logic"`
	Matching            MatchingConfig            `yaml:"matching"`
	CustomProfile       CustomProfileConfig       `yaml:"custom_profile"`
	JSONLogger          JSONLoggerConfig          `yaml:"json_logger"`
	GameLogger          GameLoggerConfig          `yaml:"game_logger"`
	RealtimeBroadcaster RealtimeBroadcasterConfig `yaml:"realtime_broadcaster"`
	TTSBroadcaster      TTSBroadcasterConfig      `yaml:"tts_broadcaster"`
	TeamHealth          TeamHealthConfig          `yaml:"team_health"`
	SlackNotifier       SlackNotifierConfig       `yaml:"slack_notifier"`
}

type ServerConfig struct {
	WebSocket struct {
		Host string `yaml:"host"`
		Port int    `yaml:"port"`
	} `yaml:"web_socket"`
	Authentication struct {
		Enable bool `yaml:"enable"`
	} `yaml:"authentication"`
	Timeout struct {
		Action     time.Duration `yaml:"action"`
		Response   time.Duration `yaml:"response"`
		Acceptable time.Duration `yaml:"acceptable"`
	} `yaml:"timeout"`
	MaxContinueErrorRatio float64 `yaml:"max_continue_error_ratio"`
}

type GameConfig struct {
	AgentCount     int        `yaml:"agent_count"`
	MaxDay         int        `yaml:"max_day"`
	VoteVisibility bool       `yaml:"vote_visibility"`
	Talk           TalkConfig `yaml:"talk"`
	Whisper        TalkConfig `yaml:"whisper"`
	Vote           struct {
		MaxCount      int  `yaml:"max_count"`
		AllowSelfVote bool `yaml:"allow_self_vote"`
	} `yaml:"vote"`
	AttackVote struct {
		MaxCount      int  `yaml:"max_count"`
		AllowSelfVote bool `yaml:"allow_self_vote"`
		AllowNoTarget bool `yaml:"allow_no_target"`
	} `yaml:"attack_vote"`
}

type TalkConfig struct {
	Duration *time.Duration `yaml:"duration,omitempty"`
	MaxCount struct {
		PerAgent int `yaml:"per_agent"`
		PerDay   int `yaml:"per_day"`
	} `yaml:"max_count"`
	MaxLength struct {
		CountInWord   bool `yaml:"count_in_word"`
		CountSpaces   bool `yaml:"count_spaces"`
		PerTalk       int  `yaml:"per_talk"`
		MentionLength int  `yaml:"mention_length"`
		PerAgent      int  `yaml:"per_agent"`
		BaseLength    int  `yaml:"base_length"`
	} `yaml:"max_length"`
	MaxSkip int `yaml:"max_skip"`
}

type LogicConfig struct {
	DayPhases   []Phase                `yaml:"day_phases"`
	NightPhases []Phase                `yaml:"night_phases"`
	Roles       map[int]map[string]int `yaml:"roles"`
}

type Phase struct {
	Name      string   `yaml:"name"`
	Actions   []string `yaml:"actions"`
	OnlyDay   *int     `yaml:"only_day,omitempty"`
	ExceptDay *int     `yaml:"except_day,omitempty"`
}

type MatchingConfig struct {
	SelfMatch    bool   `yaml:"self_match"`
	IsOptimize   bool   `yaml:"is_optimize"`
	TeamCount    int    `yaml:"team_count"`
	GameCount    int    `yaml:"game_count"`
	OutputPath   string `yaml:"output_path"`
	InfiniteLoop bool   `yaml:"infinite_loop"`
}

type CustomProfileConfig struct {
	Enable          bool                 `yaml:"enable"`
	ProfileEncoding map[string]string    `yaml:"profile_encoding"`
	Profiles        []Profile            `yaml:"profiles"`
	DynamicProfile  DynamicProfileConfig `yaml:"dynamic_profile"`
}

type Profile struct {
	Name      string            `yaml:"name"`
	AvatarURL string            `yaml:"avatar_url"`
	VoiceID   int               `yaml:"voice_id"`
	Arguments map[string]string `yaml:",inline"`
}

type DynamicProfileConfig struct {
	Enable    bool     `yaml:"enable"`
	Prompt    string   `yaml:"prompt"`
	Attempts  int      `yaml:"attempts"`
	Model     string   `yaml:"model"`
	MaxTokens int      `yaml:"max_tokens"`
	Avatars   []string `yaml:"avatars"`
}

type JSONLoggerConfig struct {
	Enable    bool   `yaml:"enable"`
	OutputDir string `yaml:"output_dir"`
	Filename  string `yaml:"filename"`
}

type GameLoggerConfig struct {
	Enable    bool   `yaml:"enable"`
	OutputDir string `yaml:"output_dir"`
	Filename  string `yaml:"filename"`
}

type RealtimeBroadcasterConfig struct {
	Enable    bool          `yaml:"enable"`
	Delay     time.Duration `yaml:"delay"`
	OutputDir string        `yaml:"output_dir"`
	Filename  string        `yaml:"filename"`
}

type TTSBroadcasterConfig struct {
	Enable         bool          `yaml:"enable"`
	Async          bool          `yaml:"async"`
	TargetDuration time.Duration `yaml:"target_duration"`
	SegmentDir     string        `yaml:"segment_dir"`
	TempDir        string        `yaml:"temp_dir"`
	Host           string        `yaml:"host"`
	Timeout        time.Duration `yaml:"timeout"`
	FfmpegPath     string        `yaml:"ffmpeg_path"`
	FfprobePath    string        `yaml:"ffprobe_path"`
	ConvertArgs    []string      `yaml:"convert_args"`
	DurationArgs   []string      `yaml:"duration_args"`
	PreConvertArgs []string      `yaml:"pre_convert_args"`
	SplitArgs      []string      `yaml:"split_args"`
}

// チームごとの失敗率を直近 Window 試合で評価し、マッチの重みと隔離に反映するための設定。
type TeamHealthConfig struct {
	Enable   bool `yaml:"enable"`
	Window   int  `yaml:"window"`
	MinGames int  `yaml:"min_games"`
	// 失敗率は3つの指標の加重平均で、重みの比だけが意味を持つ。
	Scores struct {
		RequestError float64 `yaml:"request_error"`
		Fatal        float64 `yaml:"fatal"`
		Abort        float64 `yaml:"abort"`
	} `yaml:"scores"`
	WeightFloor        float64       `yaml:"weight_floor"`
	QuarantineRate     float64       `yaml:"quarantine_rate"`
	QuarantineDuration time.Duration `yaml:"quarantine_duration"`
	AbortWeightFactor  float64       `yaml:"abort_weight_factor"`
}

type SlackNotifierConfig struct {
	Enable bool `yaml:"enable"`
	// 空のときは環境変数 SLACK_WEBHOOK_URL を使う。URLは秘匿情報なので設定ファイルへ直接書かない運用を想定する。
	WebhookURL     string        `yaml:"webhook_url"`
	Username       string        `yaml:"username"`
	IconEmoji      string        `yaml:"icon_emoji"`
	Timeout        time.Duration `yaml:"timeout"`
	MinInterval    time.Duration `yaml:"min_interval"`
	Events         []string      `yaml:"events"`
	StallThreshold time.Duration `yaml:"stall_threshold"`
	MilestoneEvery int           `yaml:"milestone_every"`
}

func LoadFromPath(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		slog.Error("設定ファイルの読み込みに失敗しました", "error", err)
		return nil, err
	}
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		slog.Error("設定ファイルのパースに失敗しました", "error", err)
		return nil, err
	}
	return &config, nil
}
