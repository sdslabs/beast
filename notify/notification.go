package notify

type NotificationType int

const (
	Success NotificationType = iota
	Warning
	Error
)

type NotificationColor string

const (
	ErrorColor   NotificationColor = "#FF0000"
	WarningColor NotificationColor = "#FF4500"
	SuccessColor NotificationColor = "#32CD32"
)
