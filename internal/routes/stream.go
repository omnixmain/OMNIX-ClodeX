package routes

import (
	"EverythingSuckz/fsb/internal/bot"
	"EverythingSuckz/fsb/internal/utils"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gotd/td/tg"
	range_parser "github.com/quantumsheep/range-parser"
	"go.uber.org/zap"

	"github.com/gin-gonic/gin"
)

var log *zap.Logger

func (e *allRoutes) LoadHome(r *Route) {
	log = e.log.Named("Stream")
	defer log.Info("Loaded stream route")
	r.Engine.GET("/stream/:messageID", getStreamRoute)
	r.Engine.GET("/stream/:messageID/:hash", getStreamRoute)
	r.Engine.GET("/stream/:messageID/:hash/:filename", getStreamRoute)
	r.Engine.GET("/download/:messageID", getStreamRoute)
	r.Engine.GET("/download/:messageID/:hash", getStreamRoute)
	r.Engine.GET("/download/:messageID/:hash/:filename", getStreamRoute)
	r.Engine.GET("/view/:messageID", getStreamRoute)
	r.Engine.GET("/view/:messageID/:hash", getStreamRoute)
	r.Engine.GET("/view/:messageID/:hash/:filename", getStreamRoute)
	r.Engine.GET("/master.m3u8", masterPlaylistRoute)
}

func getStreamRoute(ctx *gin.Context) {
	w := ctx.Writer
	r := ctx.Request

	messageIDParm := ctx.Param("messageID")
	messageID, err := strconv.Atoi(messageIDParm)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	authHash := ctx.Query("hash")
	if authHash == "" {
		authHash = ctx.Param("hash")
	}
	if authHash == "" {
		http.Error(w, "missing hash param", http.StatusBadRequest)
		return
	}

	worker := bot.GetNextWorker()

	file, err := utils.FileFromMessage(ctx, worker.Client, messageID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	expectedHash := utils.PackFile(
		file.FileName,
		file.FileSize,
		file.MimeType,
		file.ID,
	)
	if !utils.CheckHash(authHash, expectedHash) {
		http.Error(w, "invalid hash", http.StatusBadRequest)
		return
	}

	// for photo messages
	if file.FileSize == 0 {
		res, err := worker.Client.API().UploadGetFile(ctx, &tg.UploadGetFileRequest{
			Location: file.Location,
			Offset:   0,
			Limit:    1024 * 1024,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		result, ok := res.(*tg.UploadFile)
		if !ok {
			http.Error(w, "unexpected response", http.StatusInternalServerError)
			return
		}
		fileBytes := result.GetBytes()
		ctx.Header("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", file.FileName))
		if r.Method != "HEAD" {
			ctx.Data(http.StatusOK, file.MimeType, fileBytes)
		}
		return
	}

	ctx.Header("Accept-Ranges", "bytes")
	var start, end int64
	rangeHeader := r.Header.Get("Range")

	if rangeHeader == "" {
		start = 0
		end = file.FileSize - 1
		w.WriteHeader(http.StatusOK)
	} else {
		ranges, err := range_parser.Parse(file.FileSize, r.Header.Get("Range"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		start = ranges[0].Start
		end = ranges[0].End
		ctx.Header("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, file.FileSize))
		log.Info("Content-Range", zap.Int64("start", start), zap.Int64("end", end), zap.Int64("fileSize", file.FileSize))
		w.WriteHeader(http.StatusPartialContent)
	}

	contentLength := end - start + 1
	mimeType := file.MimeType

	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	ctx.Header("Content-Type", mimeType)
	ctx.Header("Content-Length", strconv.FormatInt(contentLength, 10))

	disposition := "inline"

	if ctx.Query("d") == "true" || strings.HasPrefix(r.URL.Path, "/download/") {
		disposition = "attachment"
	}

	ctx.Header("Content-Disposition", fmt.Sprintf("%s; filename=\"%s\"", disposition, file.FileName))

	if r.Method != "HEAD" {
		lr, _ := utils.NewTelegramReader(ctx, worker.Client, file.Location, start, end, contentLength)
		if _, err := io.CopyN(w, lr, contentLength); err != nil {
			log.Error("Error while copying stream", zap.Error(err))
		}
	}
}

func masterPlaylistRoute(ctx *gin.Context) {
	w := ctx.Writer
	r := ctx.Request

	// Generate M3U8 master playlist from query parameters
	// URL example: /master.m3u8?1080p=base64(path1)&720p=base64(path2)
	query := r.URL.Query()

	var m3u8 strings.Builder
	m3u8.WriteString("#EXTM3U\n")

	qualityMap := map[string]string{
		"2160p": "BANDWIDTH=15000000,RESOLUTION=3840x2160",
		"1080p": "BANDWIDTH=5000000,RESOLUTION=1920x1080",
		"720p":  "BANDWIDTH=3000000,RESOLUTION=1280x720",
		"480p":  "BANDWIDTH=1500000,RESOLUTION=854x480",
		"360p":  "BANDWIDTH=800000,RESOLUTION=640x360",
	}

	for key, values := range query {
		if len(values) == 0 {
			continue
		}
		
		quality := strings.ToLower(key)
		streamInfo, ok := qualityMap[quality]
		if !ok {
			streamInfo = "BANDWIDTH=1500000" // Default for unknown
		}

		decodedPathBytes, err := base64.RawURLEncoding.DecodeString(values[0])
		if err == nil {
			m3u8.WriteString(fmt.Sprintf("#EXT-X-STREAM-INF:%s\n%s\n", streamInfo, string(decodedPathBytes)))
		}
	}

	ctx.Header("Content-Type", "application/vnd.apple.mpegurl")
	ctx.Header("Content-Disposition", "attachment; filename=\"master.m3u8\"")
	ctx.String(http.StatusOK, m3u8.String())
}

