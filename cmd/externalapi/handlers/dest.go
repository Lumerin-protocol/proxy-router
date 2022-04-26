package handlers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gitlab.com/TitanInd/lumerin/cmd/msgbus"
	"gitlab.com/TitanInd/lumerin/cmd/msgbus/msgdata"
)

func DestsGET(dest *msgdata.DestRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		results := dest.GetAllDests()
		c.JSON(http.StatusOK, results)
	}
}

func DestGET(dest *msgdata.DestRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		result, err := dest.GetDest(id)
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		c.JSON(http.StatusOK, result)
	}
}

func DestPOST(dest *msgdata.DestRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		var requestBody []msgdata.DestJSON
		if err := c.ShouldBindJSON(&requestBody); err != nil {
			c.Status(http.StatusUnprocessableEntity)
			return
		}
		for i := range requestBody {
			dest.AddDest(requestBody[i])
			destMsg := msgdata.ConvertDestJSONtoDestMSG(requestBody[i])
			_, err := dest.Ps.PubWait(msgbus.DestMsg, msgbus.IDString(destMsg.ID), destMsg)
			if err != nil {
				log.Printf("Dest POST request failed to update msgbus: %s", err)
			}
		}

		c.Status(http.StatusOK)
	}
}

func DestPUT(dest *msgdata.DestRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var requestBody msgdata.DestJSON
		if err := c.ShouldBindJSON(&requestBody); err != nil {
			c.Status(http.StatusUnprocessableEntity)
			return
		}

		if err := dest.UpdateDest(id, requestBody); err != nil {
			c.Status(http.StatusNotFound)
			return
		}

		destMsg := msgdata.ConvertDestJSONtoDestMSG(requestBody)
		_, err := dest.Ps.SetWait(msgbus.DestMsg, msgbus.IDString(destMsg.ID), destMsg)
		if err != nil {
			log.Printf("Dest PUT request failed to update msgbus: %s", err)
		}

		c.Status(http.StatusOK)
	}
}

func DestDELETE(dest *msgdata.DestRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		if err := dest.DeleteDest(id); err != nil {
			c.Status(http.StatusNotFound)
			return
		}

		_, err := dest.Ps.UnpubWait(msgbus.DestMsg, msgbus.IDString(id))
		if err != nil {
			log.Printf("Dest DELETE request failed to update msgbus: %s", err)
		}

		c.Status(http.StatusOK)
	}
}
