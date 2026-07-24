package media

import waProto "go.mau.fi/whatsmeow/binary/proto"

type Type int

const (
    Image Type = iota
    Video
)

type Media struct {
    Type  Type

    Image *waProto.ImageMessage
    Video *waProto.VideoMessage
}
