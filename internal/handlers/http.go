package handlers

import (
	"context"
	"net/url"

	"github.com/gin-gonic/gin"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/interfaces"
)

type Proxy interface {
	ChangeDest(ctx context.Context, dest *url.URL) error
}

type HTTPHandler struct {
	proxy Proxy
	log   interfaces.ILogger
}

func NewHTTPHandler(proxy Proxy, log interfaces.ILogger) *HTTPHandler {
	return &HTTPHandler{
		proxy: proxy,
		log:   log,
	}
}

func (h *HTTPHandler) ChangeDest(ctx *gin.Context) {
	urlString := ctx.Query("dest")
	if urlString == "" {
		ctx.JSON(400, gin.H{"error": "empty destination"})
		return
	}
	dest, err := url.Parse(urlString)
	if err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}
	err = h.proxy.ChangeDest(context.Background(), dest)
	if err != nil {
		ctx.JSON(500, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(200, gin.H{"status": "ok"})
}
