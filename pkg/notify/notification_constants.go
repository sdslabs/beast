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

const (
	USERNAME     string = "Beast"
	ICON_URL     string = "https://i.ibb.co/sjC5dRY/beast-eye-39371.png"
	CHANNEL_NAME string = "#beast"
)
