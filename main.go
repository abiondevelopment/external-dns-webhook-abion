/*
Copyright 2024 Abion AB

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"fmt"

	"github.com/abiondevelopment/external-dns-webhook-abion/webhook/dnsprovider"

	"github.com/abiondevelopment/external-dns-webhook-abion/webhook"
	"github.com/abiondevelopment/external-dns-webhook-abion/webhook/configuration"
	"github.com/abiondevelopment/external-dns-webhook-abion/webhook/logging"
	"github.com/abiondevelopment/external-dns-webhook-abion/webhook/server"
	log "github.com/sirupsen/logrus"
)

const banner = `
 external-dns-webhook-abion
 version: %s
`

var Version = "v0.0.1"

func main() {
	fmt.Printf(banner, Version)
	config := configuration.Init()
	logging.Init(&config)

	provider, err := dnsprovider.NewAbionProvider(&config)
	if err != nil {
		log.Fatalf("Failed to initialize DNS provider: %v", err)
	}
	srv := server.Init(config, webhook.New(provider))
	server.ShutdownGracefully(srv)
}
