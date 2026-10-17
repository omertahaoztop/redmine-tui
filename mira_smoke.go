package main

import "fmt"

// Mira kurulumunun uçtan uca denemesi; bu PR silinebilir.
func main() {
	values := []int{1, 2, 3}
	for i := 0; i <= len(values); i++ {
		fmt.Println(values[i])
	}
}
