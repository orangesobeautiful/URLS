package server

import (
	"log"

	"URLS/internal/common"
	"URLS/link/configs"
	"URLS/link/controllers"
	linkPB "URLS/proto/gen/go/link/v1"
)

const serviceName string = "link"

func Run() {
	cfgInfo := new(configs.LSCfgInfo)

	s := common.NewService(
		serviceName, cfgInfo,
		controllers.NewLinkController, linkPB.RegisterLinkServiceServer,
		linkPB.RegisterLinkServiceHandler)
	if s == nil {
		log.Fatal()
	}

	s.Run()
}
