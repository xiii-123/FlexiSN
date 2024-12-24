package websocket

import (
	"context"
	"github.com/gorilla/websocket"
	"log"
	"main/manager"
	"net/url"
	"time"
)

const (
	webSocketURL = "ws://localhost:8888/subscribe"
)

type WebSocketClient struct {
	conn *websocket.Conn
}

func RunWebSocket(ctx context.Context) {

	u, err := url.Parse(webSocketURL)
	if err != nil {
		log.Fatal("Error parsing URL: ", err)
	}

	client := &WebSocketClient{}
	err = client.connect(u)
	if err != nil {
		log.Fatal("Error connecting to WebSocket server: ", err)
	}
	defer client.conn.Close()

	// Start the heartbeat ticker
	heartbeatTicker := time.NewTicker(30 * time.Second)
	defer heartbeatTicker.Stop()

	// Start the read pump
	go client.readPump(ctx)

	// Main loop
	for {
		select {
		case <-heartbeatTicker.C:
			client.sendHeartbeat()
		case <-ctx.Done():
			log.Println("Context cancelled, closing connection...")
			client.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			return
		}
	}
}

func (c *WebSocketClient) connect(u *url.URL) error {
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return err
	}
	c.conn = conn
	c.sendSubscriptionMessage()
	return nil
}

func (c *WebSocketClient) sendSubscriptionMessage() {
	message := `{"address":"0a0f870f81376f77db1981f94f39b719f5eb3f7c","type":"data"}`
	err := c.conn.WriteMessage(websocket.TextMessage, []byte(message))
	if err != nil {
		log.Println("Error sending subscription message: ", err)
	}
	//log.Println("Sent subscription message")
}

func (c *WebSocketClient) sendHeartbeat() {
	message := `{"type":"heartbeat"}`
	err := c.conn.WriteMessage(websocket.TextMessage, []byte(message))
	if err != nil {
		log.Println("Error sending heartbeat: ", err)
	}
	//log.Println("Sent heartbeat")
}

func (c *WebSocketClient) readPump(ctx context.Context) {
	defer func() {
		c.conn.Close()
	}()

	for {
		select {
		case <-ctx.Done():
			log.Println("Context cancelled, closing connection...")
			return
		default:
			_, message, err := c.conn.ReadMessage()
			if err != nil {
				log.Println("Error reading message: ", err)
				return
			}
			log.Printf("Received message: %s\n", message)

			// parse data and build a fileTree
			metaData, err := ParseTxValue(string(message))
			if err != nil {
				log.Println("Error parsing message: ", err)
				break
			}

			// Persist the fileTree using sqlite
			err = manager.GetDBManager().SaveToMemory(string(metaData.RootHash), metaData)
			if err != nil {
				log.Println("Error saving to memory: ", err)
				break
			}
		}
	}
}
