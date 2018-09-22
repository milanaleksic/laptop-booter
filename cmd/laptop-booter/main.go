package main

import (
	"flag"
	"log"
	"os"

	"github.com/milanaleksic/laptop-booter"
)

func main() {
	username := flag.String("username", "", "Username for the AMT interface")
	password := flag.String("password", "", "Password for the AMT interface")
	// FIXME: should be optional following two, meaning direct access available
	bastionHost := flag.String("bastionHost", "", "Bastion hostname")
	bastionPort := flag.Int("bastionPort", 22, "Bastion port")
	amtHost := flag.String("amtHost", "", "AMT computer hostname")
	amtPort := flag.Int("amtPort", 16992, "AMT computer port")
	dropbearHost := flag.String("dropbearHost", "", "Dropbear (SSH) computer hostname")
	dropbearPort := flag.Int("dropbearPort", 4748, "Dropbear (SSH) computer port")
	realSSHHost := flag.String("realSSHHost", "", "Real SSH computer hostname")
	realSSHPort := flag.Int("realSSHPort", 22, "Real SSH computer port")
	diskUnlockPassword := flag.String("diskUnlockPassword", "", "Disk unlock password")
	command := flag.String("command", "", "Command (one of: status, up, down, decrypt)")
	flag.Parse()

	config := &laptop_booter.Configuration{
		Username:           *username,
		Password:           *password,
		BastionHost:        *bastionHost,
		BastionPort:        *bastionPort,
		AmtHost:            *amtHost,
		AmtPort:            *amtPort,
		DropbearHost:       *dropbearHost,
		DropbearPort:       *dropbearPort,
		RealSSHHost:        *realSSHHost,
		RealSSHPort:        *realSSHPort,
		DiskUnlockPassword: *diskUnlockPassword,
		Command:            *command,
		LocalRealSSHPort:   16887,
		LocalAmtPort:       16888,
		LocalDropbearPort:  16889,
	}
	output, err := laptop_booter.Execute(config)
	if err != nil {
		log.Printf("Execution failed, err: %v", err)
		os.Exit(1)
	} else {
		log.Printf("Execution succeeded with output: %s", output)
		os.Exit(0)
	}

}
