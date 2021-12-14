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

package cai

import (
	"context"
	"fmt"
	"log"
	"reflect"

	asset "cloud.google.com/go/asset/apiv1"
	"google.golang.org/api/iterator"
	assetpb "google.golang.org/genproto/googleapis/cloud/asset/v1"
)

type CaiSecondaryRange struct {
	Name string
	Cidr string
}
type CaiRange struct {
	Name            string
	Id              string
	Network         string
	Cidr            string
	SecondaryRanges []CaiSecondaryRange
}

func GetRangesForNetwork(parent string, networks []string) ([]CaiRange, error) {
	ctx := context.Background()
	client, err := asset.NewClient(ctx)
	if err != nil {
		return nil, err
	}

	defer client.Close()

	itr := client.ListAssets(ctx, &assetpb.ListAssetsRequest{
		Parent:      parent,
		AssetTypes:  []string{"compute.googleapis.com/Subnetwork"},
		ContentType: assetpb.ContentType_RESOURCE,
	})

	var ranges []CaiRange = make([]CaiRange, 0)

	asset, err := itr.Next()
	for {
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		if containsValue(networks, asset.Resource.Data.Fields["network"].GetStringValue()) {
			var secondaryRanges []CaiSecondaryRange = make([]CaiSecondaryRange, 0)
			secondary := asset.Resource.Data.Fields["secondaryIpRanges"].GetListValue().AsSlice()
			for i := 0; i < len(secondary); i++ {
				var rangeName string
				var ipCidrRange string

				iter := reflect.ValueOf(secondary[i]).MapRange()
				for iter.Next() {
					key := iter.Key().Interface()
					value := iter.Value().Interface()
					if key == "ipCidrRange" {
						ipCidrRange = fmt.Sprintf("%s", value)
					}
					if key == "rangeName" {
						rangeName = fmt.Sprintf("%s", value)
					}
				}
				secondaryRanges = append(secondaryRanges, CaiSecondaryRange{
					Name: rangeName,
					Cidr: ipCidrRange,
				})
			}
			ranges = append(ranges, CaiRange{
				Id:              asset.Resource.Data.Fields["id"].GetStringValue(),
				Name:            asset.Name,
				Network:         asset.Resource.Data.Fields["network"].GetStringValue(),
				Cidr:            asset.Resource.Data.Fields["ipCidrRange"].GetStringValue(),
				SecondaryRanges: secondaryRanges,
			})
		} else {
			log.Printf("Ignoring network %s", asset.Resource.Data.Fields["network"].GetStringValue())
		}
		asset, err = itr.Next()
	}
	return ranges, nil
}

func containsValue(array []string, lookup string) bool {
	for i := 0; i < len(array); i++ {
		if lookup == array[i] {
			return true
		}
	}
	return false
}
