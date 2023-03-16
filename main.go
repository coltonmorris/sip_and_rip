package main

import (
	"fmt"
	"sip_and_rip/adapters"
	"sip_and_rip/domain"
	"sip_and_rip/ports"
)

func main() {
	api := domain.NewApi()

	addr := "0.0.0.0:5061"
	server := getServer(addr, api)

	fmt.Println("Listening on: ", addr)
	if err := server.Serve(); err != nil {
		panic(err)
	}
}

func getServer(addr string, api ports.Api) ports.PublicServer {
	var server ports.PublicServer
	var err error

	server, err = adapters.NewUDPServer(addr, api)
	if err != nil {
		panic(err)
	}

	return server
}
