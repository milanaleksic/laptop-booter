package main

import (
	"flag"
	"github.com/milanaleksic/amtgo/amt"
)

func main() {
	username := flag.String("username", "", "Username for the AMT interface")
	password := flag.String("password", "", "Password for the AMT interface")

	options := amt.Optionset{
		SwSkipcertchk: 1,
		SwUseTLS:      0,
		Username:      *username,
		Password:      *password,
		Port:          16888,
	}
	amt.CliCommand(amt.CmdInfo, []string{"localhost"}, options)
}
