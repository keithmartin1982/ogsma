package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

type Client struct {
	ID       string
	targetID string
	Conn     *websocket.Conn
	// SelfSigned Disables checking CA store for cert
	Addr        string
	SelfSigned  bool
	wsPath      string
	MessageChan chan []byte
}

type Msg struct {
	ID        string    `json:"id"` // id of target user
	Message   []byte    `json:"msg"`
	TimeStamp time.Time `json:"timestamp"`
	FromID    string    `json:"from"`
}

func (c *Client) Connect() error {
	var err error
	dd := websocket.DefaultDialer
	dd.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: c.SelfSigned,
	}
	dd.HandshakeTimeout = 10 * time.Second
	c.Conn, _, err = dd.Dial(fmt.Sprintf("wss://%s/%s", c.Addr, c.wsPath), nil)
	if err != nil {
		return fmt.Errorf("dial: %v", err)
	}
	c.listener()
	mts := struct {
		ID string `json:"id"`
	}{}
	mts.ID = c.ID
	loginJson, err := json.Marshal(mts)
	if err != nil {
		return fmt.Errorf("error: json marshal: %v", err)
	}
	if err = c.Conn.WriteMessage(websocket.BinaryMessage, loginJson); err != nil {
		return fmt.Errorf("write %v", err)
	}
	return err
}

func (c *Client) listener() {
	go func() {
		for {
			mt, message, err := c.Conn.ReadMessage()
			if err != nil {
				return
			}
			switch mt {
			case websocket.TextMessage:
				c.MessageChan <- message
			}
		}
	}()
}

func (c *Client) SendMsg(msg *Msg) error {
	jm, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("error: json marshal: %v", err)
	}
	return c.Conn.WriteMessage(websocket.TextMessage, jm)
}

func (c *Client) disconnect() {
	err := c.Conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		log.Println("write close:", err)
		return
	}
	if err := c.Conn.Close(); err != nil {
		log.Printf("error: websocket Conn close: %v", err)
	}
}
