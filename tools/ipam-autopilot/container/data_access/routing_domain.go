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

package data_access

import (
	"database/sql"
	"strings"

	"github.com/GoogleCloudPlatform/professional-services/ipam-autopilot/model"
)

func GetDefaultRoutingDomainFromDB(tx *sql.Tx) (*model.RoutingDomain, error) {
	var routing_domain_id int
	var name string
	var vpcs sql.NullString

	err := tx.QueryRow("SELECT routing_domain_id, name, vpcs FROM routing_domains LIMIT 1 FOR UPDATE").Scan(&routing_domain_id, &name, &vpcs)
	if err != nil {
		return nil, err
	}

	return &model.RoutingDomain{
		Id:   routing_domain_id,
		Name: name,
		Vpcs: vpcs.String,
	}, nil
}

func GetRoutingDomainsFromDB() ([]model.RoutingDomain, error) {
	var domains []model.RoutingDomain
	rows, err := Db.Query("SELECT routing_domain_id, name, vpcs FROM routing_domains")
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var routing_domain_id int
		var name string
		var vpcs sql.NullString
		err := rows.Scan(&routing_domain_id, &name, &vpcs)
		if err != nil {
			return nil, err
		}
		domains = append(domains, model.RoutingDomain{
			Id:   routing_domain_id,
			Name: name,
			Vpcs: vpcs.String,
		})
	}
	return domains, nil
}

func FindRoutingDomainFromDBWithVpcName(tx *sql.Tx, vpc string) (*model.RoutingDomain, error) {
	var routing_domain_id int
	var name string
	var vpcs sql.NullString

	err := Db.QueryRow("SELECT routing_domain_id, name, vpcs FROM routing_domains WHERE vpcs = $?$", vpc).Scan(&routing_domain_id, &name, &vpcs)
	if err != nil {
		return nil, err
	}

	return &model.RoutingDomain{
		Id:   routing_domain_id,
		Name: name,
		Vpcs: vpcs.String,
	}, nil
}

func GetRoutingDomainFromDB(id int64) (*model.RoutingDomain, error) {
	var routing_domain_id int
	var name string
	var vpcs sql.NullString

	err := Db.QueryRow("SELECT routing_domain_id, name, vpcs FROM routing_domains WHERE routing_domain_id = ?", id).Scan(&routing_domain_id, &name, &vpcs)
	if err != nil {
		return nil, err
	}

	return &model.RoutingDomain{
		Id:   routing_domain_id,
		Name: name,
		Vpcs: vpcs.String,
	}, nil
}

func UpdateRoutingDomainOnDb(id int64, name model.JSONString, vpcs model.JSONStringArray) error {
	if name.Set && vpcs.Set {
		_, err := Db.Query("UPDATE routing_domains SET name = ?, vpcs = ? WHERE routing_domain_id = ?", id, name.Value, strings.Join(vpcs.Value, ","))
		if err != nil {
			return err
		}
	} else if vpcs.Set {
		_, err := Db.Query("UPDATE routing_domains SET vpcs = ? WHERE routing_domain_id = ?", id, strings.Join(vpcs.Value, ","))
		if err != nil {
			return err
		}
	} else if name.Set {
		_, err := Db.Query("UPDATE routing_domains SET name = ? WHERE routing_domain_id = ?", id, name.Value)
		if err != nil {
			return err
		}
	}
	return nil
}

func CreateRoutingDomainOnDb(name string, vpcs []string) (int64, error) {
	res, err := Db.Exec("INSERT INTO routing_domains (name, vpcs) VALUES (?,?);", name, strings.Join(vpcs, ","))
	if err != nil {
		return -1, err
	}
	domain_id, err := res.LastInsertId()
	if err != nil {
		return -1, err
	}
	return domain_id, nil
}

func DeleteRoutingDomainFromDB(id int64) error {
	_, err := Db.Query("DELETE FROM routing_domains WHERE routing_domain_id = ?", id)

	if err != nil {
		return err
	}
	return nil
}
