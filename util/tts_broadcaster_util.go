package util

import (
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"

	"golang.org/x/exp/slog"
)

func executeCommand(name string, arg ...string) ([]byte, error) {
	cmd := exec.Command(name, arg...)
	out, err := cmd.Output()
	if err != nil {
		slog.Error("コマンドの実行に失敗しました", "name", name, "arg", arg, "error", err)
		return nil, err
	}
	slog.Debug("コマンドの実行に成功しました", "name", name, "arg", arg, "output", string(out))
	return out, nil
}

func BuildSilenceTemplate(ffmpegPath string, args []string, duration float64, outputPath string) error {
	args = append(
		args,
		"-t",
		fmt.Sprintf("%f", duration),
		outputPath,
	)
	if _, err := executeCommand(ffmpegPath, args...); err != nil {
		slog.Error("無音セグメントの作成に失敗しました", "error", err)
		return err
	}
	slog.Info("無音セグメントの作成が完了しました")
	return nil
}

func CopyFile(sourcePath string, destinationPath string) error {
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	if err := os.MkdirAll(filepath.Dir(destinationPath), 0755); err != nil {
		return err
	}

	destFile, err := os.Create(destinationPath)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

type ConvertWavToSegmentParams struct {
	FfmpegPath      string
	FfprobePath     string
	DurationArgs    []string
	ConvertArgs     []string
	PreConvertArgs  []string
	SplitArgs       []string
	TempDir         string
	SegmentDuration float64
	Data            []byte
	BaseDir         string
	BaseName        string
}

func ConvertWavToSegment(params ConvertWavToSegmentParams) ([]string, error) {
	tempFile, err := os.CreateTemp(params.TempDir, params.BaseName+"*.wav")
	if err != nil {
		return nil, err
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath)

	if _, err := tempFile.Write(params.Data); err != nil {
		tempFile.Close()
		return nil, err
	}
	tempFile.Close()

	duration, err := GetDuration(params.FfprobePath, params.DurationArgs, tempPath)
	if err != nil {
		return nil, err
	}

	if duration <= params.SegmentDuration {
		segmentName := params.BaseName + ".ts"
		segmentPath := filepath.Join(params.BaseDir, segmentName)
		args := []string{"-i", tempPath}
		args = append(args, params.ConvertArgs...)
		args = append(args, segmentPath)

		if _, err := executeCommand(params.FfmpegPath, args...); err != nil {
			slog.Error("音声ファイルの変換に失敗しました", "error", err)
			return nil, err
		}
		slog.Info("音声ファイルの変換が完了しました", "segmentPath", segmentPath)
		return []string{segmentName}, nil
	}

	tempDir, err := os.MkdirTemp(params.TempDir, params.BaseName+"*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tempDir)

	aacPath := filepath.Join(tempDir, params.BaseName+".aac")
	convertArgs := []string{"-i", tempPath}
	convertArgs = append(convertArgs, params.PreConvertArgs...)
	convertArgs = append(convertArgs, aacPath)

	if _, err := executeCommand(params.FfmpegPath, convertArgs...); err != nil {
		slog.Error("音声ファイルの事前変換に失敗しました", "error", err)
		return nil, err
	}
	slog.Info("音声ファイルの事前変換が完了しました", "outputPath", aacPath)

	segmentCount := int(math.Ceil(duration / params.SegmentDuration))
	segmentNames := make([]string, segmentCount)
	for i := range segmentCount {
		segmentName := fmt.Sprintf("%s_%d.ts", params.BaseName, i)
		segmentNames = append(segmentNames, segmentName)
		segmentPath := filepath.Join(params.BaseDir, segmentName)
		startTime := float64(i) * params.SegmentDuration
		duration := params.SegmentDuration
		if i == segmentCount-1 {
			duration = duration - startTime
		}
		splitArgs := []string{"-i", aacPath, "-ss", fmt.Sprintf("%f", startTime), "-t", fmt.Sprintf("%f", duration)}
		splitArgs = append(splitArgs, params.SplitArgs...)
		splitArgs = append(splitArgs, segmentPath)
		if _, err := executeCommand(params.FfmpegPath, splitArgs...); err != nil {
			slog.Error("音声ファイルの分割に失敗しました", "error", err)
			return nil, err
		}
		slog.Info("音声ファイルの分割が完了しました", "segmentPath", segmentPath)
	}
	slog.Info("音声ファイルのセグメント化が完了しました", "segmentCount", segmentCount)
	return segmentNames, nil
}

func GetDuration(ffprobePath string, args []string, targetPath string) (float64, error) {
	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		return 0, err
	}
	args = append(args, targetPath)

	out, err := executeCommand(ffprobePath, args...)
	if err != nil {
		return 0, err
	}

	var duration float64
	if _, err := fmt.Sscanf(string(out), "%f", &duration); err != nil {
		slog.Error("音声ファイルの長さの取得に失敗しました", "error", err)
		return 0, err
	}
	slog.Info("音声ファイルの長さを取得しました", "duration", duration)
	return duration, nil
}
