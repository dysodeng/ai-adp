package main

import (
	"log"

	"github.com/dysodeng/ai-adp/cmd/app"
)

func main() {
	if err := app.Execute(); err != nil {
		log.Fatal(err)
	}
}
