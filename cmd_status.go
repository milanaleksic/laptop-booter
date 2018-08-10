package main

import (
	"github.com/milanaleksic/amtgo/amt"
)

const (
	amtStateOn        = 0
	amtStateSleep     = 3
	amtStateHibernate = 4
	amtStateSoftOff   = 5
)

var legacyPowerstateTextMap = map[int]string{
	0:  "On",
	1:  "unimplemented",
	2:  "unimplemented",
	3:  "Sleep",
	4:  "Hibernate",
	5:  "Soft-Off",
	6:  "unimplemented",
	7:  "unimplemented",
	8:  "unimplemented",
	9:  "unimplemented",
	10: "unimplemented",
	11: "unimplemented",
	12: "unimplemented",
	13: "unimplemented",
	14: "unimplemented",
	15: "unimplemented",
	16: "unimplemented",
}

func getAmtStatus(username, password string) amt.Laststate {
	options := amt.Optionset{
		SwSkipcertchk: 1,
		SwUseTLS:      0,
		Username:      username,
		Password:      password,
		Port:          localForwardedPort,
	}
	var client amt.Laststate
	client.Hostname = "localhost"
	return amt.Command(client, amt.CmdInfo, options)
}
