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

type DiscordPostPayload struct {
	Parse       string       `json:"parse,omitempty"`
	Username    string       `json:"username,omitempty"`
	IconUrl     string       `json:"icon_url,omitempty"`
	IconEmoji   string       `json:"icon_emoji,omitempty"`
	Channel     string       `json:"channel,omitempty"`
	Text        string       `json:"text,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
	Markdown    bool         `json:"mrkdwn,omitempty"`
}

type DiscordNotifier struct {
	WebHookURL         string
	DiscordPostPayload DiscordPostPayload
}

func NewDiscordNotifier(webhookUrl string) *DiscordNotifier {
	return &DiscordNotifier{
		WebHookURL: webhookUrl + "/slack",
	}
}

func (notifier *DiscordNotifier) SendNotification() error {
	if notifier.WebHookURL == "" {
		return fmt.Errorf("Need a WebHookURL to send notification.")
	}

	if notifier.DiscordPostPayload.Channel == "" || notifier.DiscordPostPayload.Username == "" {
		return fmt.Errorf("Username and Channel required to send the notification.")
	}

	payload, err := json.Marshal(notifier.DiscordPostPayload)
	if err != nil {
		return fmt.Errorf("Error while converting payload to JSON : %s", err)
	}

	payloadReader := bytes.NewReader(payload)
	req, err := http.NewRequest("POST", notifier.WebHookURL, payloadReader)
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

	discordNotifier := NewDiscordNotifier(config.Cfg.DiscordWebHookURL)
	discordNotifier.DiscordPostPayload = DiscordPostPayload{
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

	discordNotifier.DiscordPostPayload.Attachments = []Attachment{nAttachment}
	err := discordNotifier.SendNotification()
	if err != nil {
		log.Errorf("Error while sending notification to discord : %s", err)
		return fmt.Errorf("NOTIFICATION_SEND_ERROR: %s", err)
	}

	log.Infof("Notfication sent to discord.")
	return nil
}
