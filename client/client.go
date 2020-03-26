package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Printf("Usage : %s <ip> <port> <user_name>\n", os.Args[0])
		os.Exit(1)
	}

}