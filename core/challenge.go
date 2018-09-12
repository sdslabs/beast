package core

const CONFIG_FILE_NAME string = "beast.toml"

type BeastConfig struct {
	Challenge Challenge `toml:"challenge"`
	Author    Author    `toml:"author"`
}

type Challenge struct {
	Id               string           `toml:"id"`
	Name             string           `toml:"name"`
	ChallengeType    string           `toml:"challenge_type"`
	ChallengeDetails ChallengeDetails `toml:"details"`
}

type ChallengeDetails struct {
	Flag             string   `toml:"flag"`
	AptDeps          []string `toml:"flag"`
	SetupScript      string   `toml:"setup_script"`
	StaticContentDir string   `toml:"static_content_dir"`
	RunCmd           string   `toml:"run_cmd"`
}

type Author struct {
	Name   string `toml:"name"`
	Email  string `toml:"email"`
	SSHKey string `toml:"ssh_key"`
}
