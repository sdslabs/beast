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
	clients          Clients
	connect          chan SseClient
	disconnect       chan SseClient
	BroadcastChannel chan database.Notification
}

func Init() {
	h = &Hub{
		clients: Clients{
			data: make(map[string]SseClient),
			mu:   sync.Mutex{},
		},
		connect:          make(chan SseClient), // TODO: Make them Buffered to avoid blocking due to bad internet speed
		disconnect:       make(chan SseClient),
		BroadcastChannel: make(chan database.Notification),
	}
	go Listen()
	log.Debug("SSE hub initialized....")

}

func Listen() {
	for {
		select {
		case user := <-h.connect:
			h.clients.Add(user)
			log.Print("New client connected: ", user.Id)
			log.Print("Num client: ", h.clients.Count())
		case user := <-h.disconnect:
			close(user.NotifyChan)
			h.clients.Remove(user)
			log.Print("Client disconnected: ", user.Id)
			log.Print("Num client: ", h.clients.Count())
		case notif := <-h.BroadcastChannel:
			log.Print("Broadcasting notification: ", notif)
			for _, client := range h.clients.Clients().data {
				client.NotifyChan <- notif
				log.Print("Notification sent to ", client.Id)
			}
			// send notifications
		}
	}
}

func Close() {
	close(h.connect)
	close(h.disconnect)
	close(h.BroadcastChannel)
}

func BroadcastChannel() chan database.Notification {
	return h.BroadcastChannel
}

func AddClient(user SseClient) {
	h.connect <- user
}

func RemoveClient(user SseClient) {
	h.disconnect <- user
	// handle connection cleanup in the main loop
}

func BroadcastNotification(notif database.Notification) {
	h.BroadcastChannel <- notif
}
