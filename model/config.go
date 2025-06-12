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
	MaxCount struct {
		PerAgent int `yaml:"per_agent"`
		PerDay   int `yaml:"per_day"`
	} `yaml:"max_count"`
	MaxLength struct {
		CountInWord   bool `yaml:"count_in_word"`
		PerTalk       int  `yaml:"per_talk"`
		MentionLength int  `yaml:"mention_length"`
		PerAgent      int  `yaml:"per_agent"`
		BaseLength    int  `yaml:"base_length"`
	} `yaml:"max_length"`
	MaxSkip int `yaml:"max_skip"`
}

type LogicConfig struct {
	DayPhases   []Phase `yaml:"day_phases"`
	NightPhases []Phase `yaml:"night_phases"`
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
	Enable         bool      `yaml:"enable"`
	Profiles       []Profile `yaml:"profiles"`
	DynamicProfile struct {
		Enable   bool     `yaml:"enable"`
		Prompt   string   `yaml:"prompt"`
		Attempts int      `yaml:"attempts"`
		Avatars  []string `yaml:"avatars"`
	} `yaml:"dynamic_profile"`
}

type Profile struct {
	Name        string `yaml:"name"`
	AvatarURL   string `yaml:"avatar_url"`
	VoiceID     int    `yaml:"voice_id"`
	Age         int    `yaml:"age"`
	Gender      string `yaml:"gender"`
	Personality string `yaml:"personality"`
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
