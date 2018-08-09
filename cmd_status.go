package main

import "github.com/milanaleksic/amtgo/amt"

func printAmtStatus(username, password string) {
	options := amt.Optionset{
		SwSkipcertchk: 1,
		SwUseTLS:      0,
		Username:      username,
		Password:      password,
		Port:          localForwardedPort,
	}
	amt.CliCommand(amt.CmdInfo, []string{"localhost"}, options)
}
