package main

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/gorilla/websocket"
)

// 1 web server ktory nam vrati index html

type Message struct {
	// pozor na medzery, je to json, citlive na tieto shity
	Name    string `json:"name"`
	Content string `json:"content"`
}

type Chat struct {
	upgrader      websocket.Upgrader
	clients       map[chan Message]struct{}
	enterChannel  chan chan Message
	leaveChannel  chan chan Message
	messageChanel chan Message
}

func (c *Chat) Run() {
	// nekonecny cyklus
	for {
		select {
		case ch := <-c.enterChannel:
			c.clients[ch] = struct{}{}
		case ch := <-c.leaveChannel:
			delete(c.clients, ch)
		case msg := <-c.messageChanel:
			for ch := range c.clients {
				ch <- msg
			}
		}
	}
}

func (c *Chat) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := c.upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer conn.Close()

	channel := make(chan Message)

	go func() {
		for msg := range channel {
			conn.WriteJSON(msg)
		}
	}()

	c.enterChannel <- channel
	defer func() {
		c.leaveChannel <- channel
	}()

	msg := Message{}

	for {
		err := conn.ReadJSON(&msg)
		if err != nil {
			fmt.Println(err)
			return
		}
		c.messageChanel <- msg
	}
}

func main() {

	chat := &Chat{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		clients:       make(map[chan Message]struct{}),
		enterChannel:  make(chan chan Message),
		leaveChannel:  make(chan chan Message),
		messageChanel: make(chan Message),
	}
	go chat.Run()

	http.Handle("/socket", chat)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		f, err := os.Open("./index.html")
		if err != nil {
			fmt.Println(err)
		}
		defer f.Close()

		io.Copy(w, f)
	})
	// tymto sme spustili webovy server
	http.ListenAndServe(":8080", nil)

}
