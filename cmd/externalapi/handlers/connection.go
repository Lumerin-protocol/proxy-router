package handlers

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gitlab.com/TitanInd/lumerin/cmd/msgbus"
	"gitlab.com/TitanInd/lumerin/cmd/msgbus/msgdata"

	"github.com/gorilla/websocket"
)

func ConnectionSTREAM(conn *msgdata.ConnectionRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		upgrader := websocket.Upgrader{
			EnableCompression: true,
			CheckOrigin: func(req *http.Request) bool {
				return true
			},
			ReadBufferSize:   1024,
			WriteBufferSize:  1024,
			HandshakeTimeout: time.Minute * 60,
		}

		ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Printf("Http failed to upgrade to a Websocket connection: %s", err)
			return
		}

		writer, err := ws.NextWriter(websocket.TextMessage)
		if err != nil {
			log.Printf("Websocket connection failed to open up writer: %s", err)
		}
		defer writer.Close()

		initialCxns := conn.GetAllConnections()

		writer.Write([]byte(fmt.Sprintf("{ type: 'cxns', connections: %v }", initialCxns)))

		// TODO: create real-time socket connection publisher logic
		// go func() {
		// 	for {
		// 		_, messageBytes, err := ws.ReadMessage()
		// 		if err != nil {
		// 			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
		// 				log.Printf("error: %v", err)
		// 			}
		// 			break
		// 		}
		// 		fmt.Println(messageBytes)

		// 		e := <-conn.EventChan

		// 		log.Printf("event: %v", e)
		// 		writer.Write([]byte("{ 'cxns': {  }}"))
		// 		fmt.Println(e)
		// 	}

		// }()
	}
}

func ConnectionsGET(conn *msgdata.ConnectionRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		results := conn.GetAllConnections()
		c.JSON(http.StatusOK, results)
	}
}

func ConnectionGET(conn *msgdata.ConnectionRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		result, err := conn.GetConnection(id)
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		c.JSON(http.StatusOK, result)
	}
}

func ConnectionPOST(conn *msgdata.ConnectionRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		var requestBody []msgdata.ConnectionJSON
		if err := c.ShouldBindJSON(&requestBody); err != nil {
			c.Status(http.StatusUnprocessableEntity)
			return
		}
		for i := range requestBody {
			conn.AddConnection(requestBody[i])
			connMsg := msgdata.ConvertConnectionJSONtoConnectionMSG(requestBody[i])
			_, err := conn.Ps.PubWait(msgbus.ConnectionMsg, msgbus.IDString(connMsg.ID), connMsg)
			if err != nil {
				log.Printf("Connection POST request failed to update msgbus: %s", err)
			}
		}

		c.Status(http.StatusOK)
	}
}

func ConnectionPUT(conn *msgdata.ConnectionRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var requestBody msgdata.ConnectionJSON
		if err := c.ShouldBindJSON(&requestBody); err != nil {
			c.Status(http.StatusUnprocessableEntity)
			return
		}

		if err := conn.UpdateConnection(id, requestBody); err != nil {
			c.Status(http.StatusNotFound)
			return
		}

		connMsg := msgdata.ConvertConnectionJSONtoConnectionMSG(requestBody)
		_, err := conn.Ps.SetWait(msgbus.ConnectionMsg, msgbus.IDString(connMsg.ID), connMsg)
		if err != nil {
			log.Printf("Connection PUT request failed to update msgbus: %s", err)
		}

		c.Status(http.StatusOK)
	}
}

func ConnectionDELETE(conn *msgdata.ConnectionRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		if err := conn.DeleteConnection(id); err != nil {
			c.Status(http.StatusNotFound)
			return
		}

		_, err := conn.Ps.UnpubWait(msgbus.ConnectionMsg, msgbus.IDString(id))
		if err != nil {
			log.Printf("Connection DELETE request failed to update msgbus: %s", err)
		}

		c.Status(http.StatusOK)
	}
}
