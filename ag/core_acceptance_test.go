package ag

import (
	"testing"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccAWSVpc_coreMismatchedDiffs(t *testing.T) {
	var vpc ec2.Vpc

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, ec2.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVpcDestroy,
		Steps: []resource.TestStep{
			{
				Config: testMatchedDiffs,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVpcExists("aws_vpc.test", &vpc),
					testAccCheckVpcCidr(&vpc, "10.0.0.0/16"),
					resource.TestCheckResourceAttr(
						"aws_vpc.test", "cidr_block", "10.0.0.0/16"),
				),
			},
		},
	})
}

const testMatchedDiffs = `
resource "aws_vpc" "test" {
  cidr_block = "10.0.0.0/16"

  tags = {
    Name = "terraform-testacc-repro-GH-4965"
  }

  lifecycle {
    ignore_changes = ["tags"]
  }
}`
