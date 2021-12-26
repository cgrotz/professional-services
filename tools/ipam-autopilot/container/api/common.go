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
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/GoogleCloudPlatform/professional-services/ipam-autopilot/cai"
	"github.com/GoogleCloudPlatform/professional-services/ipam-autopilot/data_access"
	"github.com/GoogleCloudPlatform/professional-services/ipam-autopilot/ip"
	"github.com/GoogleCloudPlatform/professional-services/ipam-autopilot/model"
	"github.com/apparentlymart/go-cidr/cidr"
)

func directInsert(tx *sql.Tx, p RangeRequest, routingDomain *model.RoutingDomain) (int, string, error) {
	var err error
	domain_id, err := strconv.ParseInt(p.Domain, 10, 64)
	if err != nil {
		return -1, "", NewBadInputError(fmt.Sprintf("Domain needs to be an integer %v", err))
	}

	parent_id := int64(-1)
	if p.Parent != "" {
		parent_id, err = strconv.ParseInt(p.Parent, 10, 64)
		if err != nil {
			rangeFromDb, err := data_access.GetRangeByCidrAndRoutingDomain(tx, p.Parent, int(domain_id))
			if err != nil {
				return -1, "", NewBadInputError(fmt.Sprintf("Parent needs to be either a cidr range within the routing domain or the id of a valid range %v", err))
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
		return -1, "", NewServerError(fmt.Sprintf("Unable to create new Subnet Lease %v", err))
	}

	err = tx.Commit()
	if err != nil {
		log.Fatal(err)
	}

	return int(id), p.Cidr, nil
}

func findNewLeaseAndInsert(tx *sql.Tx, rangeName string, parentId string, range_size int, routingDomain *model.RoutingDomain) (int, string, error) {
	var err error
	var parent *model.Range
	if parentId != "" {
		parent_id, err := strconv.ParseInt(parentId, 10, 64)
		if err != nil {
			parent, err = data_access.GetRangeByCidrAndRoutingDomain(tx, parentId, routingDomain.Id)
			if err != nil {
				return -1, "", NewBadInputError(fmt.Sprintf("Parent needs to be either a cidr range within the routing domain or the id of a valid range %v", err))
			}
		} else {
			parent, err = data_access.GetRangeFromDBWithTx(tx, parent_id)
			if err != nil {
				log.Printf("Failed retrieving Parent range %v", err)
				tx.Rollback()
				return -1, "", NewServerError(fmt.Sprintf("Unable to create new Subnet Lease  %v", err))
			}
		}
	} else {
		return -1, "", NewBadInputError("Please provide the ID of a parent range")
	}
	subnet_ranges, err := data_access.GetRangesForParentFromDB(tx, int64(parent.Subnet_id))
	if err != nil {
		log.Printf("Failed getting child ranges for parent range %v", err)
		tx.Rollback()
		return -1, "", NewServerError(fmt.Sprintf("Unable to create new Subnet Lease  %v", err))
	}
	if os.Getenv("CAI_ORG_ID") != "" {
		log.Printf("CAI for org %s enabled", os.Getenv("CAI_ORG_ID"))
		// Integrating ranges from the VPC -- start
		vpcs := strings.Split(routingDomain.Vpcs, ",")
		log.Printf("Looking for subnets in vpcs %v", vpcs)
		ranges, err := cai.GetRangesForNetwork(fmt.Sprintf("organizations/%s", os.Getenv("CAI_ORG_ID")), vpcs)
		if err != nil {
			log.Printf("Failed retrieving ranges from CAI %v", err)
			tx.Rollback()
			return -1, "", NewServerError(fmt.Sprintf("error %v", err))
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
		log.Printf("Failed finding next subnet %v", err)
		tx.Rollback()
		return -1, "", NewServerError(fmt.Sprintf("Unable to create new Subnet Lease %v", err))
	}
	nextSubnet, _ := cidr.NextSubnet(subnet, int(range_size))
	log.Printf("next subnet will be starting with %s", nextSubnet.IP.String())

	id, err := data_access.CreateRangeInDb(tx, int64(parent.Subnet_id), routingDomain.Id, rangeName, fmt.Sprintf("%s/%d", subnet.IP.To4().String(), subnetOnes))

	if err != nil {
		log.Printf("Failed inserting range %v", err)
		tx.Rollback()
		return -1, "", NewServerError(fmt.Sprintf("Unable to create new Subnet Lease %v", err))
	}

	err = tx.Commit()
	if err != nil {
		log.Printf("Failed commiting transaction %v", err)
		log.Fatal(err)
	}

	return int(id), fmt.Sprintf("%s/%d", subnet.IP.To4().String(), subnetOnes), nil
}

func checkOverlap(tx *sql.Tx, routingDomain *model.RoutingDomain, ipCidrRange string) error {
	subnet_ranges, err := data_access.GetRangesForRoutingDomainFromDB(tx, routingDomain.Id)
	if err != nil {
		return err
	}
	if os.Getenv("CAI_ORG_ID") != "" {
		log.Printf("CAI for org %s enabled", os.Getenv("CAI_ORG_ID"))
		// Integrating ranges from the VPC -- start
		vpcs := strings.Split(routingDomain.Vpcs, ",")
		log.Printf("Looking for subnets in vpcs %v", vpcs)
		ranges, err := cai.GetRangesForNetwork(fmt.Sprintf("organizations/%s", os.Getenv("CAI_ORG_ID")), vpcs)
		if err != nil {
			log.Printf("Failed retrieving ranges from CAI %v", err)
			tx.Rollback()
			return NewServerError(fmt.Sprintf("error %v", err))
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

	_, net, err := net.ParseCIDR(ipCidrRange)
	if err != nil {
		return err
	}
	return ip.VerifyNoOverlap(subnet_ranges, net)
}

func ContainsRange(array []model.Range, cidr string) bool {
	for i := 0; i < len(array); i++ {
		if cidr == array[i].Cidr {
			return true
		}
	}
	return false
}

type BadInputError struct {
	msg string
}

func NewBadInputError(msg string) *BadInputError {
	return &BadInputError{
		msg: msg,
	}
}

func (e *BadInputError) Error() string {
	return e.msg
}

type ServerError struct {
	msg string
}

func NewServerError(msg string) *ServerError {
	return &ServerError{
		msg: msg,
	}
}

func (e *ServerError) Error() string {
	return e.msg
}
