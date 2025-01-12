package ag

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go/service/elasticbeanstalk"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccAWSDataSourceElasticBeanstalkHostedZone_basic(t *testing.T) {
	dataSourceName := "data.aws_elastic_beanstalk_hosted_zone.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:   func() { testAccPreCheck(t) },
		ErrorCheck: testAccErrorCheck(t, elasticbeanstalk.EndpointsID),
		Providers:  testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckAwsElasticBeanstalkHostedZoneDataSource_currentRegion,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsElasticBeanstalkHostedZone(dataSourceName, testAccGetRegion()),
				),
			},
		},
	})
}

func TestAccAWSDataSourceElasticBeanstalkHostedZone_Region(t *testing.T) {
	dataSourceName := "data.aws_elastic_beanstalk_hosted_zone.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:   func() { testAccPreCheck(t) },
		ErrorCheck: testAccErrorCheck(t, elasticbeanstalk.EndpointsID),
		Providers:  testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckAwsElasticBeanstalkHostedZoneDataSource_byRegion("ap-southeast-2"), //lintignore:AWSAT003 // passes in GovCloud
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsElasticBeanstalkHostedZone(dataSourceName, "ap-southeast-2"), //lintignore:AWSAT003 // passes in GovCloud
				),
			},
			{
				Config: testAccCheckAwsElasticBeanstalkHostedZoneDataSource_byRegion("eu-west-1"), //lintignore:AWSAT003 // passes in GovCloud
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsElasticBeanstalkHostedZone(dataSourceName, "eu-west-1"), //lintignore:AWSAT003 // passes in GovCloud
				),
			},
			{
				Config:      testAccCheckAwsElasticBeanstalkHostedZoneDataSource_byRegion("ss-pluto-1"),
				ExpectError: regexp.MustCompile("Unsupported region"),
			},
		},
	})
}

func testAccCheckAwsElasticBeanstalkHostedZone(resourceName string, region string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		expectedValue, ok := elasticBeanstalkHostedZoneIds[region]

		if !ok {
			return fmt.Errorf("Unsupported region: %s", region)
		}

		return resource.TestCheckResourceAttr(resourceName, "id", expectedValue)(s)
	}
}

const testAccCheckAwsElasticBeanstalkHostedZoneDataSource_currentRegion = `
data "aws_elastic_beanstalk_hosted_zone" "test" {}
`

func testAccCheckAwsElasticBeanstalkHostedZoneDataSource_byRegion(r string) string {
	return fmt.Sprintf(`
data "aws_elastic_beanstalk_hosted_zone" "test" {
  region = "%s"
}
`, r)
}
