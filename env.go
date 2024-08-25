package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
)

type Port int

func (p Port) Addr() string {
	return fmt.Sprintf(":%d", p)
}

type NodeEnv string

func (ne NodeEnv) IsProduction() bool {
	return ne == "production"
}

type Envs struct {
	Port    Port
	NodeEnv NodeEnv
}

func GetEnvs() Envs {
	port, err := strconv.Atoi(os.Getenv("PORT"))
	if err != nil {
		log.Fatalf("Invalid port %q", port)
	}
	return Envs{
		Port:    Port(port),
		NodeEnv: NodeEnv(os.Getenv("NODE_ENV")),
	}
}
