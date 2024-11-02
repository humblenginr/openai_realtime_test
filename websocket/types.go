package websocket

import (
	"fmt"

	"github.com/gorilla/websocket"
)

type MessageType string

const (
	ResponseAudioDeltaMsgType MessageType = "response.audio.delta"
	ResponseAudioDoneMsgType  MessageType = "response.audio.done"
)

// Message defines the structure of incoming messages
type Message struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

type WebsocketClient struct {
	c *websocket.Conn
}

func (wc *WebsocketClient) SendResponseAudioDeltaMessage(base64pcm16audio string) error {
	msg := Message{Type: string(ResponseAudioDeltaMsgType), Data: base64pcm16audio}
	if wc == nil {
		return fmt.Errorf("Websocket connection is not present")

	}
	return wc.c.WriteJSON(msg)
}

func (wc *WebsocketClient) SendResponseAudioDoneMessage() error {
	msg := Message{Type: string(ResponseAudioDoneMsgType), Data: ""}
	if wc == nil {
		return fmt.Errorf("Websocket connection is not present")

	}
	return wc.c.WriteJSON(msg)
}

func (wc *WebsocketClient) handleResponseAudio(respDeltaCh chan string, respDoneCh chan bool) {
	for {
		select {
		case delta := <-respDeltaCh:
			err := wc.SendResponseAudioDeltaMessage(delta)
			if err != nil {
				fmt.Println(err)
			}
		case <-respDoneCh:
			err := wc.SendResponseAudioDoneMessage()
			if err != nil {
				fmt.Println(err)
			}

		}

	}
}
