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
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/GoogleCloudPlatform/professional-services/ipam-autopilot/cai"
	"github.com/GoogleCloudPlatform/professional-services/ipam-autopilot/data_access"
	"github.com/GoogleCloudPlatform/professional-services/ipam-autopilot/ip"
	"github.com/GoogleCloudPlatform/professional-services/ipam-autopilot/model"
	"github.com/apparentlymart/go-cidr/cidr"
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
		return directInsert(c, tx, p, routingDomain)
	} else {
		return findNewLeaseAndInsert(c, tx, p, routingDomain)
	}
}

func directInsert(c *fiber.Ctx, tx *sql.Tx, p RangeRequest, routingDomain *model.RoutingDomain) error {
	var err error
	domain_id, err := strconv.ParseInt(p.Domain, 10, 64)
	if err != nil {
		return c.Status(400).JSON(&fiber.Map{
			"success": false,
			"message": fmt.Sprintf("Domain needs to be an integer %v", err),
		})
	}

	parent_id := int64(-1)
	if p.Parent != "" {
		parent_id, err = strconv.ParseInt(p.Parent, 10, 64)
		if err != nil {
			rangeFromDb, err := data_access.GetRangeByCidrAndRoutingDomain(tx, p.Parent, int(domain_id))
			if err != nil {
				return c.Status(400).JSON(&fiber.Map{
					"success": false,
					"message": fmt.Sprintf("Parent needs to be either a cidr range within the routing domain or the id of a valid range %v", err),
				})
			}
			parent_id = int64(rangeFromDb.Subnet_id)
		}
	}

	id, err := data_access.CreateRangeInDb(tx, parent_id,
		int(domain_id),
		p.Name,
		p.Cidr)

	if err != nil {
		tx.Rollback()
		return c.Status(503).JSON(&fiber.Map{
			"success": false,
			"message": fmt.Sprintf("Unable to create new Subnet Lease %v", err),
		})
	}

	err = tx.Commit()
	if err != nil {
		log.Fatal(err)
	}

	return c.Status(200).JSON(&fiber.Map{
		"id":   id,
		"cidr": p.Cidr,
	})
}

func findNewLeaseAndInsert(c *fiber.Ctx, tx *sql.Tx, p RangeRequest, routingDomain *model.RoutingDomain) error {
	var err error
	var parent *model.Range
	if p.Parent != "" {
		parent_id, err := strconv.ParseInt(p.Parent, 10, 64)
		if err != nil {
			parent, err = data_access.GetRangeByCidrAndRoutingDomain(tx, p.Parent, routingDomain.Id)
			if err != nil {
				return c.Status(400).JSON(&fiber.Map{
					"success": false,
					"message": fmt.Sprintf("Parent needs to be either a cidr range within the routing domain or the id of a valid range %v", err),
				})
			}
		} else {
			parent, err = data_access.GetRangeFromDBWithTx(tx, parent_id)
			if err != nil {
				tx.Rollback()
				return c.Status(503).JSON(&fiber.Map{
					"success": false,
					"message": fmt.Sprintf("Unable to create new Subnet Lease  %v", err),
				})
			}
		}
	} else {
		return c.Status(400).JSON(&fiber.Map{
			"success": false,
			"message": "Please provide the ID of a parent range",
		})
	}
	range_size := p.Range_size
	subnet_ranges, err := data_access.GetRangesForParentFromDB(tx, int64(parent.Subnet_id))
	if err != nil {
		tx.Rollback()
		return c.Status(503).JSON(&fiber.Map{
			"success": false,
			"message": fmt.Sprintf("Unable to create new Subnet Lease  %v", err),
		})
	}
	if os.Getenv("CAI_ORG_ID") != "" {
		log.Printf("CAI for org %s enabled", os.Getenv("CAI_ORG_ID"))
		// Integrating ranges from the VPC -- start
		vpcs := strings.Split(routingDomain.Vpcs, ",")
		log.Printf("Looking for subnets in vpcs %v", vpcs)
		ranges, err := cai.GetRangesForNetwork(fmt.Sprintf("organizations/%s", os.Getenv("CAI_ORG_ID")), vpcs)
		if err != nil {
			tx.Rollback()
			return c.Status(503).JSON(&fiber.Map{
				"success": false,
				"message": fmt.Sprintf("error %v", err),
			})
		}
		log.Printf("Found %d subnets in vpcs %v", len(ranges), vpcs)

		for j := 0; j < len(ranges); j++ {
			vpc_range := ranges[j]
			if !ContainsRange(subnet_ranges, vpc_range.Cidr) {
				log.Printf("Adding range %s from CAI", vpc_range.Cidr)
				subnet_ranges = append(subnet_ranges, model.Range{
					Cidr: vpc_range.Cidr,
				})
			}

			for k := 0; k < len(vpc_range.SecondaryRanges); k++ {
				secondaryRange := vpc_range.SecondaryRanges[k]
				if !ContainsRange(subnet_ranges, secondaryRange.Cidr) {
					log.Printf("Adding secondary range %s from CAI", vpc_range.Cidr)
					subnet_ranges = append(subnet_ranges, model.Range{
						Cidr: secondaryRange.Cidr,
					})
				}
			}
		}
		// Integrating ranges from the VPC -- end
	} else {
		log.Printf("Not checking CAI, env variable with Org ID not set")
	}

	subnet, subnetOnes, err := ip.FindNextSubnet(int(range_size), parent.Cidr, subnet_ranges)
	if err != nil {
		tx.Rollback()
		return c.Status(503).JSON(&fiber.Map{
			"success": false,
			"message": fmt.Sprintf("Unable to create new Subnet Lease %v", err),
		})
	}
	nextSubnet, _ := cidr.NextSubnet(subnet, int(range_size))
	log.Printf("next subnet will be starting with %s", nextSubnet.IP.String())

	id, err := data_access.CreateRangeInDb(tx, int64(parent.Subnet_id), routingDomain.Id, p.Name, fmt.Sprintf("%s/%d", subnet.IP.To4().String(), subnetOnes))

	if err != nil {
		tx.Rollback()
		return c.Status(503).JSON(&fiber.Map{
			"success": false,
			"message": fmt.Sprintf("Unable to create new Subnet Lease %v", err),
		})
	}

	err = tx.Commit()
	if err != nil {
		log.Fatal(err)
	}

	return c.Status(200).JSON(&fiber.Map{
		"id":   id,
		"cidr": fmt.Sprintf("%s/%d", subnet.IP.To4().String(), subnetOnes),
	})
}

func ContainsRange(array []model.Range, cidr string) bool {
	for i := 0; i < len(array); i++ {
		if cidr == array[i].Cidr {
			return true
		}
	}
	return false
}
