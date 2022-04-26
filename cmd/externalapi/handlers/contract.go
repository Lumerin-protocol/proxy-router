package handlers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gitlab.com/TitanInd/lumerin/cmd/msgbus"
	"gitlab.com/TitanInd/lumerin/cmd/msgbus/msgdata"
)

func ContractsGET(contract *msgdata.ContractRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		results := contract.GetAllContracts()
		c.JSON(http.StatusOK, results)
	}
}

func ContractGET(contract *msgdata.ContractRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		result, err := contract.GetContract(id)
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		c.JSON(http.StatusOK, result)
	}
}

func ContractPOST(contract *msgdata.ContractRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		var requestBody []msgdata.ContractJSON
		if err := c.ShouldBindJSON(&requestBody); err != nil {
			c.Status(http.StatusUnprocessableEntity)
			return
		}
		for i := range requestBody {
			contract.AddContract(requestBody[i])
			contractMsg := msgdata.ConvertContractJSONtoContractMSG(requestBody[i])
			_, err := contract.Ps.PubWait(msgbus.ContractMsg, msgbus.IDString(contractMsg.ID), contractMsg)
			if err != nil {
				log.Printf("Contract POST request failed to update msgbus: %s", err)
			}
		}

		c.Status(http.StatusOK)
	}
}

func ContractPUT(contract *msgdata.ContractRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var requestBody msgdata.ContractJSON
		if err := c.ShouldBindJSON(&requestBody); err != nil {
			c.Status(http.StatusUnprocessableEntity)
			return
		}

		if err := contract.UpdateContract(id, requestBody); err != nil {
			c.Status(http.StatusNotFound)
			return
		}

		contractMsg := msgdata.ConvertContractJSONtoContractMSG(requestBody)
		_, err := contract.Ps.SetWait(msgbus.ContractMsg, msgbus.IDString(contractMsg.ID), contractMsg)
		if err != nil {
			log.Printf("Contract PUT request failed to update msgbus: %s", err)
		}

		c.Status(http.StatusOK)
	}
}

func ContractDELETE(contract *msgdata.ContractRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		if err := contract.DeleteContract(id); err != nil {
			c.Status(http.StatusNotFound)
			return
		}

		_, err := contract.Ps.UnpubWait(msgbus.ContractMsg, msgbus.IDString(id))
		if err != nil {
			log.Printf("Contract DELETE request failed to update msgbus: %s", err)
		}

		c.Status(http.StatusOK)
	}
}
