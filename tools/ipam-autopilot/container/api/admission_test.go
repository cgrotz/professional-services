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
	"bytes"
	"io"
	"log"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/GoogleCloudPlatform/professional-services/ipam-autopilot/data_access"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

var mock sqlmock.Sqlmock

func InitMockDb() {
	var err error
	data_access.Db, mock, err = sqlmock.New()
	if err != nil {
		log.Fatal(err)
	}
}

func TestValidatingWithoutOverlap(t *testing.T) {
	InitMockDb()
	defer data_access.Close()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT routing_domain_id, name, vpcs FROM routing_domains LIMIT 1 FOR UPDATE").WillReturnRows(sqlmock.NewRows([]string{"routing_domain_id", "name", "vpcs"}).FromCSVString("1,DEFAULT,"))
	mock.ExpectQuery("SELECT subnet_id, parent_id, routing_domain_id, name, cidr FROM subnets WHERE routing_domain_id = (.+) FOR UPDATE").WillReturnRows(sqlmock.NewRows([]string{"subnet_id", "parent_id", "routing_domain_id", "name", "cidr"}))

	mock.ExpectCommit()
	// Define Fiber app.
	app := fiber.New()

	// Create route with GET method for test
	app.Post("/admission/validating", ValidatingWebhook)

	data := []byte(`{
		"kind": "AdmissionReview",
		"apiVersion": "admission.k8s.io/v1",
		"request": {
		  "uid": "9ec4fd53-d3cf-4e9b-af6a-25a398697e3c",
		  "kind": {
			"group": "compute.cnrm.cloud.google.com",
			"version": "v1beta1",
			"kind": "ComputeSubnetwork"
		  },
		  "resource": {
			"group": "compute.cnrm.cloud.google.com",
			"version": "v1beta1",
			"resource": "computesubnetworks"
		  },
		  "requestKind": {
			"group": "compute.cnrm.cloud.google.com",
			"version": "v1beta1",
			"kind": "ComputeSubnetwork"
		  },
		  "requestResource": {
			"group": "compute.cnrm.cloud.google.com",
			"version": "v1beta1",
			"resource": "computesubnetworks"
		  },
		  "name": "computesubnetwork-sample2",
		  "namespace": "default",
		  "operation": "CREATE",
		  "userInfo": {
			"username": "grotz@grotz.joonix.net",
			"groups": [
			  "system:authenticated"
			],
			"extra": {
			  "iam.gke.io/user-assertion": [
				"AK8TC8Lh3gRR8JYyhlGL4hyHQAe90AEOafQmNdeFYfw7j1hofgD5VZUQmQdKtUVWz9z99G/KAG1B82rtCx20TX1CWDVTL9bjVKWwvH44rlztiQf8x1RuJdyusNdbetCHG1kK2k9UWTkvTkyrK8WL4RXHItGUlvVJtv1jCQsUJVMM3PUx7fT45SHB+OZlWM0IZ6vzFLWjDxem0Hh/skXjKRm2ao2ELqlkTDWA+IDPzvy8m2qGW53u7C6bYdmzw3pY9FR7ze9YMbAuc0HlpnE8ORWfCaqalqNLab82b9ICQErE5S5zgkv98uKLv2B2en9QuHzFA9fADny7ldQfhEc2qJVyS2P2dZ+wJqQgZnOpgJ0S9Eow7jOqN083qiVbpn50WKIJ4gMzpt1cffB9v/rqH/ruPBaJxzJ3z8+dIyCXoy0Lrx6pxHi9oINRzOJJtBqj56kGbW8h31lwuTi41Up81+puIAcUD/1xG3jDjnh6EHmZWdjUGL9tBb+lIvUcp1JpGYmN1nfo/SdL26x86brQfYu8EjJ13BsVkLjp6Z1Ba3CHrVriBwWppozXxhjIdUUd/mxL92JddUBYaJ17JDBvP62aaABof6vQubbp5zrL2s5/iZvHrI6dTDKf6RWq+UKnO25iWVzKqTZ0J6kWPItNZkSt7RC/IJaoHCr+CB+WzckNddY4KIn9ilOfdwGyr4g3yOGuwaX2knHLkjSYYiJeJgsUri1Zna3ZurWfk/oAqhHVBey0HQ52R3u7lsrf5Px191SGVzzcHOhL0lXCNH/nP5SHql0B7fo2Q85oYIgYeVwP1xSTu5RXEgBTrV8sHq6/OxHQ12P8aV01xiRQlh+v5RlFMvUmB1k0zl6/eu/mM7+EQNKkl4+KeT6rpO7J2YeZeD4pe215UHRfEtFa6wBJvDHfSntfPxSOJVp7oWRe0jBC7CJLY35KXsagwIofIG7AYxFnY3Qi+9fpKuAPf91s7b7Khz7J8iPVMPMqMEqrpXO11/Ue4e0Aw0KP+wZ7L9AL4XgrGPPhj5qIZpcZafYEJyivLjOpTWcUW9LZp36BM7443loY86bGwce6+1PW7UMGkSb+0CeYNzKdGO7JUPpm8X5n7xG4p2fdV6bUuHrI/Iyy+vI4vACNnjqmlg9EFQ=="
			  ],
			  "user-assertion.cloud.google.com": [
				"AK8TC8Jjdjp4VSyb8kHLQWhJImroENwF+9ioTbRIbowZHE25m6MLxAPLtLVH+M2Li9mtiBzRaAwRAaZdHCQSH3ZOqD/1vvh6eRmwHyGdm8jwfk56xLyeNLHjZht/CphA8RZW2dsxjMFiVbwlwMaPpS1qrMTa0Mqt5zuB0aW/fmsMEs/avtTMyFilPpVhWbKFKtCcfo5ZAOmHi91aTcoqoXl6tYvbzY9WjLcYUFqe0Ic="
			  ]
			}
		  },
		  "object": {
			"apiVersion": "compute.cnrm.cloud.google.com/v1beta1",
			"kind": "ComputeSubnetwork",
			"metadata": {
			  "annotations": {
				"cnrm.cloud.google.com/management-conflict-prevention-policy": "none",
				"cnrm.cloud.google.com/project-id": "default",
				"cnrm.cloud.google.com/state-into-spec": "merge",
				"kubectl.kubernetes.io/last-applied-configuration": "{\"apiVersion\":\"compute.cnrm.cloud.google.com/v1beta1\",\"kind\":\"ComputeSubnetwork\",\"metadata\":{\"annotations\":{},\"labels\":{\"label-one\":\"value-one\"},\"name\":\"computesubnetwork-sample2\",\"namespace\":\"default\"},\"spec\":{\"description\":\"My subnet2\",\"ipCidrRange\":\"10.2.0.0/16\",\"logConfig\":{\"aggregationInterval\":\"INTERVAL_10_MIN\",\"flowSampling\":0.5,\"metadata\":\"INCLUDE_ALL_METADATA\"},\"networkRef\":{\"name\":\"computesubnetwork-dep2\"},\"privateIpGoogleAccess\":false,\"region\":\"us-central1\"}}\n"
			  },
			  "creationTimestamp": "2021-12-24T08:36:45Z",
			  "generation": 1,
			  "labels": {
				"label-one": "value-one"
			  },
			  "managedFields": [
				{
				  "apiVersion": "compute.cnrm.cloud.google.com/v1beta1",
				  "fieldsType": "FieldsV1",
				  "fieldsV1": {
					"f:metadata": {
					  "f:annotations": {
						".": {},
						"f:kubectl.kubernetes.io/last-applied-configuration": {}
					  },
					  "f:labels": {
						".": {},
						"f:label-one": {}
					  }
					},
					"f:spec": {
					  ".": {},
					  "f:description": {},
					  "f:ipCidrRange": {},
					  "f:logConfig": {
						".": {},
						"f:aggregationInterval": {},
						"f:flowSampling": {},
						"f:metadata": {}
					  },
					  "f:networkRef": {
						".": {},
						"f:name": {}
					  },
					  "f:privateIpGoogleAccess": {},
					  "f:region": {}
					}
				  },
				  "manager": "kubectl-client-side-apply",
				  "operation": "Update",
				  "time": "2021-12-24T08:36:45Z"
				}
			  ],
			  "name": "computesubnetwork-sample2",
			  "namespace": "default",
			  "uid": "96527522-64fe-40f1-9c73-98d6311dfae3"
			},
			"spec": {
			  "description": "My subnet2",
			  "ipCidrRange": "10.2.0.0/16",
			  "logConfig": {
				"aggregationInterval": "INTERVAL_10_MIN",
				"flowSampling": 0.5,
				"metadata": "INCLUDE_ALL_METADATA"
			  },
			  "networkRef": {
				"name": "computesubnetwork-dep2"
			  },
			  "privateIpGoogleAccess": false,
			  "region": "us-central1"
			}
		  },
		  "oldObject": null,
		  "dryRun": false,
		  "options": {
			"kind": "CreateOptions",
			"apiVersion": "meta.k8s.io/v1",
			"fieldManager": "kubectl-client-side-apply"
		  }
		}
	  } 
	`)
	req := httptest.NewRequest("POST", "/admission/validating", bytes.NewReader(data))
	req.Header["Content-Type"] = []string{"application/json"}
	resp, err := app.Test(req, -1)
	if err != nil {
		log.Fatal(err)
	}
	assert.Equalf(t, 200, resp.StatusCode, "Status Code")
	buf := new(strings.Builder)
	io.Copy(buf, resp.Body)
	respBody := buf.String()
	assert.Equalf(t, "{\"kind\":\"AdmissionReview\",\"apiVersion\":\"admission.k8s.io/v1\",\"response\":{\"uid\":\"9ec4fd53-d3cf-4e9b-af6a-25a398697e3c\",\"allowed\":true}}", respBody, "Body")
}

func TestValidatingWithOverlap(t *testing.T) {
	InitMockDb()
	defer data_access.Close()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT routing_domain_id, name, vpcs FROM routing_domains WHERE routing_domain_id = (.+)").WillReturnRows(sqlmock.NewRows([]string{"routing_domain_id", "name", "vpcs"}).FromCSVString("1,DEFAULT,"))
	mock.ExpectQuery("SELECT subnet_id, parent_id, routing_domain_id, name, cidr FROM subnets WHERE routing_domain_id = (.+) FOR UPDATE").WillReturnRows(sqlmock.NewRows([]string{"subnet_id", "parent_id", "routing_domain_id", "name", "cidr"}).FromCSVString("1,-1,1,test,10.0.0.0/8"))

	mock.ExpectCommit()
	// Define Fiber app.
	app := fiber.New()

	// Create route with GET method for test
	app.Post("/admission/validating", ValidatingWebhook)

	data := []byte(`{
		"kind": "AdmissionReview",
		"apiVersion": "admission.k8s.io/v1",
		"request": {
		  "uid": "9ec4fd53-d3cf-4e9b-af6a-25a398697e3c",
		  "kind": {
			"group": "compute.cnrm.cloud.google.com",
			"version": "v1beta1",
			"kind": "ComputeSubnetwork"
		  },
		  "resource": {
			"group": "compute.cnrm.cloud.google.com",
			"version": "v1beta1",
			"resource": "computesubnetworks"
		  },
		  "requestKind": {
			"group": "compute.cnrm.cloud.google.com",
			"version": "v1beta1",
			"kind": "ComputeSubnetwork"
		  },
		  "requestResource": {
			"group": "compute.cnrm.cloud.google.com",
			"version": "v1beta1",
			"resource": "computesubnetworks"
		  },
		  "name": "computesubnetwork-sample2",
		  "namespace": "default",
		  "operation": "CREATE",
		  "userInfo": {
			"username": "grotz@grotz.joonix.net",
			"groups": [
			  "system:authenticated"
			],
			"extra": {
			  "iam.gke.io/user-assertion": [
				"AK8TC8Lh3gRR8JYyhlGL4hyHQAe90AEOafQmNdeFYfw7j1hofgD5VZUQmQdKtUVWz9z99G/KAG1B82rtCx20TX1CWDVTL9bjVKWwvH44rlztiQf8x1RuJdyusNdbetCHG1kK2k9UWTkvTkyrK8WL4RXHItGUlvVJtv1jCQsUJVMM3PUx7fT45SHB+OZlWM0IZ6vzFLWjDxem0Hh/skXjKRm2ao2ELqlkTDWA+IDPzvy8m2qGW53u7C6bYdmzw3pY9FR7ze9YMbAuc0HlpnE8ORWfCaqalqNLab82b9ICQErE5S5zgkv98uKLv2B2en9QuHzFA9fADny7ldQfhEc2qJVyS2P2dZ+wJqQgZnOpgJ0S9Eow7jOqN083qiVbpn50WKIJ4gMzpt1cffB9v/rqH/ruPBaJxzJ3z8+dIyCXoy0Lrx6pxHi9oINRzOJJtBqj56kGbW8h31lwuTi41Up81+puIAcUD/1xG3jDjnh6EHmZWdjUGL9tBb+lIvUcp1JpGYmN1nfo/SdL26x86brQfYu8EjJ13BsVkLjp6Z1Ba3CHrVriBwWppozXxhjIdUUd/mxL92JddUBYaJ17JDBvP62aaABof6vQubbp5zrL2s5/iZvHrI6dTDKf6RWq+UKnO25iWVzKqTZ0J6kWPItNZkSt7RC/IJaoHCr+CB+WzckNddY4KIn9ilOfdwGyr4g3yOGuwaX2knHLkjSYYiJeJgsUri1Zna3ZurWfk/oAqhHVBey0HQ52R3u7lsrf5Px191SGVzzcHOhL0lXCNH/nP5SHql0B7fo2Q85oYIgYeVwP1xSTu5RXEgBTrV8sHq6/OxHQ12P8aV01xiRQlh+v5RlFMvUmB1k0zl6/eu/mM7+EQNKkl4+KeT6rpO7J2YeZeD4pe215UHRfEtFa6wBJvDHfSntfPxSOJVp7oWRe0jBC7CJLY35KXsagwIofIG7AYxFnY3Qi+9fpKuAPf91s7b7Khz7J8iPVMPMqMEqrpXO11/Ue4e0Aw0KP+wZ7L9AL4XgrGPPhj5qIZpcZafYEJyivLjOpTWcUW9LZp36BM7443loY86bGwce6+1PW7UMGkSb+0CeYNzKdGO7JUPpm8X5n7xG4p2fdV6bUuHrI/Iyy+vI4vACNnjqmlg9EFQ=="
			  ],
			  "user-assertion.cloud.google.com": [
				"AK8TC8Jjdjp4VSyb8kHLQWhJImroENwF+9ioTbRIbowZHE25m6MLxAPLtLVH+M2Li9mtiBzRaAwRAaZdHCQSH3ZOqD/1vvh6eRmwHyGdm8jwfk56xLyeNLHjZht/CphA8RZW2dsxjMFiVbwlwMaPpS1qrMTa0Mqt5zuB0aW/fmsMEs/avtTMyFilPpVhWbKFKtCcfo5ZAOmHi91aTcoqoXl6tYvbzY9WjLcYUFqe0Ic="
			  ]
			}
		  },
		  "object": {
			"apiVersion": "compute.cnrm.cloud.google.com/v1beta1",
			"kind": "ComputeSubnetwork",
			"metadata": {
			  "annotations": {
				"cnrm.cloud.google.com/management-conflict-prevention-policy": "none",
				"cnrm.cloud.google.com/project-id": "default",
				"cnrm.cloud.google.com/state-into-spec": "merge",
				"kubectl.kubernetes.io/last-applied-configuration": "{\"apiVersion\":\"compute.cnrm.cloud.google.com/v1beta1\",\"kind\":\"ComputeSubnetwork\",\"metadata\":{\"annotations\":{},\"labels\":{\"label-one\":\"value-one\"},\"name\":\"computesubnetwork-sample2\",\"namespace\":\"default\"},\"spec\":{\"description\":\"My subnet2\",\"ipCidrRange\":\"10.2.0.0/16\",\"logConfig\":{\"aggregationInterval\":\"INTERVAL_10_MIN\",\"flowSampling\":0.5,\"metadata\":\"INCLUDE_ALL_METADATA\"},\"networkRef\":{\"name\":\"computesubnetwork-dep2\"},\"privateIpGoogleAccess\":false,\"region\":\"us-central1\"}}\n",
				"ipam.cloud.google.com/routing-domain-id": "1"
			  },
			  "creationTimestamp": "2021-12-24T08:36:45Z",
			  "generation": 1,
			  "labels": {
				"label-one": "value-one"
			  },
			  "managedFields": [
				{
				  "apiVersion": "compute.cnrm.cloud.google.com/v1beta1",
				  "fieldsType": "FieldsV1",
				  "fieldsV1": {
					"f:metadata": {
					  "f:annotations": {
						".": {},
						"f:kubectl.kubernetes.io/last-applied-configuration": {}
					  },
					  "f:labels": {
						".": {},
						"f:label-one": {}
					  }
					},
					"f:spec": {
					  ".": {},
					  "f:description": {},
					  "f:ipCidrRange": {},
					  "f:logConfig": {
						".": {},
						"f:aggregationInterval": {},
						"f:flowSampling": {},
						"f:metadata": {}
					  },
					  "f:networkRef": {
						".": {},
						"f:name": {}
					  },
					  "f:privateIpGoogleAccess": {},
					  "f:region": {}
					}
				  },
				  "manager": "kubectl-client-side-apply",
				  "operation": "Update",
				  "time": "2021-12-24T08:36:45Z"
				}
			  ],
			  "name": "computesubnetwork-sample2",
			  "namespace": "default",
			  "uid": "96527522-64fe-40f1-9c73-98d6311dfae3"
			},
			"spec": {
			  "description": "My subnet2",
			  "ipCidrRange": "10.2.0.0/16",
			  "logConfig": {
				"aggregationInterval": "INTERVAL_10_MIN",
				"flowSampling": 0.5,
				"metadata": "INCLUDE_ALL_METADATA"
			  },
			  "networkRef": {
				"name": "computesubnetwork-dep2"
			  },
			  "privateIpGoogleAccess": false,
			  "region": "us-central1"
			}
		  },
		  "oldObject": null,
		  "dryRun": false,
		  "options": {
			"kind": "CreateOptions",
			"apiVersion": "meta.k8s.io/v1",
			"fieldManager": "kubectl-client-side-apply"
		  }
		}
	  } 
	`)
	req := httptest.NewRequest("POST", "/admission/validating", bytes.NewReader(data))
	req.Header["Content-Type"] = []string{"application/json"}
	resp, err := app.Test(req, -1)
	if err != nil {
		log.Fatal(err)
	}
	assert.Equalf(t, 200, resp.StatusCode, "Status Code")
	buf := new(strings.Builder)
	io.Copy(buf, resp.Body)
	respBody := buf.String()
	assert.Equalf(t, "{\"kind\":\"AdmissionReview\",\"apiVersion\":\"admission.k8s.io/v1\",\"response\":{\"uid\":\"9ec4fd53-d3cf-4e9b-af6a-25a398697e3c\",\"allowed\":false,\"status\":{\"metadata\":{},\"status\":\"Failure\",\"message\":\"Overlap for routing domain detected\"}}}", respBody, "Body")
}

func TestValidatingNoMetadata(t *testing.T) {
	InitMockDb()
	defer data_access.Close()
	mock.ExpectBegin()
	mock.ExpectCommit()
	// Define Fiber app.
	app := fiber.New()

	// Create route with GET method for test
	app.Post("/admission/validating", ValidatingWebhook)

	data := []byte(`{"kind":"AdmissionReview","apiVersion":"admission.k8s.io/v1","request":{"uid":"99a9c86a-1d7a-44df-9711-7521b30121ff","kind":{"group":"compute.cnrm.cloud.google.com","version":"v1beta1","kind":"ComputeSubnetwork"},"resource":{"group":"compute.cnrm.cloud.google.com","version":"v1beta1","resource":"computesubnetworks"},"requestKind":{"group":"compute.cnrm.cloud.google.com","version":"v1beta1","kind":"ComputeSubnetwork"},"requestResource":{"group":"compute.cnrm.cloud.google.com","version":"v1beta1","resource":"computesubnetworks"},"name":"computesubnetwork-sample2","namespace":"default","operation":"CREATE","userInfo":{"username":"grotz@grotz.joonix.net","groups":["system:authenticated"],"extra":{"iam.gke.io/user-assertion":["AK8TC8L1fOuuYjXQdgv5DvXGqbLtML/iFEhzaCFllbaI5E5FcafaajxnmCiGLEwLWLsgFMJ7OydHFZMlpUYJklnxJ73yd4w2lJoOIDwdQFSsmKTy9m8lbJo5WRRKnSjXRnpl4Upi0d43sB05qwBdhCeei2Y/TWEqs6d8PEZMoWRAsDweVcORzghoWSA8gV2YmUY2ZRMkIt2ZFm8AUuHZGeVylhCNz5p4YalFUL4Hb1ED0JOA4nAAHN5gMP6HY6fjUHG77RQWCGyF3LpFzeD0PseDm9Dbbjshwp+HcFdIC4yq1uVidi3tbcoYXKD0l5/EXB1EhfYaXdnh+IAPh7ZzFNwDZvp2LBHJaCkweos5s7cm8jpHsE0movoExywbRDvHwPvdp/I0zSvWUa8Smvju+FncpKQ1Bp49vocWSfEFzMEPEf+MyO7DtHe5suFObaqHdH2NQ8ouA+RopPEFM+DfCFzM76wYHzS6n4E5pfhhTSZYw47cDg5UWCozDqBW2jhvHC43mW7nTcpkSkVWtbBXhLX5Ssrb+bC/WR9vm/ZHPpsBr89SXLRrrdYL1jnsJRUg24WMbHOzRo9hsX4y5S3JV4OOYUEK5q5wJtUAzwXVMtiyLwYq6Zj0mXLwlt9HYAYhPugdwEX/ovLGOrUb/hIgNjtSGPJoMn5Kg6mbQOV4K7tN4Bd1hsdCYLUhZwDTeUAdb0stzJLQxccH9bwt0bMNMnD+bw/UpIHyuGh0j19dnGp+BW/9HHcdB+IXbTpH6oFypsuknNmgZ6v4HVlc/5fA2EvqgthLCpl0Zqlbjq3GUeZ5oZ9nmvbiOJdZbkUOArEH0t/51UoXdT9qy/mEteBCh44DeA3dwNPTvdwZRnsWp1DrkLvStIoZ4gsZYrLDsM7f/I13No4VcJMYtSTHOVfwZTC8H6dq53bxaCQKj+lL2NlR4B3Q6BKZU6rUoIWG1vFZsI+h7nY5yzcG5OL/wHfROX6QxQwBe0XxPG7xaBGo0YxVlrsIZnO8jAVxAfiIgcCG6KcEIhBx+XecaAc8LVwg0151dJrVJcJRn6OnKlMkUQ5yrjAyHm2MdZrcG1E4SKbTzcDDU/bh9uXva3mAZ/KLjcRfnOETL5q69LUgr6+fn9EKFYjQ61Gvh5ZMVKgLpQ=="],"user-assertion.cloud.google.com":["AK8TC8JLmuvX3YK1HTVggi5NSd6rdPKZH3/TUQDVzhFTN/70O0K3+s8pdOl5v/L322/QnYEG07NwzJkatlCoLuD3WCUzsypcskqYimw94sXQUtU2mMp7cnKcB3LXYas7Ez3MPCSj3uW2KdA2lpimIUAYU1ao1VoYNRJ4cBKXckvatLgGo7tBR3sn51qxWA/fQogR7aoWrDYGEh9jKGmuHSJKAE212B0NoAdh0oqnDvU="]}},"object":{"apiVersion":"compute.cnrm.cloud.google.com/v1beta1","kind":"ComputeSubnetwork","metadata":{"annotations":{"cnrm.cloud.google.com/management-conflict-prevention-policy":"none","cnrm.cloud.google.com/project-id":"default","cnrm.cloud.google.com/state-into-spec":"merge","kubectl.kubernetes.io/last-applied-configuration":"{\"apiVersion\":\"compute.cnrm.cloud.google.com/v1beta1\",\"kind\":\"ComputeSubnetwork\",\"metadata\":{\"annotations\":{},\"labels\":{\"label-one\":\"value-one\"},\"name\":\"computesubnetwork-sample2\",\"namespace\":\"default\"},\"spec\":{\"description\":\"My subnet2\",\"ipCidrRange\":\"10.2.0.0/16\",\"logConfig\":{\"aggregationInterval\":\"INTERVAL_10_MIN\",\"flowSampling\":0.5,\"metadata\":\"INCLUDE_ALL_METADATA\"},\"networkRef\":{\"name\":\"computesubnetwork-dep2\"},\"privateIpGoogleAccess\":false,\"region\":\"us-central1\"}}\n"},"creationTimestamp":"2021-12-25T21:56:00Z","generation":1,"labels":{"label-one":"value-one"},"managedFields":[{"apiVersion":"compute.cnrm.cloud.google.com/v1beta1","fieldsType":"FieldsV1","fieldsV1":{"f:metadata":{"f:annotations":{".":{},"f:kubectl.kubernetes.io/last-applied-configuration":{}},"f:labels":{".":{},"f:label-one":{}}},"f:spec":{".":{},"f:description":{},"f:ipCidrRange":{},"f:logConfig":{".":{},"f:aggregationInterval":{},"f:flowSampling":{},"f:metadata":{}},"f:networkRef":{".":{},"f:name":{}},"f:privateIpGoogleAccess":{},"f:region":{}}},"manager":"kubectl-client-side-apply","operation":"Update","time":"2021-12-25T21:55:59Z"}],"name":"computesubnetwork-sample2","namespace":"default","uid":"a4f9ede5-e4c1-4999-8514-f7340e2f69de"},"spec":{"description":"My subnet2","ipCidrRange":"10.2.0.0/16","logConfig":{"aggregationInterval":"INTERVAL_10_MIN","flowSampling":0.5,"metadata":"INCLUDE_ALL_METADATA"},"networkRef":{"name":"computesubnetwork-dep2"},"privateIpGoogleAccess":false,"region":"us-central1"}},"oldObject":null,"dryRun":false,"options":{"kind":"CreateOptions","apiVersion":"meta.k8s.io/v1","fieldManager":"kubectl-client-side-apply"}}}
	`)
	req := httptest.NewRequest("POST", "/admission/validating", bytes.NewReader(data))
	req.Header["Content-Type"] = []string{"application/json"}
	resp, err := app.Test(req, -1)
	if err != nil {
		log.Fatal(err)
	}
	assert.Equalf(t, 200, resp.StatusCode, "Status Code")
	buf := new(strings.Builder)
	io.Copy(buf, resp.Body)
	respBody := buf.String()
	assert.Equalf(t, "{\"kind\":\"AdmissionReview\",\"apiVersion\":\"admission.k8s.io/v1\",\"response\":{\"uid\":\"99a9c86a-1d7a-44df-9711-7521b30121ff\",\"allowed\":true}}", respBody, "Body")
}

func TestMutatingNoDataGiven(t *testing.T) {
	// Define Fiber app.
	app := fiber.New()

	// Create route with GET method for test
	app.Post("/admission/mutating", MutatingWebhook)

	data := []byte(`{
		"kind": "AdmissionReview",
		"apiVersion": "admission.k8s.io/v1",
		"request": {
		  "uid": "9ec4fd53-d3cf-4e9b-af6a-25a398697e3c",
		  "kind": {
			"group": "compute.cnrm.cloud.google.com",
			"version": "v1beta1",
			"kind": "ComputeSubnetwork"
		  },
		  "resource": {
			"group": "compute.cnrm.cloud.google.com",
			"version": "v1beta1",
			"resource": "computesubnetworks"
		  },
		  "requestKind": {
			"group": "compute.cnrm.cloud.google.com",
			"version": "v1beta1",
			"kind": "ComputeSubnetwork"
		  },
		  "requestResource": {
			"group": "compute.cnrm.cloud.google.com",
			"version": "v1beta1",
			"resource": "computesubnetworks"
		  },
		  "name": "computesubnetwork-sample2",
		  "namespace": "default",
		  "operation": "CREATE",
		  "userInfo": {
			"username": "grotz@grotz.joonix.net",
			"groups": [
			  "system:authenticated"
			],
			"extra": {
			  "iam.gke.io/user-assertion": [
				"AK8TC8Lh3gRR8JYyhlGL4hyHQAe90AEOafQmNdeFYfw7j1hofgD5VZUQmQdKtUVWz9z99G/KAG1B82rtCx20TX1CWDVTL9bjVKWwvH44rlztiQf8x1RuJdyusNdbetCHG1kK2k9UWTkvTkyrK8WL4RXHItGUlvVJtv1jCQsUJVMM3PUx7fT45SHB+OZlWM0IZ6vzFLWjDxem0Hh/skXjKRm2ao2ELqlkTDWA+IDPzvy8m2qGW53u7C6bYdmzw3pY9FR7ze9YMbAuc0HlpnE8ORWfCaqalqNLab82b9ICQErE5S5zgkv98uKLv2B2en9QuHzFA9fADny7ldQfhEc2qJVyS2P2dZ+wJqQgZnOpgJ0S9Eow7jOqN083qiVbpn50WKIJ4gMzpt1cffB9v/rqH/ruPBaJxzJ3z8+dIyCXoy0Lrx6pxHi9oINRzOJJtBqj56kGbW8h31lwuTi41Up81+puIAcUD/1xG3jDjnh6EHmZWdjUGL9tBb+lIvUcp1JpGYmN1nfo/SdL26x86brQfYu8EjJ13BsVkLjp6Z1Ba3CHrVriBwWppozXxhjIdUUd/mxL92JddUBYaJ17JDBvP62aaABof6vQubbp5zrL2s5/iZvHrI6dTDKf6RWq+UKnO25iWVzKqTZ0J6kWPItNZkSt7RC/IJaoHCr+CB+WzckNddY4KIn9ilOfdwGyr4g3yOGuwaX2knHLkjSYYiJeJgsUri1Zna3ZurWfk/oAqhHVBey0HQ52R3u7lsrf5Px191SGVzzcHOhL0lXCNH/nP5SHql0B7fo2Q85oYIgYeVwP1xSTu5RXEgBTrV8sHq6/OxHQ12P8aV01xiRQlh+v5RlFMvUmB1k0zl6/eu/mM7+EQNKkl4+KeT6rpO7J2YeZeD4pe215UHRfEtFa6wBJvDHfSntfPxSOJVp7oWRe0jBC7CJLY35KXsagwIofIG7AYxFnY3Qi+9fpKuAPf91s7b7Khz7J8iPVMPMqMEqrpXO11/Ue4e0Aw0KP+wZ7L9AL4XgrGPPhj5qIZpcZafYEJyivLjOpTWcUW9LZp36BM7443loY86bGwce6+1PW7UMGkSb+0CeYNzKdGO7JUPpm8X5n7xG4p2fdV6bUuHrI/Iyy+vI4vACNnjqmlg9EFQ=="
			  ],
			  "user-assertion.cloud.google.com": [
				"AK8TC8Jjdjp4VSyb8kHLQWhJImroENwF+9ioTbRIbowZHE25m6MLxAPLtLVH+M2Li9mtiBzRaAwRAaZdHCQSH3ZOqD/1vvh6eRmwHyGdm8jwfk56xLyeNLHjZht/CphA8RZW2dsxjMFiVbwlwMaPpS1qrMTa0Mqt5zuB0aW/fmsMEs/avtTMyFilPpVhWbKFKtCcfo5ZAOmHi91aTcoqoXl6tYvbzY9WjLcYUFqe0Ic="
			  ]
			}
		  },
		  "object": {
			"apiVersion": "compute.cnrm.cloud.google.com/v1beta1",
			"kind": "ComputeSubnetwork",
			"metadata": {
			  "annotations": {
				"cnrm.cloud.google.com/management-conflict-prevention-policy": "none",
				"cnrm.cloud.google.com/project-id": "default",
				"cnrm.cloud.google.com/state-into-spec": "merge",
				"kubectl.kubernetes.io/last-applied-configuration": "{\"apiVersion\":\"compute.cnrm.cloud.google.com/v1beta1\",\"kind\":\"ComputeSubnetwork\",\"metadata\":{\"annotations\":{},\"labels\":{\"label-one\":\"value-one\"},\"name\":\"computesubnetwork-sample2\",\"namespace\":\"default\"},\"spec\":{\"description\":\"My subnet2\",\"ipCidrRange\":\"10.2.0.0/16\",\"logConfig\":{\"aggregationInterval\":\"INTERVAL_10_MIN\",\"flowSampling\":0.5,\"metadata\":\"INCLUDE_ALL_METADATA\"},\"networkRef\":{\"name\":\"computesubnetwork-dep2\"},\"privateIpGoogleAccess\":false,\"region\":\"us-central1\"}}\n"
			  },
			  "creationTimestamp": "2021-12-24T08:36:45Z",
			  "generation": 1,
			  "labels": {
				"label-one": "value-one"
			  },
			  "managedFields": [
				{
				  "apiVersion": "compute.cnrm.cloud.google.com/v1beta1",
				  "fieldsType": "FieldsV1",
				  "fieldsV1": {
					"f:metadata": {
					  "f:annotations": {
						".": {},
						"f:kubectl.kubernetes.io/last-applied-configuration": {}
					  },
					  "f:labels": {
						".": {},
						"f:label-one": {}
					  }
					},
					"f:spec": {
					  ".": {},
					  "f:description": {},
					  "f:ipCidrRange": {},
					  "f:logConfig": {
						".": {},
						"f:aggregationInterval": {},
						"f:flowSampling": {},
						"f:metadata": {}
					  },
					  "f:networkRef": {
						".": {},
						"f:name": {}
					  },
					  "f:privateIpGoogleAccess": {},
					  "f:region": {}
					}
				  },
				  "manager": "kubectl-client-side-apply",
				  "operation": "Update",
				  "time": "2021-12-24T08:36:45Z"
				}
			  ],
			  "name": "computesubnetwork-sample2",
			  "namespace": "default",
			  "uid": "96527522-64fe-40f1-9c73-98d6311dfae3"
			},
			"spec": {
			  "description": "My subnet2",
			  "ipCidrRange": "10.2.0.0/16",
			  "logConfig": {
				"aggregationInterval": "INTERVAL_10_MIN",
				"flowSampling": 0.5,
				"metadata": "INCLUDE_ALL_METADATA"
			  },
			  "networkRef": {
				"name": "computesubnetwork-dep2"
			  },
			  "privateIpGoogleAccess": false,
			  "region": "us-central1"
			}
		  },
		  "oldObject": null,
		  "dryRun": false,
		  "options": {
			"kind": "CreateOptions",
			"apiVersion": "meta.k8s.io/v1",
			"fieldManager": "kubectl-client-side-apply"
		  }
		}
	  } 
	`)
	req := httptest.NewRequest("POST", "/admission/mutating", bytes.NewReader(data))
	req.Header["Content-Type"] = []string{"application/json"}
	resp, err := app.Test(req, -1)
	if err != nil {
		log.Fatal(err)
	}
	assert.Equalf(t, 200, resp.StatusCode, "Status Code")
	buf := new(strings.Builder)
	io.Copy(buf, resp.Body)
	respBody := buf.String()
	assert.Equalf(t, "{\"kind\":\"AdmissionReview\",\"apiVersion\":\"admission.k8s.io/v1\",\"response\":{\"uid\":\"9ec4fd53-d3cf-4e9b-af6a-25a398697e3c\",\"allowed\":true}}", respBody, "Body")
}

func TestMutatingWithDefaultRoutingDomain(t *testing.T) {
	InitMockDb()
	defer data_access.Close()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT routing_domain_id, name, vpcs FROM routing_domains LIMIT 1 FOR UPDATE").WillReturnRows(sqlmock.NewRows([]string{"routing_domain_id", "name", "vpcs"}).FromCSVString("1,DEFAULT,"))
	mock.ExpectQuery("SELECT subnet_id, parent_id, routing_domain_id, name, cidr FROM subnets WHERE subnet_id = (.+) FOR UPDATE").WithArgs(1).WillReturnRows(sqlmock.NewRows([]string{"subnet_id", "parent_id", "routing_domain_id", "name", "cidr"}).FromCSVString("1,-1,1,10.0.0.0/8,10.0.0.0/8"))
	mock.ExpectQuery("SELECT subnet_id, parent_id, routing_domain_id, name, cidr FROM subnets WHERE parent_id = (.+) FOR UPDATE").WithArgs(1).WillReturnRows(sqlmock.NewRows([]string{"subnet_id", "parent_id", "routing_domain_id", "name", "cidr"}))
	mock.ExpectExec("INSERT INTO subnets").WithArgs(1, 1, "computesubnetwork-sample2", "10.0.0.0/26").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	// Define Fiber app.
	app := fiber.New()

	// Create route with GET method for test
	app.Post("/admission/mutating", MutatingWebhook)

	data := []byte(`{
		"kind": "AdmissionReview",
		"apiVersion": "admission.k8s.io/v1",
		"request": {
		  "uid": "9ec4fd53-d3cf-4e9b-af6a-25a398697e3c",
		  "kind": {
			"group": "compute.cnrm.cloud.google.com",
			"version": "v1beta1",
			"kind": "ComputeSubnetwork"
		  },
		  "resource": {
			"group": "compute.cnrm.cloud.google.com",
			"version": "v1beta1",
			"resource": "computesubnetworks"
		  },
		  "requestKind": {
			"group": "compute.cnrm.cloud.google.com",
			"version": "v1beta1",
			"kind": "ComputeSubnetwork"
		  },
		  "requestResource": {
			"group": "compute.cnrm.cloud.google.com",
			"version": "v1beta1",
			"resource": "computesubnetworks"
		  },
		  "name": "computesubnetwork-sample2",
		  "namespace": "default",
		  "operation": "CREATE",
		  "userInfo": {
			"username": "grotz@grotz.joonix.net",
			"groups": [
			  "system:authenticated"
			],
			"extra": {
			  "iam.gke.io/user-assertion": [
				"AK8TC8Lh3gRR8JYyhlGL4hyHQAe90AEOafQmNdeFYfw7j1hofgD5VZUQmQdKtUVWz9z99G/KAG1B82rtCx20TX1CWDVTL9bjVKWwvH44rlztiQf8x1RuJdyusNdbetCHG1kK2k9UWTkvTkyrK8WL4RXHItGUlvVJtv1jCQsUJVMM3PUx7fT45SHB+OZlWM0IZ6vzFLWjDxem0Hh/skXjKRm2ao2ELqlkTDWA+IDPzvy8m2qGW53u7C6bYdmzw3pY9FR7ze9YMbAuc0HlpnE8ORWfCaqalqNLab82b9ICQErE5S5zgkv98uKLv2B2en9QuHzFA9fADny7ldQfhEc2qJVyS2P2dZ+wJqQgZnOpgJ0S9Eow7jOqN083qiVbpn50WKIJ4gMzpt1cffB9v/rqH/ruPBaJxzJ3z8+dIyCXoy0Lrx6pxHi9oINRzOJJtBqj56kGbW8h31lwuTi41Up81+puIAcUD/1xG3jDjnh6EHmZWdjUGL9tBb+lIvUcp1JpGYmN1nfo/SdL26x86brQfYu8EjJ13BsVkLjp6Z1Ba3CHrVriBwWppozXxhjIdUUd/mxL92JddUBYaJ17JDBvP62aaABof6vQubbp5zrL2s5/iZvHrI6dTDKf6RWq+UKnO25iWVzKqTZ0J6kWPItNZkSt7RC/IJaoHCr+CB+WzckNddY4KIn9ilOfdwGyr4g3yOGuwaX2knHLkjSYYiJeJgsUri1Zna3ZurWfk/oAqhHVBey0HQ52R3u7lsrf5Px191SGVzzcHOhL0lXCNH/nP5SHql0B7fo2Q85oYIgYeVwP1xSTu5RXEgBTrV8sHq6/OxHQ12P8aV01xiRQlh+v5RlFMvUmB1k0zl6/eu/mM7+EQNKkl4+KeT6rpO7J2YeZeD4pe215UHRfEtFa6wBJvDHfSntfPxSOJVp7oWRe0jBC7CJLY35KXsagwIofIG7AYxFnY3Qi+9fpKuAPf91s7b7Khz7J8iPVMPMqMEqrpXO11/Ue4e0Aw0KP+wZ7L9AL4XgrGPPhj5qIZpcZafYEJyivLjOpTWcUW9LZp36BM7443loY86bGwce6+1PW7UMGkSb+0CeYNzKdGO7JUPpm8X5n7xG4p2fdV6bUuHrI/Iyy+vI4vACNnjqmlg9EFQ=="
			  ],
			  "user-assertion.cloud.google.com": [
				"AK8TC8Jjdjp4VSyb8kHLQWhJImroENwF+9ioTbRIbowZHE25m6MLxAPLtLVH+M2Li9mtiBzRaAwRAaZdHCQSH3ZOqD/1vvh6eRmwHyGdm8jwfk56xLyeNLHjZht/CphA8RZW2dsxjMFiVbwlwMaPpS1qrMTa0Mqt5zuB0aW/fmsMEs/avtTMyFilPpVhWbKFKtCcfo5ZAOmHi91aTcoqoXl6tYvbzY9WjLcYUFqe0Ic="
			  ]
			}
		  },
		  "object": {
			"apiVersion": "compute.cnrm.cloud.google.com/v1beta1",
			"kind": "ComputeSubnetwork",
			"metadata": {
			  "annotations": {
				"cnrm.cloud.google.com/management-conflict-prevention-policy": "none",
				"cnrm.cloud.google.com/project-id": "default",
				"cnrm.cloud.google.com/state-into-spec": "merge",
				"kubectl.kubernetes.io/last-applied-configuration": "{\"apiVersion\":\"compute.cnrm.cloud.google.com/v1beta1\",\"kind\":\"ComputeSubnetwork\",\"metadata\":{\"annotations\":{},\"labels\":{\"label-one\":\"value-one\"},\"name\":\"computesubnetwork-sample2\",\"namespace\":\"default\"},\"spec\":{\"description\":\"My subnet2\",\"ipCidrRange\":\"10.2.0.0/16\",\"logConfig\":{\"aggregationInterval\":\"INTERVAL_10_MIN\",\"flowSampling\":0.5,\"metadata\":\"INCLUDE_ALL_METADATA\"},\"networkRef\":{\"name\":\"computesubnetwork-dep2\"},\"privateIpGoogleAccess\":false,\"region\":\"us-central1\"}}\n",
			    "ipam.cloud.google.com/size": "26",
			    "ipam.cloud.google.com/parent": "1"
			  },
			  "creationTimestamp": "2021-12-24T08:36:45Z",
			  "generation": 1,
			  "labels": {
				"label-one": "value-one"
			  },
			  "managedFields": [
				{
				  "apiVersion": "compute.cnrm.cloud.google.com/v1beta1",
				  "fieldsType": "FieldsV1",
				  "fieldsV1": {
					"f:metadata": {
					  "f:annotations": {
						".": {},
						"f:kubectl.kubernetes.io/last-applied-configuration": {}
					  },
					  "f:labels": {
						".": {},
						"f:label-one": {}
					  }
					},
					"f:spec": {
					  ".": {},
					  "f:description": {},
					  "f:ipCidrRange": {},
					  "f:logConfig": {
						".": {},
						"f:aggregationInterval": {},
						"f:flowSampling": {},
						"f:metadata": {}
					  },
					  "f:networkRef": {
						".": {},
						"f:name": {}
					  },
					  "f:privateIpGoogleAccess": {},
					  "f:region": {}
					}
				  },
				  "manager": "kubectl-client-side-apply",
				  "operation": "Update",
				  "time": "2021-12-24T08:36:45Z"
				}
			  ],
			  "name": "computesubnetwork-sample2",
			  "namespace": "default",
			  "uid": "96527522-64fe-40f1-9c73-98d6311dfae3"
			},
			"spec": {
			  "description": "My subnet2",
			  "logConfig": {
				"aggregationInterval": "INTERVAL_10_MIN",
				"flowSampling": 0.5,
				"metadata": "INCLUDE_ALL_METADATA"
			  },
			  "networkRef": {
				"name": "computesubnetwork-dep2"
			  },
			  "privateIpGoogleAccess": false,
			  "region": "us-central1"
			}
		  },
		  "oldObject": null,
		  "dryRun": false,
		  "options": {
			"kind": "CreateOptions",
			"apiVersion": "meta.k8s.io/v1",
			"fieldManager": "kubectl-client-side-apply"
		  }
		}
	  } 
	`)
	req := httptest.NewRequest("POST", "/admission/mutating", bytes.NewReader(data))
	req.Header["Content-Type"] = []string{"application/json"}
	resp, err := app.Test(req, -1)
	if err != nil {
		log.Fatal(err)
	}
	assert.Equalf(t, 200, resp.StatusCode, "Status Code")
	buf := new(strings.Builder)
	io.Copy(buf, resp.Body)
	respBody := buf.String()
	assert.Equalf(t, "{\"kind\":\"AdmissionReview\",\"apiVersion\":\"admission.k8s.io/v1\",\"response\":{\"uid\":\"9ec4fd53-d3cf-4e9b-af6a-25a398697e3c\",\"allowed\":true,\"patch\":\"W3sib3AiOiJhZGQiLCJwYXRoIjoiL21ldGFkYXRhL2Fubm90YXRpb25zL2lwYW0uY2xvdWQuZ29vZ2xlLmNvbS9yYW5nZS1pZCIsInZhbHVlIjoiMSJ9LHsib3AiOiJyZXBsYWNlIiwicGF0aCI6Ii9zcGVjL2lwQ2lkclJhbmdlIiwidmFsdWUiOiIxMC4wLjAuMC8yNiJ9XQ==\",\"patchType\":\"JSONPatch\"}}", respBody, "Body")
}

func TestMutatingGivenCidrRange(t *testing.T) {
	InitMockDb()
	defer data_access.Close()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT routing_domain_id, name, vpcs FROM routing_domains LIMIT 1 FOR UPDATE").WillReturnRows(sqlmock.NewRows([]string{"routing_domain_id", "name", "vpcs"}).FromCSVString("1,DEFAULT,"))
	mock.ExpectQuery("SELECT subnet_id, parent_id, routing_domain_id, name, cidr FROM subnets WHERE subnet_id = (.+) FOR UPDATE").WithArgs(1).WillReturnRows(sqlmock.NewRows([]string{"subnet_id", "parent_id", "routing_domain_id", "name", "cidr"}).FromCSVString("1,-1,1,10.0.0.0/8,10.0.0.0/8"))
	mock.ExpectQuery("SELECT subnet_id, parent_id, routing_domain_id, name, cidr FROM subnets WHERE parent_id = (.+) FOR UPDATE").WithArgs(1).WillReturnRows(sqlmock.NewRows([]string{"subnet_id", "parent_id", "routing_domain_id", "name", "cidr"}))
	mock.ExpectExec("INSERT INTO subnets").WithArgs(1, 1, "My subnet2", "10.0.0.0/26").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	// Define Fiber app.
	app := fiber.New()

	// Create route with GET method for test
	app.Post("/admission/mutating", MutatingWebhook)

	data := []byte(`{
		"kind": "AdmissionReview",
		"apiVersion": "admission.k8s.io/v1",
		"request": {
			"uid": "2e60c831-f9a3-4e57-b748-9a510452e48a",
			"kind": {
				"group": "compute.cnrm.cloud.google.com",
				"version": "v1beta1",
				"kind": "ComputeSubnetwork"
			},
			"resource": {
				"group": "compute.cnrm.cloud.google.com",
				"version": "v1beta1",
				"resource": "computesubnetworks"
			},
			"requestKind": {
				"group": "compute.cnrm.cloud.google.com",
				"version": "v1beta1",
				"kind": "ComputeSubnetwork"
			},
			"requestResource": {
				"group": "compute.cnrm.cloud.google.com",
				"version": "v1beta1",
				"resource": "computesubnetworks"
			},
			"name": "computesubnetwork-sample2",
			"namespace": "default",
			"operation": "CREATE",
			"userInfo": {
				"username": "grotz@grotz.joonix.net",
				"groups": ["system:authenticated"],
				"extra": {
					"iam.gke.io/user-assertion": ["AK8TC8KNQKa1n5v8KUn6tOBsS6pYbMwWBdz5RCpK9YdtAL1gzhD0rC/zIKqevi5QVV00fZ+9ceEAw9ZZKFAcrhj4+zyD+FntB+zJeXsPPjNcTPir+W5BadEdYwp6MchK9C8oZwGqYnEnJTzK+TYEYo0+7qTEe7y4mH/ohZxCVd7MYWhqGuzP7fCsO/3YkelfkIUC7C+QxN67GvkNbE48wFRphSoUpsni9wv9rzZ7C1u0oyYCFJ9Axru7dkhwPRuY7ZQ6fNnHE8Yl07j/GU5YH8JKalRBuHgctB4l1b8C4KXHhTGjT3lEdtwdErMLoHnODO99VfeN19sYvQWDiyBDy1hHcYWm+a1Yzau23Kw1aUs7QQnHlcxwTa+rEp9aSDzy046vlzdPTU5IjbCac4gWPGbmOjC8TDKWSANrF3U/vhY2+0hWZMl375eN1Xwb6FSJ8iwevd/IimhdRuhwfiEdZJL9QWN0BihDTcoDzGQKKcwfSLwHAtDrckwZmOdNjyQopM4Hj/4xe5CKl/MN2UHWb3CcTEEuAszv8NtMEhIJFwuEc/Km6tPeVwXELTWBPdNEUxm12Q5hTBGrlpmhgyscxTFAehwwYGoEivqnoeXuvFk0I+YNwmzAn8yzh7s8B4pg7lIDctfX5qygSMrkCQ2hQMN9DoEa5/s9nir+r6nRARXip2IBohNBfeDn9l6D6CjYL/gxsWk4rj+QBDv8Vz03dcHOdTZ4lFR6vrdcHwXR5SUU4Fo8s8GVk1Gf5uIRI6g7bsw8kevXEacBH778eHzksWpejEj140D6SbbZhFtSaFEEtb4oU5IWy69ws+xe8rb57yZbhh/+StghtoWYxZUz2QOb/X+E/noUige6qGBMGVmxgLZNdcMqTR+7Rm8N5xwuRNkUKMX6HnelkWcTHLTmgfZYi/zjbycZGi3ACSpOPdguWFrnjcUgZx5omfuvf+VRVYR86+0A8FOORDtkhI8WvevqY6eg29XrG0cdzi37F52a2UjtaqvOGQpEEh6pIW8M/23eLEJBtbwH2Z5qQ22akZE5dnjDeTBzut5rHUcxHv7qXNsP9adcMwMlDCeT8T8vlU/sVKqiakSNIzdyKtwbSPY5fq6UNiPgawFEqptqeI/8L+P9cjGAlBxEnN10kD4="],
					"user-assertion.cloud.google.com": ["AK8TC8JgXlAzQyz4f3fuEeFm5gw/62CeQ1Fswy2FYxeAkHoSdXC2Eg5ItYFQxm4cJbYGTbzAD8+67gXhnRxuy9dzYKNiy12jCe1aiXPn6e9oAUyQnNe2Y3TOVY139pnODw/uLOj61DhmWBsIlSq4KTR9Cm3VYa5Q8EqvkdtwEO/nsHDf/9J0hZPEu6lRIzMWbPYG1gdcib40k21lV5O+iRevJgLevNkKQ1F2xgELS00="]
				}
			},
			"object": {
				"apiVersion": "compute.cnrm.cloud.google.com/v1beta1",
				"kind": "ComputeSubnetwork",
				"metadata": {
					"annotations": {
						"cnrm.cloud.google.com/management-conflict-prevention-policy": "none",
						"cnrm.cloud.google.com/project-id": "default",
						"cnrm.cloud.google.com/state-into-spec": "merge",
						"kubectl.kubernetes.io/last-applied-configuration": "{\"apiVersion\":\"compute.cnrm.cloud.google.com/v1beta1\",\"kind\":\"ComputeSubnetwork\",\"metadata\":{\"annotations\":{},\"labels\":{\"label-one\":\"value-one\"},\"name\":\"computesubnetwork-sample2\",\"namespace\":\"default\"},\"spec\":{\"description\":\"My subnet2\",\"ipCidrRange\":\"10.2.0.0/16\",\"logConfig\":{\"aggregationInterval\":\"INTERVAL_10_MIN\",\"flowSampling\":0.5,\"metadata\":\"INCLUDE_ALL_METADATA\"},\"networkRef\":{\"name\":\"computesubnetwork-dep2\"},\"privateIpGoogleAccess\":false,\"region\":\"us-central1\"}}\n"
					},
					"creationTimestamp": null,
					"labels": {
						"label-one": "value-one"
					},
					"managedFields": [{
						"apiVersion": "compute.cnrm.cloud.google.com/v1beta1",
						"fieldsType": "FieldsV1",
						"fieldsV1": {
							"f:metadata": {
								"f:annotations": {
									".": {},
									"f:kubectl.kubernetes.io/last-applied-configuration": {}
								},
								"f:labels": {
									".": {},
									"f:label-one": {}
								}
							},
							"f:spec": {
								".": {},
								"f:description": {},
								"f:ipCidrRange": {},
								"f:logConfig": {
									".": {},
									"f:aggregationInterval": {},
									"f:flowSampling": {},
									"f:metadata": {}
								},
								"f:networkRef": {
									".": {},
									"f:name": {}
								},
								"f:privateIpGoogleAccess": {},
								"f:region": {}
							}
						},
						"manager": "kubectl-client-side-apply",
						"operation": "Update",
						"time": "2021-12-25T22:13:52Z"
					}],
					"name": "computesubnetwork-sample2",
					"namespace": "default"
				},
				"spec": {
					"description": "My subnet2",
					"ipCidrRange": "10.2.0.0/16",
					"logConfig": {
						"aggregationInterval": "INTERVAL_10_MIN",
						"flowSampling": 0.5,
						"metadata": "INCLUDE_ALL_METADATA"
					},
					"networkRef": {
						"name": "computesubnetwork-dep2"
					},
					"privateIpGoogleAccess": false,
					"region": "us-central1"
				}
			},
			"oldObject": null,
			"dryRun": false,
			"options": {
				"kind": "CreateOptions",
				"apiVersion": "meta.k8s.io/v1",
				"fieldManager": "kubectl-client-side-apply"
			}
		}
	}
	`)
	req := httptest.NewRequest("POST", "/admission/mutating", bytes.NewReader(data))
	req.Header["Content-Type"] = []string{"application/json"}
	resp, err := app.Test(req, -1)
	if err != nil {
		log.Fatal(err)
	}
	assert.Equalf(t, 200, resp.StatusCode, "Status Code")
	buf := new(strings.Builder)
	io.Copy(buf, resp.Body)
	respBody := buf.String()
	assert.Equalf(t, "{\"kind\":\"AdmissionReview\",\"apiVersion\":\"admission.k8s.io/v1\",\"response\":{\"uid\":\"2e60c831-f9a3-4e57-b748-9a510452e48a\",\"allowed\":true}}", respBody, "Body")
}

func TestDeletion(t *testing.T) {
	InitMockDb()
	defer data_access.Close()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT routing_domain_id, name, vpcs FROM routing_domains LIMIT 1 FOR UPDATE").WillReturnRows(sqlmock.NewRows([]string{"routing_domain_id", "name", "vpcs"}).FromCSVString("1,DEFAULT,"))
	mock.ExpectQuery("SELECT subnet_id, parent_id, routing_domain_id, name, cidr FROM subnets WHERE subnet_id = (.+) FOR UPDATE").WithArgs(1).WillReturnRows(sqlmock.NewRows([]string{"subnet_id", "parent_id", "routing_domain_id", "name", "cidr"}).FromCSVString("1,-1,1,10.0.0.0/8,10.0.0.0/8"))
	mock.ExpectQuery("SELECT subnet_id, parent_id, routing_domain_id, name, cidr FROM subnets WHERE parent_id = (.+) FOR UPDATE").WithArgs(1).WillReturnRows(sqlmock.NewRows([]string{"subnet_id", "parent_id", "routing_domain_id", "name", "cidr"}))
	mock.ExpectExec("INSERT INTO subnets").WithArgs(1, 1, "My subnet2", "10.0.0.0/26").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	// Define Fiber app.
	app := fiber.New()

	// Create route with GET method for test
	app.Post("/admission/mutating", MutatingWebhook)

	data := []byte(`{
		"kind": "AdmissionReview",
		"apiVersion": "admission.k8s.io/v1",
		"request": {
			"uid": "d465b0b4-e28a-4ac0-a3cf-acccf36c4fb3",
			"kind": {
				"group": "compute.cnrm.cloud.google.com",
				"version": "v1beta1",
				"kind": "ComputeSubnetwork"
			},
			"resource": {
				"group": "compute.cnrm.cloud.google.com",
				"version": "v1beta1",
				"resource": "computesubnetworks"
			},
			"requestKind": {
				"group": "compute.cnrm.cloud.google.com",
				"version": "v1beta1",
				"kind": "ComputeSubnetwork"
			},
			"requestResource": {
				"group": "compute.cnrm.cloud.google.com",
				"version": "v1beta1",
				"resource": "computesubnetworks"
			},
			"name": "computesubnetwork-sample2",
			"namespace": "default",
			"operation": "DELETE",
			"userInfo": {
				"username": "grotz@grotz.joonix.net",
				"groups": ["system:authenticated"],
				"extra": {
					"iam.gke.io/user-assertion": ["AK8TC8LHfURbLp+9dTK47i34YJsBRnguWkTNQQ1bYgmMkFe1+NKE52MmqdTAhPst4TF15OCGy+Km0XfvxTpffZ+CPYbA4KgRXzBu1yg9Gis4mQSBb1suKXbxNjrvIRr2Rwr4ZOYPXk7Ig62fdWhYvagVoPQunOueFOkWfpRGphQpGnB10H1U8p2mOY/aQxS/RZw66SQ8lipNBODcSKGnkeqcRMKr3aDKc/PqW6IF9shSyh7lFAs8rLFyGsGq9oAfA68lg2gG3YzBDb6PJsb99knP9j9DrKUBRqz2h4vupTLPjq9N9ILd4Imb102I+bxQLRquIgOTaBApmsdLz9TPgxP6vE8OnNKZcmthyOWp68lNZ0GnbiTMnJwbI+0OGO2m6lscijAynKDZYBbkfVHXzPFY0YrmSfLHrCjyDnEf4a+1NxzvWDSw2AgsTZ9eZ9FHJy2JIVreCGA1HtICqADzed5VsyaOldoOr7TzDEztu9pTNfaDUnNgp9pNAe8wyCDwq7SPuqfKq5+EIwytz+TDA6dcuAVGIHv3XZv78A1OtRaCoaHhjXuDE6Q8ljH2Vdi8ItP6LmZDvujq6dVXosRaDRsDgT+Ug6KpB5W+qvVp0XqX3RH+YLJ/c0r9/qKKa9k9SO4X+dIvoO8dWJO7UHAUPF6/k0LSQxOFc6DScNDiHd71zh2VHx6R+Vxepn1T0S5d3n2xpvx3+WbD/x8/1a8YqFeSbcY3HMxErNClG88J8dMAa5ps4DOmnKZS/nuCXC5/js8zV7NBdi3LEj4LECIYnYZCITFwQOHQTmvtNhTosv/xcLvelTAqc6+ZO1UA1OVQs9bcPShZNTY7lUtwYsjlGa3wIXgyU0R/T5LUuHxdU9J6gjWQRC5JCUQv7InX1GT4YaKO9Vh9WDMKppmuW+nvnkrPUlr/IbhQlKl7hItZ9okB8IEv+9y2gmmKRj+/JD4wOhf1siDcEBt2hWd06YCQnG0aVLMX4RIVGtWWKCmUGMZNHiVf9kL6n3hI4Hq2Z3iTSp9QNUarLptN4abcKLTF9ccqHVyWa5REhsQcGTXNtIjgVae/lBaoNr5S1F5AkSg4kzA5pgs2DIB/0PoT/E1wqybJBIAASBUOf/kn052R7xmj9/6OUlcaUK+CAZ/2kw=="],
					"user-assertion.cloud.google.com": ["AK8TC8JDr97iF+Ur1HNL9n7zqnHxNo2yEofAjejlrwYVp69YMPanB979JUw6rlqdrMSQ+6qglyQ++GQ9Gzb4FCw+RkqxvoWLFjF8p07K9+ex00fgj95v1WMFBy3oOvGnwSgfmKOjSY7uA8+pN7JzFFbRTAzHUAiY+q4X2XCzmKHX/AMG/sXsu/9huGR4hGx+URt5g6gkqR/mQImmsZpsxhqYARM+vZvXOLzzrRB5y8Q="]
				}
			},
			"object": null,
			"oldObject": {
				"apiVersion": "compute.cnrm.cloud.google.com/v1beta1",
				"kind": "ComputeSubnetwork",
				"metadata": {
					"annotations": {
						"cnrm.cloud.google.com/management-conflict-prevention-policy": "none",
						"cnrm.cloud.google.com/project-id": "default",
						"cnrm.cloud.google.com/state-into-spec": "merge",
						"kubectl.kubernetes.io/last-applied-configuration": "{\"apiVersion\":\"compute.cnrm.cloud.google.com/v1beta1\",\"kind\":\"ComputeSubnetwork\",\"metadata\":{\"annotations\":{},\"labels\":{\"label-one\":\"value-one\"},\"name\":\"computesubnetwork-sample2\",\"namespace\":\"default\"},\"spec\":{\"description\":\"My subnet2\",\"ipCidrRange\":\"10.2.0.0/16\",\"logConfig\":{\"aggregationInterval\":\"INTERVAL_10_MIN\",\"flowSampling\":0.5,\"metadata\":\"INCLUDE_ALL_METADATA\"},\"networkRef\":{\"name\":\"computesubnetwork-dep2\"},\"privateIpGoogleAccess\":false,\"region\":\"us-central1\"}}\n"
					},
					"creationTimestamp": "2021-12-25T22:35:42Z",
					"generation": 1,
					"labels": {
						"label-one": "value-one"
					},
					"managedFields": [{
						"apiVersion": "compute.cnrm.cloud.google.com/v1beta1",
						"fieldsType": "FieldsV1",
						"fieldsV1": {
							"f:metadata": {
								"f:annotations": {
									".": {},
									"f:kubectl.kubernetes.io/last-applied-configuration": {}
								},
								"f:labels": {
									".": {},
									"f:label-one": {}
								}
							},
							"f:spec": {
								".": {},
								"f:description": {},
								"f:ipCidrRange": {},
								"f:logConfig": {
									".": {},
									"f:aggregationInterval": {},
									"f:flowSampling": {},
									"f:metadata": {}
								},
								"f:networkRef": {
									".": {},
									"f:name": {}
								},
								"f:privateIpGoogleAccess": {},
								"f:region": {}
							}
						},
						"manager": "kubectl-client-side-apply",
						"operation": "Update",
						"time": "2021-12-25T22:35:41Z"
					}],
					"name": "computesubnetwork-sample2",
					"namespace": "default",
					"resourceVersion": "878208",
					"uid": "9e5e2519-0483-4a57-a084-354fcdd0d3c7"
				},
				"spec": {
					"description": "My subnet2",
					"ipCidrRange": "10.2.0.0/16",
					"logConfig": {
						"aggregationInterval": "INTERVAL_10_MIN",
						"flowSampling": 0.5,
						"metadata": "INCLUDE_ALL_METADATA"
					},
					"networkRef": {
						"name": "computesubnetwork-dep2"
					},
					"privateIpGoogleAccess": false,
					"region": "us-central1"
				}
			},
			"dryRun": false,
			"options": {
				"kind": "DeleteOptions",
				"apiVersion": "meta.k8s.io/v1",
				"propagationPolicy": "Background"
			}
		}
	}
	`)
	req := httptest.NewRequest("POST", "/admission/mutating", bytes.NewReader(data))
	req.Header["Content-Type"] = []string{"application/json"}
	resp, err := app.Test(req, -1)
	if err != nil {
		log.Fatal(err)
	}
	assert.Equalf(t, 200, resp.StatusCode, "Status Code")
	buf := new(strings.Builder)
	io.Copy(buf, resp.Body)
	respBody := buf.String()
	assert.Equalf(t, "{\"kind\":\"AdmissionReview\",\"apiVersion\":\"admission.k8s.io/v1\",\"response\":{\"uid\":\"d465b0b4-e28a-4ac0-a3cf-acccf36c4fb3\",\"allowed\":true}}", respBody, "Body")
}

func TestMutatingWithoutParentId(t *testing.T) {
	InitMockDb()
	defer data_access.Close()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT routing_domain_id, name, vpcs FROM routing_domains LIMIT 1 FOR UPDATE").WillReturnRows(sqlmock.NewRows([]string{"routing_domain_id", "name", "vpcs"}).FromCSVString("1,DEFAULT,"))
	mock.ExpectCommit()

	// Define Fiber app.
	app := fiber.New()

	// Create route with GET method for test
	app.Post("/admission/mutating", MutatingWebhook)

	data := []byte(`{
		"kind": "AdmissionReview",
		"apiVersion": "admission.k8s.io/v1",
		"request": {
			"uid": "66306aed-20e2-4b59-9903-5aa3e13ca7a0",
			"kind": {
				"group": "compute.cnrm.cloud.google.com",
				"version": "v1beta1",
				"kind": "ComputeSubnetwork"
			},
			"resource": {
				"group": "compute.cnrm.cloud.google.com",
				"version": "v1beta1",
				"resource": "computesubnetworks"
			},
			"requestKind": {
				"group": "compute.cnrm.cloud.google.com",
				"version": "v1beta1",
				"kind": "ComputeSubnetwork"
			},
			"requestResource": {
				"group": "compute.cnrm.cloud.google.com",
				"version": "v1beta1",
				"resource": "computesubnetworks"
			},
			"name": "computesubnetwork-sample3",
			"namespace": "default",
			"operation": "CREATE",
			"userInfo": {
				"username": "grotz@grotz.joonix.net",
				"groups": ["system:authenticated"],
				"extra": {
					"iam.gke.io/user-assertion": ["AK8TC8KxuovtcaGeM8nhIFsr+uTEHinqQqHYKONDsNRRzhoLMJzlN84HW4X1v++FDLuSotaIIdi0rKDT7eTWdC5C98RFnn6nfbRkn3FS7UBDeyfu5gs8bfLBSpU2FNYhDhlwSdzeBis9Q9/QHi4nIjGBnNlcoE/VidTiw53IKw7CjrGoyB4RJgTw6NILulTug0aNvWN+EkcS5Io+dCMQD9MkI+xpP4MteuxaElbfdWqyKjWnsGCGioEQ+knbhbR1+m2YcoVIsorb3krH5If+pspQzdHsRjw4xxTfXwJlAlQqneXXmaPkQeToIpZOxatYkn9dVQ1Qy/BV3I5Yzqu8YoVFVOM0fU/j2RrtnWOn0FRR8sMJ9frVNj+3bPxleHLJtZt0bUytWBSPt1ARHqpaISx7s7DXXsKtW4A+oUcRsmZqi6Eb6mcT7xVvKzkOIBIPjQ622+bJw6qhJos3WeRmbUUotY2kGhWiS0fNHhD1ZKTtWtfDxC0AT7A+XTou23D3BIBfcxrDeo4lceV+8SDxcC856DN0DmLkKMrJpBbpaZ6tqsUN98xu3m9biGH838k8K+yyMfkLcKh0+VsrlJnhXHxdPmRx3Y1arnhrr64h9QjIOcdXW/GEaTF1iJi0K2dCpWVFjDVHhFaZ4WSx8Kf0GsMsIu0g/mUs1YYnPq1pErq17kbMvoSz/sYjnM92zRlhHALyXEKKdlfB04HhtdkT7+mDozQOkrRYks9EFJYVQPFkqC6PTPpSWYYpMoxd2bK6/X15AAwi7X+8hsAz00Gqgm6CdtjqTTJbndDSU4A4KYixFxik20U0Zcg7TRMZ+sNY7U+crJaRbB7lBDYjzbvWPhFv3BsVmtytuuJfdQqTIXIKi6qz4zH/siiiJlR80QHmEdLCquzytbiufh0Bl9seu14ZvM3NGLcZZUBVw13XNPh396QkR1hxIrOX/hnwYumnBfP2DnYAW15fk4c5S9zow6Pkc4xXRhiOibWL9aidurhuThKJeoFg7gH9LvcoW0/qSK/SqXSxZul8HrYwYBnDZisVxIpGqmVVU4rVHbl7PZsoS4ep8+91MeeOp9hrQCPvU1rfN1ZqNVf/M58lYqw75cqrffh1D5gInvNW8V6P5ChHib/PGqH/WqNAFuv48Ac="],
					"user-assertion.cloud.google.com": ["AK8TC8JxXIuFydhuAaKNQIH7eGOnYspAvqeOaUJk9yL4NY9++BqDM/iORtInaYGC+/ZIz3lW0Ik8OmloT7TB/KtR86jTHu6f9aCE4/XsqY0FaGDuK/3OK7shm1K0U0LNutCnQnZzxX18wFFhs/bDDqcrWp9mLI+k51oNjYd6EXaJd1TG0hb2X1A24toiIrsuQqcj3yt6BoC/yi/b45g5lF4N3K25HavmBqKmp6iqlgo="]
				}
			},
			"object": {
				"apiVersion": "compute.cnrm.cloud.google.com/v1beta1",
				"kind": "ComputeSubnetwork",
				"metadata": {
					"annotations": {
						"cnrm.cloud.google.com/management-conflict-prevention-policy": "none",
						"cnrm.cloud.google.com/project-id": "default",
						"cnrm.cloud.google.com/state-into-spec": "merge",
						"ipam.cloud.google.com/size": "26",
						"kubectl.kubernetes.io/last-applied-configuration": "{\"apiVersion\":\"compute.cnrm.cloud.google.com/v1beta1\",\"kind\":\"ComputeSubnetwork\",\"metadata\":{\"annotations\":{\"ipam.cloud.google.com/size\":\"26\"},\"labels\":{\"label-one\":\"value-one\"},\"name\":\"computesubnetwork-sample3\",\"namespace\":\"default\"},\"spec\":{\"description\":\"My subnet3\",\"logConfig\":{\"aggregationInterval\":\"INTERVAL_10_MIN\",\"flowSampling\":0.5,\"metadata\":\"INCLUDE_ALL_METADATA\"},\"networkRef\":{\"name\":\"computesubnetwork-dep2\"},\"privateIpGoogleAccess\":false,\"region\":\"us-central1\"}}\n"
					},
					"creationTimestamp": null,
					"labels": {
						"label-one": "value-one"
					},
					"managedFields": [{
						"apiVersion": "compute.cnrm.cloud.google.com/v1beta1",
						"fieldsType": "FieldsV1",
						"fieldsV1": {
							"f:metadata": {
								"f:annotations": {
									".": {},
									"f:ipam.cloud.google.com/size": {},
									"f:kubectl.kubernetes.io/last-applied-configuration": {}
								},
								"f:labels": {
									".": {},
									"f:label-one": {}
								}
							},
							"f:spec": {
								".": {},
								"f:description": {},
								"f:logConfig": {
									".": {},
									"f:aggregationInterval": {},
									"f:flowSampling": {},
									"f:metadata": {}
								},
								"f:networkRef": {
									".": {},
									"f:name": {}
								},
								"f:privateIpGoogleAccess": {},
								"f:region": {}
							}
						},
						"manager": "kubectl-client-side-apply",
						"operation": "Update",
						"time": "2021-12-25T22:56:59Z"
					}],
					"name": "computesubnetwork-sample3",
					"namespace": "default"
				},
				"spec": {
					"description": "My subnet3",
					"logConfig": {
						"aggregationInterval": "INTERVAL_10_MIN",
						"flowSampling": 0.5,
						"metadata": "INCLUDE_ALL_METADATA"
					},
					"networkRef": {
						"name": "computesubnetwork-dep2"
					},
					"privateIpGoogleAccess": false,
					"region": "us-central1"
				}
			},
			"oldObject": null,
			"dryRun": false,
			"options": {
				"kind": "CreateOptions",
				"apiVersion": "meta.k8s.io/v1",
				"fieldManager": "kubectl-client-side-apply"
			}
		}
	}
	`)
	req := httptest.NewRequest("POST", "/admission/mutating", bytes.NewReader(data))
	req.Header["Content-Type"] = []string{"application/json"}
	resp, err := app.Test(req, -1)
	if err != nil {
		log.Fatal(err)
	}
	assert.Equalf(t, 200, resp.StatusCode, "Status Code")
	buf := new(strings.Builder)
	io.Copy(buf, resp.Body)
	respBody := buf.String()
	assert.Equalf(t, "{\"kind\":\"AdmissionReview\",\"apiVersion\":\"admission.k8s.io/v1\",\"response\":{\"uid\":\"66306aed-20e2-4b59-9903-5aa3e13ca7a0\",\"allowed\":false,\"status\":{\"metadata\":{},\"status\":\"Failure\",\"message\":\"Please provide the ID of a parent range\"}}}", respBody, "Body")
}

func TestMutatingCreation(t *testing.T) {
	InitMockDb()
	defer data_access.Close()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT routing_domain_id, name, vpcs FROM routing_domains LIMIT 1 FOR UPDATE").WillReturnRows(sqlmock.NewRows([]string{"routing_domain_id", "name", "vpcs"}).FromCSVString("1,DEFAULT,"))
	mock.ExpectQuery("SELECT subnet_id, parent_id, routing_domain_id, name, cidr FROM subnets WHERE subnet_id = (.+) FOR UPDATE").WithArgs(1).WillReturnRows(sqlmock.NewRows([]string{"subnet_id", "parent_id", "routing_domain_id", "name", "cidr"}).FromCSVString("1,-1,1,10.0.0.0/8,10.0.0.0/8"))
	mock.ExpectQuery("SELECT subnet_id, parent_id, routing_domain_id, name, cidr FROM subnets WHERE parent_id = (.+) FOR UPDATE").WithArgs(1).WillReturnRows(sqlmock.NewRows([]string{"subnet_id", "parent_id", "routing_domain_id", "name", "cidr"}))
	mock.ExpectExec("INSERT INTO subnets").WithArgs(1, 1, "computesubnetwork-sample3", "10.0.0.0/26").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	// Define Fiber app.
	app := fiber.New()

	// Create route with GET method for test
	app.Post("/admission/mutating", MutatingWebhook)

	data := []byte(`{
		"kind": "AdmissionReview",
		"apiVersion": "admission.k8s.io/v1",
		"request": {
			"uid": "0a0dffd5-99a1-4638-b539-03e51e761088",
			"kind": {
				"group": "compute.cnrm.cloud.google.com",
				"version": "v1beta1",
				"kind": "ComputeSubnetwork"
			},
			"resource": {
				"group": "compute.cnrm.cloud.google.com",
				"version": "v1beta1",
				"resource": "computesubnetworks"
			},
			"requestKind": {
				"group": "compute.cnrm.cloud.google.com",
				"version": "v1beta1",
				"kind": "ComputeSubnetwork"
			},
			"requestResource": {
				"group": "compute.cnrm.cloud.google.com",
				"version": "v1beta1",
				"resource": "computesubnetworks"
			},
			"name": "computesubnetwork-sample3",
			"namespace": "default",
			"operation": "CREATE",
			"userInfo": {
				"username": "grotz@grotz.joonix.net",
				"groups": ["system:authenticated"],
				"extra": {
					"iam.gke.io/user-assertion": ["AK8TC8LFRvtnoDTrG58hcF5+evI5AtUh3zTV/TMduBLDjTo0vO/PTJzsCJWaFV6iKuK5Oqf/xCUWiMHOtpWBeL77PgJoWqjf3xvBfY+NBKvAMqAcId9IEsIjIMsYgcTi8knVb9J7asxxxlu7+NzcykNgf4rO9x9sF3tbz0TtcuVgyfQHfJgFjD2Rz+lLocTrYyA4RhwifGQqEWWqDMReKxgjBUm57lhn9esNf/iYDZrZpKhjxyUmgq42heYvfCTfnukJhscLjBnMY6IrWGr6AaT7SQ29orQAWPAuyBykSDJIlgsmHbUB38PIrzNHl8oAFw9wuHnlnEYAhQhFmfFOHkNT9+ckl7HAlEglBlIgOSCNB/sbODm38+KDV8vGKXZ91n9Jmsmcb6GlwAWRbABky6qnpAdnzch5ooVvpSiEoTbEZpSlUEmgEmyCkwujAzMA7VgGMONLnbYVGpWiLfPInP+khz0K9HyMMMztt+YsEFd5LDSEiLAJHWlPS/MJDZ9ErjYQ7ZT67ec3i7QNtjNPN5Y5qXuZf1yEUqVOlrTHWbZKcYXob8UfoAosXcGid56AIYGb10xLlPVdLAj0WUe8mt+pizUtqLOshXIISWlHwdjeyyOQqCqaqgzmZf2GZUfD43cm2FE2Xh+EYugE32NFObgw4aY/xTkSYBTZQZYy4x5eaYEzNIc9yXtcts5r5pf832Fu/DcHt0an/IhQOMebiM7hbP8idcE2XJL57wdKjoTsVX+WcTn5Pq4OEUttXiVWWAUA7trteQi/eJNxO3foo1m33VTQBoVF6CaGygMUv/fOGBOxGVyiiopMUyCWPGQpwPlChixhzeQu1YfiAL//CkXwjw0t6LbkusveWQj12l8ssoKAegVRacnmuwgh+/a3IwC+qwamZBmojsDAs0zg29VtdPDyTAFKRizUxECdR4UskD3v4Wor9LFFCnFMQiWmpfsL2MX1tQZvYuhtzVtur0ln1LV+uC4KzlxRMnMJ12gh/KEg64mTSuYF5xNpPHdzUkh3OxEXfRA9jtCUks37RXwjX5srRJ7Qrw9EeD8M6XfdHH9yhvqV1y26ZNz598hNHQnZfCaGlXJzx7qk3WirPiSV/0yRGcyNmXcYL7XDZR3ZAUacV5pEMlOVZXPH4g=="],
					"user-assertion.cloud.google.com": ["AK8TC8LQD5O311Yyu+k6On6wURtTn8bbmPMh6UkBcN2aEiwEZ1+LKmatkCfWABrQKcdMAp1mSb1CVzOGgJNDIKcIBRddHNOxyBQ/Z+bRrIP8r6jbJv9ZhQA5CHVq1h1S8jX8CfSreeYApN3hyyoZUr5K1+LnnVFL7tjujsm2XnfXYM3Jyuk3gPMhyuou7h0OYOMKKpLNHoVFa0eJZ9hcL0l7ueo2f8rQGhMEpAgb+7A="]
				}
			},
			"object": {
				"apiVersion": "compute.cnrm.cloud.google.com/v1beta1",
				"kind": "ComputeSubnetwork",
				"metadata": {
					"annotations": {
						"cnrm.cloud.google.com/management-conflict-prevention-policy": "none",
						"cnrm.cloud.google.com/project-id": "default",
						"cnrm.cloud.google.com/state-into-spec": "merge",
						"ipam.cloud.google.com/parent": "1",
						"ipam.cloud.google.com/size": "26",
						"kubectl.kubernetes.io/last-applied-configuration": "{\"apiVersion\":\"compute.cnrm.cloud.google.com/v1beta1\",\"kind\":\"ComputeSubnetwork\",\"metadata\":{\"annotations\":{\"ipam.cloud.google.com/parent-id\":\"1\",\"ipam.cloud.google.com/size\":\"26\"},\"labels\":{\"label-one\":\"value-one\"},\"name\":\"computesubnetwork-sample3\",\"namespace\":\"default\"},\"spec\":{\"description\":\"My subnet3\",\"logConfig\":{\"aggregationInterval\":\"INTERVAL_10_MIN\",\"flowSampling\":0.5,\"metadata\":\"INCLUDE_ALL_METADATA\"},\"networkRef\":{\"name\":\"computesubnetwork-dep2\"},\"privateIpGoogleAccess\":false,\"region\":\"us-central1\"}}\n"
					},
					"creationTimestamp": null,
					"labels": {
						"label-one": "value-one"
					},
					"managedFields": [{
						"apiVersion": "compute.cnrm.cloud.google.com/v1beta1",
						"fieldsType": "FieldsV1",
						"fieldsV1": {
							"f:metadata": {
								"f:annotations": {
									".": {},
									"f:ipam.cloud.google.com/parent-id": {},
									"f:ipam.cloud.google.com/size": {},
									"f:kubectl.kubernetes.io/last-applied-configuration": {}
								},
								"f:labels": {
									".": {},
									"f:label-one": {}
								}
							},
							"f:spec": {
								".": {},
								"f:description": {},
								"f:logConfig": {
									".": {},
									"f:aggregationInterval": {},
									"f:flowSampling": {},
									"f:metadata": {}
								},
								"f:networkRef": {
									".": {},
									"f:name": {}
								},
								"f:privateIpGoogleAccess": {},
								"f:region": {}
							}
						},
						"manager": "kubectl-client-side-apply",
						"operation": "Update",
						"time": "2021-12-25T23:09:23Z"
					}],
					"name": "computesubnetwork-sample3",
					"namespace": "default"
				},
				"spec": {
					"description": "My subnet3",
					"logConfig": {
						"aggregationInterval": "INTERVAL_10_MIN",
						"flowSampling": 0.5,
						"metadata": "INCLUDE_ALL_METADATA"
					},
					"networkRef": {
						"name": "computesubnetwork-dep2"
					},
					"privateIpGoogleAccess": false,
					"region": "us-central1"
				}
			},
			"oldObject": null,
			"dryRun": false,
			"options": {
				"kind": "CreateOptions",
				"apiVersion": "meta.k8s.io/v1",
				"fieldManager": "kubectl-client-side-apply"
			}
		}
	}
	`)
	req := httptest.NewRequest("POST", "/admission/mutating", bytes.NewReader(data))
	req.Header["Content-Type"] = []string{"application/json"}
	resp, err := app.Test(req, -1)
	if err != nil {
		log.Fatal(err)
	}
	assert.Equalf(t, 200, resp.StatusCode, "Status Code")
	buf := new(strings.Builder)
	io.Copy(buf, resp.Body)
	respBody := buf.String()
	assert.Equalf(t, "{\"kind\":\"AdmissionReview\",\"apiVersion\":\"admission.k8s.io/v1\",\"response\":{\"uid\":\"0a0dffd5-99a1-4638-b539-03e51e761088\",\"allowed\":true,\"patch\":\"W3sib3AiOiJhZGQiLCJwYXRoIjoiL21ldGFkYXRhL2Fubm90YXRpb25zL2lwYW0uY2xvdWQuZ29vZ2xlLmNvbS9yYW5nZS1pZCIsInZhbHVlIjoiMSJ9LHsib3AiOiJyZXBsYWNlIiwicGF0aCI6Ii9zcGVjL2lwQ2lkclJhbmdlIiwidmFsdWUiOiIxMC4wLjAuMC8yNiJ9XQ==\",\"patchType\":\"JSONPatch\"}}", respBody, "Body")
}
