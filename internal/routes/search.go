package routes

import (
	"EverythingSuckz/fsb/config"
	"EverythingSuckz/fsb/internal/bot"
	"EverythingSuckz/fsb/internal/utils"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gotd/td/tg"
)

func (e *allRoutes) LoadSearch(r *Route) {
	log = e.log.Named("Search")
	defer log.Info("Loaded search route")
	r.Engine.GET("/api/search", getSearchRoute)
}

func getSearchRoute(ctx *gin.Context) {
	query := ctx.Query("q")
	if query == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "missing query param q"})
		return
	}

	worker := bot.GetNextWorker()
	client := worker.Client

	peer, err := utils.GetLogChannelPeer(ctx, client.API(), client.PeerStorage)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get log channel peer: " + err.Error()})
		return
	}

	// Search for messages in the channel
	searchReq := &tg.MessagesSearchRequest{
		Peer:   &tg.InputPeerChannel{ChannelID: peer.ChannelID, AccessHash: peer.AccessHash},
		Q:      query,
		Filter: &tg.InputMessagesFilterDocument{},
		Limit:  1,
	}

	res, err := client.API().MessagesSearch(ctx, searchReq)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "search failed: " + err.Error()})
		return
	}

	var msg *tg.Message
	switch results := res.(type) {
	case *tg.MessagesChannelMessages:
		if len(results.Messages) > 0 {
			if m, ok := results.Messages[0].(*tg.Message); ok {
				msg = m
			}
		}
	case *tg.MessagesMessages:
		if len(results.Messages) > 0 {
			if m, ok := results.Messages[0].(*tg.Message); ok {
				msg = m
			}
		}
	case *tg.MessagesMessagesSlice:
		if len(results.Messages) > 0 {
			if m, ok := results.Messages[0].(*tg.Message); ok {
				msg = m
			}
		}
	}

	if msg == nil || msg.Media == nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "no results found or no media found in best match"})
		return
	}

	file, err := utils.FileFromMedia(msg.Media)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to extract file info: " + err.Error()})
		return
	}

	hash := utils.PackFile(
		file.FileName,
		file.FileSize,
		file.MimeType,
		file.ID,
	)
	shortHash := utils.GetShortHash(hash)

	host := config.ValueOf.Host
	if host == "" {
		scheme := "http"
		if ctx.Request.TLS != nil {
			scheme = "https"
		}
		host = fmt.Sprintf("%s://%s", scheme, ctx.Request.Host)
	}

	streamURL := fmt.Sprintf("%s/stream/%d?hash=%s", strings.TrimSuffix(host, "/"), msg.ID, shortHash)

	ctx.JSON(http.StatusOK, gin.H{
		"success":    true,
		"fileName":   file.FileName,
		"messageID":  msg.ID,
		"hash":       shortHash,
		"streamURL":  streamURL,
	})
}
