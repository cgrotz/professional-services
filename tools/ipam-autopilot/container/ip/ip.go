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

package ip

import (
	"fmt"
	"log"
	"net"

	"github.com/GoogleCloudPlatform/professional-services/ipam-autopilot/model"
	"github.com/apparentlymart/go-cidr/cidr"
)

func VerifyNoOverlap(parentCidr string, subnetRanges []model.Range, newSubnet *net.IPNet) error {
	_, parentNetwork, err := net.ParseCIDR(parentCidr)
	if err != nil {
		return fmt.Errorf("can't parse CIDR %v", err)
	}
	log.Printf("Checking Overlap\nparentCidr:\t%s", parentCidr)
	var subnets []*net.IPNet
	subnets = append(subnets, newSubnet)
	log.Printf("newSubnet:\t%s/%d", newSubnet.IP.String(), netMask(newSubnet.Mask))
	for i := 0; i < len(subnetRanges); i++ {
		subnetRange := subnetRanges[i]
		netAddr, subnetCidr, err := net.ParseCIDR(subnetRange.Cidr)
		if err != nil {
			return fmt.Errorf("can't parse CIDR %v", err)
		}
		if parentNetwork.Contains(netAddr) {
			subnets = append(subnets, subnetCidr)
		}
	}

	return cidr.VerifyNoOverlap(subnets, parentNetwork)
}

func FindNextSubnet(range_size int, sourceRange string, existingRanges []model.Range) (*net.IPNet, int, error) {
	_, parentNet, err := net.ParseCIDR(sourceRange)
	if err != nil {
		return nil, -1, err
	}

	subnet, subnetOnes, err := CreateNewSubnetLease(sourceRange, range_size, 0)
	if err != nil {
		return nil, -1, err
	}
	log.Printf("new subnet lease %s/%d", subnet.IP.String(), subnetOnes)

	var lastSubnet = false
	for {
		err = VerifyNoOverlap(sourceRange, existingRanges, subnet)
		if err == nil {
			break
		} else if !lastSubnet {
			subnet, lastSubnet = cidr.NextSubnet(subnet, int(range_size))
			if !parentNet.Contains(subnet.IP) {
				return nil, -1, fmt.Errorf("no_address_range_available_in_parent")
			}
		} else {
			return nil, -1, err
		}
	}

	return subnet, subnetOnes, nil
}

func CreateNewSubnetLease(prevCidr string, range_size int, subnetIndex int) (*net.IPNet, int, error) {
	_, network, err := net.ParseCIDR(prevCidr)
	if err != nil {
		return nil, -1, fmt.Errorf("unable to calculate subnet %v", err)
	}
	ones, size := network.Mask.Size()
	subnet, err := cidr.Subnet(network, int(range_size)-ones, subnetIndex)
	if err != nil {
		return nil, -1, fmt.Errorf("unable to calculate subnet %v", err)
	}
	subnet.Mask = net.CIDRMask(range_size, size)
	return subnet, range_size, nil
}

func netMask(mask net.IPMask) int {
	ones, _ := mask.Size()
	return ones
}
