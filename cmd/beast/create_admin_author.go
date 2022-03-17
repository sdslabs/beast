package main

import (
	"fmt"
	"os"
	"path/filepath"
	"encoding/csv"
	"log"
	"io"

	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/core/utils"
	"github.com/sdslabs/beastv4/pkg/auth"
	"github.com/spf13/cobra"
)

var (
	BEAST_GLOBAL_DIR string = filepath.Join(os.Getenv("HOME"), ".beast/comp")
)

var createAuthorCmd = &cobra.Command{
	Use:   "create-author",
	Short: "Creates new author",
	Long:  "Creates new author using command line arguments",
	PreRun: func(cmd *cobra.Command, args []string) {
		if Name == "" {
			fmt.Printf("Name of Author not provided")
			os.Exit(1)
		}
		if Username == "" {
			fmt.Printf("Username of Author not provided")
			os.Exit(1)
		}

		if Email == "" {
			fmt.Printf("Email not provided")
			os.Exit(1)
		}

		if PublicKeyPath == "" {
			fmt.Printf("Public Key Path not provided")
		}

		if Password == "" {
			fmt.Printf("Password not provided")
			os.Exit(1)
		}
	},

	Run: func(cmd *cobra.Command, args []string) {
		config.InitConfig()

		auth.Init(core.ITERATIONS, core.HASH_LENGTH, core.TIMEPERIOD, core.ISSUER, config.Cfg.JWTSecret, []string{core.USER_ROLES["author"]}, []string{core.USER_ROLES["admin"]}, []string{core.USER_ROLES["contestant"]})

		utils.CreateAdminOrAuthor(Name, Username, Email, PublicKeyPath, Password, "author")
	},
}

var createAdminCmd = &cobra.Command{
	Use:   "create-admin",
	Short: "Creates new admin",
	Long:  "Creates new admin using command line arguments",
	PreRun: func(cmd *cobra.Command, args []string) {
		if Name == "" {
			fmt.Printf("Name of Admin not provided")
			os.Exit(1)
		}
		if Username == "" {
			fmt.Printf("Username of Admin not provided")
			os.Exit(1)
		}

		if Email == "" {
			fmt.Printf("Email not provided")
			os.Exit(1)
		}

		if PublicKeyPath == "" {
			fmt.Printf("Public Key Path not provided")
		}

		if Password == "" {
			fmt.Printf("Password not provided")
			os.Exit(1)
		}
	},

	Run: func(cmd *cobra.Command, args []string) {
		config.InitConfig()

		auth.Init(core.ITERATIONS, core.HASH_LENGTH, core.TIMEPERIOD, core.ISSUER, config.Cfg.JWTSecret, []string{core.USER_ROLES["author"]}, []string{core.USER_ROLES["admin"]}, []string{core.USER_ROLES["contestant"]})

		utils.CreateAdminOrAuthor(Name, Username, Email, PublicKeyPath, Password, "admin")
	},
}

var createMultipleAuthorCmd = &cobra.Command{
	Use:   "create-multiple-author",
	Short: "Creates multiple new authors",
	Long:  "Creates multiple new authors using given .csv file",
	PreRun: func(cmd *cobra.Command, args []string) {
		if CsvFile == "" {
			fmt.Printf("Csv File path not provided")
			os.Exit(1)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		csvFile := filepath.Join(BEAST_GLOBAL_DIR, CsvFile)

		config.InitConfig()

		auth.Init(core.ITERATIONS, core.HASH_LENGTH, core.TIMEPERIOD, core.ISSUER, config.Cfg.JWTSecret, []string{core.USER_ROLES["author"]}, []string{core.USER_ROLES["admin"]}, []string{core.USER_ROLES["contestant"]})


		f, err := os.Open(csvFile)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		
		csvReader := csv.NewReader(f)
		for {
			rec, err := csvReader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Fatal(err)
			}
			Name := rec[0]
			Username := rec[1]
			Email := rec[2]
			PublicKeyPath := rec[3]
			Password := rec[4]

			if Name == "" {
				fmt.Printf("Name of Author not provided")
				os.Exit(1)
			}
			if Username == "" {
				fmt.Printf("Username of Author not provided")
				os.Exit(1)
			}
	
			if Email == "" {
				fmt.Printf("Email not provided")
				os.Exit(1)
			}
	
			if PublicKeyPath == "" {
				fmt.Printf("Public Key Path not provided")
			}
	
			if Password == "" {
				fmt.Printf("Password not provided")
				os.Exit(1)
			}

			utils.CreateAdminOrAuthor(Name, Username, Email, PublicKeyPath, Password, "author")
		}

	},
}

var createMultipleAdminCmd = &cobra.Command{
	Use:   "create-multiple-admin",
	Short: "Creates multiple new admins",
	Long:  "Creates multiple new admins using given .csv file",
	PreRun: func(cmd *cobra.Command, args []string) {
		if CsvFile == "" {
			fmt.Printf("Csv File path not provided")
			os.Exit(1)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		csvFile := filepath.Join(BEAST_GLOBAL_DIR, CsvFile)

		config.InitConfig()

		auth.Init(core.ITERATIONS, core.HASH_LENGTH, core.TIMEPERIOD, core.ISSUER, config.Cfg.JWTSecret, []string{core.USER_ROLES["author"]}, []string{core.USER_ROLES["admin"]}, []string{core.USER_ROLES["contestant"]})


		f, err := os.Open(csvFile)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		
		csvReader := csv.NewReader(f)
		for {
			rec, err := csvReader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Fatal(err)
			}
			Name := rec[0]
			Username := rec[1]
			Email := rec[2]
			PublicKeyPath := rec[3]
			Password := rec[4]

			if Name == "" {
				fmt.Printf("Name of Author not provided")
				os.Exit(1)
			}
			if Username == "" {
				fmt.Printf("Username of Author not provided")
				os.Exit(1)
			}
	
			if Email == "" {
				fmt.Printf("Email not provided")
				os.Exit(1)
			}
	
			if PublicKeyPath == "" {
				fmt.Printf("Public Key Path not provided")
			}
	
			if Password == "" {
				fmt.Printf("Password not provided")
				os.Exit(1)
			}

			utils.CreateAdminOrAuthor(Name, Username, Email, PublicKeyPath, Password, "admin")
		}

	},
}
