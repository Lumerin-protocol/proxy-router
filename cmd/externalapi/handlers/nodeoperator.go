package handlers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gitlab.com/TitanInd/lumerin/cmd/msgbus"
	"gitlab.com/TitanInd/lumerin/cmd/msgbus/msgdata"
)

func NodeOperatorsGET(nodeOperator *msgdata.NodeOperatorRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		results := nodeOperator.GetAllNodeOperators()
		c.JSON(http.StatusOK, results)
	}
}

func NodeOperatorGET(nodeOperator *msgdata.NodeOperatorRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		result, err := nodeOperator.GetNodeOperator(id)
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		c.JSON(http.StatusOK, result)
	}
}

func NodeOperatorPOST(nodeOperator *msgdata.NodeOperatorRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		var requestBody []msgdata.NodeOperatorJSON
		if err := c.ShouldBindJSON(&requestBody); err != nil {
			c.Status(http.StatusUnprocessableEntity)
			return
		}
		for i := range requestBody {
			nodeOperator.AddNodeOperator(requestBody[i])
			nodeOperatorMsg := msgdata.ConvertNodeOperatorJSONtoNodeOperatorMSG(requestBody[i])
			_, err := nodeOperator.Ps.PubWait(msgbus.NodeOperatorMsg, msgbus.IDString(nodeOperatorMsg.ID), nodeOperatorMsg)
			if err != nil {
				log.Printf("NodeOperator POST request failed to update msgbus: %s", err)
			}
		}

		c.Status(http.StatusOK)
	}
}

func NodeOperatorPUT(nodeOperator *msgdata.NodeOperatorRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var requestBody msgdata.NodeOperatorJSON
		if err := c.ShouldBindJSON(&requestBody); err != nil {
			c.Status(http.StatusUnprocessableEntity)
			return
		}

		if err := nodeOperator.UpdateNodeOperator(id, requestBody); err != nil {
			c.Status(http.StatusNotFound)
			return
		}

		nodeOperatorMsg := msgdata.ConvertNodeOperatorJSONtoNodeOperatorMSG(requestBody)
		_, err := nodeOperator.Ps.SetWait(msgbus.NodeOperatorMsg, msgbus.IDString(nodeOperatorMsg.ID), nodeOperatorMsg)
		if err != nil {
			log.Printf("NodeOperator PUT request failed to update msgbus: %s", err)
		}

		c.Status(http.StatusOK)
	}
}

func NodeOperatorDELETE(nodeOperator *msgdata.NodeOperatorRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		if err := nodeOperator.DeleteNodeOperator(id); err != nil {
			c.Status(http.StatusNotFound)
			return
		}

		_, err := nodeOperator.Ps.UnpubWait(msgbus.NodeOperatorMsg, msgbus.IDString(id))
		if err != nil {
			log.Printf("NodeOperator DELETE request failed to update msgbus: %s", err)
		}

		c.Status(http.StatusOK)
	}
}
