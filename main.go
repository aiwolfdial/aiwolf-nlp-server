package main

import (
	"flag"
	"log/slog"
	"os"

	"github.com/aiwolfdial/aiwolf-nlp-server/matchmaking"
	"github.com/aiwolfdial/aiwolf-nlp-server/model"
	"github.com/aiwolfdial/aiwolf-nlp-server/transport"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

var (
	version  string
	revision string
	build    string
)

func main() {
	gin.SetMode(gin.ReleaseMode)

	if version == "" {
		if godotenv.Load("./config/.env") != nil {
			slog.Error("環境変数の読み込みに失敗しました")
		}
	} else {
		if godotenv.Load("./.env") != nil {
			slog.Error("環境変数の読み込みに失敗しました")
		}
	}

	transport.SetVersion(version, revision, build)

	var (
		configPath    = flag.String("c", "./default.yml", "設定ファイルのパス")
		analyzerMode  = flag.Bool("a", false, "解析モード")
		reductionMode = flag.Bool("r", false, "縮約モード")
		srcConfigPath = flag.String("s", "", "ソース設定ファイルのパス")
		dstConfigPath = flag.String("d", "", "デスティネーション設定ファイルのパス")
		showVersion   = flag.Bool("v", false, "バージョンを表示")
		showHelp      = flag.Bool("h", false, "ヘルプを表示")
	)
	flag.Parse()

	if *showVersion {
		println("version:", transport.Version.Version)
		println("revision:", transport.Version.Revision)
		println("build:", transport.Version.Build)
		os.Exit(0)
	}

	if *showHelp {
		flag.Usage()
		os.Exit(0)
	}

	config, err := model.LoadFromPath(*configPath)
	if err != nil {
		panic(err)
	}
	config.ApplyEnvOverrides()

	if *analyzerMode {
		matchmaking.Analyzer(*config)
		return
	}

	if *reductionMode {
		srcConfig, err := model.LoadFromPath(*srcConfigPath)
		if err != nil {
			panic(err)
		}
		dstConfig, err := model.LoadFromPath(*dstConfigPath)
		if err != nil {
			panic(err)
		}
		matchmaking.Reduction(*srcConfig, *dstConfig)
		return
	}

	server, err := transport.NewServer(*config)
	if err != nil {
		panic(err)
	}
	server.Run()
}
