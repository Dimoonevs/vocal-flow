package lib

import (
	"fmt"
	"github.com/Dimoonevs/vocal-flow/app/internal/models"
	"os/exec"
	"strconv"
	"time"
)

func StitchVideoSubtitles(pathSave, pathVideo, filename string, subtitlesData []models.SubtitlesData) (string, error) {
	videoSavePath := fmt.Sprintf("%s/%s_sub_%s", pathSave, strconv.FormatInt(time.Now().Unix(), 10), filename)
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

	args = append(args, "-c:v", "copy", "-c:a", "copy", videoSavePath)

	cmd := exec.Command("ffmpeg", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("ffmpeg error: %s, %v", string(output), err)
	}

	return videoSavePath, nil
}
