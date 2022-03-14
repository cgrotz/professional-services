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
	"encoding/json"
	"fmt"
	"log"
	"strconv"

	"github.com/GoogleCloudPlatform/professional-services/ipam-autopilot/data_access"
	"github.com/GoogleCloudPlatform/professional-services/ipam-autopilot/model"
	"github.com/gofiber/fiber/v2"
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

/*
Check if Range is collision free. (Routing Domain Lookup either via Label or via Network)
*/
func ValidatingWebhook(c *fiber.Ctx) error {
	//log.Printf("Received Validation request %s", c.Body())
	request := admissionv1.AdmissionReview{}
	if err := c.BodyParser(&request); err != nil {
		log.Printf("Unable to unmarshal body: %v", err)
		return err
	}

	if request.Request.Kind.Group != "compute.cnrm.cloud.google.com" || request.Request.Kind.Version != "v1beta1" || request.Request.Kind.Kind != "ComputeSubnetwork" {
		resp := admissionv1.AdmissionReview{
			Response: &admissionv1.AdmissionResponse{
				UID:     request.Request.UID,
				Allowed: true,
				Warnings: []string{
					fmt.Sprintf("This admission hook is not able to handle the provided resources group=%s,version=%s,kind=%s", request.Request.Kind.Group, request.Request.Kind.Version, request.Request.Kind.Kind),
				},
			},
		}
		resp.Kind = "AdmissionReview"
		resp.APIVersion = "admission.k8s.io/v1"
		return c.Status(200).JSON(&resp)
	}

	obj := make(map[string]interface{})
	json.Unmarshal(request.Request.Object.Raw, &obj)
	if err := c.BodyParser(&request); err != nil {
		log.Printf("Unable to unmarshal body: %v", err)
		return err
	}

	tx, err := data_access.GetTransaction()
	if err != nil {
		log.Fatal(err)
	}

	metadata := obj["metadata"].(map[string]interface{})
	name := ""
	if metadata["name"] != nil {
		name = metadata["name"].(string)
	}
	annotations := metadata["annotations"].(map[string]interface{})
	// projectId doesn't contain the project projectId := annotations["cnrm.cloud.google.com/project-id"]
	routingDomainId := ""
	if annotations["ipam.cloud.google.com/routing-domain-id"] != nil {
		routingDomainId = annotations["ipam.cloud.google.com/routing-domain-id"].(string)
	}

	if routingDomainId == "" {
		resp := admissionv1.AdmissionReview{
			Response: &admissionv1.AdmissionResponse{
				UID:     request.Request.UID,
				Allowed: true,
			},
		}
		resp.Kind = "AdmissionReview"
		resp.APIVersion = "admission.k8s.io/v1"
		return c.Status(200).JSON(&resp)
	}

	routingDomain, err := getRoutingDomain(tx, routingDomainId)
	if err != nil {
		return c.Status(400).JSON(&fiber.Map{
			"success": false,
			"message": fmt.Sprintf("%v", err),
		})
	}

	rangeId := ""
	if annotations["ipam.cloud.google.com/range-id"] != nil {
		rangeId = annotations["ipam.cloud.google.com/range-id"].(string)
	}
	if rangeId == "" {
		spec := obj["spec"].(map[string]interface{})
		ipCidrRange := ""
		if spec["ipCidrRange"] != nil {
			ipCidrRange = spec["ipCidrRange"].(string)
		}

		//networkRef := spec["networkRef"].(map[string]interface{})["name"]
		if name != "" && ipCidrRange != "" {
			err := checkOverlap(tx, routingDomain, ipCidrRange)
			if err != nil {
				resp := admissionv1.AdmissionReview{
					Response: &admissionv1.AdmissionResponse{
						UID:     request.Request.UID,
						Allowed: false,
						Result: &metav1.Status{
							Status:  "Failure",
							Message: "Overlap for routing domain detected",
						},
					},
				}
				resp.Kind = "AdmissionReview"
				resp.APIVersion = "admission.k8s.io/v1"
				return c.Status(200).JSON(&resp)
			} else {
				resp := admissionv1.AdmissionReview{
					Response: &admissionv1.AdmissionResponse{
						UID:     request.Request.UID,
						Allowed: true,
					},
				}
				resp.Kind = "AdmissionReview"
				resp.APIVersion = "admission.k8s.io/v1"
				return c.Status(200).JSON(&resp)
			}
		} else {
			resp := admissionv1.AdmissionReview{
				Response: &admissionv1.AdmissionResponse{
					UID:     request.Request.UID,
					Allowed: true,
				},
			}
			resp.Kind = "AdmissionReview"
			resp.APIVersion = "admission.k8s.io/v1"
			return c.Status(200).JSON(&resp)
		}
	} else {
		// If it has a rangeId it's managed by IPAM Autopilot, nothing to see here
		resp := admissionv1.AdmissionReview{
			Response: &admissionv1.AdmissionResponse{
				UID:     request.Request.UID,
				Allowed: true,
			},
		}
		resp.Kind = "AdmissionReview"
		resp.APIVersion = "admission.k8s.io/v1"
		return c.Status(200).JSON(&resp)
	}
}

/*
Add free IP Range, Size from Label, (Routing Domain Lookup either via Label or via Network, Parent either via Label or implicit if only one top range in Routing Domain)
*/
func MutatingWebhook(c *fiber.Ctx) error {
	//log.Printf("Received Mutating request %s", c.Body())
	request := admissionv1.AdmissionReview{}
	if err := c.BodyParser(&request); err != nil {
		log.Printf("Unable to unmarshal body: %v", err)
		return err
	}

	if request.Request.Kind.Group != "compute.cnrm.cloud.google.com" || request.Request.Kind.Version != "v1beta1" || request.Request.Kind.Kind != "ComputeSubnetwork" {
		resp := admissionv1.AdmissionReview{
			Response: &admissionv1.AdmissionResponse{
				UID:     request.Request.UID,
				Allowed: true,
				Warnings: []string{
					fmt.Sprintf("This admission hook is not able to handle the provided resources group=%s,version=%s,kind=%s", request.Request.Kind.Group, request.Request.Kind.Version, request.Request.Kind.Kind),
				},
			},
		}
		resp.Kind = "AdmissionReview"
		resp.APIVersion = "admission.k8s.io/v1"
		return c.Status(200).JSON(&resp)
	}

	if request.Request.Operation == "CREATE" {
		return handleCreation(c, request)
	} else if request.Request.Operation == "DELETE" {
		return handleDeletion(c, request)
	} else {
		// Should this Admission webhook be able to handle alterations?
		resp := admissionv1.AdmissionReview{
			Response: &admissionv1.AdmissionResponse{
				UID:     request.Request.UID,
				Allowed: true,
				Warnings: []string{
					fmt.Sprintf("This admission hook is not able to handle the provided resources group=%s,version=%s,kind=%s", request.Request.Kind.Group, request.Request.Kind.Version, request.Request.Kind.Kind),
				},
			},
		}
		resp.Kind = "AdmissionReview"
		resp.APIVersion = "admission.k8s.io/v1"
		return c.Status(200).JSON(&resp)
	}
}

func handleCreation(c *fiber.Ctx, request admissionv1.AdmissionReview) error {
	var err error
	obj := make(map[string]interface{})
	json.Unmarshal(request.Request.Object.Raw, &obj)
	if err = c.BodyParser(&request); err != nil {
		log.Printf("Unable to unmarshal body: %v", err)
		return err
	}

	// TODO handle DryRuns request.Request.DryRun
	metadata := obj["metadata"].(map[string]interface{})
	name := ""
	if metadata["name"] != nil {
		name = metadata["name"].(string)
	}
	annotations := metadata["annotations"].(map[string]interface{})

	// projectId doesn't necessarily contain the project, it might just contain the namespace name https://cloud.google.com/config-connector/docs/how-to/organizing-resources/project-scoped-resources
	projectId := ""
	if annotations["cnrm.cloud.google.com/project-id"] != nil {
		projectId = annotations["cnrm.cloud.google.com/project-id"].(string)
	}

	routingDomainId := ""
	if annotations["ipam.cloud.google.com/routing-domain-id"] != nil {
		routingDomainId = annotations["ipam.cloud.google.com/routing-domain-id"].(string)
	}
	if annotations["ipam.cloud.google.com/size"] != nil {
		size := annotations["ipam.cloud.google.com/size"].(string)
		parent := ""
		if annotations["ipam.cloud.google.com/parent"] != nil {
			parent = annotations["ipam.cloud.google.com/parent"].(string)
		}

		sizeInteger, err := strconv.ParseInt(size, 10, 64)
		if err != nil {
			return c.Status(400).JSON(&fiber.Map{
				"success": false,
				"message": fmt.Sprintf("%v", err),
			})
		}

		tx, err := data_access.GetTransaction()
		if err != nil {
			log.Fatal(err)
		}
		var routingDomain *model.RoutingDomain
		if routingDomainId != "" {
			routingDomain, err = getRoutingDomain(tx, routingDomainId)
			if err != nil {
				err = tx.Commit()
				if err != nil {
					log.Fatal(err)
				}
				return c.Status(400).JSON(&fiber.Map{
					"success": false,
					"message": fmt.Sprintf("%v", err),
				})
			}
		} else {
			spec := metadata["spec"].(map[string]interface{})
			networkRef := spec["networkRef"].(map[string]interface{})
			networkName := networkRef["name"].(string)
			// TODO can the routing ID be infered
			routingDomain, err = findRoutingDomainByVpc(tx, fmt.Sprintf("https://www.googleapis.com/compute/v1/projects/%s/global/networks/%s", projectId, networkName))
		}

		if *(request.Request.DryRun) {
			// What should be the DryRun behavior? Pass back a range without inserting to database?
			resp := admissionv1.AdmissionReview{
				Response: &admissionv1.AdmissionResponse{
					UID:     request.Request.UID,
					Allowed: true,
				},
			}
			resp.Kind = "AdmissionReview"
			resp.APIVersion = "admission.k8s.io/v1"
			return c.Status(200).JSON(&resp)
		} else {
			id, cidr, err := findNewLeaseAndInsert(tx, name, parent, int(sizeInteger), routingDomain)
			if err != nil {
				resp := admissionv1.AdmissionReview{
					Response: &admissionv1.AdmissionResponse{
						UID:     request.Request.UID,
						Allowed: false,
						Result: &metav1.Status{
							Status:  "Failure",
							Message: err.Error(),
						},
					},
				}
				resp.Kind = "AdmissionReview"
				resp.APIVersion = "admission.k8s.io/v1"
				err = tx.Commit()
				if err != nil {
					log.Fatal(err)
				}
				return c.Status(200).JSON(&resp)
			} else {
				patchType := admissionv1.PatchTypeJSONPatch
				patches := []Patch{}
				// Slash / needs to be encoded https://www.rfc-editor.org/rfc/rfc6901#section-3
				patches = append(patches, NewPatch("add", "/metadata/annotations/ipam.cloud.google.com~1range-id", fmt.Sprintf("%d", id)))
				patches = append(patches, NewPatch("replace", "/spec/ipCidrRange", cidr))
				data, err := c.App().Config().JSONEncoder(patches)
				if err != nil {
					return err

				}
				resp := admissionv1.AdmissionReview{
					Response: &admissionv1.AdmissionResponse{
						UID:       request.Request.UID,
						Allowed:   true,
						PatchType: &patchType,
						Patch:     data,
					},
				}
				resp.Kind = "AdmissionReview"
				resp.APIVersion = "admission.k8s.io/v1"
				bodyBytes, _ := json.Marshal(&resp)
				log.Printf("Responding to Mutating request %s", string(bodyBytes))
				return c.Status(200).JSON(&resp)
			}
		}
	} else {
		resp := admissionv1.AdmissionReview{
			Response: &admissionv1.AdmissionResponse{
				UID:     request.Request.UID,
				Allowed: true,
			},
		}
		resp.Kind = "AdmissionReview"
		resp.APIVersion = "admission.k8s.io/v1"
		return c.Status(200).JSON(&resp)
	}
}

func handleDeletion(c *fiber.Ctx, request admissionv1.AdmissionReview) error {
	var err error
	obj := make(map[string]interface{})
	json.Unmarshal(request.Request.OldObject.Raw, &obj)
	if err = c.BodyParser(&request); err != nil {
		log.Printf("Unable to unmarshal body: %v", err)
		return err
	}

	metadata := obj["metadata"].(map[string]interface{})
	annotations := metadata["annotations"].(map[string]interface{})

	rangeId := ""
	if annotations["ipam.cloud.google.com/range-id"] != nil {
		rangeId = annotations["ipam.cloud.google.com/range-id"].(string)
	}

	if rangeId == "" {
		resp := admissionv1.AdmissionReview{
			Response: &admissionv1.AdmissionResponse{
				UID:     request.Request.UID,
				Allowed: true,
			},
		}
		resp.Kind = "AdmissionReview"
		resp.APIVersion = "admission.k8s.io/v1"
		return c.Status(200).JSON(&resp)
	} else {
		rangeIdInt, err := strconv.ParseInt(rangeId, 10, 64)
		if err != nil {
			return c.SendStatus(400)
		}
		err = data_access.DeleteRangeFromDb(rangeIdInt)
		if err != nil {
			resp := admissionv1.AdmissionReview{
				Response: &admissionv1.AdmissionResponse{
					UID:     request.Request.UID,
					Allowed: false,
					Result: &metav1.Status{
						Status:  "Failure",
						Message: "Unable to delete range",
					},
				},
			}
			resp.Kind = "AdmissionReview"
			resp.APIVersion = "admission.k8s.io/v1"
			return c.Status(200).JSON(&resp)
		} else {
			resp := admissionv1.AdmissionReview{
				Response: &admissionv1.AdmissionResponse{
					UID:     request.Request.UID,
					Allowed: true,
				},
			}
			resp.Kind = "AdmissionReview"
			resp.APIVersion = "admission.k8s.io/v1"
			return c.Status(200).JSON(&resp)
		}
	}
}

func getRoutingDomain(tx *sql.Tx, routingDomainId string) (*model.RoutingDomain, error) {
	var err error
	var routingDomain *model.RoutingDomain
	if routingDomainId == "" {
		routingDomain, err = data_access.GetDefaultRoutingDomainFromDB(tx)
		if err != nil {
			log.Printf("Error %v", err)
			tx.Rollback()
			return nil, fmt.Errorf("Couldn't retrieve default routing domain")
		}
	} else {
		domain_id, err := strconv.ParseInt(routingDomainId, 10, 64)
		if err != nil {
			return nil, err

		}
		routingDomain, err = data_access.GetRoutingDomainFromDB(domain_id)
		if err != nil {
			log.Printf("Error %v", err)
			tx.Rollback()
			return nil, fmt.Errorf("Couldn't retrieve default routing domain")
		}
	}
	return routingDomain, nil
}

func findRoutingDomainByVpc(tx *sql.Tx, vpcName string) (*model.RoutingDomain, error) {
	var err error
	var routingDomain *model.RoutingDomain
	if routingDomainId == "" {
		routingDomain, err = data_access.GetDefaultRoutingDomainFromDB(tx)
		if err != nil {
			log.Printf("Error %v", err)
			tx.Rollback()
			return nil, fmt.Errorf("Couldn't retrieve default routing domain")
		}
	} else {
		domain_id, err := strconv.ParseInt(routingDomainId, 10, 64)
		if err != nil {
			return nil, err

		}
		routingDomain, err = data_access.GetRoutingDomainFromDB(domain_id)
		if err != nil {
			log.Printf("Error %v", err)
			tx.Rollback()
			return nil, fmt.Errorf("Couldn't retrieve default routing domain")
		}
	}
	return routingDomain, nil
}

type Patch struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value string `json:"value"`
}

func NewPatch(op string, path string, value string) Patch {
	return Patch{
		Op:    op,
		Path:  path,
		Value: value,
	}
}
