package notify

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type Attachment struct {
	Fallback     *string   `json:"fallback"`
	Color        *string   `json:"color"`
	PreText      *string   `json:"pretext"`
	AuthorName   *string   `json:"author_name"`
	AuthorLink   *string   `json:"author_link"`
	AuthorIcon   *string   `json:"author_icon"`
	Title        *string   `json:"title"`
	TitleLink    *string   `json:"title_link"`
	Text         *string   `json:"text"`
	ImageUrl     *string   `json:"image_url"`
	Fields       []*Field  `json:"fields"`
	Footer       *string   `json:"footer"`
	FooterIcon   *string   `json:"footer_icon"`
	Timestamp    *int64    `json:"ts"`
	MarkdownIn   *[]string `json:"mrkdwn_in"`
	Actions      []*Action `json:"actions"`
	CallbackID   *string   `json:"callback_id"`
	ThumbnailUrl *string   `json:"thumb_url"`
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

	req, err := http.NewRequest("POST", notifier.WebHookURL, payload)
	if err != nil {
		return fmt.Errorf("Error while connecting to webhook url host : %s", err)
	}

	req.Header.Set("Content-Type", "application/json")
	client := http.Client{}
	_, err = client.Do(req)

	if err != nil {
		fmt.Errorf("Error while posting payload for notification : %s", err)
	}
}
