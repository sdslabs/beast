package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
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

type PostPayload struct {
	Parse       string       `json:"parse,omitempty"`
	Username    string       `json:"username,omitempty"`
	IconUrl     string       `json:"icon_url,omitempty"`
	IconEmoji   string       `json:"icon_emoji,omitempty"`
	Channel     string       `json:"channel,omitempty"`
	Text        string       `json:"text,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
	Markdown    bool         `json:"mrkdwn,omitempty"`
}

type ProviderType struct {
	Slack   SlackNotificationProvider
	Discord DiscordNotificationProvider
}

type Notifier interface {
	SendNotification() error
	PostPayload
}

func NewNotifier(providerType ProviderType) Notifier {
	if providerType == ProviderType.Slack {
		return &SlackNotificationProvider{
			SlackWebHookURL: SlackWebHookURL,
		}
	} else if providerType == ProviderType.Discord {
		return &DiscordNotificationProvider{
			DiscordWebHookURL: SlackWebHookURL + "/slack",
		}
	}
	return nil
}

func (s *SlackNotificationProvider) SendNotification() error {
	if notifier.WebHookURL == "" {
		return fmt.Errorf("Need a WebHookURL to send notification.")
	}

	if notifier.PostPayload.Channel == "" || notifier.PostPayload.Username == "" {
		return fmt.Errorf("Username and Channel required to send the notification.")
	}

	payload, err := json.Marshal(notifier.PostPayload)
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

func (d *DiscordNotificationProvider) SendNotification() error {
	if notifier.WebHookURL == "" {
		return fmt.Errorf("Need a WebHookURL to send notification.")
	}

	if notifier.PostPayload.Channel == "" || notifier.PostPayload.Username == "" {
		return fmt.Errorf("Username and Channel required to send the notification.")
	}

	payload, err := json.Marshal(notifier.PostPayload)
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
