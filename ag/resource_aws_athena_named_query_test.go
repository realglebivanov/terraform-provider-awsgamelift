package ag

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/athena"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccAWSAthenaNamedQuery_basic(t *testing.T) {
	resourceName := "aws_athena_named_query.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, athena.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSAthenaNamedQueryDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAthenaNamedQueryConfig(acctest.RandInt(), acctest.RandString(5)),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAthenaNamedQueryExists(resourceName),
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

func TestAccAWSAthenaNamedQuery_withWorkGroup(t *testing.T) {
	resourceName := "aws_athena_named_query.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, athena.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSAthenaNamedQueryDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAthenaNamedWorkGroupQueryConfig(acctest.RandInt(), acctest.RandString(5)),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAthenaNamedQueryExists(resourceName),
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

func testAccCheckAWSAthenaNamedQueryDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).athenaconn
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_athena_named_query" {
			continue
		}

		input := &athena.GetNamedQueryInput{
			NamedQueryId: aws.String(rs.Primary.ID),
		}

		resp, err := conn.GetNamedQuery(input)
		if err != nil {
			if isAWSErr(err, athena.ErrCodeInvalidRequestException, rs.Primary.ID) {
				return nil
			}
			return err
		}
		if resp.NamedQuery != nil {
			return fmt.Errorf("Athena Named Query (%s) found", rs.Primary.ID)
		}
	}
	return nil
}

func testAccCheckAWSAthenaNamedQueryExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		conn := testAccProvider.Meta().(*AWSClient).athenaconn

		input := &athena.GetNamedQueryInput{
			NamedQueryId: aws.String(rs.Primary.ID),
		}

		_, err := conn.GetNamedQuery(input)
		return err
	}
}

func testAccAthenaNamedQueryConfig(rInt int, rName string) string {
	return fmt.Sprintf(`
resource "aws_s3_bucket" "test" {
  bucket        = "tf-test-athena-db-%s-%d"
  force_destroy = true
}

resource "aws_athena_database" "test" {
  name   = "%s"
  bucket = aws_s3_bucket.test.bucket
}

resource "aws_athena_named_query" "test" {
  name        = "tf-athena-named-query-%s"
  database    = aws_athena_database.test.name
  query       = "SELECT * FROM ${aws_athena_database.test.name} limit 10;"
  description = "tf test"
}
`, rName, rInt, rName, rName)
}

func testAccAthenaNamedWorkGroupQueryConfig(rInt int, rName string) string {
	return fmt.Sprintf(`
resource "aws_s3_bucket" "test" {
  bucket        = "tf-test-athena-db-%s-%d"
  force_destroy = true
}

resource "aws_athena_workgroup" "test" {
  name = "tf-athena-workgroup-%s-%d"
}

resource "aws_athena_database" "test" {
  name   = "%s"
  bucket = aws_s3_bucket.test.bucket
}

resource "aws_athena_named_query" "test" {
  name        = "tf-athena-named-query-%s"
  workgroup   = aws_athena_workgroup.test.id
  database    = aws_athena_database.test.name
  query       = "SELECT * FROM ${aws_athena_database.test.name} limit 10;"
  description = "tf test"
}
`, rName, rInt, rName, rInt, rName, rName)
}
