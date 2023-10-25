package main

import (
	"fmt"

	"github.com/mateogo42/goi/cmd"
)

func main() {
	err := cmd.Execute()
	if err != nil {
		fmt.Println(err)
	}
}
