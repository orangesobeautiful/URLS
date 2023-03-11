package server

import (
	"log"

	"URLS/internal/common"
	userPB "URLS/proto/gen/go/user/v1"
	"URLS/user/configs"
	"URLS/user/controllers"
)

const serviceName string = "user"

func Run() {
	cfgInfo := new(configs.USCfgInfo)

	s := common.NewService(
		serviceName, cfgInfo,
		controllers.NewUserController, userPB.RegisterUserServiceServer,
		userPB.RegisterUserServiceHandler)
	if s == nil {
		log.Fatal()
	}

	s.Run()
}
