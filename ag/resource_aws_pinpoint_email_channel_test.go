package ag

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/pinpoint"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccAWSPinpointEmailChannel_basic(t *testing.T) {
	var channel pinpoint.EmailChannelResponse
	resourceName := "aws_pinpoint_email_channel.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t); testAccPreCheckAWSPinpointApp(t) },
		ErrorCheck:   testAccErrorCheck(t, pinpoint.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSPinpointEmailChannelDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSPinpointEmailChannelConfig_FromAddress("user@example.com"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSPinpointEmailChannelExists(resourceName, &channel),
					resource.TestCheckResourceAttr(resourceName, "enabled", "false"),
					resource.TestCheckResourceAttrSet(resourceName, "messages_per_second"),
					resource.TestCheckResourceAttrPair(resourceName, "role_arn", "aws_iam_role.test", "arn"),
					resource.TestCheckResourceAttrPair(resourceName, "identity", "aws_ses_domain_identity.test", "arn"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccAWSPinpointEmailChannelConfig_FromAddress("userupdated@example.com"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSPinpointEmailChannelExists(resourceName, &channel),
					resource.TestCheckResourceAttr(resourceName, "enabled", "false"),
					resource.TestCheckResourceAttrSet(resourceName, "messages_per_second"),
				),
			},
		},
	})
}

func TestAccAWSPinpointEmailChannel_configurationSet(t *testing.T) {
	var channel pinpoint.EmailChannelResponse
	resourceName := "aws_pinpoint_email_channel.test"
	rName := acctest.RandomWithPrefix("tf-acc-test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t); testAccPreCheckAWSPinpointApp(t) },
		ErrorCheck:   testAccErrorCheck(t, pinpoint.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSPinpointEmailChannelDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSPinpointEmailChannelConfigConfigurationSet("user@example.com", rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSPinpointEmailChannelExists(resourceName, &channel),
					resource.TestCheckResourceAttrPair(resourceName, "configuration_set", "aws_ses_configuration_set.test", "arn"),
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

func TestAccAWSPinpointEmailChannel_noRole(t *testing.T) {
	var channel pinpoint.EmailChannelResponse
	resourceName := "aws_pinpoint_email_channel.test"
	rName := acctest.RandomWithPrefix("tf-acc-test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t); testAccPreCheckAWSPinpointApp(t) },
		ErrorCheck:   testAccErrorCheck(t, pinpoint.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSPinpointEmailChannelDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSPinpointEmailChannelConfigNoRole("user@example.com", rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSPinpointEmailChannelExists(resourceName, &channel),
					resource.TestCheckResourceAttrPair(resourceName, "configuration_set", "aws_ses_configuration_set.test", "arn"),
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

func TestAccAWSPinpointEmailChannel_disappears(t *testing.T) {
	var channel pinpoint.EmailChannelResponse
	resourceName := "aws_pinpoint_email_channel.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t); testAccPreCheckAWSPinpointApp(t) },
		ErrorCheck:   testAccErrorCheck(t, pinpoint.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSPinpointEmailChannelDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSPinpointEmailChannelConfig_FromAddress("user@example.com"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSPinpointEmailChannelExists(resourceName, &channel),
					testAccCheckResourceDisappears(testAccProvider, resourceAwsPinpointEmailChannel(), resourceName),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccCheckAWSPinpointEmailChannelExists(n string, channel *pinpoint.EmailChannelResponse) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Pinpoint Email Channel with that application ID exists")
		}

		conn := testAccProvider.Meta().(*AWSClient).pinpointconn

		// Check if the app exists
		params := &pinpoint.GetEmailChannelInput{
			ApplicationId: aws.String(rs.Primary.ID),
		}
		output, err := conn.GetEmailChannel(params)

		if err != nil {
			return err
		}

		*channel = *output.EmailChannelResponse

		return nil
	}
}

func testAccAWSPinpointEmailChannelConfig_FromAddress(fromAddress string) string {
	return fmt.Sprintf(`
resource "aws_pinpoint_app" "test" {}

resource "aws_pinpoint_email_channel" "test" {
  application_id = aws_pinpoint_app.test.application_id
  enabled        = "false"
  from_address   = %[1]q
  identity       = aws_ses_domain_identity.test.arn
  role_arn       = aws_iam_role.test.arn
}

resource "aws_ses_domain_identity" "test" {
  domain = "example.com"
}

resource "aws_iam_role" "test" {
  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "pinpoint.amazonaws.com"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
EOF
}

resource "aws_iam_role_policy" "test" {
  name = "test"
  role = aws_iam_role.test.id

  policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": {
    "Action": [
      "mobileanalytics:PutEvents",
      "mobileanalytics:PutItems"
    ],
    "Effect": "Allow",
    "Resource": [
      "*"
    ]
  }
}
EOF
}
`, fromAddress)
}

func testAccAWSPinpointEmailChannelConfigConfigurationSet(fromAddress, rName string) string {
	return fmt.Sprintf(`
resource "aws_pinpoint_app" "test" {}

resource "aws_ses_configuration_set" "test" {
  name = %[2]q
}

resource "aws_pinpoint_email_channel" "test" {
  application_id    = aws_pinpoint_app.test.application_id
  enabled           = "false"
  from_address      = %[1]q
  identity          = aws_ses_domain_identity.test.arn
  role_arn          = aws_iam_role.test.arn
  configuration_set = aws_ses_configuration_set.test.arn
}

resource "aws_ses_domain_identity" "test" {
  domain = "example.com"
}

resource "aws_iam_role" "test" {
  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "pinpoint.amazonaws.com"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
EOF
}

resource "aws_iam_role_policy" "test" {
  name = "test"
  role = aws_iam_role.test.id

  policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": {
    "Action": [
      "mobileanalytics:PutEvents",
      "mobileanalytics:PutItems"
    ],
    "Effect": "Allow",
    "Resource": [
      "*"
    ]
  }
}
EOF
}
`, fromAddress, rName)
}

func testAccAWSPinpointEmailChannelConfigNoRole(fromAddress, rName string) string {
	return fmt.Sprintf(`
resource "aws_pinpoint_app" "test" {}

resource "aws_ses_configuration_set" "test" {
  name = %[2]q
}

resource "aws_pinpoint_email_channel" "test" {
  application_id    = aws_pinpoint_app.test.application_id
  enabled           = "false"
  from_address      = %[1]q
  identity          = aws_ses_domain_identity.test.arn
  configuration_set = aws_ses_configuration_set.test.arn
}

resource "aws_ses_domain_identity" "test" {
  domain = "example.com"
}
`, fromAddress, rName)
}

func testAccCheckAWSPinpointEmailChannelDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).pinpointconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_pinpoint_email_channel" {
			continue
		}

		// Check if the event stream exists
		params := &pinpoint.GetEmailChannelInput{
			ApplicationId: aws.String(rs.Primary.ID),
		}
		_, err := conn.GetEmailChannel(params)
		if err != nil {
			if isAWSErr(err, pinpoint.ErrCodeNotFoundException, "") {
				continue
			}
			return err
		}
		return fmt.Errorf("Email Channel exists when it should be destroyed!")
	}

	return nil
}
