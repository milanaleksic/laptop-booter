package laptop_booter

import "os"

func getCurrentUser() string {
	return os.Getenv("USER")
}
