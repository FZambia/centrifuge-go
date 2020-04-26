// Simple Go chat client for https://github.com/centrifugal/centrifuge/tree/master/examples/events example.
package main

import (
	"bufio"
	"encoding/json"
	"log"
	"os"

	"github.com/centrifugal/centrifuge-go"
	"github.com/dgrijalva/jwt-go"
)

// Actually in real life clients should never know secret key.
// This is only for example purposes to quickly generate JWT for
// connection.
const exampleTokenHmacSecret = "secret"

func connToken(user string, exp int64) string {
	// NOTE that JWT must be generated on backend side of your application!
	// Here we are generating it on client side only for example simplicity.
	claims := jwt.MapClaims{"sub": user}
	if exp > 0 {
		claims["exp"] = exp
	}
	t, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(exampleTokenHmacSecret))
	if err != nil {
		panic(err)
	}
	return t
}

// ChatMessage is chat app specific message struct.
type ChatMessage struct {
	Input string `json:"input"`
}

type eventHandler struct{}

func (h *eventHandler) OnConnect(c *centrifuge.Client, e centrifuge.ConnectEvent) {
	log.Printf("Connected to chat with ID %s", e.ClientID)
	return
}

func (h *eventHandler) OnError(c *centrifuge.Client, e centrifuge.ErrorEvent) {
	log.Printf("Error: %s", e.Message)
	return
}

func (h *eventHandler) OnDisconnect(c *centrifuge.Client, e centrifuge.DisconnectEvent) {
	log.Printf("Disconnected from chat: %s", e.Reason)
	return
}

func (h *eventHandler) OnPublish(sub *centrifuge.Subscription, e centrifuge.PublishEvent) {
	var chatMessage *ChatMessage
	err := json.Unmarshal(e.Data, &chatMessage)
	if err != nil {
		return
	}
	log.Printf("Someone says: %s", chatMessage.Input)
}

func (h *eventHandler) OnJoin(sub *centrifuge.Subscription, e centrifuge.JoinEvent) {
	log.Printf("Someone joined: user id %s, client id %s", e.User, e.Client)
}

func (h *eventHandler) OnLeave(sub *centrifuge.Subscription, e centrifuge.LeaveEvent) {
	log.Printf("Someone left: user id %s, client id %s", e.User, e.Client)
}

func (h *eventHandler) OnSubscribeSuccess(sub *centrifuge.Subscription, e centrifuge.SubscribeSuccessEvent) {
	log.Printf("Subscribed on channel %s, resubscribed: %v, recovered: %v", sub.Channel(), e.Resubscribed, e.Recovered)
}

func (h *eventHandler) OnSubscribeError(sub *centrifuge.Subscription, e centrifuge.SubscribeErrorEvent) {
	log.Printf("Subscribed on channel %s failed, error: %s", sub.Channel(), e.Error)
}

func (h *eventHandler) OnUnsubscribe(sub *centrifuge.Subscription, e centrifuge.UnsubscribeEvent) {
	log.Printf("Unsubscribed from channel %s", sub.Channel())
}

func main() {
	url := "ws://localhost:8000/connection/websocket?format=protobuf"

	log.Printf("Connect to %s\n", url)
	log.Printf("Print something and press ENTER to send\n")

	c := centrifuge.New(url, centrifuge.DefaultConfig())
	// Uncomment to make it work with Centrifugo and JWT auth.
	//c.SetToken(connToken("49", 0))
	defer c.Close()
	handler := &eventHandler{}
	c.OnConnect(handler)
	c.OnError(handler)
	c.OnDisconnect(handler)

	sub, err := c.NewSubscription("chat:index")
	if err != nil {
		log.Fatalln(err)
	}

	sub.OnPublish(handler)
	sub.OnJoin(handler)
	sub.OnLeave(handler)
	sub.OnSubscribeSuccess(handler)
	sub.OnSubscribeError(handler)
	sub.OnUnsubscribe(handler)

	err = sub.Subscribe()
	if err != nil {
		log.Fatalln(err)
	}

	err = c.Connect()
	if err != nil {
		log.Fatalln(err)
	}

	// Read input from stdin.
	go func(sub *centrifuge.Subscription) {
		reader := bufio.NewReader(os.Stdin)
		for {
			text, _ := reader.ReadString('\n')
			msg := &ChatMessage{
				Input: text,
			}
			data, _ := json.Marshal(msg)
			err := sub.Publish(data)
			if err != nil {
				log.Printf("publish error: %v", err)
			}
		}
	}(sub)

	// Run until CTRL+C.
	select {}
}
