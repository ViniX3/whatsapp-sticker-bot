package handler

import (
        "fmt"
        "strings"
        "time"

        "whatsapp-sticker-bot/internal/auth"
        "whatsapp-sticker-bot/internal/gold"
        "whatsapp-sticker-bot/internal/logger"
        "whatsapp-sticker-bot/internal/media"
        "whatsapp-sticker-bot/internal/processor"
        "whatsapp-sticker-bot/internal/whatsapp"

        "go.mau.fi/whatsmeow"
        "go.mau.fi/whatsmeow/types/events"
)

func ProcessMessage(
        client *whatsmeow.Client,
        msgEvent *events.Message,
        botStartedAt time.Time,
) {

        // Ignora mensagens antigas
        if msgEvent.Info.Timestamp.Before(botStartedAt) {
                logger.Debug("Mensagem antiga ignorada")
                return
        }

        logger.Info("==============================")
        logger.Info("Mensagem recebida")

        // Aplica whitelist apenas para grupos
        if msgEvent.Info.IsGroup {
                groupID := msgEvent.Info.Chat.String()

                if !auth.IsGroupAllowed(groupID) {
                        logger.Debug("Grupo não autorizado:", groupID)
                        logger.Info("==============================")
                        return
                }

                logger.Info("Grupo autorizado:", groupID)
        }

        text := extractText(msgEvent)

        logger.Info("Texto recebido:", text)

        // =========================
        // Comando !gold
        // =========================
        if strings.EqualFold(strings.TrimSpace(text), "!gold") {

                jid := msgEvent.Info.Sender.String()
                name := msgEvent.Info.PushName

                logger.Info("Comando !gold recebido de:", jid)

                user, created, err := gold.GetOrCreateWallet(jid, name)
                if err != nil {
                        logger.Error("Erro ao acessar carteira Gold:", err)

                        _ = whatsapp.SendText(
                                client,
                                msgEvent.Info.Chat,
                                "❌ Erro ao acessar carteira Gold.",
                        )

                        logger.Info("==============================")
                        return
                }

                var response string

                if created {
                        response = fmt.Sprintf(`🪙 *Carteira Gold criada!*

Você recebeu *1000 Gold* iniciais.
Saldo atual: *%d Gold*`, user.Gold)

                        logger.Success("Nova carteira Gold criada para:", jid)
                } else {
                        response = fmt.Sprintf(`🪙 *Carteira Gold*

Saldo atual: *%d Gold*`, user.Gold)
                }

                _ = whatsapp.SendText(
                        client,
                        msgEvent.Info.Chat,
                        response,
                )

                logger.Info("Resposta Gold enviada para:", jid)
                logger.Info("==============================")
                return
        }

        // =========================
        // Comando !f (figurinha)
        // =========================
        if !IsStickerCommand(text) {
                logger.Debug("Sem comando reconhecido, ignorando")
                logger.Info("==============================")
                return
        }

        logger.Info("Comando !f recebido")

        mediaMessage := getMedia(msgEvent)

        if mediaMessage == nil {
                logger.Warn("Nenhuma mídia encontrada")
                logger.Info("==============================")
                return
        }

        processStickerCommand(client, msgEvent, mediaMessage)

        logger.Info("==============================")
}

func extractText(msg *events.Message) string {

        // Texto simples
        if msg.Message.GetConversation() != "" {
                return strings.TrimSpace(msg.Message.GetConversation())
        }

        // Texto expandido
        if msg.Message.ExtendedTextMessage != nil {
                return strings.TrimSpace(
                        msg.Message.GetExtendedTextMessage().GetText(),
                )
        }

        // Legenda de imagem
        if msg.Message.ImageMessage != nil {
                return strings.TrimSpace(
                        msg.Message.GetImageMessage().GetCaption(),
                )
        }

        // Legenda de vídeo
        if msg.Message.VideoMessage != nil {
                return strings.TrimSpace(
                        msg.Message.GetVideoMessage().GetCaption(),
                )
        }

        return ""
}

func getMedia(msg *events.Message) *media.Media {

        // Imagem enviada diretamente
        if msg.Message.ImageMessage != nil {
                return &media.Media{
                        Type:  media.Image,
                        Image: msg.Message.ImageMessage,
                }
        }

        // Vídeo enviado diretamente
        if msg.Message.VideoMessage != nil {
                return &media.Media{
                        Type:  media.Video,
                        Video: msg.Message.VideoMessage,
                }
        }

        // Mensagem respondida
        if msg.Message.ExtendedTextMessage != nil {
                contextInfo := msg.Message.ExtendedTextMessage.ContextInfo

                if contextInfo != nil && contextInfo.QuotedMessage != nil {

                        // Imagem respondida
                        if contextInfo.QuotedMessage.ImageMessage != nil {
                                return &media.Media{
                                        Type:  media.Image,
                                        Image: contextInfo.QuotedMessage.ImageMessage,
                                }
                        }

                        // Vídeo respondido
                        if contextInfo.QuotedMessage.VideoMessage != nil {
                                return &media.Media{
                                        Type:  media.Video,
                                        Video: contextInfo.QuotedMessage.VideoMessage,
                                }
                        }
                }
        }

        return nil
}

func processStickerCommand(
        client *whatsmeow.Client,
        msg *events.Message,
        mediaMessage *media.Media,
) {

        err := processor.ProcessSticker(
                client,
                msg.Info.Chat,
                mediaMessage,
        )

        if err != nil {
                logger.Error("Erro ao processar figurinha:", err)
                return
        }

        logger.Success("Processamento concluído")
}
