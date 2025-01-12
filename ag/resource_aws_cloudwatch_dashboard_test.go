package ag

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccAWSCloudWatchDashboard_basic(t *testing.T) {
	var dashboard cloudwatch.GetDashboardOutput
	resourceName := "aws_cloudwatch_dashboard.test"
	rInt := acctest.RandInt()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, cloudwatch.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCloudWatchDashboardDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSCloudWatchDashboardConfig(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudWatchDashboardExists(resourceName, &dashboard),
					resource.TestCheckResourceAttr(resourceName, "dashboard_name", testAccAWSCloudWatchDashboardName(rInt)),
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

func TestAccAWSCloudWatchDashboard_update(t *testing.T) {
	var dashboard cloudwatch.GetDashboardOutput
	resourceName := "aws_cloudwatch_dashboard.test"
	rInt := acctest.RandInt()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, cloudwatch.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCloudWatchDashboardDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSCloudWatchDashboardConfig(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudWatchDashboardExists(resourceName, &dashboard),
					testAccCloudWatchCheckDashboardBodyIsExpected(resourceName, basicWidget),
					resource.TestCheckResourceAttr(resourceName, "dashboard_name", testAccAWSCloudWatchDashboardName(rInt)),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccAWSCloudWatchDashboardConfig_updateBody(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudWatchDashboardExists(resourceName, &dashboard),
					testAccCloudWatchCheckDashboardBodyIsExpected(resourceName, updatedWidget),
					resource.TestCheckResourceAttr(resourceName, "dashboard_name", testAccAWSCloudWatchDashboardName(rInt)),
				),
			},
		},
	})
}

func TestAccAWSCloudWatchDashboard_updateName(t *testing.T) {
	var dashboard cloudwatch.GetDashboardOutput
	resourceName := "aws_cloudwatch_dashboard.test"
	rInt := acctest.RandInt()
	rInt2 := acctest.RandInt()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, cloudwatch.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCloudWatchDashboardDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSCloudWatchDashboardConfig(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudWatchDashboardExists(resourceName, &dashboard),
					testAccCloudWatchCheckDashboardBodyIsExpected(resourceName, basicWidget),
					resource.TestCheckResourceAttr(resourceName, "dashboard_name", testAccAWSCloudWatchDashboardName(rInt)),
				),
			},
			{
				Config: testAccAWSCloudWatchDashboardConfig(rInt2),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudWatchDashboardExists(resourceName, &dashboard),
					testAccCloudWatchCheckDashboardBodyIsExpected(resourceName, basicWidget),
					resource.TestCheckResourceAttr(resourceName, "dashboard_name", testAccAWSCloudWatchDashboardName(rInt2)),
					testAccCheckAWSCloudWatchDashboardDestroyPrevious(testAccAWSCloudWatchDashboardName(rInt)),
				),
			},
		},
	})
}

func testAccCheckCloudWatchDashboardExists(n string, dashboard *cloudwatch.GetDashboardOutput) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		conn := testAccProvider.Meta().(*AWSClient).cloudwatchconn
		params := cloudwatch.GetDashboardInput{
			DashboardName: aws.String(rs.Primary.ID),
		}

		resp, err := conn.GetDashboard(&params)
		if err != nil {
			return err
		}

		*dashboard = *resp

		return nil
	}
}

func testAccCheckAWSCloudWatchDashboardDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).cloudwatchconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_cloudwatch_dashboard" {
			continue
		}

		params := cloudwatch.GetDashboardInput{
			DashboardName: aws.String(rs.Primary.ID),
		}

		_, err := conn.GetDashboard(&params)
		if err == nil {
			return fmt.Errorf("Dashboard still exists: %s", rs.Primary.ID)
		}
		if !isCloudWatchDashboardNotFoundErr(err) {
			return err
		}
	}

	return nil
}

func testAccCheckAWSCloudWatchDashboardDestroyPrevious(dashboardName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := testAccProvider.Meta().(*AWSClient).cloudwatchconn

		params := cloudwatch.GetDashboardInput{
			DashboardName: aws.String(dashboardName),
		}

		_, err := conn.GetDashboard(&params)

		if err == nil {
			return fmt.Errorf("Dashboard still exists: %s", dashboardName)
		}

		if !isCloudWatchDashboardNotFoundErr(err) {
			return err
		}

		return nil
	}
}

const (
	basicWidget = `{
  "widgets": [
    {
      "type": "text",
      "x": 0,
      "y": 0,
      "width": 6,
      "height": 6,
      "properties": {
        "markdown": "Hi there from Terraform: CloudWatch"
      }
    }
  ]
}`

	updatedWidget = `{
  "widgets": [
    {
      "type": "text",
      "x": 0,
      "y": 0,
      "width": 6,
      "height": 6,
      "properties": {
        "markdown": "Hi there from Terraform: CloudWatch - updated"
      }
    }
  ]
}`
)

func testAccAWSCloudWatchDashboardName(rInt int) string {
	return fmt.Sprintf("terraform-test-dashboard-%d", rInt)
}

func testAccAWSCloudWatchDashboardConfig(rInt int) string {
	return fmt.Sprintf(`
resource "aws_cloudwatch_dashboard" "test" {
  dashboard_name = "terraform-test-dashboard-%d"

  dashboard_body = <<EOF
  %s
EOF
}
`, rInt, basicWidget)
}

func testAccAWSCloudWatchDashboardConfig_updateBody(rInt int) string {
	return fmt.Sprintf(`
resource "aws_cloudwatch_dashboard" "test" {
  dashboard_name = "terraform-test-dashboard-%d"

  dashboard_body = <<EOF
  %s
EOF
}
`, rInt, updatedWidget)
}

func testAccCloudWatchCheckDashboardBodyIsExpected(resourceName, expected string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("Not found: %s", resourceName)
		}

		conn := testAccProvider.Meta().(*AWSClient).cloudwatchconn
		params := cloudwatch.GetDashboardInput{
			DashboardName: aws.String(rs.Primary.ID),
		}

		resp, err := conn.GetDashboard(&params)
		if err != nil {
			return err
		}

		var (
			bodyReader     = strings.NewReader(*resp.DashboardBody)
			expectedReader = strings.NewReader(expected)

			body         = make(map[string]interface{})
			expectedBody = make(map[string]interface{})
		)

		if err := json.NewDecoder(bodyReader).Decode(&body); err != nil {
			return fmt.Errorf("failed to parse received body: %s", err)
		} else if err := json.NewDecoder(expectedReader).Decode(&expectedBody); err != nil {
			return fmt.Errorf("failed to parse expected body: %s", err)
		}

		if !reflect.DeepEqual(body, expectedBody) {
			return fmt.Errorf("Expected %q dashboard body, got %q", expectedBody, body)
		}

		return nil
	}
}
