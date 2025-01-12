package ag

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceAwsVpc_basic(t *testing.T) {
	rand.Seed(time.Now().UTC().UnixNano())
	rInt := rand.Intn(254)
	cidr := fmt.Sprintf("10.%d.0.0/16", rInt+1) // Prevent common 10.0.0.0/16 cidr_block matches
	tag := fmt.Sprintf("terraform-testacc-vpc-data-source-basic-%d", rInt)

	vpcResourceName := "aws_vpc.test"
	ds1ResourceName := "data.aws_vpc.by_id"
	ds2ResourceName := "data.aws_vpc.by_cidr"
	ds3ResourceName := "data.aws_vpc.by_tag"
	ds4ResourceName := "data.aws_vpc.by_filter"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:   func() { testAccPreCheck(t) },
		ErrorCheck: testAccErrorCheck(t, ec2.EndpointsID),
		Providers:  testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceAwsVpcConfig(cidr, tag),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						ds1ResourceName, "id", vpcResourceName, "id"),
					resource.TestCheckResourceAttrPair(
						ds1ResourceName, "owner_id", vpcResourceName, "owner_id"),
					resource.TestCheckResourceAttr(
						ds1ResourceName, "cidr_block", cidr),
					resource.TestCheckResourceAttr(
						ds1ResourceName, "tags.Name", tag),
					resource.TestCheckResourceAttr(
						ds1ResourceName, "enable_dns_support", "true"),
					resource.TestCheckResourceAttr(
						ds1ResourceName, "enable_dns_hostnames", "false"),
					resource.TestCheckResourceAttrSet(
						ds1ResourceName, "arn"),
					resource.TestCheckResourceAttrPair(
						ds1ResourceName, "main_route_table_id", vpcResourceName, "main_route_table_id"),

					resource.TestCheckResourceAttrPair(
						ds2ResourceName, "id", vpcResourceName, "id"),
					resource.TestCheckResourceAttrPair(
						ds2ResourceName, "owner_id", vpcResourceName, "owner_id"),
					resource.TestCheckResourceAttr(
						ds2ResourceName, "cidr_block", cidr),
					resource.TestCheckResourceAttr(
						ds2ResourceName, "tags.Name", tag),

					resource.TestCheckResourceAttrPair(
						ds3ResourceName, "id", vpcResourceName, "id"),
					resource.TestCheckResourceAttrPair(
						ds3ResourceName, "owner_id", vpcResourceName, "owner_id"),
					resource.TestCheckResourceAttr(
						ds3ResourceName, "cidr_block", cidr),
					resource.TestCheckResourceAttr(
						ds3ResourceName, "tags.Name", tag),

					resource.TestCheckResourceAttrPair(
						ds4ResourceName, "id", vpcResourceName, "id"),
					resource.TestCheckResourceAttrPair(
						ds4ResourceName, "owner_id", vpcResourceName, "owner_id"),
					resource.TestCheckResourceAttr(
						ds4ResourceName, "cidr_block", cidr),
					resource.TestCheckResourceAttr(
						ds4ResourceName, "tags.Name", tag),
				),
			},
		},
	})
}

func TestAccDataSourceAwsVpc_ipv6Associated(t *testing.T) {
	rand.Seed(time.Now().UTC().UnixNano())
	rInt := rand.Intn(255)
	cidr := fmt.Sprintf("10.%d.0.0/16", rInt)
	tag := fmt.Sprintf("terraform-testacc-vpc-data-source-ipv6-associated-%d", rInt)

	vpcResourceName := "aws_vpc.test"
	ds1ResourceName := "data.aws_vpc.by_id"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:   func() { testAccPreCheck(t) },
		ErrorCheck: testAccErrorCheck(t, ec2.EndpointsID),
		Providers:  testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceAwsVpcConfigIpv6(cidr, tag),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						ds1ResourceName, "id", vpcResourceName, "id"),
					resource.TestCheckResourceAttrPair(
						ds1ResourceName, "owner_id", vpcResourceName, "owner_id"),
					resource.TestCheckResourceAttr(
						ds1ResourceName, "cidr_block", cidr),
					resource.TestCheckResourceAttr(
						ds1ResourceName, "tags.Name", tag),
					resource.TestCheckResourceAttrSet(
						"data.aws_vpc.by_id", "ipv6_association_id"),
					resource.TestCheckResourceAttrSet(
						"data.aws_vpc.by_id", "ipv6_cidr_block"),
				),
			},
		},
	})
}

func TestAccDataSourceAwsVpc_CidrBlockAssociations_Multiple(t *testing.T) {
	dataSourceName := "data.aws_vpc.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, ec2.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVpcDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceAwsVpcConfigCidrBlockAssociationsMultiple(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "cidr_block_associations.#", "2"),
				),
			},
		},
	})
}

func testAccDataSourceAwsVpcConfigIpv6(cidr, tag string) string {
	return fmt.Sprintf(`
resource "aws_vpc" "test" {
  cidr_block                       = "%s"
  assign_generated_ipv6_cidr_block = true

  tags = {
    Name = "%s"
  }
}

data "aws_vpc" "by_id" {
  id = aws_vpc.test.id
}
`, cidr, tag)
}

func testAccDataSourceAwsVpcConfig(cidr, tag string) string {
	return fmt.Sprintf(`
resource "aws_vpc" "test" {
  cidr_block = "%s"

  tags = {
    Name = "%s"
  }
}

data "aws_vpc" "by_id" {
  id = aws_vpc.test.id
}

data "aws_vpc" "by_cidr" {
  cidr_block = aws_vpc.test.cidr_block
}

data "aws_vpc" "by_tag" {
  tags = {
    Name = aws_vpc.test.tags["Name"]
  }
}

data "aws_vpc" "by_filter" {
  filter {
    name   = "vpc-id"
    values = [aws_vpc.test.id]
  }
}
`, cidr, tag)
}

func testAccDataSourceAwsVpcConfigCidrBlockAssociationsMultiple() string {
	return `
resource "aws_vpc" "test" {
  cidr_block = "10.0.0.0/16"
}

resource "aws_vpc_ipv4_cidr_block_association" "test" {
  vpc_id     = aws_vpc.test.id
  cidr_block = "172.0.0.0/16"
}

data "aws_vpc" "test" {
  id = aws_vpc_ipv4_cidr_block_association.test.vpc_id
}
`
}
