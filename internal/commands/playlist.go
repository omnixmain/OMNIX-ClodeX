package commands

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"

	"EverythingSuckz/fsb/config"
	"EverythingSuckz/fsb/internal/utils"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/dispatcher/handlers"
	"github.com/celestix/gotgproto/ext"
	"github.com/celestix/gotgproto/storage"
	"github.com/gotd/td/tg"
)

func (m *command) LoadPlaylist(dispatcher dispatcher.Dispatcher) {
	log := m.log.Named("playlist")
	defer log.Sugar().Info("Loaded")
	dispatcher.AddHandler(handlers.NewCommand("m3u8", createMasterPlaylist))
}

func createMasterPlaylist(ctx *ext.Context, u *ext.Update) error {
	chatId := u.EffectiveChat().GetID()
	peerChatId := ctx.PeerStorage.GetPeerById(chatId)
	if peerChatId.Type != int(storage.TypeUser) {
		return dispatcher.EndGroups
	}
	if len(config.ValueOf.AllowedUsers) != 0 && !utils.Contains(config.ValueOf.AllowedUsers, chatId) {
		ctx.Reply(u, "You are not allowed to use this bot.", nil)
		return dispatcher.EndGroups
	}

	// Command syntax: /m3u8 1080p:link1 720p:link2 480p:link3
	text := u.EffectiveMessage.Text
	parts := strings.Split(text, " ")
	if len(parts) < 2 {
		ctx.Reply(u, "⚠️ **Usage:** `/m3u8 <quality>:<link> <quality>:<link> ...`\n\n**Example:**\n`/m3u8 1080p:http://host/stream/... 720p:http://host/stream/...`", nil)
		return dispatcher.EndGroups
	}

	var qs []string

	for _, p := range parts[1:] {
		// e.g. 1080p:http://host/stream/123/hash/file.mp4
		sp := strings.SplitN(p, ":", 2)
		if len(sp) != 2 {
			continue
		}
		quality := sp[0]
		link := sp[1]

		// Extract the path from the link
		parsedUrl, err := url.Parse(link)
		if err != nil {
			continue
		}

		// Just take the path, e.g. /stream/123/hash/file.mp4
		linkPath := parsedUrl.Path
		// Base64 encode the path to avoid URL parameter parsing issues
		b64Path := base64.RawURLEncoding.EncodeToString([]byte(linkPath))
		qs = append(qs, fmt.Sprintf("%s=%s", quality, b64Path))
	}

	if len(qs) == 0 {
		ctx.Reply(u, "❌ Invalid links provided. Please make sure to follow the format: `<quality>:<link>`", nil)
		return dispatcher.EndGroups
	}

	masterLink := fmt.Sprintf("%s/master.m3u8?%s", config.ValueOf.Host, strings.Join(qs, "&"))

	response := fmt.Sprintf("✅ **Master M3U8 Playlist Generated!**\n\n🔗 %s\n\n⚡ **Powered by OMNIX ClodeX**", masterLink)

	row := tg.KeyboardButtonRow{
		Buttons: []tg.KeyboardButtonClass{
			&tg.KeyboardButtonURL{
				Text: "▶️ Play M3U8",
				URL:  masterLink,
			},
		},
	}
	markup := &tg.ReplyInlineMarkup{
		Rows: []tg.KeyboardButtonRow{row},
	}

	_, err := ctx.Reply(u, response, &ext.ReplyOpts{
		Markup:           markup,
		ReplyToMessageId: u.EffectiveMessage.ID,
		NoWebpage:        true,
	})
	if err != nil {
		utils.Logger.Sugar().Error(err)
		ctx.Reply(u, fmt.Sprintf("Error - %s", err.Error()), nil)
	}

	return dispatcher.EndGroups
}
