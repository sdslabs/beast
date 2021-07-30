package git

import (
	"fmt"
	"io/ioutil"

	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
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

// IsAlreadyUpToDate checks if the local repo is already up to date with the remote repo
func IsAlreadyUpToDate(gitDir string, sshKeyFile string, branch string, remoteName string) (bool, error) {
	auth, err := getSSHAuth(sshKeyFile)
	if err != nil {
		return false, fmt.Errorf("Error while generating auth for git: %s", err)
	}

	repo, err := git.PlainOpen(gitDir)
	if err != nil {
		return false, fmt.Errorf("Error while opening the path: %s", err)
	}

	refName := plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", branch))
	currRef, err := repo.Reference(refName, true)
	if err != nil {
		return false, fmt.Errorf("Error while getting reference from local repo: %s", err)
	}
	currHash := currRef.Hash()

	remote, err := repo.Remote(remoteName)
	if err != nil {
		return false, fmt.Errorf("Error while getting remote for git directory: %s", err)
	}

	refs, err := remote.List(&git.ListOptions{
		Auth: auth,
	})
	if err != nil {
		return false, fmt.Errorf("Error while listing remote references: %s", err)
	}

	for _, ref := range refs {
		if ref.Name() == refName {
			upToDate := ref.Hash() == currHash
			return upToDate, nil
		}
	}

	return false, fmt.Errorf("Reference %s not found in the remote repo", refName)
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

// getLatestCommit gets the latest commit (as pointed by the HEAD reference) from the repository
func getLatestCommit(repo *git.Repository) (*object.Commit, error) {
	headRef, err := repo.Head()
	if err != nil {
		return nil, fmt.Errorf("Error while getting HEAD reference of git repository")
	}

	latestCommit, err := repo.CommitObject(headRef.Hash())
	if err != nil {
		return nil, fmt.Errorf("Error while getting latest commit from git repository")
	}

	return latestCommit, nil
}

// PullAndGetChanges pulls changes from remote and returns an array of file names which were changed
func PullAndGetChanges(gitDir string, sshkeyFile string, branch string, remote string) ([]string, error) {
	auth, err := getSSHAuth(sshkeyFile)
	if err != nil {
		return nil, fmt.Errorf("Error while generating auth for git : %s", err)
	}

	repo, err := git.PlainOpen(gitDir)
	if err != nil {
		return nil, fmt.Errorf("Error while opening the path : %s", err)
	}

	oldCommit, err := getLatestCommit(repo)
	if err != nil {
		return nil, err
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("Error while getting Worktree for git directory : %s", err)
	}

	refName := plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", branch))
	err = worktree.Pull(&git.PullOptions{
		RemoteName:    remote,
		ReferenceName: refName,
		SingleBranch:  true,
		Auth:          auth,
	})
	if err != nil {
		return nil, fmt.Errorf("Error while pulling from remote branch %s : %s", branch, err)
	}

	log.Debugf("Git pull completed for %s", gitDir)

	newCommit, err := getLatestCommit(repo)
	if err != nil {
		return nil, err
	}

	patch, err := oldCommit.Patch(newCommit)
	if err != nil {
		return nil, fmt.Errorf("Error while getting patch from old commit to new commit")
	}

	fileStats := patch.Stats()
	var filesChanged []string
	for _, fileStat := range fileStats {
		filesChanged = append(filesChanged, fileStat.Name)
	}

	return filesChanged, nil
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
