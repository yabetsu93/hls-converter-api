package video

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"

	"github.com/yabetsu93/hls-converter-api/models"
)

func GetVideoInfo(videoPath string) (*models.VideoInfo, error) {
	cmd := exec.Command("ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		videoPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("ffprobe error: %v, output: %s", err, string(output))
	}

	var result struct {
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
		Streams []struct {
			CodecType string `json:"codec_type"`
			CodecName string `json:"code_name"`
			Width     int    `json:"width"`
			Height    int    `json:"height"`
		} `json:"streams"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse ffprobe output: %v", err)
	}

	info := &models.VideoInfo{
		Codec: "unknown",
	}

	// Parse Duration
	if result.Format.Duration != "" {
		if duration, err := strconv.ParseFloat(result.Format.Duration, 64); err == nil {
			info.Duration = duration
		}
	}

	// Find video stream
	for _, stream := range result.Streams {
		if stream.CodecType == "video" {
			info.Width = stream.Width
			info.Height = stream.Height
			info.Codec = stream.CodecName
			break
		}
	}

	return info, nil
}
