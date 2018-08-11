package main

import (
	"fmt"

	"github.com/milanaleksic/amtgo/amt"
)

func setPowerStateOff(username string, password string) {
	options := amt.Optionset{
		SwSkipcertchk: 1,
		SwUseTLS:      0,
		Username:      username,
		Password:      password,
		Port:          localAmtPort,
	}
	var client amt.Laststate
	client.Hostname = "localhost"
	result := amt.Command(client, amt.CmdDown, options)
	fmt.Printf("%+v", result)
}