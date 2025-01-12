package ag

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ram"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccAwsRamResourceShareAccepter_basic(t *testing.T) {
	var providers []*schema.Provider
	resourceName := "aws_ram_resource_share_accepter.test"
	principalAssociationResourceName := "aws_ram_principal_association.test"
	rName := acctest.RandomWithPrefix("tf-acc-test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccAlternateAccountPreCheck(t)
		},
		ErrorCheck:        testAccErrorCheck(t, ram.EndpointsID),
		ProviderFactories: testAccProviderFactoriesAlternate(&providers),
		CheckDestroy:      testAccCheckAwsRamResourceShareAccepterDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAwsRamResourceShareAccepterBasic(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsRamResourceShareAccepterExists(resourceName),
					resource.TestCheckResourceAttrPair(resourceName, "share_arn", principalAssociationResourceName, "resource_share_arn"),
					testAccMatchResourceAttrRegionalARNAccountID(resourceName, "invitation_arn", "ram", `\d{12}`, regexp.MustCompile(fmt.Sprintf("resource-share-invitation/%s$", uuidRegexPattern))),
					resource.TestMatchResourceAttr(resourceName, "share_id", regexp.MustCompile(fmt.Sprintf(`^rs-%s$`, uuidRegexPattern))),
					resource.TestCheckResourceAttr(resourceName, "status", ram.ResourceShareStatusActive),
					testAccCheckResourceAttrAccountID(resourceName, "receiver_account_id"),
					resource.TestMatchResourceAttr(resourceName, "sender_account_id", regexp.MustCompile(`\d{12}`)),
					resource.TestCheckResourceAttr(resourceName, "share_name", rName),
					resource.TestCheckResourceAttr(resourceName, "resources.%", "0"),
				),
			},
			{
				Config:            testAccAwsRamResourceShareAccepterBasic(rName),
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAwsRamResourceShareAccepter_disappears(t *testing.T) {
	var providers []*schema.Provider
	resourceName := "aws_ram_resource_share_accepter.test"
	rName := acctest.RandomWithPrefix("tf-acc-test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccAlternateAccountPreCheck(t)
		},
		ErrorCheck:        testAccErrorCheck(t, ram.EndpointsID),
		ProviderFactories: testAccProviderFactoriesAlternate(&providers),
		CheckDestroy:      testAccCheckAwsRamResourceShareAccepterDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAwsRamResourceShareAccepterBasic(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsRamResourceShareAccepterExists(resourceName),
					testAccCheckResourceDisappears(testAccProvider, resourceAwsRamResourceShareAccepter(), resourceName),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccCheckAwsRamResourceShareAccepterDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).ramconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_ram_resource_share_accepter" {
			continue
		}

		input := &ram.GetResourceSharesInput{
			ResourceShareArns: []*string{aws.String(rs.Primary.Attributes["share_arn"])},
			ResourceOwner:     aws.String(ram.ResourceOwnerOtherAccounts),
		}

		output, err := conn.GetResourceShares(input)
		if err != nil {
			if isAWSErr(err, ram.ErrCodeUnknownResourceException, "") {
				return nil
			}
			return fmt.Errorf("Error deleting RAM resource share: %s", err)
		}

		if len(output.ResourceShares) > 0 && aws.StringValue(output.ResourceShares[0].Status) != ram.ResourceShareStatusDeleted {
			return fmt.Errorf("RAM resource share invitation found, should be destroyed")
		}
	}

	return nil
}

func testAccCheckAwsRamResourceShareAccepterExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]

		if !ok || rs.Type != "aws_ram_resource_share_accepter" {
			return fmt.Errorf("RAM resource share invitation not found: %s", name)
		}

		conn := testAccProvider.Meta().(*AWSClient).ramconn

		input := &ram.GetResourceSharesInput{
			ResourceShareArns: []*string{aws.String(rs.Primary.Attributes["share_arn"])},
			ResourceOwner:     aws.String(ram.ResourceOwnerOtherAccounts),
		}

		output, err := conn.GetResourceShares(input)
		if err != nil || len(output.ResourceShares) == 0 {
			return fmt.Errorf("Error finding RAM resource share: %s", err)
		}

		return nil
	}
}

func testAccAwsRamResourceShareAccepterBasic(rName string) string {
	return testAccAlternateAccountProviderConfig() + fmt.Sprintf(`
resource "aws_ram_resource_share_accepter" "test" {
  share_arn = aws_ram_principal_association.test.resource_share_arn
}

resource "aws_ram_resource_share" "test" {
  provider = "awsalternate"

  name                      = %[1]q
  allow_external_principals = true

  tags = {
    Name = %[1]q
  }
}

resource "aws_ram_principal_association" "test" {
  provider = "awsalternate"

  principal          = data.aws_caller_identity.receiver.account_id
  resource_share_arn = aws_ram_resource_share.test.arn
}

data "aws_caller_identity" "receiver" {}
`, rName)
}
