package utils

import (
	"fmt"
	"os/exec"
)
	
// Shell command to be executed :
// find newChallDir -type f -exec grep -l "#!/bin/bash" {} \ | xargs shellcheck

//Shellcheck func is...
func ShellCheck(newChallDir string) {
	findCommand := "find"
	findParameters := `-type f -exec grep -l "#!/bin/bash" {} \ | xargs shellcheck`
	cmd := exec.Command(findCommand,newChallDir,findParameters)

	stdout, err := cmd.Output()

    if err != nil {
        fmt.Println(err.Error())
        return
    }

    // Print the output
    fmt.Println(string(stdout))
}
