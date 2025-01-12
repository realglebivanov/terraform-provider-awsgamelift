package ag

import (
	"fmt"
	"log"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/configservice"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func init() {
	resource.AddTestSweepers("aws_config_aggregate_authorization", &resource.Sweeper{
		Name: "aws_config_aggregate_authorization",
		F:    testSweepConfigAggregateAuthorizations,
	})
}

func testSweepConfigAggregateAuthorizations(region string) error {
	client, err := sharedClientForRegion(region)
	if err != nil {
		return fmt.Errorf("Error getting client: %s", err)
	}
	conn := client.(*AWSClient).configconn

	aggregateAuthorizations, err := describeConfigAggregateAuthorizations(conn)
	if err != nil {
		if testSweepSkipSweepError(err) {
			log.Printf("[WARN] Skipping Config Aggregate Authorizations sweep for %s: %s", region, err)
			return nil
		}
		return fmt.Errorf("Error retrieving config aggregate authorizations: %s", err)
	}

	if len(aggregateAuthorizations) == 0 {
		log.Print("[DEBUG] No config aggregate authorizations to sweep")
		return nil
	}

	log.Printf("[INFO] Found %d config aggregate authorizations", len(aggregateAuthorizations))

	for _, auth := range aggregateAuthorizations {
		log.Printf("[INFO] Deleting config authorization %s", *auth.AggregationAuthorizationArn)
		_, err := conn.DeleteAggregationAuthorization(&configservice.DeleteAggregationAuthorizationInput{
			AuthorizedAccountId: auth.AuthorizedAccountId,
			AuthorizedAwsRegion: auth.AuthorizedAwsRegion,
		})
		if err != nil {
			return fmt.Errorf("Error deleting config aggregate authorization %s: %s", *auth.AggregationAuthorizationArn, err)
		}
	}

	return nil
}

func TestAccAWSConfigAggregateAuthorization_basic(t *testing.T) {
	rString := acctest.RandStringFromCharSet(12, "0123456789")
	resourceName := "aws_config_aggregate_authorization.example"
	dataSourceName := "data.aws_region.current"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, configservice.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSConfigAggregateAuthorizationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSConfigAggregateAuthorizationConfig_basic(rString),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "account_id", rString),
					resource.TestCheckResourceAttrPair(resourceName, "region", dataSourceName, "name"),
					testAccMatchResourceAttrRegionalARN(resourceName, "arn", "config", regexp.MustCompile(fmt.Sprintf(`aggregation-authorization/%s/%s$`, rString, testAccGetRegion()))),
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

func TestAccAWSConfigAggregateAuthorization_tags(t *testing.T) {
	rString := acctest.RandStringFromCharSet(12, "0123456789")
	resourceName := "aws_config_aggregate_authorization.example"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, configservice.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSConfigAggregateAuthorizationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSConfigAggregateAuthorizationConfig_tags(rString, "foo", "bar", "fizz", "buzz"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "tags.%", "3"),
					resource.TestCheckResourceAttr(resourceName, "tags.Name", rString),
					resource.TestCheckResourceAttr(resourceName, "tags.foo", "bar"),
					resource.TestCheckResourceAttr(resourceName, "tags.fizz", "buzz"),
				),
			},
			{
				Config: testAccAWSConfigAggregateAuthorizationConfig_tags(rString, "foo", "bar2", "fizz2", "buzz2"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "tags.%", "3"),
					resource.TestCheckResourceAttr(resourceName, "tags.Name", rString),
					resource.TestCheckResourceAttr(resourceName, "tags.foo", "bar2"),
					resource.TestCheckResourceAttr(resourceName, "tags.fizz2", "buzz2"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccAWSConfigAggregateAuthorizationConfig_basic(rString),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "tags.%", "0"),
				),
			},
		},
	})
}

func testAccCheckAWSConfigAggregateAuthorizationDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).configconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_config_aggregate_authorization" {
			continue
		}

		accountId, region, err := resourceAwsConfigAggregateAuthorizationParseID(rs.Primary.ID)
		if err != nil {
			return err
		}

		aggregateAuthorizations, err := describeConfigAggregateAuthorizations(conn)

		if err != nil {
			return err
		}

		for _, auth := range aggregateAuthorizations {
			if accountId == aws.StringValue(auth.AuthorizedAccountId) && region == aws.StringValue(auth.AuthorizedAwsRegion) {
				return fmt.Errorf("Config aggregate authorization still exists: %s", rs.Primary.ID)
			}
		}
	}

	return nil
}

func testAccAWSConfigAggregateAuthorizationConfig_basic(rString string) string {
	return fmt.Sprintf(`
data "aws_region" "current" {}

resource "aws_config_aggregate_authorization" "example" {
  account_id = %[1]q
  region     = data.aws_region.current.name
}
`, rString)
}

func testAccAWSConfigAggregateAuthorizationConfig_tags(rString, tagKey1, tagValue1, tagKey2, tagValue2 string) string {
	return fmt.Sprintf(`
data "aws_region" "current" {}

resource "aws_config_aggregate_authorization" "example" {
  account_id = %[1]q
  region     = data.aws_region.current.name

  tags = {
    Name = %[1]q

    %[2]s = %[3]q
    %[4]s = %[5]q
  }
}
`, rString, tagKey1, tagValue1, tagKey2, tagValue2)
}
