package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type Response struct {
	Message   string `json:"message"`
	Challenge []byte `json:"challenge"`
	Token     string `json:"token"`
}

func Authorize(password string, host string, username string) error {
	base, err := url.Parse(host)
	if err != nil || base.Host == "" || (base.Scheme != "http" && base.Scheme != "https") {
		return fmt.Errorf("invalid Beast server URL")
	}

	httpClient := &http.Client{Timeout: 15 * time.Second}
	res, err := httpClient.PostForm(base.JoinPath("auth", "login").String(), url.Values{
		"username": {username},
		"password": {password},
	})
	if err != nil {
		return fmt.Errorf("request Beast token: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("Beast login returned %s", res.Status)
	}

	var response Response
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return fmt.Errorf("decode Beast login response: %w", err)
	}
	if response.Token == "" {
		return fmt.Errorf("Beast login response did not contain a token")
	}

	fmt.Printf("\nThe response:\nToken \t: %v\nMessage\t: %v\n", response.Token, response.Message)
	return nil
}
