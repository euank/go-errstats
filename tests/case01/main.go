package main

import (
	"errors"
	"fmt"
	"os"
)

// meaningless comment

func main() {
	_, err := fmt.Println("vim-go")
	if err != nil {
		panic("1")
	}
	if err := errors.New(""); err != nil {
		// meaningless comment
		panic("2")
	}
	if err := (&os.File{}); err != nil {
		panic("3")
	}
}
