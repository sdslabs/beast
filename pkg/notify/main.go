package notify

import (
	"fmt"
	"net/url"
	"strings"

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
func NewNotifier(url *url.URL, ProviderType ProviderTypeEnum) Notifier {
	log.Info("Inside notifier: Webhook URL: " + url.String())

	switch ProviderType {
	case SlackProvider:
		return &SlackNotificationProvider{
			Request{
				WebHookURL: url.String(),
			},
		}
	case DiscordProvider:
		return &DiscordNotificationProvider{
			Request{
				WebHookURL: url.String() + "/slack",
			},
		}
	}
	return nil
}

func (req *Request) FillReqParams() error {
	req.PostPayload = PostPayload{
		Username: USERNAME,
		IconUrl:  ICON_URL,
		Channel:  CHANNEL_NAME,
	}
	return nil
}

func SendNotification(nType NotificationType, message string) error {
	var errs []string

	for _, webhook := range config.Cfg.NotificationWebhooks {
		if webhook.ServiceName != "" && webhook.Active == true && webhook.URL != "" {
			var provider ProviderTypeEnum

			url, err := url.ParseRequestURI(webhook.URL)
			if url == nil || err != nil {
				log.Info("Invalid notification webhook URL")
				errs = append(errs, fmt.Sprintf("Invalid URL for notifier webhook %s", webhook.URL))
				continue
			}

			switch webhook.ServiceName {
			case "slack":
				provider = SlackProvider
			case "discord":
				provider = DiscordProvider
			default:
				log.Errorf("Not a valid service name for notification.")
				errs = append(errs, fmt.Sprintf("Not a valid service name for notification %s", webhook.ServiceName))
				continue
			}
			notifier := NewNotifier(url, provider)

			err = notifier.SendNotification(nType, message)
			if err != nil {
				e := fmt.Sprintf("Error while sending notification to %s : %s", webhook.ServiceName, err)
				log.Errorf("NOTIFICATION_SEND_ERROR: %s", err)

				errs = append(errs, e)
				continue
			}

			log.Infof("Notfication sent to %s.", webhook.ServiceName)
		} else {
			log.Warnf("No %s webhook url provided in beast config, cannot send notification.", webhook.ServiceName)
			log.Errorf("Notification webhook configuration not valid")
		}
	}

	if len(errs) == 0 {
		return nil
	}

	return fmt.Errorf(strings.Join(errs, "\n"))
}
