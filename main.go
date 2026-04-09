package main

import "github.com/harris-ahmad/gitpull/cmd"

var version = "dev"

func main() {
	cmd.SetVersion(version)
	cmd.Execute()
}
