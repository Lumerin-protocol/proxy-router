package httphandlers

import (
	"github.com/Lumerin-protocol/proxy-router/internal/config"
	"github.com/gin-gonic/gin"
)

func (h *HTTPHandler) GetConfig(ctx *gin.Context) {
	ctx.JSON(200, ConfigResponse{
		Version:       config.BuildVersion,
		Commit:        config.Commit,
		Config:        h.config.GetSanitized(),
		DerivedConfig: h.derivedConfig,
	})
}
