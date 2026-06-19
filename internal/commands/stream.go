package commands

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"EverythingSuckz/fsb/config"
	"EverythingSuckz/fsb/internal/utils"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/dispatcher/handlers"
	"github.com/celestix/gotgproto/ext"
	"github.com/celestix/gotgproto/storage"
	"github.com/celestix/gotgproto/types"
	"github.com/gotd/td/telegram/message/styling"
	"github.com/gotd/td/tg"
)

func (m *command) LoadStream(dispatcher dispatcher.Dispatcher) {
	log := m.log.Named("start")
	defer log.Sugar().Info("Loaded")
	dispatcher.AddHandler(
		handlers.NewMessage(nil, sendLink),
	)
}

func supportedMediaFilter(m *types.Message) (bool, error) {
	if not := m.Media == nil; not {
		return false, dispatcher.EndGroups
	}
	switch m.Media.(type) {
	case *tg.MessageMediaDocument:
		return true, nil
	case *tg.MessageMediaPhoto:
		return true, nil
	case tg.MessageMediaClass:
		return false, dispatcher.EndGroups
	default:
		return false, nil
	}
}

var albumMu sync.Mutex
var albumLinks = make(map[string]map[string]string)

func sendLink(ctx *ext.Context, u *ext.Update) error {
	chatId := u.EffectiveChat().GetID()
	peerChatId := ctx.PeerStorage.GetPeerById(chatId)
	if peerChatId.Type != int(storage.TypeUser) && peerChatId.Type != int(storage.TypeChat) && peerChatId.Type != int(storage.TypeChannel) {
		return dispatcher.EndGroups
	}
	if len(config.ValueOf.AllowedUsers) != 0 && !utils.Contains(config.ValueOf.AllowedUsers, chatId) {
		ctx.Reply(u, "You are not allowed to use this bot.", nil)
		return dispatcher.EndGroups
	}
	supported, err := supportedMediaFilter(u.EffectiveMessage)
	if err != nil {
		return err
	}
	if !supported {
		if peerChatId.Type == int(storage.TypeUser) {
			ctx.Reply(u, "Sorry, this message type is unsupported.", nil)
		}
		return dispatcher.EndGroups
	}
	update, err := utils.ForwardMessages(ctx, chatId, config.ValueOf.LogChannelID, u.EffectiveMessage.ID)
	if err != nil {
		utils.Logger.Sugar().Error(err)
		ctx.Reply(u, fmt.Sprintf("Error - %s", err.Error()), nil)
		return dispatcher.EndGroups
	}
	messageID := update.Updates[0].(*tg.UpdateMessageID).ID
	doc := update.Updates[1].(*tg.UpdateNewChannelMessage).Message.(*tg.Message).Media
	file, err := utils.FileFromMedia(doc)
	if err != nil {
		ctx.Reply(u, fmt.Sprintf("Error - %s", err.Error()), nil)
		return dispatcher.EndGroups
	}
	fullHash := utils.PackFile(
		file.FileName,
		file.FileSize,
		file.MimeType,
		file.ID,
	)
	hash := utils.GetShortHash(fullHash)

	safeName := "file"
	if file.FileName != "" {
		safeName = url.PathEscape(file.FileName)
	}

	var linkType, actionText, actionEmoji string
	if strings.HasPrefix(file.MimeType, "video") {
		linkType = "stream"
		actionText = "Watch Now"
		actionEmoji = "▶️"
	} else if strings.HasPrefix(file.MimeType, "image") {
		linkType = "view"
		actionText = "View Now"
		actionEmoji = "🖼️"
	} else if strings.HasPrefix(file.MimeType, "audio") {
		linkType = "stream"
		actionText = "Listen Now"
		actionEmoji = "🎧"
	} else {
		linkType = "download"
		actionText = "Download Now"
		actionEmoji = "⬇️"
	}

	link := fmt.Sprintf("%s/%s/%d/%s/%s", config.ValueOf.Host, linkType, messageID, hash, safeName)
	dlLink := fmt.Sprintf("%s/download/%d/%s/%s", config.ValueOf.Host, messageID, hash, safeName)

	beautifiedName, quality, seasonEpisode := utils.BeautifyFileName(file.FileName)
	readableSize := utils.HumanReadableSize(file.FileSize)

	text := []styling.StyledTextOption{
		styling.Plain("📁 "), styling.Bold("File Name: "), styling.Plain(beautifiedName),
	}

	if seasonEpisode != "" {
		text = append(text, styling.Plain("\n🎞️ "), styling.Bold(seasonEpisode))
	}

	text = append(text,
		styling.Plain("\n💿 "), styling.Bold("Quality: "), styling.Code(quality),
		styling.Plain("\n⚖️ "), styling.Bold("Size: "), styling.Code(readableSize),
		styling.Plain(fmt.Sprintf("\n\n%s ", actionEmoji)), styling.Bold(fmt.Sprintf("%s:", actionText)),
		styling.Plain("\n🔗 "), styling.Code(link),
		styling.Plain("\n\n⚡ "), styling.Bold("Powered by OMNIX ClodeX"),
	)

	row := tg.KeyboardButtonRow{
		Buttons: []tg.KeyboardButtonClass{},
	}

	if linkType == "stream" || linkType == "view" {
		row.Buttons = append(row.Buttons, &tg.KeyboardButtonURL{
			Text: actionText,
			URL:  link,
		})
	}

	row.Buttons = append(row.Buttons, &tg.KeyboardButtonURL{
		Text: "Download",
		URL:  dlLink,
	})

	markup := &tg.ReplyInlineMarkup{
		Rows: []tg.KeyboardButtonRow{row},
	}
	if strings.Contains(link, "http://localhost") {
		_, err = ctx.Reply(u, text, &ext.ReplyOpts{
			NoWebpage:        false,
			ReplyToMessageId: u.EffectiveMessage.ID,
		})
	} else {
		_, err = ctx.Reply(u, text, &ext.ReplyOpts{
			Markup:           markup,
			NoWebpage:        false,
			ReplyToMessageId: u.EffectiveMessage.ID,
		})
	}
	if err != nil {
		utils.Logger.Sugar().Error(err)
		ctx.Reply(u, fmt.Sprintf("Error - %s", err.Error()), nil)
	}

	groupKey := fmt.Sprintf("%d_%s", chatId, beautifiedName)
	if linkType == "stream" {
		albumMu.Lock()
		isFirst := false
		if _, ok := albumLinks[groupKey]; !ok {
			albumLinks[groupKey] = make(map[string]string)
			isFirst = true
		}
		
		// Use short path (with a generic video extension) to keep master URL clean and player-compatible
		shortPath := fmt.Sprintf("/%s/%d/%s/video.mp4", linkType, messageID, hash)
		
		q := strings.ToLower(quality)
		if q == "unknown" {
			q = fmt.Sprintf("unknown_%d", len(albumLinks[groupKey]))
		}
		albumLinks[groupKey][q] = shortPath
		albumMu.Unlock()

		if isFirst {
			go func(gKey string, context *ext.Context, update *ext.Update) {
				// Wait for all parts to arrive (10 seconds should be plenty)
				time.Sleep(15 * time.Second)
				albumMu.Lock()
				links := albumLinks[gKey]
				delete(albumLinks, gKey)
				albumMu.Unlock()

				if len(links) > 1 {
					var masterText []styling.StyledTextOption
					masterText = append(masterText, styling.Plain("✅ "), styling.Bold("Auto-Generated Master M3U8 Playlist!\n\n"))

					var qs []string
					for q, l := range links {
						b64Path := base64.RawURLEncoding.EncodeToString([]byte(l))
						qs = append(qs, fmt.Sprintf("%s=%s", q, b64Path))
						
						displayQ := strings.ToUpper(q)
						if strings.HasPrefix(displayQ, "UNKNOWN_") {
							displayQ = "PART " + strings.TrimPrefix(displayQ, "UNKNOWN_")
						}
						
						fullLink := fmt.Sprintf("%s%s", config.ValueOf.Host, l)
						masterText = append(masterText, styling.Bold(displayQ), styling.Plain(" - "), styling.Code(fullLink), styling.Plain("\n"))
					}
					
					masterLink := fmt.Sprintf("%s/master.m3u8?%s", config.ValueOf.Host, strings.Join(qs, "&"))
					
					masterText = append(masterText, 
						styling.Plain("\n🎬 "), styling.Bold("Master URL"), styling.Plain(" - "), styling.Code(masterLink),
						styling.Plain("\n\n⚡ "), styling.Bold("Powered by OMNIX ClodeX"),
					)
					
					masterMarkup := &tg.ReplyInlineMarkup{
						Rows: []tg.KeyboardButtonRow{
							{
								Buttons: []tg.KeyboardButtonClass{
									&tg.KeyboardButtonURL{
										Text: "▶️ Play M3U8",
										URL:  masterLink,
									},
								},
							},
						},
					}

					context.Reply(update, masterText, &ext.ReplyOpts{
						Markup:           masterMarkup,
						ReplyToMessageId: update.EffectiveMessage.ID,
						NoWebpage:        true,
					})
				}
			}(groupKey, ctx, u)
		}
	}

	return dispatcher.EndGroups
}
