package ag

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sns"
	multierror "github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// The preferences are account-wide, so the tests must be serialized
func TestAccAWSSNSSMSPreferences_serial(t *testing.T) {
	testCases := map[string]func(t *testing.T){
		"almostAll":      testAccAWSSNSSMSPreferences_almostAll,
		"defaultSMSType": testAccAWSSNSSMSPreferences_defaultSMSType,
		"deliveryRole":   testAccAWSSNSSMSPreferences_deliveryRole,
		"empty":          testAccAWSSNSSMSPreferences_empty,
	}

	for name, tc := range testCases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			tc(t)
		})
	}
}

func testAccAWSSNSSMSPreferences_empty(t *testing.T) {
	resourceName := "aws_sns_sms_preferences.test_pref"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, sns.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSNSSMSPrefsDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSNSSMSPreferencesConfig_empty,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr(resourceName, "monthly_spend_limit"),
					resource.TestCheckNoResourceAttr(resourceName, "delivery_status_iam_role_arn"),
					resource.TestCheckNoResourceAttr(resourceName, "delivery_status_success_sampling_rate"),
					resource.TestCheckNoResourceAttr(resourceName, "default_sender_id"),
					resource.TestCheckNoResourceAttr(resourceName, "default_sms_type"),
					resource.TestCheckNoResourceAttr(resourceName, "usage_report_s3_bucket"),
				),
			},
		},
	})
}

func testAccAWSSNSSMSPreferences_defaultSMSType(t *testing.T) {
	resourceName := "aws_sns_sms_preferences.test_pref"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, sns.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSNSSMSPrefsDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSNSSMSPreferencesConfig_defSMSType,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr(resourceName, "monthly_spend_limit"),
					resource.TestCheckNoResourceAttr(resourceName, "delivery_status_iam_role_arn"),
					resource.TestCheckNoResourceAttr(resourceName, "delivery_status_success_sampling_rate"),
					resource.TestCheckNoResourceAttr(resourceName, "default_sender_id"),
					resource.TestCheckResourceAttr(resourceName, "default_sms_type", "Transactional"),
					resource.TestCheckNoResourceAttr(resourceName, "usage_report_s3_bucket"),
				),
			},
		},
	})
}

func testAccAWSSNSSMSPreferences_almostAll(t *testing.T) {
	resourceName := "aws_sns_sms_preferences.test_pref"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, sns.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSNSSMSPrefsDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSNSSMSPreferencesConfig_almostAll,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "monthly_spend_limit", "1"),
					resource.TestCheckResourceAttr(resourceName, "default_sms_type", "Transactional"),
					resource.TestCheckResourceAttr(resourceName, "usage_report_s3_bucket", "some-bucket"),
				),
			},
		},
	})
}

func testAccAWSSNSSMSPreferences_deliveryRole(t *testing.T) {
	resourceName := "aws_sns_sms_preferences.test_pref"
	iamRoleName := "aws_iam_role.test_smsdelivery_role"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, sns.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSNSSMSPrefsDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSNSSMSPreferencesConfig_deliveryRole,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair(resourceName, "delivery_status_iam_role_arn", iamRoleName, "arn"),
					resource.TestCheckResourceAttr(resourceName, "delivery_status_success_sampling_rate", "75"),
				),
			},
		},
	})
}

func testAccCheckAWSSNSSMSPrefsDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_sns_sms_preferences" {
			continue
		}

		conn := testAccProvider.Meta().(*AWSClient).snsconn
		attrs, err := conn.GetSMSAttributes(&sns.GetSMSAttributesInput{})
		if err != nil {
			return fmt.Errorf("error getting SMS attributes: %s", err)
		}
		if attrs == nil || len(attrs.Attributes) == 0 {
			return nil
		}

		var attrErrs *multierror.Error

		// The API is returning undocumented keys, e.g. "UsageReportS3Enabled". Only check the keys we're aware of.
		for _, snsAttrName := range smsAttributeMap {
			v := aws.StringValue(attrs.Attributes[snsAttrName])
			if v != "" {
				attrErrs = multierror.Append(attrErrs, fmt.Errorf("expected SMS attribute %q to be empty, but received: %q", snsAttrName, v))
			}
		}

		return attrErrs.ErrorOrNil()
	}

	return nil
}

const testAccAWSSNSSMSPreferencesConfig_empty = `
resource "aws_sns_sms_preferences" "test_pref" {}
`
const testAccAWSSNSSMSPreferencesConfig_defSMSType = `
resource "aws_sns_sms_preferences" "test_pref" {
  default_sms_type = "Transactional"
}
`

const testAccAWSSNSSMSPreferencesConfig_almostAll = `
resource "aws_sns_sms_preferences" "test_pref" {
  monthly_spend_limit    = "1"
  default_sms_type       = "Transactional"
  usage_report_s3_bucket = "some-bucket"
}
`

const testAccAWSSNSSMSPreferencesConfig_deliveryRole = `
resource "aws_sns_sms_preferences" "test_pref" {
  delivery_status_iam_role_arn          = aws_iam_role.test_smsdelivery_role.arn
  delivery_status_success_sampling_rate = "75"
}

resource "aws_iam_role" "test_smsdelivery_role" {
  name = "test_smsdelivery_role"
  path = "/"

  assume_role_policy = <<POLICY
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "sns.amazonaws.com"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
POLICY
}

resource "aws_iam_role_policy" "test_smsdelivery_role_policy" {
  name = "test_smsdelivery_role_policy"
  role = aws_iam_role.test_smsdelivery_role.id

  policy = <<POLICY
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": [
        "logs:CreateLogGroup",
        "logs:CreateLogStream",
        "logs:PutLogEvents",
        "logs:PutMetricFilter",
        "logs:PutRetentionPolicy"
      ],
      "Resource": "*",
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
POLICY
}
`
