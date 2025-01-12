package ag

import (
	"fmt"
	"log"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/configservice"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func init() {
	resource.AddTestSweepers("aws_config_configuration_recorder", &resource.Sweeper{
		Name: "aws_config_configuration_recorder",
		F:    testSweepConfigConfigurationRecorder,
	})
}

func testSweepConfigConfigurationRecorder(region string) error {
	client, err := sharedClientForRegion(region)
	if err != nil {
		return fmt.Errorf("error getting client: %s", err)
	}
	conn := client.(*AWSClient).configconn

	req := &configservice.DescribeConfigurationRecordersInput{}
	resp, err := conn.DescribeConfigurationRecorders(req)
	if err != nil {
		if testSweepSkipSweepError(err) {
			log.Printf("[WARN] Skipping Config Configuration Recorders sweep for %s: %s", region, err)
			return nil
		}
		return fmt.Errorf("Error describing Configuration Recorders: %s", err)
	}

	if len(resp.ConfigurationRecorders) == 0 {
		log.Print("[DEBUG] No AWS Config Configuration Recorder to sweep")
		return nil
	}

	for _, cr := range resp.ConfigurationRecorders {
		_, err := conn.StopConfigurationRecorder(&configservice.StopConfigurationRecorderInput{
			ConfigurationRecorderName: cr.Name,
		})
		if err != nil {
			return err
		}

		_, err = conn.DeleteConfigurationRecorder(&configservice.DeleteConfigurationRecorderInput{
			ConfigurationRecorderName: cr.Name,
		})
		if err != nil {
			return fmt.Errorf(
				"Error deleting Configuration Recorder (%s): %s",
				*cr.Name, err)
		}
	}

	return nil
}

func testAccConfigConfigurationRecorder_basic(t *testing.T) {
	var cr configservice.ConfigurationRecorder
	rInt := acctest.RandInt()
	expectedName := fmt.Sprintf("tf-acc-test-%d", rInt)
	expectedRoleName := fmt.Sprintf("tf-acc-test-awsconfig-%d", rInt)

	resourceName := "aws_config_configuration_recorder.foo"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, configservice.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckConfigConfigurationRecorderDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccConfigConfigurationRecorderConfig_basic(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckConfigConfigurationRecorderExists(resourceName, &cr),
					testAccCheckConfigConfigurationRecorderName(resourceName, expectedName, &cr),
					testAccCheckResourceAttrGlobalARN(resourceName, "role_arn", "iam", fmt.Sprintf("role/%s", expectedRoleName)),
					resource.TestCheckResourceAttr(resourceName, "name", expectedName),
				),
			},
		},
	})
}

func testAccConfigConfigurationRecorder_allParams(t *testing.T) {
	var cr configservice.ConfigurationRecorder
	rInt := acctest.RandInt()
	expectedName := fmt.Sprintf("tf-acc-test-%d", rInt)
	expectedRoleName := fmt.Sprintf("tf-acc-test-awsconfig-%d", rInt)

	resourceName := "aws_config_configuration_recorder.foo"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, configservice.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckConfigConfigurationRecorderDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccConfigConfigurationRecorderConfig_allParams(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckConfigConfigurationRecorderExists(resourceName, &cr),
					testAccCheckConfigConfigurationRecorderName(resourceName, expectedName, &cr),
					testAccCheckResourceAttrGlobalARN(resourceName, "role_arn", "iam", fmt.Sprintf("role/%s", expectedRoleName)),
					resource.TestCheckResourceAttr(resourceName, "name", expectedName),
					resource.TestCheckResourceAttr(resourceName, "recording_group.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "recording_group.0.all_supported", "false"),
					resource.TestCheckResourceAttr(resourceName, "recording_group.0.include_global_resource_types", "false"),
					resource.TestCheckResourceAttr(resourceName, "recording_group.0.resource_types.#", "2"),
				),
			},
		},
	})
}

func testAccConfigConfigurationRecorder_importBasic(t *testing.T) {
	resourceName := "aws_config_configuration_recorder.foo"
	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, configservice.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckConfigConfigurationRecorderDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccConfigConfigurationRecorderConfig_basic(rInt),
			},

			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckConfigConfigurationRecorderName(n string, desired string, obj *configservice.ConfigurationRecorder) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		_, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if *obj.Name != desired {
			return fmt.Errorf("Expected configuration recorder %q name to be %q, given: %q",
				n, desired, *obj.Name)
		}

		return nil
	}
}

func testAccCheckConfigConfigurationRecorderExists(n string, obj *configservice.ConfigurationRecorder) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not Found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No configuration recorder ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).configconn
		out, err := conn.DescribeConfigurationRecorders(&configservice.DescribeConfigurationRecordersInput{
			ConfigurationRecorderNames: []*string{aws.String(rs.Primary.Attributes["name"])},
		})
		if err != nil {
			return fmt.Errorf("Failed to describe configuration recorder: %s", err)
		}
		if len(out.ConfigurationRecorders) < 1 {
			return fmt.Errorf("No configuration recorder found when describing %q", rs.Primary.Attributes["name"])
		}

		cr := out.ConfigurationRecorders[0]
		*obj = *cr

		return nil
	}
}

func testAccCheckConfigConfigurationRecorderDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).configconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_config_configuration_recorder_status" {
			continue
		}

		resp, err := conn.DescribeConfigurationRecorders(&configservice.DescribeConfigurationRecordersInput{
			ConfigurationRecorderNames: []*string{aws.String(rs.Primary.Attributes["name"])},
		})

		if err == nil {
			if len(resp.ConfigurationRecorders) != 0 &&
				*resp.ConfigurationRecorders[0].Name == rs.Primary.Attributes["name"] {
				return fmt.Errorf("Configuration recorder still exists: %s", rs.Primary.Attributes["name"])
			}
		}
	}

	return nil
}

func testAccConfigConfigurationRecorderConfig_basic(randInt int) string {
	return fmt.Sprintf(`
resource "aws_config_configuration_recorder" "foo" {
  name     = "tf-acc-test-%d"
  role_arn = aws_iam_role.r.arn
}

resource "aws_iam_role" "r" {
  name = "tf-acc-test-awsconfig-%d"

  assume_role_policy = <<POLICY
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "config.amazonaws.com"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
POLICY
}

resource "aws_iam_role_policy" "p" {
  name = "tf-acc-test-awsconfig-%d"
  role = aws_iam_role.r.id

  policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": [
        "s3:*"
      ],
      "Effect": "Allow",
      "Resource": [
        "${aws_s3_bucket.b.arn}",
        "${aws_s3_bucket.b.arn}/*"
      ]
    }
  ]
}
EOF
}

resource "aws_s3_bucket" "b" {
  bucket        = "tf-acc-test-awsconfig-%d"
  force_destroy = true
}

resource "aws_config_delivery_channel" "foo" {
  name           = "tf-acc-test-awsconfig-%d"
  s3_bucket_name = aws_s3_bucket.b.bucket
  depends_on     = [aws_config_configuration_recorder.foo]
}
`, randInt, randInt, randInt, randInt, randInt)
}

func testAccConfigConfigurationRecorderConfig_allParams(randInt int) string {
	return fmt.Sprintf(`
resource "aws_config_configuration_recorder" "foo" {
  name     = "tf-acc-test-%d"
  role_arn = aws_iam_role.r.arn

  recording_group {
    all_supported                 = false
    include_global_resource_types = false
    resource_types                = ["AWS::EC2::Instance", "AWS::CloudTrail::Trail"]
  }
}

resource "aws_iam_role" "r" {
  name = "tf-acc-test-awsconfig-%d"

  assume_role_policy = <<POLICY
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "config.amazonaws.com"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
POLICY
}

resource "aws_iam_role_policy" "p" {
  name = "tf-acc-test-awsconfig-%d"
  role = aws_iam_role.r.id

  policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": [
        "s3:*"
      ],
      "Effect": "Allow",
      "Resource": [
        "${aws_s3_bucket.b.arn}",
        "${aws_s3_bucket.b.arn}/*"
      ]
    }
  ]
}
EOF
}

resource "aws_s3_bucket" "b" {
  bucket        = "tf-acc-test-awsconfig-%d"
  force_destroy = true
}

resource "aws_config_delivery_channel" "foo" {
  name           = "tf-acc-test-awsconfig-%d"
  s3_bucket_name = aws_s3_bucket.b.bucket
  depends_on     = [aws_config_configuration_recorder.foo]
}
`, randInt, randInt, randInt, randInt, randInt)
}
