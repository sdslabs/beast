package notify

import (
	"fmt"
	"time"

	"github.com/sdslabs/beastv4/core/config"
	log "github.com/sirupsen/logrus"
)

func NewDiscordNotifier(webhookUrl string) *Notifier {
	return &Notifier{
		WebHookURL: webhookUrl + "/slack",
	}
}

func SendNotificationToDiscord(nType NotificationType, msg string) error {
	if config.Cfg.DiscordWebHookURL == "" {
		log.Warnf("No discord webhook url provided in beast config, cannot send notification.")
		return fmt.Errorf("No webhook URL in beast config.")
	}

	discordNotifier := NewDiscordNotifier(config.Cfg.DiscordWebHookURL)
	discordNotifier.PostPayload = PostPayload{
		Username: "Beast",
		IconUrl:  "https://i.ibb.co/sjC5dRY/beast-eye-39371.png",
		Channel:  "#beast",
	}

	nAttachment := Attachment{
		AuthorName: "Beast Notifier",
		AuthorLink: "https://backdoor.sdslabs.co",
		Footer:     "Beast Discord API",
		FooterIcon: "https://platform.discord-edge.com/img/default_application_icon.png",
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
