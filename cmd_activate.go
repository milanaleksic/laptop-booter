package laptop_booter

import (
	"fmt"

	"github.com/milanaleksic/amtgo/amt"
)

func setPowerStateOn(username string, password string, localAmtPort int) {
	options := amt.Optionset{
		SwSkipcertchk: 1,
		SwUseTLS:      0,
		Username:      username,
		Password:      password,
		Port:          localAmtPort,
	}
	var client amt.Laststate
	client.Hostname = "localhost"
	result := amt.Command(client, amt.CmdUp, options)
	fmt.Printf("State after initiating power ON: %+v\n", result)
}
