package handlers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gitlab.com/TitanInd/lumerin/cmd/msgbus"
	"gitlab.com/TitanInd/lumerin/cmd/msgbus/msgdata"
)

func ContractManagerConfigsGET(contractConfig *msgdata.ContractManagerConfigRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		results := contractConfig.GetAllContractManagerConfigs()
		c.JSON(http.StatusOK, results)
	}
}

func ContractManagerConfigGET(contractConfig *msgdata.ContractManagerConfigRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		result, err := contractConfig.GetContractManagerConfig(id)
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		c.JSON(http.StatusOK, result)
	}
}

func ContractManagerConfigPOST(contractConfig *msgdata.ContractManagerConfigRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		var requestBody []msgdata.ContractManagerConfigJSON
		if err := c.ShouldBindJSON(&requestBody); err != nil {
			c.Status(http.StatusUnprocessableEntity)
			return
		}
		for i := range requestBody {
			contractConfig.AddContractManagerConfig(requestBody[i])
			contractConfigMsg := msgdata.ConvertContractManagerConfigJSONtoContractManagerConfigMSG(requestBody[i])
			_, err := contractConfig.Ps.PubWait(msgbus.ContractManagerConfigMsg, msgbus.IDString(contractConfigMsg.ID), contractConfigMsg)
			if err != nil {
				log.Printf("Contract Config POST request failed to update msgbus: %s", err)
			}
		}

		c.Status(http.StatusOK)
	}
}

func ContractManagerConfigPUT(contractConfig *msgdata.ContractManagerConfigRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var requestBody msgdata.ContractManagerConfigJSON
		if err := c.ShouldBindJSON(&requestBody); err != nil {
			c.Status(http.StatusUnprocessableEntity)
			return
		}

		if err := contractConfig.UpdateContractManagerConfig(id, requestBody); err != nil {
			c.Status(http.StatusNotFound)
			return
		}

		contractConfigMsg := msgdata.ConvertContractManagerConfigJSONtoContractManagerConfigMSG(requestBody)
		_, err := contractConfig.Ps.SetWait(msgbus.ContractManagerConfigMsg, msgbus.IDString(contractConfigMsg.ID), contractConfigMsg)
		if err != nil {
			log.Printf("Contract Config PUT request failed to update msgbus: %s", err)
		}

		c.Status(http.StatusOK)
	}
}

func ContractManagerConfigDELETE(contractConfig *msgdata.ContractManagerConfigRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		if err := contractConfig.DeleteContractManagerConfig(id); err != nil {
			c.Status(http.StatusNotFound)
			return
		}

		_, err := contractConfig.Ps.UnpubWait(msgbus.ContractManagerConfigMsg, msgbus.IDString(id))
		if err != nil {
			log.Printf("Contract Config DELETE request failed to update msgbus: %s", err)
		}

		c.Status(http.StatusOK)
	}
}
