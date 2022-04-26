package handlers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gitlab.com/TitanInd/lumerin/cmd/msgbus"
	"gitlab.com/TitanInd/lumerin/cmd/msgbus/msgdata"
)

func ConfigsGET(config *msgdata.ConfigInfoRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		results := config.GetAllConfigInfos()
		c.JSON(http.StatusOK, results)
	}
}

func ConfigGET(config *msgdata.ConfigInfoRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		result, err := config.GetConfigInfo(id)
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		c.JSON(http.StatusOK, result)
	}
}

func ConfigPOST(config *msgdata.ConfigInfoRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		var requestBody []msgdata.ConfigInfoJSON
		if err := c.ShouldBindJSON(&requestBody); err != nil {
			c.Status(http.StatusUnprocessableEntity)
			return
		}
		for i := range requestBody {
			config.AddConfigInfo(requestBody[i])
			configMsg := msgdata.ConvertConfigInfoJSONtoConfigInfoMSG(requestBody[i])
			_, err := config.Ps.PubWait(msgbus.ConfigMsg, msgbus.IDString(configMsg.ID), configMsg)
			if err != nil {
				log.Printf("Config POST request failed to update msgbus: %s", err)
			}
		}

		c.Status(http.StatusOK)
	}
}

func ConfigPUT(config *msgdata.ConfigInfoRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var requestBody msgdata.ConfigInfoJSON
		if err := c.ShouldBindJSON(&requestBody); err != nil {
			c.Status(http.StatusUnprocessableEntity)
			return
		}

		if err := config.UpdateConfigInfo(id, requestBody); err != nil {
			c.Status(http.StatusNotFound)
			return
		}

		configMsg := msgdata.ConvertConfigInfoJSONtoConfigInfoMSG(requestBody)
		_, err := config.Ps.SetWait(msgbus.ConfigMsg, msgbus.IDString(configMsg.ID), configMsg)
		if err != nil {
			log.Printf("Config PUT request failed to update msgbus: %s", err)
		}

		c.Status(http.StatusOK)
	}
}

func ConfigDELETE(config *msgdata.ConfigInfoRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		if err := config.DeleteConfigInfo(id); err != nil {
			c.Status(http.StatusNotFound)
			return
		}

		_, err := config.Ps.UnpubWait(msgbus.ConfigMsg, msgbus.IDString(id))
		if err != nil {
			log.Printf("Config DELETE request failed to update msgbus: %s", err)
		}

		c.Status(http.StatusOK)
	}
}
