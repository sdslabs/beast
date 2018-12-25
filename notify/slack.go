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

type Attachment struct {
	Fallback     string            `json:"fallback,omitempty"`
	Color        NotificationColor `json:"color,omitempty"`
	PreText      string            `json:"pretext,omitempty"`
	AuthorName   string            `json:"author_name,omitempty"`
	AuthorLink   string            `json:"author_link,omitempty"`
	AuthorIcon   string            `json:"author_icon,omitempty"`
	Title        string            `json:"title,omitempty"`
	TitleLink    string            `json:"title_link,omitempty"`
	Text         string            `json:"text,omitempty"`
	ImageUrl     string            `json:"image_url,omitempty"`
	Footer       string            `json:"footer,omitempty"`
	FooterIcon   string            `json:"footer_icon,omitempty"`
	Timestamp    int64             `json:"ts,omitempty"`
	MarkdownIn   []string          `json:"mrkdwn_in,omitempty"`
	CallbackID   string            `json:"callback_id,omitempty"`
	ThumbnailUrl string            `json:"thumb_url,omitempty"`
}

type SlackPostPayload struct {
	Parse       string       `json:"parse,omitempty"`
	Username    string       `json:"username,omitempty"`
	IconUrl     string       `json:"icon_url,omitempty"`
	IconEmoji   string       `json:"icon_emoji,omitempty"`
	Channel     string       `json:"channel,omitempty"`
	Text        string       `json:"text,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
	Markdown    bool         `json:"mrkdwn,omitempty"`
}

type SlackNotifier struct {
	WebHookURL       string
	SlackPostPayload SlackPostPayload
}

func NewSlackNotifier(webhookUrl string) *SlackNotifier {
	return &SlackNotifier{
		WebHookURL: webhookUrl,
	}
}

func (notifier *SlackNotifier) SendNotification() error {
	if notifier.WebHookURL == "" {
		return fmt.Errorf("Need a WebHookURL to send notification.")
	}

	if notifier.SlackPostPayload.Channel == "" || notifier.SlackPostPayload.Username == "" {
		return fmt.Errorf("Username and Channel required to send the notification.")
	}

	payload, err := json.Marshal(notifier.SlackPostPayload)
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

func SendNotificationToSlack(nType NotificationType, msg string) error {
	if config.Cfg.SlackWebHookURL == "" {
		log.Warnf("No slack webhook url provided in beast config, cannot send notification.")
		return fmt.Errorf("No webhook URL in beast config.")
	}

	slackNotifier := NewSlackNotifier(config.Cfg.SlackWebHookURL)
	slackNotifier.SlackPostPayload = SlackPostPayload{
		Username: "Beast",
		IconUrl:  "https://i.ibb.co/sjC5dRY/beast-eye-39371.png",
		Channel:  "#beast",
	}

	nAttachment := Attachment{
		AuthorName: "Beast Notifier",
		AuthorLink: "https://backdoor.sdslabs.co",
		Footer:     "Beast Slack API",
		FooterIcon: "https://platform.slack-edge.com/img/default_application_icon.png",
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

	slackNotifier.SlackPostPayload.Attachments = []Attachment{nAttachment}
	err := slackNotifier.SendNotification()
	if err != nil {
		log.Errorf("Error while sending notification to slack : %s", err)
		return fmt.Errorf("NOTIFICATION_SEND_ERROR: %s", err)
	}

	log.Infof("Notfication sent to slack.")
	return nil
}
