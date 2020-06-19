package main

import (
	log "github.com/sirupsen/logrus"
	"room/server"
)

func main() {
	discovery := server.NewUdpIpDiscovery()
	discovery.SetOnLeave(func(address string) {
		log.Printf("%s leaved", address)
	})
	discovery.SetOnJoin(func(address string) {
		log.Printf("%s joined", address)
	})
	discovery.SetOnPing(func(address string) {
		log.Printf("%s pinged", address)
	})
	discovery.Join()
	discovery.AutoLeave()
}
