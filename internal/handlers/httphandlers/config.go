package httphandlers

import (
	"github.com/gin-gonic/gin"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/config"
)

func (h *HTTPHandler) GetConfig(ctx *gin.Context) {
	ctx.JSON(200, ConfigResponse{
		Version: config.BuildVersion,
		Config:  h.config.GetSanitized(),
	})
}
