package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
	"tikmeh/cli"
	"time"
)

func main() {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("%v", err)
			println("\nPress any button to exit...")
			_, _ = bufio.NewReader(os.Stdin).ReadByte()
		}
	}()

	if len(os.Args) > 1 { // with console args
		Cli := cli.NewTikmehCli()
		if err := Cli.Run(os.Args); err != nil {
			log.Fatal(err)
		}
	} else { // interactive mode
		fmt.Printf("%s (%s) [sources and up-to-date executables: %s]\n"+
			"Enter 'help' to get help message.\n", cli.PackageName, cli.VersionInfo, cli.GithubLink)
		reader := bufio.NewReader(os.Stdin)
		for {
			print(">>> ")
			input, err := reader.ReadString('\n')
			if err != nil {
				log.Fatalf(err.Error())
			}
			input = strings.Trim(input, " \n\t")
			if input == "" {
				println("see you next time (exiting in 5 sec)")
				time.Sleep(time.Second * 5)
				os.Exit(0)
			}
			Cli := cli.NewTikmehCli()
			if err := Cli.Run(append([]string{cli.PackageName}, strings.Split(input, " ")...)); err != nil {
				log.Fatal(err)
			}
		}
	}
}
