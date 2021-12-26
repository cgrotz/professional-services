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

package api

import (
	"fmt"
	"log"
	"strconv"

	"github.com/GoogleCloudPlatform/professional-services/ipam-autopilot/data_access"
	"github.com/GoogleCloudPlatform/professional-services/ipam-autopilot/model"
	"github.com/gofiber/fiber/v2"
)

type RangeRequest struct {
	Parent     string `json:"parent"`
	Name       string `json:"name"`
	Range_size int    `json:"range_size"`
	Domain     string `json:"domain"`
	Cidr       string `json:"cidr"`
}

func GetRanges(c *fiber.Ctx) error {
	var results []*fiber.Map
	ranges, err := data_access.GetRangesFromDB()
	if err != nil {
		return c.Status(503).JSON(&fiber.Map{
			"success": false,
			"message": fmt.Sprintf("%v", err),
		})
	}

	for i := 0; i < len(ranges); i++ {
		results = append(results, &fiber.Map{
			"id":     ranges[i].Subnet_id,
			"parent": ranges[i].Parent_id,
			"name":   ranges[i].Name,
			"cidr":   ranges[i].Cidr,
		})
	}
	return c.Status(200).JSON(results)
}

func GetRange(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(&fiber.Map{
			"success": false,
			"message": fmt.Sprintf("%v", err),
		})
	}
	rang, err := data_access.GetRangeFromDB(id)
	if err != nil {
		return c.Status(503).JSON(&fiber.Map{
			"success": false,
			"message": fmt.Sprintf("%v", err),
		})
	}

	return c.Status(200).JSON(&fiber.Map{
		"id":     rang.Subnet_id,
		"parent": rang.Parent_id,
		"name":   rang.Name,
		"cidr":   rang.Cidr,
	})
}

func DeleteRange(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(&fiber.Map{
			"success": false,
			"message": fmt.Sprintf("%v", err),
		})
	}
	err = data_access.DeleteRangeFromDb(id)
	if err != nil {
		return c.Status(503).JSON(&fiber.Map{
			"success": false,
			"message": fmt.Sprintf("%v", err),
		})
	}

	return c.Status(200).JSON(&fiber.Map{
		"success": true,
	})
}

func CreateNewRange(c *fiber.Ctx) error {
	tx, err := data_access.GetTransaction()
	if err != nil {
		log.Fatal(err)
	}

	// Instantiate new RangeRequest struct
	p := RangeRequest{}
	//  Parse body into RangeRequest struct
	if err := c.BodyParser(&p); err != nil {
		fmt.Printf("Failed parsing body. %s Bad format %v", string(c.Body()), err)
		tx.Rollback()
		return c.Status(400).JSON(&fiber.Map{
			"success": false,
			"message": fmt.Sprintf("Bad format %v", err),
		})
	}

	var routingDomain *model.RoutingDomain
	if p.Domain == "" {
		routingDomain, err = data_access.GetDefaultRoutingDomainFromDB(tx)
		if err != nil {
			fmt.Printf("Error %v", err)
			tx.Rollback()
			return c.Status(503).JSON(&fiber.Map{
				"success": false,
				"message": "Couldn't retrieve default routing domain",
			})
		}
	} else {
		domain_id, err := strconv.ParseInt(p.Domain, 10, 64)
		if err != nil {
			return c.Status(400).JSON(&fiber.Map{
				"success": false,
				"message": fmt.Sprintf("%v", err),
			})
		}
		routingDomain, err = data_access.GetRoutingDomainFromDB(domain_id)
		if err != nil {
			fmt.Printf("Error %v", err)
			tx.Rollback()
			return c.Status(503).JSON(&fiber.Map{
				"success": false,
				"message": "Couldn't retrieve default routing domain",
			})
		}
	}

	if p.Cidr != "" {
		id, cidr, err := directInsert(tx, p, routingDomain)
		if err != nil {
			switch e := err.(type) {
			case *BadInputError:
				return c.Status(400).JSON(&fiber.Map{
					"success": false,
					"message": e.msg,
				})
			case *ServerError:
				return c.Status(503).JSON(&fiber.Map{
					"success": false,
					"message": e.msg,
				})
			default:
				return c.Status(503).JSON(&fiber.Map{
					"success": false,
					"message": "Failed creating range",
				})
			}
		} else {
			return c.Status(200).JSON(&fiber.Map{
				"id":   id,
				"cidr": cidr,
			})
		}
	} else {
		id, cidr, err := findNewLeaseAndInsert(tx, p.Name, p.Parent, p.Range_size, routingDomain)
		if err != nil {
			switch e := err.(type) {
			case *BadInputError:
				return c.Status(400).JSON(&fiber.Map{
					"success": false,
					"message": e.msg,
				})
			case *ServerError:
				return c.Status(503).JSON(&fiber.Map{
					"success": false,
					"message": e.msg,
				})
			default:
				return c.Status(503).JSON(&fiber.Map{
					"success": false,
					"message": "Failed creating range",
				})
			}
		} else {
			return c.Status(200).JSON(&fiber.Map{
				"id":   id,
				"cidr": cidr,
			})
		}
	}
}
