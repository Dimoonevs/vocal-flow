package lib

import (
	"flag"
	"fmt"
	"github.com/Dimoonevs/vocal-flow/app/internal/models"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

var (
	staticRootDir = flag.String("staticDir", "/var/www/file_service/", "static dir")
	publicHost    = flag.String("publicHost", "http://your-video-service.pp.ua/video/service/", "public host")
)

func StitchVideoSubtitles(pathSave, pathVideo, filename string, subtitlesData []models.SubtitlesData) (string, error) {
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	videoSavePath := fmt.Sprintf("%s/%s_sub_%s", pathSave, timestamp, filename)
	args := []string{"-i", pathVideo}

	for _, subtitleData := range subtitlesData {
		args = append(args, "-i", subtitleData.URI)
	}

	args = append(args, "-map", "0:v", "-map", "0:a")
	for i := range subtitlesData {
		args = append(args, "-map", strconv.Itoa(i+1))
	}

	for i, subtitleData := range subtitlesData {
		args = append(args, "-c:s", "mov_text", "-metadata:s:s:"+strconv.Itoa(i), "language="+subtitleData.Lang)
	}

	// Указываем копирование видео и аудио
	args = append(args, "-c:v", "copy", "-c:a", "copy", videoSavePath)

	cmd := exec.Command("ffmpeg", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("ffmpeg error: %s, %v", string(output), err)
	}

	return videoSavePath, nil
}

func GetVideoLocalLink(link string) string {
	return strings.ReplaceAll(link, *staticRootDir, *publicHost)
}

func GetVideoPublicLink(link string) string {
	return strings.ReplaceAll(link, *publicHost, *staticRootDir)
}
