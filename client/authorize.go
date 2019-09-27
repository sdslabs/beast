package client

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)

type Response struct {
	Message   string `json:"message"`
	Challenge []byte `json:"challenge"`
	Token     string `json:"token"`
}

func Authorize(password string, host string, username string) {

	u, err := url.Parse("auth/login")
	if err != nil {
		fmt.Printf("Error while parsing url : %v", err)
		return
	}

	base, err := url.Parse(host)
	if err != nil {
		fmt.Printf("Error while parsing url : %v", err)
		return
	}

	res, err := http.PostForm(base.ResolveReference(u).String(), url.Values{
		"username": {username},
		"password": {password},
	})
	if err != nil {
		fmt.Printf("Error while making post request : %v", err)
		return
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Printf("Error while making post request : %v", err)
		return
	}

	if res.StatusCode != http.StatusOK {
		fmt.Printf("Response code : %v \nBody : %v", res.StatusCode, string(body))
		return
	}

	var response Response

	err = json.Unmarshal(body, &response)
	if err != nil {
		fmt.Printf("Error while parsing response : %v", err)
		return
	}

	fmt.Printf(`
The response:
Token 	: %v
Message	: %v
	`, response.Token, response.Message)

}
