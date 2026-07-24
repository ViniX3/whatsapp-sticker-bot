package converter

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"whatsapp-sticker-bot/internal/config"
	"whatsapp-sticker-bot/internal/logger"
)

func CreateSticker(
	input string,
) (string, error) {

	output := fmt.Sprintf(
		"temp/sticker_%d.webp",
		time.Now().Unix(),
	)

	logger.Debug(
		"Iniciando conversão para sticker:",
		input,
	)

	extension := strings.ToLower(
		filepath.Ext(input),
	)

	switch extension {

	case ".jpg", ".jpeg", ".png", ".webp":

		return createImageSticker(
			input,
			output,
		)

	case ".mp4", ".gif", ".webm":

		return createVideoSticker(
			input,
			output,
		)

	default:

		return "",
			fmt.Errorf(
				"formato não suportado: %s",
				extension,
			)
	}
}


func createImageSticker(
	input string,
	output string,
) (string, error) {

	filter := fmt.Sprintf(
		"scale=%d:%d:force_original_aspect_ratio=decrease,"+
			"pad=%d:%d:(ow-iw)/2:(oh-ih)/2:color=black@0,"+
			"format=rgba",
		config.StickerWidth,
		config.StickerHeight,
		config.StickerWidth,
		config.StickerHeight,
	)

	cmd := exec.Command(
		"ffmpeg",

		"-y",

		"-i",
		input,

		"-vf",
		filter,

		"-c:v",
		"libwebp",

		"-quality",
		strconv.Itoa(config.StickerQuality),

		"-compression_level",
		"6",

		"-preset",
		"picture",

		"-lossless",
		"0",

		"-an",

		"-pix_fmt",
		"yuva420p",

		output,
	)

	return executeFFmpeg(
		cmd,
		output,
	)
}


func createVideoSticker(
	input string,
	output string,
) (string, error) {

	filter := fmt.Sprintf(
		"scale=%d:%d:force_original_aspect_ratio=decrease,"+
			"pad=%d:%d:(ow-iw)/2:(oh-ih)/2:color=black@0,"+
			"fps=%d,"+
			"format=rgba",
		config.StickerWidth,
		config.StickerHeight,
		config.StickerWidth,
		config.StickerHeight,
		config.VideoFPS,
	)

	cmd := exec.Command(
		"ffmpeg",

		"-y",

		"-i",
		input,

		"-t",
		strconv.Itoa(config.VideoMaxDuration),

		"-vf",
		filter,

		"-c:v",
		"libwebp",

		"-quality",
		strconv.Itoa(config.VideoQuality),

		"-compression_level",
		"6",

		"-preset",
		"default",

		"-loop",
		"0",

		"-an",

		"-pix_fmt",
		"yuva420p",

		output,
	)

	return executeFFmpeg(
		cmd,
		output,
	)
}


func executeFFmpeg(
	cmd *exec.Cmd,
	output string,
) (string, error) {

	result, err := cmd.CombinedOutput()

	if err != nil {

		logger.Error(
			"Erro na conversão:",
			string(result),
		)

		return "",
			fmt.Errorf(
				"erro ao converter: %v - %s",
				err,
				string(result),
			)
	}

	logger.Success(
		"Sticker convertido:",
		output,
	)

	return output, nil
}
