package internal

import (
	"log"
	"sync"
)

func Start(wg *sync.WaitGroup, controllerAddr string) {
	defer wg.Done()
	log.Println("starting controller client")
}

func Stop() {
	log.Println("controller client stopped")
}
