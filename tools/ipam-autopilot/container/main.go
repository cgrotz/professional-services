// Copyright 2021 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/GoogleCloudPlatform/professional-services/ipam-autopilot/api"
	"github.com/GoogleCloudPlatform/professional-services/ipam-autopilot/data_access"
	"github.com/GoogleCloudPlatform/professional-services/ipam-autopilot/provider"
	"github.com/gofiber/fiber/v2"
)

func main() {
	var err error

	data_access.InitDatabase()
	defer data_access.Close()

	app := fiber.New()
	// No static assets right now app.Static("/", "./public")
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("IPAM Autopilot up and running 👋!")
	})
	app.Get("/.well-known/terraform.json", provider.GetTerraformDiscovery)
	app.Get("/terraform/providers/v1/ipam-autopilot/ipam/versions", provider.GetTerraformVersions)
	app.Get("/terraform/providers/v1/ipam-autopilot/ipam/:version/download/:os/:arch", provider.GetTerraformVersionDownload)

	app.Post("/ranges", api.CreateNewRange)
	app.Get("/ranges", api.GetRanges)
	app.Get("/ranges/:id", api.GetRange)
	app.Delete("/ranges/:id", api.DeleteRange)

	app.Get("/domains", api.GetRoutingDomains)
	app.Get("/domains/:id", api.GetRoutingDomain)
	app.Put("/domains/:id", api.UpdateRoutingDomain)
	app.Post("/domains", api.CreateRoutingDomain)
	app.Delete("/domains/:id", api.DeleteRoutingDomain)

	var port int64
	if os.Getenv("PORT") != "" {
		port, err = strconv.ParseInt(os.Getenv("PORT"), 10, 64)
		if err != nil {
			log.Panicf("Can't parse value of PORT env variable %s %v", os.Getenv("PORT"), err)
		}
	} else {
		port = 8080
	}

	app.Listen(fmt.Sprintf(":%d", port))
}
