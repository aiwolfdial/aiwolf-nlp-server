package main

import (
	"flag"
	"log/slog"
	"os"

	"github.com/aiwolfdial/aiwolf-nlp-server/core"
	"github.com/aiwolfdial/aiwolf-nlp-server/model"
	"github.com/joho/godotenv"
)

var (
	version  string
	revision string
	build    string
)

func main() {
	err := godotenv.Load()
	if err != nil {
		slog.Error("環境変数の読み込みに失敗しました", "error", err)
	}

	core.SetVersion(version, revision, build)

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
		println("version:", core.Version.Version)
		println("revision:", core.Version.Revision)
		println("build:", core.Version.Build)
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

	if *analyzerMode {
		core.Analyzer(*config)
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
		core.Reduction(*srcConfig, *dstConfig)
		return
	}

	server := core.NewServer(*config)
	server.Run()
}
