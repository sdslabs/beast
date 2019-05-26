package git

import (
	"fmt"
	"io/ioutil"

	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	gitssh "gopkg.in/src-d/go-git.v4/plumbing/transport/ssh"
)

// This function returns the SSH auth for git interaction with the remote
// The paramter sshKeyFile is the path to the file containing the private
// key to be used during the transport.
func getSSHAuth(sshKeyFile string) (*gitssh.PublicKeys, error) {

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
func Pull(gitDir string, sshKeyFile string, branch string, remote string) error {
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

	refName := plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", branch))
	err = w.Pull(&git.PullOptions{
		RemoteName:    remote,
		ReferenceName: refName,
		SingleBranch:  true,
		Auth:          auth,
	})
	if err != nil {
		return fmt.Errorf("Error while pulling from remote branch %s : %s", branch, err)
	}

	log.Debugf("Git pull completed for %s", gitDir)
	return nil
}

// CLone the git repository to the specified git directory with the
// provided remote repo name.
// This function assumes that the arguments provided have been checked or
// validated earlier, for example gitDir is an empty directory or does not exist.
func Clone(gitDir string, sshKeyFile string, repoUrl string, branch string, remote string) error {
	auth, err := getSSHAuth(sshKeyFile)
	if err != nil {
		return fmt.Errorf("Error while generating auth for git : %s", err)
	}

	refName := plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", branch))
	log.Debugf("Performing clone for remote : %s & branch : %s", remote, refName)
	_, err = git.PlainClone(gitDir, false, &git.CloneOptions{
		URL:           repoUrl,
		Auth:          auth,
		RemoteName:    remote,
		ReferenceName: refName,
		SingleBranch:  true,
	})
	if err != nil {
		return fmt.Errorf("Error while cloning the repository : %s", err)
	}

	log.Debugf("Repository cloned to %s", gitDir)
	return nil
}
