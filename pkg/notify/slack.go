package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type SlackNotificationProvider struct {
	Request
}

func (s *SlackNotificationProvider) SendNotification(nType NotificationType, msg string) error {
	if s.Request.WebHookURL == "" {
		return fmt.Errorf("Need a WebHookURL to send notification.")
	}

	s.Request.FillReqParams()

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
