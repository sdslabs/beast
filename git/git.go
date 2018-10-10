package git

import (
	"fmt"
	"io/ioutil"

	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/utils"

	"golang.org/x/crypto/ssh"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	gitssh "gopkg.in/src-d/go-git.v4/plumbing/transport/ssh"
)

// This function returns the SSH auth for git interaction with the remote
// The paramter sshKeyFile is the path to the file containing the private
// key to be used during the transport.
func getSSHAuth(sshKeyFile string) (*gitssh.PublicKeys, error) {
	err := utils.ValidateFileExists(sshKeyFile)
	if err != nil {
		return nil, fmt.Errorf("The file %s does not exist", sshKeyFile)
	}

	pem, err := ioutil.ReadFile(sshKeyFile)
	if err != nil {
		return nil, fmt.Errorf("Error while reading ssh key file : %s", err)
	}

	signer, err := ssh.ParsePrivateKey(pem)
	if err != nil {
		return nil, fmt.Errorf("Error while parsing private key : %s", err)
	}

	auth := &gitssh.PublicKeys{
		User:   "git",
		Signer: signer,
	}

	return auth, nil
}

// Pull the git directory specified by gitDir using the provided sshKeyFile secret
// and branch.
func pull(gitDir string, sshKeyFile string, branch string) error {
	auth, err := getSSHAuth(sshKeyFile)
	if err != nil {
		return fmt.Errorf("Error while generating auth for git : %s", err)
	}

	r, err := git.PlainOpen(gitDir)
	if err != nil {
		return fmt.Errorf("Error while opening the path : %s", err)
	}

	w, err := r.Worktree()
	if err != nil {
		return fmt.Errorf("Error while getting Worktree for git directory : %s", err)
	}

	refName := plumbing.ReferenceName(branch)
	err = w.Pull(&git.PullOptions{
		RemoteName:    core.GIT_DEFAULT_REMOTE,
		ReferenceName: refName,
		SingleBranch:  true,
		Auth:          auth,
	})
	if err != nil {
		return fmt.Errorf("Error while pulling from remote branch %s : %s", branch, err)
	}

	return nil
}
