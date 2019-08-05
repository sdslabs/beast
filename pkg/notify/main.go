package notify

import (
	"fmt"

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

type Notifier interface {
	SendNotification(nType NotificationType, msg string) error
}

type Request struct {
	WebHookURL string
	PostPayload
}

type ProviderTypeEnum int

const (
	DiscordProvider ProviderTypeEnum = 1 + iota
	SlackProvider
)

//In the Discord notification provider it was using the same payload which was used for slack.
//By writing "/slack" in the discord WebHookURL, it execute Slack-Compatible Webhook
func NewNotifier(URL string, ProviderType ProviderTypeEnum) Notifier {
	switch ProviderType {
	case SlackProvider:
		return &SlackNotificationProvider{
			Request{
				WebHookURL: URL,
			},
		}
	case DiscordProvider:
		return &DiscordNotificationProvider{
			Request{
				WebHookURL: URL + "/slack",
			},
		}
	}
	return nil
}

func (req *Request) Post() error {
	req.PostPayload = PostPayload{
		Username: "Beast",
		IconUrl:  "https://i.ibb.co/sjC5dRY/beast-eye-39371.png",
		Channel:  "#beast",
	}
	return nil
}

func SendNotification(nType NotificationType, message string) error {
	if config.Cfg.SlackWebHookURL != "" {
		slackNotifier := NewNotifier(config.Cfg.SlackWebHookURL, SlackProvider)

		err := slackNotifier.SendNotification(nType, message)
		if err != nil {
			log.Errorf("Error while sending notification to slack : %s", err)
			fmt.Errorf("NOTIFICATION_SEND_ERROR: %s", err)
		}

		log.Infof("Notfication sent to slack.")
	} else {
		log.Warnf("No slack webhook url provided in beast config, cannot send notification.")
		fmt.Errorf("No webhook URL in beast config.")
	}

	if config.Cfg.DiscordWebHookURL != "" {
		discordNotifier := NewNotifier(config.Cfg.DiscordWebHookURL, DiscordProvider)

		err := discordNotifier.SendNotification(nType, message)
		if err != nil {
			log.Errorf("Error while sending notification to discord : %s", err)
			return fmt.Errorf("NOTIFICATION_SEND_ERROR: %s", err)
		}

		log.Infof("Notfication sent to discord.")
		return nil
	} else {
		log.Warnf("No discord webhook url provided in beast config, cannot send notification.")
		return fmt.Errorf("No webhook URL in beast config.")
	}
}
