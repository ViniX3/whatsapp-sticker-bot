package handler


import "strings"



func IsStickerCommand(text string) bool {


	return strings.TrimSpace(text) == "!f"

}

