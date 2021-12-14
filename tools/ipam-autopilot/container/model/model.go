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

package model

import "encoding/json"

type RoutingDomain struct {
	Id   int    `db:"routing_domain_id"`
	Name string `db:"name"`
	Vpcs string `db:"vpcs"` // associated VPCs that should be tracked for subnet creation
}

type Range struct {
	Subnet_id         int    `db:"subnet_id"`
	Parent_id         int    `db:"parent_id"`
	Routing_domain_id int    `db:"routing_domain_id"`
	Name              string `db:"name"`
	Cidr              string `db:"cidr"`
}

type JSONString struct {
	Value string
	Set   bool
}

func (i *JSONString) UnmarshalJSON(data []byte) error {
	i.Set = true
	var val string
	if err := json.Unmarshal(data, &val); err != nil {
		return err
	}
	i.Value = val
	return nil
}

type JSONStringArray struct {
	Value []string
	Set   bool
}

func (i *JSONStringArray) UnmarshalJSON(data []byte) error {
	i.Set = true
	var val []string
	if err := json.Unmarshal(data, &val); err != nil {
		return err
	}
	i.Value = val
	return nil
}
