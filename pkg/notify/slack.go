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

type SlackNotificationProvider struct {
	Request
}

func (s *SlackNotificationProvider) SendNotification(nType NotificationType, msg string) error {
	if s.Request.WebHookURL == "" {
		return fmt.Errorf("Need a WebHookURL to send notification.")
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
	s.Request.PostPayload.Attachments = []Attachment{nAttachment}

	if s.Request.PostPayload.Channel == "" || s.Request.PostPayload.Username == "" {
		return fmt.Errorf("Username and Channel required to send the notification.")
	}

	payload, err := json.Marshal(s.PostPayload)
	if err != nil {
		return fmt.Errorf("Error while converting payload to JSON : %s", err)
	}

	payloadReader := bytes.NewReader(payload)
	req, err := http.NewRequest("POST", s.Request.WebHookURL, payloadReader)
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

	slackNotifier := NewNotifier(config.Cfg.SlackWebHookURL, SlackProvider)

	err := slackNotifier.SendNotification(nType, msg)
	if err != nil {
		log.Errorf("Error while sending notification to slack : %s", err)
		return fmt.Errorf("NOTIFICATION_SEND_ERROR: %s", err)
	}

	log.Infof("Notfication sent to slack.")
	return nil
}
