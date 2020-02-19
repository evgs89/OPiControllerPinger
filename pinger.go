package main

import (
	"fmt"
	"bufio"
	"os"
)

func main () {
	fmt.Println("Hello")
	file, err := os.Open("settings.ini")
	if err != nil {
		fmt.Println("Error opening file")
	}

}