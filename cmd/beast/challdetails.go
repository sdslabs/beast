package main

import (
	"fmt"

	"github.com/sdslabs/beastv4/core/database"
	"github.com/spf13/cobra"
)

var challdetailsCmd = &cobra.Command{
	Use:   "challdetails",
	Short: "Lists all challenge details",

	Run: func(cmd *cobra.Command, args []string) {
		// fmt.Println("challdetails called")
		challengeInfo(args)
	},
}

func init() {
	rootCmd.AddCommand(challdetailsCmd)
}

func challengeInfo(args []string) {

	challenges, err := database.QueryAllChallenges()

	if err != nil {
		fmt.Println("DATABASE ERROR while processing the request.")
		return
	}

	if len(challenges) > 0 {
		fmt.Println("Chall Name | Chall State | Chall Container ID | Chall Image ID")
		for i := 0; i < len(challenges); i++ {
			challenge := challenges[i]
			fmt.Print(challenge.Name + " | ")
			fmt.Print(challenge.Status + " | ")
			fmt.Print(challenge.ContainerId + " | ")
			fmt.Println(challenge.ImageId)
		}

	} else {
		fmt.Println("No challenges present.")
	}

	return
}
