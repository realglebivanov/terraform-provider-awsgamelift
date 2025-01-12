package ag

import (
	"fmt"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccAWSUserPolicyAttachment_basic(t *testing.T) {
	var out iam.ListAttachedUserPoliciesOutput
	rName := acctest.RandString(10)
	policyName1 := fmt.Sprintf("test-policy-%s", acctest.RandString(10))
	policyName2 := fmt.Sprintf("test-policy-%s", acctest.RandString(10))
	policyName3 := fmt.Sprintf("test-policy-%s", acctest.RandString(10))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, iam.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSUserPolicyAttachmentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSUserPolicyAttachConfig(rName, policyName1),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSUserPolicyAttachmentExists("aws_iam_user_policy_attachment.test-attach", 1, &out),
					testAccCheckAWSUserPolicyAttachmentAttributes([]string{policyName1}, &out),
				),
			},
			{
				ResourceName:      "aws_iam_user_policy_attachment.test-attach",
				ImportState:       true,
				ImportStateIdFunc: testAccAWSIAMUserPolicyAttachmentImportStateIdFunc("aws_iam_user_policy_attachment.test-attach"),
				// We do not have a way to align IDs since the Create function uses resource.PrefixedUniqueId()
				// Failed state verification, resource with ID USER-POLICYARN not found
				// ImportStateVerify: true,
				ImportStateCheck: func(s []*terraform.InstanceState) error {
					if len(s) != 1 {
						return fmt.Errorf("expected 1 state: %#v", s)
					}

					rs := s[0]

					if !strings.HasPrefix(rs.Attributes["policy_arn"], "arn:") {
						return fmt.Errorf("expected policy_arn attribute to be set and begin with arn:, received: %s", rs.Attributes["policy_arn"])
					}

					return nil
				},
			},
			{
				Config: testAccAWSUserPolicyAttachConfigUpdate(rName, policyName1, policyName2, policyName3),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSUserPolicyAttachmentExists("aws_iam_user_policy_attachment.test-attach", 2, &out),
					testAccCheckAWSUserPolicyAttachmentAttributes([]string{policyName2, policyName3}, &out),
				),
			},
		},
	})
}

func testAccCheckAWSUserPolicyAttachmentDestroy(s *terraform.State) error {
	return nil
}

func testAccCheckAWSUserPolicyAttachmentExists(n string, c int, out *iam.ListAttachedUserPoliciesOutput) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No policy name is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).iamconn
		user := rs.Primary.Attributes["user"]

		attachedPolicies, err := conn.ListAttachedUserPolicies(&iam.ListAttachedUserPoliciesInput{
			UserName: aws.String(user),
		})
		if err != nil {
			return fmt.Errorf("Error: Failed to get attached policies for user %s (%s)", user, n)
		}
		if c != len(attachedPolicies.AttachedPolicies) {
			return fmt.Errorf("Error: User (%s) has wrong number of policies attached on initial creation", n)
		}

		*out = *attachedPolicies
		return nil
	}
}

func testAccCheckAWSUserPolicyAttachmentAttributes(policies []string, out *iam.ListAttachedUserPoliciesOutput) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		matched := 0

		for _, p := range policies {
			for _, ap := range out.AttachedPolicies {
				// *ap.PolicyArn like arn:aws:iam::111111111111:policy/test-policy
				parts := strings.Split(*ap.PolicyArn, "/")
				if len(parts) == 2 && p == parts[1] {
					matched++
				}
			}
		}
		if matched != len(policies) || matched != len(out.AttachedPolicies) {
			return fmt.Errorf("Error: Number of attached policies was incorrect: expected %d matched policies, matched %d of %d", len(policies), matched, len(out.AttachedPolicies))
		}
		return nil
	}
}

func testAccAWSIAMUserPolicyAttachmentImportStateIdFunc(resourceName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("Not found: %s", resourceName)
		}

		return fmt.Sprintf("%s/%s", rs.Primary.Attributes["user"], rs.Primary.Attributes["policy_arn"]), nil
	}
}

func testAccAWSUserPolicyAttachConfig(rName, policyName string) string {
	return fmt.Sprintf(`
resource "aws_iam_user" "user" {
  name = "test-user-%s"
}

resource "aws_iam_policy" "policy" {
  name        = "%s"
  description = "A test policy"

  policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": [
        "iam:ChangePassword"
      ],
      "Resource": "*",
      "Effect": "Allow"
    }
  ]
}
EOF
}

resource "aws_iam_user_policy_attachment" "test-attach" {
  user       = aws_iam_user.user.name
  policy_arn = aws_iam_policy.policy.arn
}
`, rName, policyName)
}

func testAccAWSUserPolicyAttachConfigUpdate(rName, policyName1, policyName2, policyName3 string) string {
	return fmt.Sprintf(`
resource "aws_iam_user" "user" {
  name = "test-user-%s"
}

resource "aws_iam_policy" "policy" {
  name        = "%s"
  description = "A test policy"

  policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": [
        "iam:ChangePassword"
      ],
      "Resource": "*",
      "Effect": "Allow"
    }
  ]
}
EOF
}

resource "aws_iam_policy" "policy2" {
  name        = "%s"
  description = "A test policy"

  policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": [
        "iam:ChangePassword"
      ],
      "Resource": "*",
      "Effect": "Allow"
    }
  ]
}
EOF
}

resource "aws_iam_policy" "policy3" {
  name        = "%s"
  description = "A test policy"

  policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": [
        "iam:ChangePassword"
      ],
      "Resource": "*",
      "Effect": "Allow"
    }
  ]
}
EOF
}

resource "aws_iam_user_policy_attachment" "test-attach" {
  user       = aws_iam_user.user.name
  policy_arn = aws_iam_policy.policy2.arn
}

resource "aws_iam_user_policy_attachment" "test-attach2" {
  user       = aws_iam_user.user.name
  policy_arn = aws_iam_policy.policy3.arn
}
`, rName, policyName1, policyName2, policyName3)
}
