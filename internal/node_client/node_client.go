package internal

import (
	"log"
	"sync"
)

func Start(wg *sync.WaitGroup, controllerAddr string) {
	defer wg.Done()
	log.Println("starting client")
}

func Stop() {
	log.Println("client stopped")
}
