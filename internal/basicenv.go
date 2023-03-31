package internal

import "os"

func CheckIfDevelopment() bool {
	return os.Getenv("DEVELOPMENT") == "true"
}
