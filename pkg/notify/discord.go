package notify

import (
	"encoding/json"
	"fmt"
	"time"
)

type DiscordNotificationProvider struct {
	Request
}

func (d *DiscordNotificationProvider) SendNotification(nType NotificationType, msg string) error {
	if d.Request.WebHookURL == "" {
		return fmt.Errorf("Need a WebHookURL to send notification.")
	}

	d.Request.FillReqParams()

	nAttachment := Attachment{
		AuthorName: "Beast Notifier",
		AuthorLink: "https://backdoor.sdslabs.co",
		Footer:     "Beast Discord API",
		FooterIcon: "https://discordapp.com/assets/e05ead6e6ebc08df9291738d0aa6986d.png",
		Timestamp:  time.Now().Unix(),
		Text:       msg,
	}

	switch nType {
	case Success:
		nAttachment.Color = SuccessColor
		nAttachment.Title = "Beast Deployment Success"
		break
	case Error:
		nAttachment.Color = ErrorColor
		nAttachment.Title = "Beast Deployment Error"
		break
	}

	d.Request.PostPayload.Attachments = []Attachment{nAttachment}

	if d.Request.PostPayload.Channel == "" || d.Request.PostPayload.Username == "" {
		return fmt.Errorf("Username and Channel required to send the notification.")
	}

	payload, err := json.Marshal(d.PostPayload)
	if err != nil {
		return fmt.Errorf("Error while converting payload to JSON : %s", err)
	}

	return postWebhookJSON(d.Request.WebHookURL, payload)
}
