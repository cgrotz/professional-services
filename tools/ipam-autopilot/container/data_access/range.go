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

	"github.com/GoogleCloudPlatform/professional-services/ipam-autopilot/model"
	"github.com/jackc/pgtype"

	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func GetRangesFromDB() ([]model.Range, error) {
	var ranges []model.Range

	rows, err := Db.Query("SELECT subnet_id, parent_id, routing_domain_id, name, cidr FROM subnets")
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var subnet_id int
		var routing_domain_id int
		tmp := pgtype.Int4{}
		var name string
		var cidr string
		err := rows.Scan(&subnet_id, &tmp, &routing_domain_id, &name, &cidr)
		if err != nil {
			return nil, err
		}
		parent_id := -1
		if tmp.Status == pgtype.Present {
			tmp.AssignTo(&parent_id)
		}

		ranges = append(ranges, model.Range{
			Subnet_id:         subnet_id,
			Parent_id:         parent_id,
			Routing_domain_id: routing_domain_id,
			Name:              name,
			Cidr:              cidr,
		})
	}
	return ranges, nil
}

func GetRangesForParentFromDB(tx *sql.Tx, parent_id int64) ([]model.Range, error) {
	var ranges []model.Range
	rows, err := tx.Query("SELECT subnet_id, parent_id, routing_domain_id, name, cidr FROM subnets WHERE parent_id = ? FOR UPDATE", parent_id)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var subnet_id int
		var routing_domain_id int
		tmp := pgtype.Int4{}
		var name string
		var cidr string
		err := rows.Scan(&subnet_id, &tmp, &routing_domain_id, &name, &cidr)
		if err != nil {
			return nil, err
		}
		parent_id := -1
		if tmp.Status == pgtype.Present {
			tmp.AssignTo(&parent_id)
		}

		ranges = append(ranges, model.Range{
			Subnet_id:         subnet_id,
			Parent_id:         parent_id,
			Routing_domain_id: routing_domain_id,
			Name:              name,
			Cidr:              cidr,
		})
	}
	return ranges, nil
}

func GetRangesForRoutingDomainFromDB(tx *sql.Tx, routingDomainId int) ([]model.Range, error) {
	var ranges []model.Range
	rows, err := tx.Query("SELECT subnet_id, parent_id, routing_domain_id, name, cidr FROM subnets WHERE routing_domain_id = ? FOR UPDATE", routingDomainId)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var subnet_id int
		var routing_domain_id int
		tmp := pgtype.Int4{}
		var name string
		var cidr string
		err := rows.Scan(&subnet_id, &tmp, &routing_domain_id, &name, &cidr)
		if err != nil {
			return nil, err
		}
		parent_id := -1
		if tmp.Status == pgtype.Present {
			tmp.AssignTo(&parent_id)
		}

		ranges = append(ranges, model.Range{
			Subnet_id:         subnet_id,
			Parent_id:         parent_id,
			Routing_domain_id: routing_domain_id,
			Name:              name,
			Cidr:              cidr,
		})
	}
	return ranges, nil
}

func GetRangeFromDB(id int64) (*model.Range, error) {
	var subnet_id int
	var routing_domain_id int
	tmp := pgtype.Int4{}
	var name string
	var cidr string

	err := Db.QueryRow("SELECT subnet_id, parent_id, routing_domain_id, name, cidr FROM subnets WHERE subnet_id = ?", id).Scan(&subnet_id, &tmp, &routing_domain_id, &name, &cidr)

	if err != nil {
		return nil, err
	}
	parent_id := -1
	if tmp.Status == pgtype.Present {
		tmp.AssignTo(&parent_id)
	}

	return &model.Range{
		Subnet_id:         subnet_id,
		Parent_id:         parent_id,
		Routing_domain_id: routing_domain_id,
		Name:              name,
		Cidr:              cidr,
	}, nil
}

func GetRangeFromDBWithTx(tx *sql.Tx, id int64) (*model.Range, error) {
	var subnet_id int
	var routing_domain_id int
	tmp := pgtype.Int4{}
	var name string
	var cidr string

	err := tx.QueryRow("SELECT subnet_id, parent_id, routing_domain_id, name, cidr FROM subnets WHERE subnet_id = ? FOR UPDATE", id).Scan(&subnet_id, &tmp, &routing_domain_id, &name, &cidr)

	if err != nil {
		return nil, err
	}
	parent_id := -1
	if tmp.Status == pgtype.Present {
		tmp.AssignTo(&parent_id)
	}

	return &model.Range{
		Subnet_id:         subnet_id,
		Parent_id:         parent_id,
		Routing_domain_id: routing_domain_id,
		Name:              name,
		Cidr:              cidr,
	}, nil
}

func GetRangeByCidrAndRoutingDomain(tx *sql.Tx, request_cidr string, routing_domain_id int) (*model.Range, error) {
	var subnet_id int
	tmp := pgtype.Int4{}
	var name string
	var cidr string

	err := tx.QueryRow("SELECT subnet_id, parent_id, name, cidr FROM subnets WHERE cidr = ? and routing_domain_id = ? FOR UPDATE", request_cidr, routing_domain_id).Scan(&subnet_id, &tmp, &name, &cidr)
	if err != nil {
		return nil, err
	}

	parent_id := -1
	if tmp.Status == pgtype.Present {
		tmp.AssignTo(&parent_id)
	}

	return &model.Range{
		Subnet_id:         subnet_id,
		Parent_id:         parent_id,
		Routing_domain_id: routing_domain_id,
		Name:              name,
		Cidr:              cidr,
	}, nil
}

func GetRangeByCidrFromDB(tx *sql.Tx, routing_domain_id int, cidr_request string) (*model.Range, error) {
	var subnet_id int
	tmp := pgtype.Int4{}
	var name string
	var cidr string

	if cidr_request != "" {
		err := tx.QueryRow("SELECT subnet_id, parent_id, name, cidr FROM subnets WHERE cidr = ? and routing_domain_id = ? FOR UPDATE", cidr_request, routing_domain_id).Scan(&subnet_id, &tmp, &name, &cidr)
		if err != nil {
			return nil, err
		}
	} else {
		err := tx.QueryRow("SELECT subnet_id, parent_id, name, cidr FROM subnets WHERE routing_domain_id = ? LIMIT 1 FOR UPDATE", routing_domain_id).Scan(&subnet_id, &tmp, &name, &cidr)
		if err != nil {
			return nil, err
		}
	}
	parent_id := -1
	if tmp.Status == pgtype.Present {
		tmp.AssignTo(&parent_id)
	}

	return &model.Range{
		Subnet_id:         subnet_id,
		Parent_id:         parent_id,
		Routing_domain_id: routing_domain_id,
		Name:              name,
		Cidr:              cidr,
	}, nil
}

func DeleteRangeFromDb(id int64) error {
	_, err := Db.Query("DELETE FROM subnets WHERE subnet_id = ?", id)

	if err != nil {
		return err
	}
	return nil
}

func CreateRangeInDb(tx *sql.Tx, parent_id int64, routing_domain_id int, name string, cidr string) (int64, error) {
	if parent_id == -1 {
		res, err := tx.Exec("INSERT INTO subnets (routing_domain_id, name, cidr) VALUES (?,?,?);", routing_domain_id, name, cidr)
		if err != nil {
			return -1, err
		}
		subnet_id, err := res.LastInsertId()
		if err != nil {
			return -1, err
		}
		return subnet_id, nil
	} else {
		res, err := tx.Exec("INSERT INTO subnets (parent_id, routing_domain_id, name, cidr) VALUES (?,?,?,?);", parent_id, routing_domain_id, name, cidr)
		if err != nil {
			return -1, err
		}
		subnet_id, err := res.LastInsertId()
		if err != nil {
			return -1, err
		}
		return subnet_id, nil
	}
}
