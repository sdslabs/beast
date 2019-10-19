package client

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/sdslabs/beastv4/core/auth"
	"github.com/sdslabs/beastv4/utils"
)

type Response struct {
	Message   string `json:"message"`
	Challenge []byte `json:"challenge"`
	Token     string `json:"token"`
}

func Authorize(keyFile string, host string, username string) {
	err := utils.ValidateFileExists(keyFile)
	if err != nil {
		fmt.Printf("File Location does not exist")
		return
	}

	key, err := auth.ParsePrivateKey(keyFile)
	if err != nil {
		fmt.Printf("Error while parsing private key : %v", err)
		return
	}

	u, err := url.Parse("auth/" + username)
	if err != nil {
		fmt.Printf("Error while parsing url : %v", err)
		return
	}

	base, err := url.Parse(host)
	if err != nil {
		fmt.Printf("Error while parsing url : %v", err)
		return
	}

	res, err := http.Get(base.ResolveReference(u).String())
	if err != nil {
		fmt.Printf("Error while making get request : %v", err)
		return
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Printf("Error while making get request : %v", err)
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

	if response.Challenge == nil {
		fmt.Printf("The response challenge is empty")
		return
	}

	decodedMessage, err := rsa.DecryptPKCS1v15(rand.Reader, key, []byte(response.Challenge))

	res, err = http.PostForm(base.ResolveReference(u).String(), url.Values{"decrmess": {string(decodedMessage)}})
	if err != nil {
		fmt.Printf("Error while making post request : %v", err)
		return
	}

	body, err = ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Printf("Error while making post request : %v", err)
		return
	}

	if res.StatusCode != http.StatusOK {
		fmt.Printf("Response code : %v \nBody : %v", res.StatusCode, string(body))
		return
	}

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
