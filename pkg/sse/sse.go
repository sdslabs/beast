package sse

import (
	"sync"

	"github.com/sdslabs/beastv4/core/database"
	log "github.com/sirupsen/logrus"
)

var (
	h *Hub
)

type Hub struct {
	data             map[string]*Client
	connect          chan *Client
	disconnect       chan *Client
	BroadcastChannel chan database.Notification

	quit chan struct{}
	done chan struct{}

	once sync.Once // only for safe closure
}

func Init() {
	h = &Hub{
		data:             make(map[string]*Client),
		connect:          make(chan *Client, SSE_CONNECT_BUFFER),
		disconnect:       make(chan *Client, SSE_DISCONNECT_BUFFER),
		BroadcastChannel: make(chan database.Notification, SSE_BROADCAST_BUFFER),

		quit: make(chan struct{}),
		done: make(chan struct{}), // Initialize the channel
	}
	go listen()
}

func listen() {
	defer close(h.done)

	log.Println("Hub started listening...")

	for {
		select {

		case user, ok := <-h.connect:
			if !ok {
				return
			}
			add(user)
		case user, ok := <-h.disconnect:
			if !ok {
				return
			}
			if client, exists := h.data[user.Id]; exists {
				close(client.NotifyChan)
				remove(user)
			}
		case notif, ok := <-h.BroadcastChannel:
			if !ok {
				return
			}
			log.Printf("Broadcasting notification: id: [%v] %s", notif.ID, notif.Title)
			for _, client := range h.data {
				select {
				case client.NotifyChan <- notif:
					log.Print("Notification sent to ", client.Id)
				default:
					// For now we are dropping the notifications if the buffer is full
					log.Printf("Skipping client %s (buffer full)", client.Id)
				}

			}
		case <-h.quit:
			log.Println("Hub shutting down...")

			for _, client := range h.data {
				close(client.NotifyChan)
			}
			log.Println("Hub shutdown complete. All clients disconnected.")
			return
		}
	}
}

func Shutdown() {
	h.once.Do(func() {
		close(h.quit) // send the signal <-h.quit in the select statement
		<-h.done
	})
}

func AddClient(user *Client) {
	select {
	case h.connect <- user:
	case <-h.quit:
	}
}

func RemoveClient(user *Client) {
	select {
	case h.disconnect <- user:
	case <-h.quit:
	}
}

func BroadcastNotification(notif database.Notification) {
	select {
	case h.BroadcastChannel <- notif:
	case <-h.quit:
	}
}

func add(user *Client) {
	h.data[user.Id] = user
}

func remove(user *Client) {
	delete(h.data, user.Id)
}
func count() int {
	return len(h.data)
}
