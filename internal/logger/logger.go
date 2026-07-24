package logger

import (
	"fmt"
	"time"

	"whatsapp-sticker-bot/internal/config"
)

const appName = "StickerBot"

func timestamp() string {
	return time.Now().Format("15:04:05")
}

func log(level string, v ...interface{}) {

	if level == "DEBUG" && !config.Debug {
		return
	}

	prefix := fmt.Sprintf(
		"[%s] [%s] [%s]",
		timestamp(),
		appName,
		level,
	)

	fmt.Println(
		append(
			[]interface{}{prefix},
			v...,
		)...,
	)
}

func Info(v ...interface{}) {
	log("INFO", v...)
}

func Warn(v ...interface{}) {
	log("WARN", v...)
}

func Error(v ...interface{}) {
	log("ERROR", v...)
}

func Success(v ...interface{}) {
	log("SUCCESS", v...)
}

func Debug(v ...interface{}) {
	log("DEBUG", v...)
}
