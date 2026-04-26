package routes

import (
	"EverythingSuckz/fsb/config"
	"EverythingSuckz/fsb/internal/bot"
	"EverythingSuckz/fsb/internal/utils"
	"context"
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

	// Try searching with multiple strategies
	var msg *tg.Message

	// Strategy 1: Full Query
	msg, _ = searchInChannel(ctx, client.API(), peer, query)

	// Strategy 2: If failed and query has parts, try the last part (usually episode code)
	if msg == nil {
		parts := strings.Fields(query)
		if len(parts) > 1 {
			code := parts[len(parts)-1]
			// Only try if it looks like a code (e.g. S01, E01, Ep01)
			if strings.ContainsAny(strings.ToUpper(code), "SE") {
				msg, _ = searchInChannel(ctx, client.API(), peer, code)
				
				// Verify if the show name exists in the found file name to avoid wrong show matches
				if msg != nil {
					file, _ := utils.FileFromMedia(msg.Media)
					if file != nil {
						showName := strings.ToLower(strings.Join(parts[:len(parts)-1], " "))
						fileName := strings.ToLower(file.FileName)
						// If show name is not in file name, maybe it's a wrong match
						if !strings.Contains(fileName, showName) {
							msg = nil // Reject match
						}
					}
				}
			}
		}
	}

	if msg == nil || msg.Media == nil {
		ctx.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "no results found for query: " + query,
		})
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

func searchInChannel(ctx context.Context, api *tg.Client, peer *tg.InputChannel, query string) (*tg.Message, error) {
	// Try with Empty filter first to catch everything
	searchReq := &tg.MessagesSearchRequest{
		Peer:   &tg.InputPeerChannel{ChannelID: peer.ChannelID, AccessHash: peer.AccessHash},
		Q:      query,
		Filter: &tg.InputMessagesFilterEmpty{},
		Limit:  5,
	}

	res, err := api.MessagesSearch(ctx, searchReq)
	if err != nil {
		return nil, err
	}

	var messages []tg.MessageClass
	switch results := res.(type) {
	case *tg.MessagesChannelMessages:
		messages = results.Messages
	case *tg.MessagesMessages:
		messages = results.Messages
	case *tg.MessagesMessagesSlice:
		messages = results.Messages
	}

	for _, m := range messages {
		if msg, ok := m.(*tg.Message); ok {
			if msg.Media != nil {
				return msg, nil
			}
		}
	}
	return nil, nil
}
