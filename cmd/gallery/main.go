package main

import "log"

func main() {
	if _, err := loadConfig(); err != nil {
		log.Fatal(err)
	}
}
