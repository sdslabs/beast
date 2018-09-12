package deploy

func DeployChallenge(challengeDir string) error {
	if err := ValidateChallengeDir(challengeDir); err != nil {
		return err
	}

	return nil
}
