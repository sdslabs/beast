package notify

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
	discord ProviderTypeEnum = 1 + iota
	slack
)

//func (p ProviderTypeEnum) String() string {
//	return [...]string{"Discord", "Slack"}[p]
//}

func NewNotifier(URL string, ProviderType ProviderTypeEnum) Notifier {
	switch ProviderType {
	case slack:
		return &SlackNotificationProvider{
			Request{
				WebHookURL: URL,
			},
		}
	case discord:
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
