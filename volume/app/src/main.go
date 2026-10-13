package main

import (
	"fmt"
	"time"
)

func main() {
	fmt.Println("template-go iniciado correctamente!")

	for {
		fmt.Printf("template-go activo y en ejecución...\n")
		time.Sleep(10 * time.Second)
	}
}