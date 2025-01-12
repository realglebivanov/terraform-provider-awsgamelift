package ag

import (
	"fmt"
	"log"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func init() {
	resource.AddTestSweepers("aws_network_acl", &resource.Sweeper{
		Name: "aws_network_acl",
		F:    testSweepNetworkAcls,
	})
}

func testSweepNetworkAcls(region string) error {
	client, err := sharedClientForRegion(region)
	if err != nil {
		return fmt.Errorf("error getting client: %s", err)
	}
	conn := client.(*AWSClient).ec2conn

	req := &ec2.DescribeNetworkAclsInput{}
	resp, err := conn.DescribeNetworkAcls(req)
	if err != nil {
		if testSweepSkipSweepError(err) {
			log.Printf("[WARN] Skipping EC2 Network ACL sweep for %s: %s", region, err)
			return nil
		}
		return fmt.Errorf("Error describing Network ACLs: %s", err)
	}

	if len(resp.NetworkAcls) == 0 {
		log.Print("[DEBUG] No Network ACLs to sweep")
		return nil
	}

	for _, nacl := range resp.NetworkAcls {
		// Delete rules first
		for _, entry := range nacl.Entries {
			// These are the rule numbers for IPv4 and IPv6 "ALL traffic" rules which cannot be deleted
			if aws.Int64Value(entry.RuleNumber) == 32767 || aws.Int64Value(entry.RuleNumber) == 32768 {
				log.Printf("[DEBUG] Skipping Network ACL rule: %q / %d", *nacl.NetworkAclId, *entry.RuleNumber)
				continue
			}

			log.Printf("[INFO] Deleting Network ACL rule: %q / %d", *nacl.NetworkAclId, *entry.RuleNumber)
			_, err := conn.DeleteNetworkAclEntry(&ec2.DeleteNetworkAclEntryInput{
				NetworkAclId: nacl.NetworkAclId,
				Egress:       entry.Egress,
				RuleNumber:   entry.RuleNumber,
			})
			if err != nil {
				return fmt.Errorf(
					"Error deleting Network ACL rule (%s / %d): %s",
					*nacl.NetworkAclId, *entry.RuleNumber, err)
			}
		}

		// Disassociate subnets
		log.Printf("[DEBUG] Found %d Network ACL associations for %q", len(nacl.Associations), *nacl.NetworkAclId)
		for _, a := range nacl.Associations {
			log.Printf("[DEBUG] Replacing subnet associations for Network ACL %q", *nacl.NetworkAclId)
			defaultAcl, err := getDefaultNetworkAcl(*nacl.VpcId, conn)
			if err != nil {
				return fmt.Errorf("Failed to find default Network ACL for VPC %q", *nacl.VpcId)
			}
			_, err = conn.ReplaceNetworkAclAssociation(&ec2.ReplaceNetworkAclAssociationInput{
				NetworkAclId:  defaultAcl.NetworkAclId,
				AssociationId: a.NetworkAclAssociationId,
			})
			if err != nil {
				return fmt.Errorf("Failed to replace subnet association for Network ACL %q: %s",
					*nacl.NetworkAclId, err)
			}
		}

		// Default Network ACLs will be deleted along with VPC
		if *nacl.IsDefault {
			log.Printf("[DEBUG] Skipping default Network ACL: %q", *nacl.NetworkAclId)
			continue
		}

		log.Printf("[INFO] Deleting Network ACL: %q", *nacl.NetworkAclId)
		_, err := conn.DeleteNetworkAcl(&ec2.DeleteNetworkAclInput{
			NetworkAclId: nacl.NetworkAclId,
		})
		if err != nil {
			return fmt.Errorf(
				"Error deleting Network ACL (%s): %s",
				*nacl.NetworkAclId, err)
		}
	}

	return nil
}

func TestAccAWSNetworkAcl_basic(t *testing.T) {
	resourceName := "aws_network_acl.test"
	var networkAcl ec2.NetworkAcl

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, ec2.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSNetworkAclDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSNetworkAclEgressNIngressConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSNetworkAclExists(resourceName, &networkAcl),
					testAccMatchResourceAttrRegionalARN(resourceName, "arn", "ec2", regexp.MustCompile(`network-acl/acl-.+`)),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "0"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAWSNetworkAcl_tags(t *testing.T) {
	resourceName := "aws_network_acl.test"
	rName := acctest.RandomWithPrefix("tf-acc-test")
	var networkAcl ec2.NetworkAcl

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, ec2.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSNetworkAclDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSNetworkAclConfigTags1(rName, "key1", "value1"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSNetworkAclExists(resourceName, &networkAcl),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "tags.key1", "value1"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccAWSNetworkAclConfigTags2(rName, "key1", "value1updated", "key2", "value2"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSNetworkAclExists(resourceName, &networkAcl),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "tags.key1", "value1updated"),
					resource.TestCheckResourceAttr(resourceName, "tags.key2", "value2"),
				),
			},
			{
				Config: testAccAWSNetworkAclConfigTags1(rName, "key2", "value2"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSNetworkAclExists(resourceName, &networkAcl),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "tags.key2", "value2"),
				),
			},
		},
	})
}

func TestAccAWSNetworkAcl_disappears(t *testing.T) {
	var networkAcl ec2.NetworkAcl
	resourceName := "aws_network_acl.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, ec2.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSNetworkAclDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSNetworkAclEgressNIngressConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSNetworkAclExists(resourceName, &networkAcl),
					testAccCheckResourceDisappears(testAccProvider, resourceAwsNetworkAcl(), resourceName),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccAWSNetworkAcl_Egress_ConfigMode(t *testing.T) {
	var networkAcl1, networkAcl2, networkAcl3 ec2.NetworkAcl
	resourceName := "aws_network_acl.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, ec2.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSNetworkAclDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSNetworkAclConfigEgressConfigModeBlocks(),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSNetworkAclExists(resourceName, &networkAcl1),
					testAccCheckAWSNetworkAclEgressRuleLength(&networkAcl1, 2),
					resource.TestCheckResourceAttr(resourceName, "egress.#", "2"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccAWSNetworkAclConfigEgressConfigModeNoBlocks(),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSNetworkAclExists(resourceName, &networkAcl2),
					testAccCheckAWSNetworkAclEgressRuleLength(&networkAcl2, 2),
					resource.TestCheckResourceAttr(resourceName, "egress.#", "2"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccAWSNetworkAclConfigEgressConfigModeZeroed(),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSNetworkAclExists(resourceName, &networkAcl3),
					testAccCheckAWSNetworkAclEgressRuleLength(&networkAcl3, 0),
					resource.TestCheckResourceAttr(resourceName, "egress.#", "0"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAWSNetworkAcl_Ingress_ConfigMode(t *testing.T) {
	var networkAcl1, networkAcl2, networkAcl3 ec2.NetworkAcl
	resourceName := "aws_network_acl.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, ec2.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSNetworkAclDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSNetworkAclConfigIngressConfigModeBlocks(),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSNetworkAclExists(resourceName, &networkAcl1),
					testIngressRuleLength(&networkAcl1, 2),
					resource.TestCheckResourceAttr(resourceName, "ingress.#", "2"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccAWSNetworkAclConfigIngressConfigModeNoBlocks(),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSNetworkAclExists(resourceName, &networkAcl2),
					testIngressRuleLength(&networkAcl2, 2),
					resource.TestCheckResourceAttr(resourceName, "ingress.#", "2"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccAWSNetworkAclConfigIngressConfigModeZeroed(),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSNetworkAclExists(resourceName, &networkAcl3),
					testIngressRuleLength(&networkAcl3, 0),
					resource.TestCheckResourceAttr(resourceName, "ingress.#", "0"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAWSNetworkAcl_EgressAndIngressRules(t *testing.T) {
	var networkAcl ec2.NetworkAcl
	resourceName := "aws_network_acl.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, ec2.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSNetworkAclDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSNetworkAclEgressNIngressConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSNetworkAclExists(resourceName, &networkAcl),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "ingress.*", map[string]string{
						"protocol":   "6",
						"rule_no":    "1",
						"from_port":  "80",
						"to_port":    "80",
						"action":     "allow",
						"cidr_block": "10.3.0.0/18",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "egress.*", map[string]string{
						"protocol":   "6",
						"rule_no":    "2",
						"from_port":  "443",
						"to_port":    "443",
						"action":     "allow",
						"cidr_block": "10.3.0.0/18",
					}),
					testAccCheckResourceAttrAccountID(resourceName, "owner_id"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAWSNetworkAcl_OnlyIngressRules_basic(t *testing.T) {
	var networkAcl ec2.NetworkAcl
	resourceName := "aws_network_acl.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, ec2.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSNetworkAclDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSNetworkAclIngressConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSNetworkAclExists(resourceName, &networkAcl),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "ingress.*", map[string]string{
						"protocol":   "6",
						"rule_no":    "2",
						"from_port":  "443",
						"to_port":    "443",
						"action":     "deny",
						"cidr_block": "10.2.0.0/18",
					}),
					testAccCheckResourceAttrAccountID(resourceName, "owner_id"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAWSNetworkAcl_OnlyIngressRules_update(t *testing.T) {
	var networkAcl ec2.NetworkAcl
	resourceName := "aws_network_acl.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, ec2.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSNetworkAclDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSNetworkAclIngressConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSNetworkAclExists(resourceName, &networkAcl),
					testIngressRuleLength(&networkAcl, 2),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "ingress.*", map[string]string{
						"protocol":  "6",
						"rule_no":   "1",
						"from_port": "0",
						"to_port":   "22",
						"action":    "deny",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "ingress.*", map[string]string{
						"cidr_block": "10.2.0.0/18",
						"from_port":  "443",
						"rule_no":    "2",
					}),
					testAccCheckResourceAttrAccountID(resourceName, "owner_id"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccAWSNetworkAclIngressConfigChange,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSNetworkAclExists(resourceName, &networkAcl),
					testIngressRuleLength(&networkAcl, 1),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "ingress.*", map[string]string{
						"protocol":   "6",
						"rule_no":    "1",
						"from_port":  "0",
						"to_port":    "22",
						"action":     "deny",
						"cidr_block": "10.2.0.0/18",
					}),
					testAccCheckResourceAttrAccountID(resourceName, "owner_id"),
				),
			},
		},
	})
}

func TestAccAWSNetworkAcl_CaseSensitivityNoChanges(t *testing.T) {
	var networkAcl ec2.NetworkAcl
	resourceName := "aws_network_acl.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, ec2.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSNetworkAclDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSNetworkAclCaseSensitiveConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSNetworkAclExists(resourceName, &networkAcl),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAWSNetworkAcl_OnlyEgressRules(t *testing.T) {
	var networkAcl ec2.NetworkAcl
	resourceName := "aws_network_acl.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, ec2.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSNetworkAclDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSNetworkAclEgressConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSNetworkAclExists(resourceName, &networkAcl),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "tags.Name", "tf-acc-acl-egress"),
					resource.TestCheckResourceAttr(resourceName, "tags.foo", "bar"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAWSNetworkAcl_SubnetChange(t *testing.T) {
	var networkAcl ec2.NetworkAcl
	resourceName := "aws_network_acl.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, ec2.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSNetworkAclDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSNetworkAclSubnetConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSNetworkAclExists(resourceName, &networkAcl),
					testAccCheckSubnetIsAssociatedWithAcl(resourceName, "aws_subnet.old"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccAWSNetworkAclSubnetConfigChange,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSNetworkAclExists(resourceName, &networkAcl),
					testAccCheckSubnetIsNotAssociatedWithAcl(resourceName, "aws_subnet.old"),
					testAccCheckSubnetIsAssociatedWithAcl(resourceName, "aws_subnet.new"),
				),
			},
		},
	})

}

func TestAccAWSNetworkAcl_Subnets(t *testing.T) {
	var networkAcl ec2.NetworkAcl
	resourceName := "aws_network_acl.test"

	checkACLSubnets := func(acl *ec2.NetworkAcl, count int) resource.TestCheckFunc {
		return func(*terraform.State) (err error) {
			if count != len(acl.Associations) {
				return fmt.Errorf("ACL association count does not match, expected %d, got %d", count, len(acl.Associations))
			}

			return nil
		}
	}

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, ec2.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSNetworkAclDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSNetworkAclSubnet_SubnetIds,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSNetworkAclExists(resourceName, &networkAcl),
					testAccCheckSubnetIsAssociatedWithAcl(resourceName, "aws_subnet.one"),
					testAccCheckSubnetIsAssociatedWithAcl(resourceName, "aws_subnet.two"),
					checkACLSubnets(&networkAcl, 2),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccAWSNetworkAclSubnet_SubnetIdsUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSNetworkAclExists(resourceName, &networkAcl),
					testAccCheckSubnetIsAssociatedWithAcl(resourceName, "aws_subnet.one"),
					testAccCheckSubnetIsAssociatedWithAcl(resourceName, "aws_subnet.three"),
					testAccCheckSubnetIsAssociatedWithAcl(resourceName, "aws_subnet.four"),
					checkACLSubnets(&networkAcl, 3),
				),
			},
		},
	})
}

func TestAccAWSNetworkAcl_SubnetsDelete(t *testing.T) {
	var networkAcl ec2.NetworkAcl
	resourceName := "aws_network_acl.test"

	checkACLSubnets := func(acl *ec2.NetworkAcl, count int) resource.TestCheckFunc {
		return func(*terraform.State) (err error) {
			if count != len(acl.Associations) {
				return fmt.Errorf("ACL association count does not match, expected %d, got %d", count, len(acl.Associations))
			}

			return nil
		}
	}

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, ec2.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSNetworkAclDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSNetworkAclSubnet_SubnetIds,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSNetworkAclExists(resourceName, &networkAcl),
					testAccCheckSubnetIsAssociatedWithAcl(resourceName, "aws_subnet.one"),
					testAccCheckSubnetIsAssociatedWithAcl(resourceName, "aws_subnet.two"),
					checkACLSubnets(&networkAcl, 2),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccAWSNetworkAclSubnet_SubnetIdsDeleteOne,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSNetworkAclExists(resourceName, &networkAcl),
					testAccCheckSubnetIsAssociatedWithAcl(resourceName, "aws_subnet.one"),
					checkACLSubnets(&networkAcl, 1),
				),
			},
		},
	})
}

func TestAccAWSNetworkAcl_ipv6Rules(t *testing.T) {
	var networkAcl ec2.NetworkAcl
	resourceName := "aws_network_acl.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, ec2.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSNetworkAclDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSNetworkAclIpv6Config,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSNetworkAclExists(resourceName, &networkAcl),
					resource.TestCheckResourceAttr(
						resourceName, "ingress.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "ingress.*", map[string]string{
						"protocol":        "6",
						"rule_no":         "1",
						"from_port":       "0",
						"to_port":         "22",
						"action":          "allow",
						"ipv6_cidr_block": "::/0",
					}),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAWSNetworkAcl_ipv6ICMPRules(t *testing.T) {
	var networkAcl ec2.NetworkAcl
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "aws_network_acl.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, ec2.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSNetworkAclDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSNetworkAclConfigIpv6ICMP(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSNetworkAclExists(resourceName, &networkAcl),
				),
			},
		},
	})
}

func TestAccAWSNetworkAcl_ipv6VpcRules(t *testing.T) {
	var networkAcl ec2.NetworkAcl
	resourceName := "aws_network_acl.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, ec2.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSNetworkAclDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSNetworkAclIpv6VpcConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSNetworkAclExists(resourceName, &networkAcl),
					resource.TestCheckResourceAttr(
						resourceName, "ingress.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "ingress.*", map[string]string{
						"ipv6_cidr_block": "2600:1f16:d1e:9a00::/56",
					}),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAWSNetworkAcl_espProtocol(t *testing.T) {
	var networkAcl ec2.NetworkAcl
	resourceName := "aws_network_acl.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, ec2.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSNetworkAclDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSNetworkAclEsp,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSNetworkAclExists(resourceName, &networkAcl),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckAWSNetworkAclDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).ec2conn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_network" {
			continue
		}

		// Retrieve the network acl
		resp, err := conn.DescribeNetworkAcls(&ec2.DescribeNetworkAclsInput{
			NetworkAclIds: []*string{aws.String(rs.Primary.ID)},
		})
		if err == nil {
			if len(resp.NetworkAcls) > 0 && *resp.NetworkAcls[0].NetworkAclId == rs.Primary.ID {
				return fmt.Errorf("Network Acl (%s) still exists.", rs.Primary.ID)
			}

			return nil
		}

		ec2err, ok := err.(awserr.Error)
		if !ok {
			return err
		}
		// Confirm error code is what we want
		if ec2err.Code() != "InvalidNetworkAclID.NotFound" {
			return err
		}
	}

	return nil
}

func testAccCheckAWSNetworkAclExists(n string, networkAcl *ec2.NetworkAcl) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set: %s", n)
		}
		conn := testAccProvider.Meta().(*AWSClient).ec2conn

		resp, err := conn.DescribeNetworkAcls(&ec2.DescribeNetworkAclsInput{
			NetworkAclIds: []*string{aws.String(rs.Primary.ID)},
		})
		if err != nil {
			return err
		}

		if len(resp.NetworkAcls) > 0 && *resp.NetworkAcls[0].NetworkAclId == rs.Primary.ID {
			*networkAcl = *resp.NetworkAcls[0]
			return nil
		}

		return fmt.Errorf("Network Acls not found")
	}
}

func testAccCheckAWSNetworkAclEgressRuleLength(networkAcl *ec2.NetworkAcl, length int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		var entries []*ec2.NetworkAclEntry
		for _, entry := range networkAcl.Entries {
			if aws.BoolValue(entry.Egress) {
				entries = append(entries, entry)
			}
		}
		// There is always a default rule (ALL Traffic ... DENY)
		// so we have to increase the length by 1
		if len(entries) != length+1 {
			return fmt.Errorf("Invalid number of ingress entries found; count = %d", len(entries))
		}
		return nil
	}
}

func testIngressRuleLength(networkAcl *ec2.NetworkAcl, length int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		var ingressEntries []*ec2.NetworkAclEntry
		for _, e := range networkAcl.Entries {
			if !*e.Egress {
				ingressEntries = append(ingressEntries, e)
			}
		}
		// There is always a default rule (ALL Traffic ... DENY)
		// so we have to increase the length by 1
		if len(ingressEntries) != length+1 {
			return fmt.Errorf("Invalid number of ingress entries found; count = %d", len(ingressEntries))
		}
		return nil
	}
}

func testAccCheckSubnetIsAssociatedWithAcl(acl string, sub string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		networkAcl := s.RootModule().Resources[acl]
		subnet := s.RootModule().Resources[sub]

		conn := testAccProvider.Meta().(*AWSClient).ec2conn
		resp, err := conn.DescribeNetworkAcls(&ec2.DescribeNetworkAclsInput{
			NetworkAclIds: []*string{aws.String(networkAcl.Primary.ID)},
			Filters: []*ec2.Filter{
				{
					Name:   aws.String("association.subnet-id"),
					Values: []*string{aws.String(subnet.Primary.ID)},
				},
			},
		})
		if err != nil {
			return err
		}
		if len(resp.NetworkAcls) > 0 {
			return nil
		}

		return fmt.Errorf("Network Acl %s is not associated with subnet %s", acl, sub)
	}
}

func testAccCheckSubnetIsNotAssociatedWithAcl(acl string, subnet string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		networkAcl := s.RootModule().Resources[acl]
		subnet := s.RootModule().Resources[subnet]

		conn := testAccProvider.Meta().(*AWSClient).ec2conn
		resp, err := conn.DescribeNetworkAcls(&ec2.DescribeNetworkAclsInput{
			NetworkAclIds: []*string{aws.String(networkAcl.Primary.ID)},
			Filters: []*ec2.Filter{
				{
					Name:   aws.String("association.subnet-id"),
					Values: []*string{aws.String(subnet.Primary.ID)},
				},
			},
		})

		if err != nil {
			return err
		}
		if len(resp.NetworkAcls) > 0 {
			return fmt.Errorf("Network Acl %s is still associated with subnet %s", acl, subnet)
		}
		return nil
	}
}

func testAccAWSNetworkAclConfigIpv6ICMP(rName string) string {
	return fmt.Sprintf(`
resource "aws_vpc" "test" {
  cidr_block = "10.1.0.0/16"

  tags = {
    Name = %q
  }
}

resource "aws_network_acl" "test" {
  vpc_id = aws_vpc.test.id

  ingress {
    action          = "allow"
    from_port       = 0
    icmp_code       = -1
    icmp_type       = -1
    ipv6_cidr_block = "::/0"
    protocol        = 58
    rule_no         = 1
    to_port         = 0
  }

  tags = {
    Name = %q
  }
}
`, rName, rName)
}

const testAccAWSNetworkAclIpv6Config = `
resource "aws_vpc" "foo" {
  cidr_block = "10.1.0.0/16"

  tags = {
    Name = "terraform-testacc-network-acl-ipv6"
  }
}

resource "aws_subnet" "blob" {
  cidr_block              = "10.1.1.0/24"
  vpc_id                  = aws_vpc.foo.id
  map_public_ip_on_launch = true

  tags = {
    Name = "tf-acc-network-acl-ipv6"
  }
}

resource "aws_network_acl" "test" {
  vpc_id = aws_vpc.foo.id

  ingress {
    protocol        = 6
    rule_no         = 1
    action          = "allow"
    ipv6_cidr_block = "::/0"
    from_port       = 0
    to_port         = 22
  }

  subnet_ids = [aws_subnet.blob.id]

  tags = {
    Name = "tf-acc-acl-ipv6"
  }
}
`

const testAccAWSNetworkAclIpv6VpcConfig = `
resource "aws_vpc" "foo" {
  cidr_block                       = "10.1.0.0/16"
  assign_generated_ipv6_cidr_block = true

  tags = {
    Name = "terraform-testacc-network-acl-ipv6-vpc-rules"
  }
}

resource "aws_network_acl" "test" {
  vpc_id = aws_vpc.foo.id

  ingress {
    protocol        = 6
    rule_no         = 1
    action          = "allow"
    ipv6_cidr_block = "2600:1f16:d1e:9a00::/56"
    from_port       = 0
    to_port         = 22
  }

  tags = {
    Name = "tf-acc-acl-ipv6"
  }
}
`

const testAccAWSNetworkAclIngressConfig = `
resource "aws_vpc" "foo" {
  cidr_block = "10.1.0.0/16"

  tags = {
    Name = "terraform-testacc-network-acl-ingress"
  }
}

resource "aws_subnet" "blob" {
  cidr_block              = "10.1.1.0/24"
  vpc_id                  = aws_vpc.foo.id
  map_public_ip_on_launch = true

  tags = {
    Name = "tf-acc-network-acl-ingress"
  }
}

resource "aws_network_acl" "test" {
  vpc_id = aws_vpc.foo.id

  ingress {
    protocol   = 6
    rule_no    = 1
    action     = "deny"
    cidr_block = "10.2.0.0/18"
    from_port  = 0
    to_port    = 22
  }

  ingress {
    protocol   = 6
    rule_no    = 2
    action     = "deny"
    cidr_block = "10.2.0.0/18"
    from_port  = 443
    to_port    = 443
  }

  subnet_ids = [aws_subnet.blob.id]

  tags = {
    Name = "tf-acc-acl-ingress"
  }
}
`

const testAccAWSNetworkAclCaseSensitiveConfig = `
resource "aws_vpc" "foo" {
  cidr_block = "10.1.0.0/16"

  tags = {
    Name = "terraform-testacc-network-acl-ingress"
  }
}

resource "aws_subnet" "blob" {
  cidr_block              = "10.1.1.0/24"
  vpc_id                  = aws_vpc.foo.id
  map_public_ip_on_launch = true

  tags = {
    Name = "tf-acc-network-acl-ingress"
  }
}

resource "aws_network_acl" "test" {
  vpc_id = aws_vpc.foo.id

  ingress {
    protocol   = 6
    rule_no    = 1
    action     = "Allow"
    cidr_block = "10.2.0.0/18"
    from_port  = 0
    to_port    = 22
  }

  subnet_ids = [aws_subnet.blob.id]

  tags = {
    Name = "tf-acc-acl-case-sensitive"
  }
}
`

const testAccAWSNetworkAclIngressConfigChange = `
resource "aws_vpc" "foo" {
  cidr_block = "10.1.0.0/16"

  tags = {
    Name = "terraform-testacc-network-acl-ingress"
  }
}

resource "aws_subnet" "blob" {
  cidr_block              = "10.1.1.0/24"
  vpc_id                  = aws_vpc.foo.id
  map_public_ip_on_launch = true

  tags = {
    Name = "tf-acc-network-acl-ingress"
  }
}

resource "aws_network_acl" "test" {
  vpc_id = aws_vpc.foo.id

  ingress {
    protocol   = 6
    rule_no    = 1
    action     = "deny"
    cidr_block = "10.2.0.0/18"
    from_port  = 0
    to_port    = 22
  }

  subnet_ids = [aws_subnet.blob.id]

  tags = {
    Name = "tf-acc-acl-ingress"
  }
}
`

const testAccAWSNetworkAclEgressConfig = `
resource "aws_vpc" "foo" {
  cidr_block = "10.2.0.0/16"

  tags = {
    Name = "terraform-testacc-network-acl-egress"
  }
}

resource "aws_subnet" "blob" {
  cidr_block              = "10.2.0.0/24"
  vpc_id                  = aws_vpc.foo.id
  map_public_ip_on_launch = true

  tags = {
    Name = "tf-acc-network-acl-egress"
  }
}

resource "aws_network_acl" "test" {
  vpc_id = aws_vpc.foo.id

  egress {
    protocol   = 6
    rule_no    = 2
    action     = "allow"
    cidr_block = "10.2.0.0/18"
    from_port  = 443
    to_port    = 443
  }

  egress {
    protocol   = "-1"
    rule_no    = 4
    action     = "allow"
    cidr_block = "0.0.0.0/0"
    from_port  = 0
    to_port    = 0
  }

  egress {
    protocol   = 6
    rule_no    = 1
    action     = "allow"
    cidr_block = "10.2.0.0/18"
    from_port  = 80
    to_port    = 80
  }

  egress {
    protocol   = 6
    rule_no    = 3
    action     = "allow"
    cidr_block = "10.2.0.0/18"
    from_port  = 22
    to_port    = 22
  }

  tags = {
    foo  = "bar"
    Name = "tf-acc-acl-egress"
  }
}
`

const testAccAWSNetworkAclEgressNIngressConfig = `
resource "aws_vpc" "foo" {
  cidr_block = "10.3.0.0/16"

  tags = {
    Name = "terraform-testacc-network-acl-egress-and-ingress"
  }
}

resource "aws_subnet" "blob" {
  cidr_block              = "10.3.0.0/24"
  vpc_id                  = aws_vpc.foo.id
  map_public_ip_on_launch = true

  tags = {
    Name = "tf-acc-network-acl-egress-and-ingress"
  }
}

resource "aws_network_acl" "test" {
  vpc_id = aws_vpc.foo.id

  egress {
    protocol   = 6
    rule_no    = 2
    action     = "allow"
    cidr_block = "10.3.0.0/18"
    from_port  = 443
    to_port    = 443
  }

  ingress {
    protocol   = 6
    rule_no    = 1
    action     = "allow"
    cidr_block = "10.3.0.0/18"
    from_port  = 80
    to_port    = 80
  }
}
`

const testAccAWSNetworkAclSubnetConfig = `
resource "aws_vpc" "foo" {
  cidr_block = "10.1.0.0/16"

  tags = {
    Name = "terraform-testacc-network-acl-subnet-change"
  }
}

resource "aws_subnet" "old" {
  cidr_block              = "10.1.111.0/24"
  vpc_id                  = aws_vpc.foo.id
  map_public_ip_on_launch = true

  tags = {
    Name = "tf-acc-network-acl-subnet-change-old"
  }
}

resource "aws_subnet" "new" {
  cidr_block              = "10.1.1.0/24"
  vpc_id                  = aws_vpc.foo.id
  map_public_ip_on_launch = true

  tags = {
    Name = "tf-acc-network-acl-subnet-change-new"
  }
}

resource "aws_network_acl" "roll" {
  vpc_id     = aws_vpc.foo.id
  subnet_ids = [aws_subnet.new.id]

  tags = {
    Name = "tf-acc-acl-subnet-change-roll"
  }
}

resource "aws_network_acl" "test" {
  vpc_id     = aws_vpc.foo.id
  subnet_ids = [aws_subnet.old.id]

  tags = {
    Name = "tf-acc-acl-subnet-change-test"
  }
}
`

const testAccAWSNetworkAclSubnetConfigChange = `
resource "aws_vpc" "foo" {
  cidr_block = "10.1.0.0/16"

  tags = {
    Name = "terraform-testacc-network-acl-subnet-change"
  }
}

resource "aws_subnet" "old" {
  cidr_block              = "10.1.111.0/24"
  vpc_id                  = aws_vpc.foo.id
  map_public_ip_on_launch = true

  tags = {
    Name = "tf-acc-network-acl-subnet-change-old"
  }
}

resource "aws_subnet" "new" {
  cidr_block              = "10.1.1.0/24"
  vpc_id                  = aws_vpc.foo.id
  map_public_ip_on_launch = true

  tags = {
    Name = "tf-acc-network-acl-subnet-change-new"
  }
}

resource "aws_network_acl" "test" {
  vpc_id     = aws_vpc.foo.id
  subnet_ids = [aws_subnet.new.id]

  tags = {
    Name = "tf-acc-acl-subnet-change-test"
  }
}
`

const testAccAWSNetworkAclSubnet_SubnetIds = `
resource "aws_vpc" "foo" {
  cidr_block = "10.1.0.0/16"

  tags = {
    Name = "terraform-testacc-network-acl-subnet-ids"
  }
}

resource "aws_subnet" "one" {
  cidr_block = "10.1.111.0/24"
  vpc_id     = aws_vpc.foo.id

  tags = {
    Name = "tf-acc-network-acl-subnet-ids-one"
  }
}

resource "aws_subnet" "two" {
  cidr_block = "10.1.1.0/24"
  vpc_id     = aws_vpc.foo.id

  tags = {
    Name = "tf-acc-network-acl-subnet-ids-two"
  }
}

resource "aws_network_acl" "test" {
  vpc_id     = aws_vpc.foo.id
  subnet_ids = [aws_subnet.one.id, aws_subnet.two.id]

  tags = {
    Name = "tf-acc-acl-subnet-ids"
  }
}
`

const testAccAWSNetworkAclSubnet_SubnetIdsUpdate = `
resource "aws_vpc" "foo" {
  cidr_block = "10.1.0.0/16"

  tags = {
    Name = "terraform-testacc-network-acl-subnet-ids"
  }
}

resource "aws_subnet" "one" {
  cidr_block = "10.1.111.0/24"
  vpc_id     = aws_vpc.foo.id

  tags = {
    Name = "tf-acc-network-acl-subnet-ids-one"
  }
}

resource "aws_subnet" "two" {
  cidr_block = "10.1.1.0/24"
  vpc_id     = aws_vpc.foo.id

  tags = {
    Name = "tf-acc-network-acl-subnet-ids-two"
  }
}

resource "aws_subnet" "three" {
  cidr_block = "10.1.222.0/24"
  vpc_id     = aws_vpc.foo.id

  tags = {
    Name = "tf-acc-network-acl-subnet-ids-three"
  }
}

resource "aws_subnet" "four" {
  cidr_block = "10.1.4.0/24"
  vpc_id     = aws_vpc.foo.id

  tags = {
    Name = "tf-acc-network-acl-subnet-ids-four"
  }
}

resource "aws_network_acl" "test" {
  vpc_id = aws_vpc.foo.id
  subnet_ids = [
    aws_subnet.one.id,
    aws_subnet.three.id,
    aws_subnet.four.id,
  ]

  tags = {
    Name = "tf-acc-acl-subnet-ids"
  }
}
`

const testAccAWSNetworkAclSubnet_SubnetIdsDeleteOne = `
resource "aws_vpc" "foo" {
  cidr_block = "10.1.0.0/16"

  tags = {
    Name = "terraform-testacc-network-acl-subnet-ids"
  }
}

resource "aws_subnet" "one" {
  cidr_block = "10.1.111.0/24"
  vpc_id     = aws_vpc.foo.id

  tags = {
    Name = "tf-acc-network-acl-subnet-ids-one"
  }
}

resource "aws_network_acl" "test" {
  vpc_id     = aws_vpc.foo.id
  subnet_ids = [aws_subnet.one.id]

  tags = {
    Name = "tf-acc-acl-subnet-ids"
  }
}
`

const testAccAWSNetworkAclEsp = `
resource "aws_vpc" "testvpc" {
  cidr_block = "10.1.0.0/16"

  tags = {
    Name = "terraform-testacc-network-acl-esp"
  }
}

resource "aws_network_acl" "test" {
  vpc_id = aws_vpc.testvpc.id

  egress {
    protocol   = "esp"
    rule_no    = 5
    action     = "allow"
    cidr_block = "10.3.0.0/18"
    from_port  = 0
    to_port    = 0
  }

  tags = {
    Name = "tf-acc-acl-esp"
  }
}
`

func testAccAWSNetworkAclConfigEgressConfigModeBlocks() string {
	return `
resource "aws_vpc" "test" {
  cidr_block = "10.0.0.0/16"

  tags = {
    Name = "terraform-testacc-network-acl-egress-computed-attribute-mode"
  }
}

resource "aws_network_acl" "test" {
  tags = {
    Name = "terraform-testacc-network-acl-egress-computed-attribute-mode"
  }

  vpc_id = aws_vpc.test.id

  egress {
    action     = "allow"
    cidr_block = aws_vpc.test.cidr_block
    from_port  = 0
    protocol   = 6
    rule_no    = 1
    to_port    = 0
  }

  egress {
    action     = "allow"
    cidr_block = aws_vpc.test.cidr_block
    from_port  = 0
    protocol   = "udp"
    rule_no    = 2
    to_port    = 0
  }
}
`
}

func testAccAWSNetworkAclConfigEgressConfigModeNoBlocks() string {
	return `
resource "aws_vpc" "test" {
  cidr_block = "10.0.0.0/16"

  tags = {
    Name = "terraform-testacc-network-acl-egress-computed-attribute-mode"
  }
}

resource "aws_network_acl" "test" {
  tags = {
    Name = "terraform-testacc-network-acl-egress-computed-attribute-mode"
  }

  vpc_id = aws_vpc.test.id
}
`
}

func testAccAWSNetworkAclConfigEgressConfigModeZeroed() string {
	return `
resource "aws_vpc" "test" {
  cidr_block = "10.0.0.0/16"

  tags = {
    Name = "terraform-testacc-network-acl-egress-computed-attribute-mode"
  }
}

resource "aws_network_acl" "test" {
  egress = []

  tags = {
    Name = "terraform-testacc-network-acl-egress-computed-attribute-mode"
  }

  vpc_id = aws_vpc.test.id
}
`
}

func testAccAWSNetworkAclConfigIngressConfigModeBlocks() string {
	return `
resource "aws_vpc" "test" {
  cidr_block = "10.0.0.0/16"

  tags = {
    Name = "terraform-testacc-network-acl-ingress-computed-attribute-mode"
  }
}

resource "aws_network_acl" "test" {
  tags = {
    Name = "terraform-testacc-network-acl-ingress-computed-attribute-mode"
  }

  vpc_id = aws_vpc.test.id

  ingress {
    action     = "allow"
    cidr_block = aws_vpc.test.cidr_block
    from_port  = 0
    protocol   = 6
    rule_no    = 1
    to_port    = 0
  }

  ingress {
    action     = "allow"
    cidr_block = aws_vpc.test.cidr_block
    from_port  = 0
    protocol   = "udp"
    rule_no    = 2
    to_port    = 0
  }
}
`
}

func testAccAWSNetworkAclConfigIngressConfigModeNoBlocks() string {
	return `
resource "aws_vpc" "test" {
  cidr_block = "10.0.0.0/16"

  tags = {
    Name = "terraform-testacc-network-acl-ingress-computed-attribute-mode"
  }
}

resource "aws_network_acl" "test" {
  tags = {
    Name = "terraform-testacc-network-acl-ingress-computed-attribute-mode"
  }

  vpc_id = aws_vpc.test.id
}
`
}

func testAccAWSNetworkAclConfigIngressConfigModeZeroed() string {
	return `
resource "aws_vpc" "test" {
  cidr_block = "10.0.0.0/16"

  tags = {
    Name = "terraform-testacc-network-acl-ingress-computed-attribute-mode"
  }
}

resource "aws_network_acl" "test" {
  ingress = []

  tags = {
    Name = "terraform-testacc-network-acl-ingress-computed-attribute-mode"
  }

  vpc_id = aws_vpc.test.id
}
`
}

func testAccAWSNetworkAclConfigTags1(rName, tagKey1, tagValue1 string) string {
	return fmt.Sprintf(`
resource "aws_vpc" "test" {
  cidr_block = "10.1.0.0/16"

  tags = {
    Name = %[1]q
  }
}

resource "aws_network_acl" "test" {
  vpc_id  = aws_vpc.test.id
  ingress = []

  tags = {
    %[2]q = %[3]q
  }
}
`, rName, tagKey1, tagValue1)
}

func testAccAWSNetworkAclConfigTags2(rName, tagKey1, tagValue1, tagKey2, tagValue2 string) string {
	return fmt.Sprintf(`
resource "aws_vpc" "test" {
  cidr_block = "10.1.0.0/16"

  tags = {
    Name = %[1]q
  }
}

resource "aws_network_acl" "test" {
  vpc_id  = aws_vpc.test.id
  ingress = []

  tags = {
    %[2]q = %[3]q
    %[4]q = %[5]q
  }
}
`, rName, tagKey1, tagValue1, tagKey2, tagValue2)
}
