package main

import (
	client "github.com/zond/mug/client"
)

func main() {
	m := client.New()
	defer m.Close()
	m.Run()
}
