package ag

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go/service/directconnect"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceAwsDxGateway_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "aws_dx_gateway.test"
	datasourceName := "data.aws_dx_gateway.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:   func() { testAccPreCheck(t) },
		ErrorCheck: testAccErrorCheck(t, directconnect.EndpointsID),
		Providers:  testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      testAccDataSourceAwsDxGatewayConfig_NonExistent,
				ExpectError: regexp.MustCompile(`Direct Connect Gateway not found`),
			},
			{
				Config: testAccDataSourceAwsDxGatewayConfig_Name(rName, acctest.RandIntRange(64512, 65534)),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair(datasourceName, "amazon_side_asn", resourceName, "amazon_side_asn"),
					resource.TestCheckResourceAttrPair(datasourceName, "id", resourceName, "id"),
					resource.TestCheckResourceAttrPair(datasourceName, "name", resourceName, "name"),
					resource.TestCheckResourceAttrPair(datasourceName, "owner_account_id", resourceName, "owner_account_id"),
				),
			},
		},
	})
}

func testAccDataSourceAwsDxGatewayConfig_Name(rName string, rBgpAsn int) string {
	return fmt.Sprintf(`
resource "aws_dx_gateway" "wrong" {
  amazon_side_asn = "%d"
  name            = "%s-wrong"
}

resource "aws_dx_gateway" "test" {
  amazon_side_asn = "%d"
  name            = "%s"
}

data "aws_dx_gateway" "test" {
  name = aws_dx_gateway.test.name
}
`, rBgpAsn+1, rName, rBgpAsn, rName)
}

const testAccDataSourceAwsDxGatewayConfig_NonExistent = `
data "aws_dx_gateway" "test" {
  name = "tf-acc-test-does-not-exist"
}
`
