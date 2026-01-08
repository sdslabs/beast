package sse

import (
	"github.com/sdslabs/beastv4/core/database"
)

type Client struct {
	Id         string
	NotifyChan chan database.Notification
}

func NewSseClient(Id string) *Client {
	return &Client{
		Id:         Id,
		NotifyChan: make(chan database.Notification, SSE_CLIENT_CHANNEL_BUFFER),
	}

}
