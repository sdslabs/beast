package deploy

import (
	"io"
	"ioutil"
	"archive/tar"
	"path/filepath"
	"time"

	"github.com/docker/docker/pkg/archive"
	"github.com/fristonio/beast/core"
	log "github.com/sirupsen/logrus"
)

func StageChallenge(challengeDir string) error {
	contextDir, err := GetContextDirPath(challengeDir)
	if err != nil {
		return err
	}

	challengeConfig := filepath.Join(contextDir, core.CONFIG_FILE_NAME)
	dockerfileCtx := GenerateChallengeDockerfileCtx(challengeConfig)

	stageCtx, err := archive.Tar(contextDir, archive.Gzip)
	if err != nil {
		return err
	}

	addDockerfileToStagingContext(dockerfileCtx, stageCtx)
}

// This is the main function which starts the deploy pipeline for a locally
// available challenge, it goes through all the stages of the challenge deployement
// and hanles any error by logging into database if it occurs.
//
// challengeDir corresponds to the directory to be used as a challenge context
//
// The pipeline goes through the following stages:
// * StageChallenge - Add the challenge to the staging area for beast creating
//		a tar for the challenge with Dockerfile embedded into the context.
// 		This challenge is then present in the staging area($BEAST_HOME/staging/challengeId/)
//		for further steps in the pipeline.
func StartDeployPipeline(challengeDir string) {
	challengeName := filepath.Base(challengeDir)
	log.Debug("Starting deploy pipeline for challenge %s", challengeName)

	err := StageChallenge(challengeDir)
	if err != nil {
		log.WithFields(log.Fields{
			"DEPLOY_ERROR": "STAGING :: " + challangeName,
		}).Errorf("%s", err)
		return
	}
}


// AddDockerfileToBuildContext from a ReadCloser, returns a new archive and
// the relative path to the dockerfile in the context.
func addDockerfileToStagingContext(dockerfileCtx io.ReadCloser, buildCtx io.ReadCloser) (io.ReadCloser, string, error) {
	file, err := ioutil.ReadAll(dockerfileCtx)
	dockerfileCtx.Close()
	if err != nil {
		return nil, "", err
	}

	now := time.Now()
	hdrTmpl := &tar.Header{
		Mode:       0600,
		Uid:        0,
		Gid:        0,
		ModTime:    now,
		Typeflag:   tar.TypeReg,
		AccessTime: now,
		ChangeTime: now,
	}
	
	dockerfile = "Dockerfile"

	buildCtx = archive.ReplaceFileTarWrapper(buildCtx, map[string]archive.TarModifierFunc{
		dockerfile: func(_ string, h *tar.Header, content io.Reader) (*tar.Header, []byte, error) {
			return hdrTmpl, file, nil
		}
	})

	return buildCtx, dockerfile, nil
}
