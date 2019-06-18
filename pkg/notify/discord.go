package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/sdslabs/beastv4/core/config"
	log "github.com/sirupsen/logrus"
)

type DiscordNotificationProvider struct {
	DiscordWebHookURL string
	PostPayload       PostPayload
}

func (d *DiscordNotificationProvider) SendNotification() error {
	if d.DiscordWebHookURL == "" {
		return fmt.Errorf("Need a WebHookURL to send notification.")
	}

	if d.PostPayload.Channel == "" || d.PostPayload.Username == "" {
		return fmt.Errorf("Username and Channel required to send the notification.")
	}

	payload, err := json.Marshal(d.PostPayload)
	if err != nil {
		return fmt.Errorf("Error while converting payload to JSON : %s", err)
	}

	payloadReader := bytes.NewReader(payload)
	req, err := http.NewRequest("POST", d.DiscordWebHookURL, payloadReader)
	if err != nil {
		return fmt.Errorf("Error while connecting to webhook url host : %s", err)
	}

	req.Header.Set("Content-Type", "application/json")
	client := http.Client{}
	_, err = client.Do(req)

	if err != nil {
		return fmt.Errorf("Error while posting payload for notification : %s", err)
	}

	return nil
}

func SendNotificationToDiscord(nType NotificationType, msg string) error {
	if config.Cfg.DiscordWebHookURL == "" {
		log.Warnf("No discord webhook url provided in beast config, cannot send notification.")
		return fmt.Errorf("No webhook URL in beast config.")
	}

	discordNotifier := NewNotifier(config.Cfg.DiscordWebHookURL, "discord")
	discordNotifier.PostPayload = PostPayload{
		Username: "Beast",
		IconUrl:  "https://i.ibb.co/sjC5dRY/beast-eye-39371.png",
		Channel:  "#beast",
	}

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

	discordNotifier.PostPayload.Attachments = []Attachment{nAttachment}
	err := discordNotifier.SendNotification()
	if err != nil {
		log.Errorf("Error while sending notification to discord : %s", err)
		return fmt.Errorf("NOTIFICATION_SEND_ERROR: %s", err)
	}

	log.Infof("Notfication sent to discord.")
	return nil
}
