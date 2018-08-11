package main

import "os"

func getCurrentUser() string {
	return os.Getenv("USER")
}
