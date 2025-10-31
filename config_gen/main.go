package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
)

type Config struct {
	Addr     string `json:"addr"`
	KeyStore string `json:"keystore"`
	Endpoint string `json:"endpoint"`
}

func main() {
	c := &Config{}
	flag.StringVar(&c.Endpoint, "endpoint", "ws", "websocket endpoint")
	flag.StringVar(&c.KeyStore, "keystore", "", "encrypted keystore string")
	flag.StringVar(&c.Addr, "addr", "10.1.10.194:8443", "address of server (10.1.10.194:8443)")
	flag.Parse()
	jb, err := json.Marshal(c)
	if err != nil {
		log.Fatalf("Error marshalling config: %v\n", err)
	}
	fmt.Println(string(jb))
}
