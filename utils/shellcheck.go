package utils

import (
	"fmt"
	"os/exec"
)

//This is a function to find all shell scripts in the challenge directory and apply shellcheck on them
func ShellCheck(challengeDir string) error {
	findCommand := "find"
	findParameters := `-type f -exec grep -l "#!/bin/bash" {} \ | xargs shellcheck`
	cmd := exec.Command(findCommand,challengeDir,findParameters)

	stdout, err := cmd.Output()

    if err != nil {
		// Print the output and error
		fmt.Println(string(stdout))
        fmt.Println(err.Error())
        return err
    }

    // Print the output
    fmt.Println(string(stdout))
	return nil
}
