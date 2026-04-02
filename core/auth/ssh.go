package auth

import (
	"bytes"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"path/filepath"

	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/database"
	"github.com/sdslabs/beastv4/templates"
	"github.com/sdslabs/beastv4/utils"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

// This function takes in the path to the authorized keys file and parses it
// producing a map with ssh-public key as key and options corresponding to that
// as the value to corresponding key in the map.
func ParseAuthorizedKeysFile(filePath string) (map[string][]string, error) {
	authorizedKeysMap := map[string][]string{}
	err := utils.ValidateFileExists(filePath)
	if err != nil {
		log.Error("Error while validating authorized_keys file path")
		return authorizedKeysMap, err
	}

	authorizedKeysBytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		eMsg := fmt.Errorf("Failed to load authorized_keys, err: %v", err)
		log.Errorf("Error : %s", eMsg)
		return authorizedKeysMap, eMsg
	}

	var eMsg error

	for len(authorizedKeysBytes) > 0 {
		pubKey, _, options, rest, err := ssh.ParseAuthorizedKey(authorizedKeysBytes)
		if err != nil {
			eMsg = fmt.Errorf("Error while parsing authorized_keys file : %s", err)
			log.Error(eMsg.Error())
			break
		}

		authorizedKeysMap[string(pubKey.Marshal())] = options
		authorizedKeysBytes = rest
	}

	return authorizedKeysMap, eMsg
}

// This function parses ssh Private Key
func ParsePrivateKey(keyFile string) (*rsa.PrivateKey, error) {
	keyString, err := ioutil.ReadFile(keyFile)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(keyString)
	if block == nil {
		return nil, errors.New("Unable to decode")
	}
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	return key, nil
}

// This function disables the SSH based container access to all the current users
func DisableUserSSH() {
	users, err := database.QueryAllUsers()
	if err != nil {
		log.Errorf("DB ERROR : %v", err)
		return
	}
	for _, user := range users {
		SHA256 := sha256.New()
		SHA256.Write([]byte(user.Email))
		scriptPath := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_SCRIPTS_DIR, fmt.Sprintf("%x", SHA256.Sum(nil)))

		data := database.ScriptFile{
			User: user.Name,
		}

		var script bytes.Buffer
		scriptTemplate, err := template.New("script").Parse(templates.SSH_RESTRICT_LOGIN_SCRIPT_TEMPLATE)
		if err != nil {
			log.Errorf("Error while parsing script template :: %v", err)
			continue
		}

		if err = scriptTemplate.Execute(&script, data); err != nil {
			log.Errorf("Error while executing script template :: %v", err)
			continue
		}

		if err = ioutil.WriteFile(scriptPath, script.Bytes(), 0755); err != nil {
			log.Errorf("Error while writing to the script file :: %v", err)
			continue
		}
	}
}
