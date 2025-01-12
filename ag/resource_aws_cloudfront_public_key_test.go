package ag

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudfront"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccAWSCloudFrontPublicKey_basic(t *testing.T) {
	rInt := acctest.RandInt()
	resourceName := "aws_cloudfront_public_key.example"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t); testAccPartitionHasServicePreCheck(cloudfront.EndpointsID, t) },
		ErrorCheck:   testAccErrorCheck(t, cloudfront.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCloudFrontPublicKeyDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSCloudFrontPublicKeyConfig(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudFrontPublicKeyExistence(resourceName),
					resource.TestCheckResourceAttr("aws_cloudfront_public_key.example", "comment", "test key"),
					resource.TestMatchResourceAttr("aws_cloudfront_public_key.example", "caller_reference", regexp.MustCompile(fmt.Sprintf("^%s", resource.UniqueIdPrefix))),
					resource.TestCheckResourceAttr("aws_cloudfront_public_key.example", "name", fmt.Sprintf("tf-acc-test-%d", rInt)),
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

func TestAccAWSCloudFrontPublicKey_disappears(t *testing.T) {
	rInt := acctest.RandInt()
	resourceName := "aws_cloudfront_public_key.example"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t); testAccPartitionHasServicePreCheck(cloudfront.EndpointsID, t) },
		ErrorCheck:   testAccErrorCheck(t, cloudfront.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCloudFrontPublicKeyDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSCloudFrontPublicKeyConfig(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudFrontPublicKeyExistence(resourceName),
					testAccCheckResourceDisappears(testAccProvider, resourceAwsCloudFrontPublicKey(), resourceName),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccAWSCloudFrontPublicKey_namePrefix(t *testing.T) {
	startsWithPrefix := regexp.MustCompile("^tf-acc-test-")
	resourceName := "aws_cloudfront_public_key.example"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t); testAccPartitionHasServicePreCheck(cloudfront.EndpointsID, t) },
		ErrorCheck:   testAccErrorCheck(t, cloudfront.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCloudFrontPublicKeyDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSCloudFrontPublicKeyConfig_namePrefix(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudFrontPublicKeyExistence(resourceName),
					resource.TestMatchResourceAttr("aws_cloudfront_public_key.example", "name", startsWithPrefix),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"name_prefix",
				},
			},
		},
	})
}

func TestAccAWSCloudFrontPublicKey_update(t *testing.T) {
	rInt := acctest.RandInt()
	resourceName := "aws_cloudfront_public_key.example"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t); testAccPartitionHasServicePreCheck(cloudfront.EndpointsID, t) },
		ErrorCheck:   testAccErrorCheck(t, cloudfront.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCloudFrontPublicKeyDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSCloudFrontPublicKeyConfig(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudFrontPublicKeyExistence(resourceName),
					resource.TestCheckResourceAttr("aws_cloudfront_public_key.example", "comment", "test key"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccAWSCloudFrontPublicKeyConfigUpdate(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudFrontPublicKeyExistence(resourceName),
					resource.TestCheckResourceAttr("aws_cloudfront_public_key.example", "comment", "test key1"),
				),
			},
		},
	})
}

func testAccCheckCloudFrontPublicKeyExistence(r string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[r]
		if !ok {
			return fmt.Errorf("Not found: %s", r)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No Id is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).cloudfrontconn

		params := &cloudfront.GetPublicKeyInput{
			Id: aws.String(rs.Primary.ID),
		}

		_, err := conn.GetPublicKey(params)
		if err != nil {
			return fmt.Errorf("Error retrieving CloudFront PublicKey: %s", err)
		}
		return nil
	}
}

func testAccCheckCloudFrontPublicKeyDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).cloudfrontconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_cloudfront_public_key" {
			continue
		}

		params := &cloudfront.GetPublicKeyInput{
			Id: aws.String(rs.Primary.ID),
		}

		_, err := conn.GetPublicKey(params)
		if isAWSErr(err, cloudfront.ErrCodeNoSuchPublicKey, "") {
			continue
		}
		if err != nil {
			return err
		}
		return fmt.Errorf("CloudFront PublicKey (%s) was not deleted", rs.Primary.ID)
	}

	return nil
}

func testAccAWSCloudFrontPublicKeyConfig(rInt int) string {
	return fmt.Sprintf(`
resource "aws_cloudfront_public_key" "example" {
  comment     = "test key"
  encoded_key = file("test-fixtures/cloudfront-public-key.pem")
  name        = "tf-acc-test-%d"
}
`, rInt)
}

func testAccAWSCloudFrontPublicKeyConfig_namePrefix() string {
	return `
resource "aws_cloudfront_public_key" "example" {
  comment     = "test key"
  encoded_key = file("test-fixtures/cloudfront-public-key.pem")
  name_prefix = "tf-acc-test-"
}
`
}

func testAccAWSCloudFrontPublicKeyConfigUpdate(rInt int) string {
	return fmt.Sprintf(`
resource "aws_cloudfront_public_key" "example" {
  comment     = "test key1"
  encoded_key = file("test-fixtures/cloudfront-public-key.pem")
  name        = "tf-acc-test-%d"
}
`, rInt)
}
