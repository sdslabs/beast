package sse

const (
	// Represents the number of connection, disconnection and broadcast request queues in the Hub
	SSE_CONNECT_BUFFER    = 100
	SSE_DISCONNECT_BUFFER = 100
	SSE_BROADCAST_BUFFER  = 100
	// Represents the number of notifications queued to be send to the user
	SSE_CLIENT_CHANNEL_BUFFER = 100
)
