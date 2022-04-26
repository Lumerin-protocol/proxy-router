package handlers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gitlab.com/TitanInd/lumerin/cmd/msgbus"
	"gitlab.com/TitanInd/lumerin/cmd/msgbus/msgdata"
)

func MinersGET(miner *msgdata.MinerRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		results := miner.GetAllMiners()
		c.JSON(http.StatusOK, results)
	}
}

func MinerGET(miner *msgdata.MinerRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		result, err := miner.GetMiner(id)
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		c.JSON(http.StatusOK, result)
	}
}

func MinerPOST(miner *msgdata.MinerRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		var requestBody []msgdata.MinerJSON
		if err := c.ShouldBindJSON(&requestBody); err != nil {
			c.Status(http.StatusUnprocessableEntity)
			return
		}
		for i := range requestBody {
			miner.AddMiner(requestBody[i])
			minerMsg := msgdata.ConvertMinerJSONtoMinerMSG(requestBody[i])
			_, err := miner.Ps.PubWait(msgbus.MinerMsg, msgbus.IDString(minerMsg.ID), minerMsg)
			if err != nil {
				log.Printf("Miner POST request failed to update msgbus: %s", err)
			}
		}

		c.Status(http.StatusOK)
	}
}

func MinerPUT(miner *msgdata.MinerRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var requestBody msgdata.MinerJSON
		if err := c.ShouldBindJSON(&requestBody); err != nil {
			c.Status(http.StatusUnprocessableEntity)
			return
		}

		if err := miner.UpdateMiner(id, requestBody); err != nil {
			c.Status(http.StatusNotFound)
			return
		}

		minerMsg := msgdata.ConvertMinerJSONtoMinerMSG(requestBody)
		_, err := miner.Ps.SetWait(msgbus.MinerMsg, msgbus.IDString(minerMsg.ID), minerMsg)
		if err != nil {
			log.Printf("Miner PUT request failed to update msgbus: %s", err)
		}

		c.Status(http.StatusOK)
	}
}

func MinerDELETE(miner *msgdata.MinerRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		if err := miner.DeleteMiner(id); err != nil {
			c.Status(http.StatusNotFound)
			return
		}

		_, err := miner.Ps.UnpubWait(msgbus.MinerMsg, msgbus.IDString(id))
		if err != nil {
			log.Printf("Miner DELETE request failed to update msgbus: %s", err)
		}

		c.Status(http.StatusOK)
	}
}
