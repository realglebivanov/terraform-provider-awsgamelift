package ag

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccAWSAPIGatewayIntegration_basic(t *testing.T) {
	var conf apigateway.Integration
	rName := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(7))
	resourceName := "aws_api_gateway_integration.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t); testAccAPIGatewayTypeEDGEPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, apigateway.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSAPIGatewayIntegrationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSAPIGatewayIntegrationConfig(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAPIGatewayIntegrationExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "type", "HTTP"),
					resource.TestCheckResourceAttr(resourceName, "integration_http_method", "GET"),
					resource.TestCheckResourceAttr(resourceName, "uri", "https://www.google.de"),
					resource.TestCheckResourceAttr(resourceName, "passthrough_behavior", "WHEN_NO_MATCH"),
					resource.TestCheckResourceAttr(resourceName, "content_handling", "CONVERT_TO_TEXT"),
					resource.TestCheckResourceAttr(resourceName, "credentials", ""),
					resource.TestCheckResourceAttr(resourceName, "request_parameters.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "request_parameters.integration.request.header.X-Authorization", "'static'"),
					resource.TestCheckResourceAttr(resourceName, "request_parameters.integration.request.header.X-Foo", "'Bar'"),
					resource.TestCheckResourceAttr(resourceName, "request_templates.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "request_templates.application/json", ""),
					resource.TestCheckResourceAttr(resourceName, "request_templates.application/xml", "#set($inputRoot = $input.path('$'))\n{ }"),
					resource.TestCheckResourceAttr(resourceName, "timeout_milliseconds", "29000"),
					resource.TestCheckResourceAttr(resourceName, "tls_config.#", "0"),
				),
			},

			{
				Config: testAccAWSAPIGatewayIntegrationConfigUpdate(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAPIGatewayIntegrationExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "type", "HTTP"),
					resource.TestCheckResourceAttr(resourceName, "integration_http_method", "GET"),
					resource.TestCheckResourceAttr(resourceName, "uri", "https://www.google.de"),
					resource.TestCheckResourceAttr(resourceName, "passthrough_behavior", "WHEN_NO_MATCH"),
					resource.TestCheckResourceAttr(resourceName, "content_handling", "CONVERT_TO_TEXT"),
					resource.TestCheckResourceAttr(resourceName, "credentials", ""),
					resource.TestCheckResourceAttr(resourceName, "request_parameters.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "request_parameters.integration.request.header.X-Authorization", "'updated'"),
					resource.TestCheckResourceAttr(resourceName, "request_parameters.integration.request.header.X-FooBar", "'Baz'"),
					resource.TestCheckResourceAttr(resourceName, "request_templates.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "request_templates.application/json", "{'foobar': 'bar}"),
					resource.TestCheckResourceAttr(resourceName, "request_templates.text/html", "<html>Foo</html>"),
					resource.TestCheckResourceAttr(resourceName, "timeout_milliseconds", "2000"),
				),
			},

			{
				Config: testAccAWSAPIGatewayIntegrationConfigUpdateURI(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAPIGatewayIntegrationExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "type", "HTTP"),
					resource.TestCheckResourceAttr(resourceName, "integration_http_method", "GET"),
					resource.TestCheckResourceAttr(resourceName, "uri", "https://www.google.de/updated"),
					resource.TestCheckResourceAttr(resourceName, "passthrough_behavior", "WHEN_NO_MATCH"),
					resource.TestCheckResourceAttr(resourceName, "content_handling", "CONVERT_TO_TEXT"),
					resource.TestCheckResourceAttr(resourceName, "credentials", ""),
					resource.TestCheckResourceAttr(resourceName, "request_parameters.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "request_parameters.integration.request.header.X-Authorization", "'static'"),
					resource.TestCheckResourceAttr(resourceName, "request_parameters.integration.request.header.X-Foo", "'Bar'"),
					resource.TestCheckResourceAttr(resourceName, "request_templates.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "request_templates.application/json", ""),
					resource.TestCheckResourceAttr(resourceName, "request_templates.application/xml", "#set($inputRoot = $input.path('$'))\n{ }"),
					resource.TestCheckResourceAttr(resourceName, "timeout_milliseconds", "2000"),
				),
			},

			{
				Config: testAccAWSAPIGatewayIntegrationConfigUpdateNoTemplates(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAPIGatewayIntegrationExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "type", "HTTP"),
					resource.TestCheckResourceAttr(resourceName, "integration_http_method", "GET"),
					resource.TestCheckResourceAttr(resourceName, "uri", "https://www.google.de"),
					resource.TestCheckResourceAttr(resourceName, "passthrough_behavior", "WHEN_NO_MATCH"),
					resource.TestCheckResourceAttr(resourceName, "content_handling", "CONVERT_TO_TEXT"),
					resource.TestCheckResourceAttr(resourceName, "credentials", ""),
					resource.TestCheckResourceAttr(resourceName, "request_parameters.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "request_templates.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "timeout_milliseconds", "2000"),
				),
			},

			{
				Config: testAccAWSAPIGatewayIntegrationConfig(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAPIGatewayIntegrationExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "type", "HTTP"),
					resource.TestCheckResourceAttr(resourceName, "integration_http_method", "GET"),
					resource.TestCheckResourceAttr(resourceName, "uri", "https://www.google.de"),
					resource.TestCheckResourceAttr(resourceName, "passthrough_behavior", "WHEN_NO_MATCH"),
					resource.TestCheckResourceAttr(resourceName, "content_handling", "CONVERT_TO_TEXT"),
					resource.TestCheckResourceAttr(resourceName, "credentials", ""),
					resource.TestCheckResourceAttr(resourceName, "request_parameters.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "request_parameters.integration.request.header.X-Authorization", "'static'"),
					resource.TestCheckResourceAttr(resourceName, "request_templates.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "request_templates.application/json", ""),
					resource.TestCheckResourceAttr(resourceName, "request_templates.application/xml", "#set($inputRoot = $input.path('$'))\n{ }"),
					resource.TestCheckResourceAttr(resourceName, "timeout_milliseconds", "29000"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testAccAWSAPIGatewayIntegrationImportStateIdFunc(resourceName),
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAWSAPIGatewayIntegration_contentHandling(t *testing.T) {
	var conf apigateway.Integration
	rName := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(7))
	resourceName := "aws_api_gateway_integration.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t); testAccAPIGatewayTypeEDGEPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, apigateway.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSAPIGatewayIntegrationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSAPIGatewayIntegrationConfig(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAPIGatewayIntegrationExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "type", "HTTP"),
					resource.TestCheckResourceAttr(resourceName, "integration_http_method", "GET"),
					resource.TestCheckResourceAttr(resourceName, "uri", "https://www.google.de"),
					resource.TestCheckResourceAttr(resourceName, "passthrough_behavior", "WHEN_NO_MATCH"),
					resource.TestCheckResourceAttr(resourceName, "content_handling", "CONVERT_TO_TEXT"),
					resource.TestCheckResourceAttr(resourceName, "credentials", ""),
					resource.TestCheckResourceAttr(resourceName, "request_parameters.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "request_parameters.integration.request.header.X-Authorization", "'static'"),
					resource.TestCheckResourceAttr(resourceName, "request_parameters.integration.request.header.X-Foo", "'Bar'"),
					resource.TestCheckResourceAttr(resourceName, "request_templates.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "request_templates.application/json", ""),
					resource.TestCheckResourceAttr(resourceName, "request_templates.application/xml", "#set($inputRoot = $input.path('$'))\n{ }"),
				),
			},

			{
				Config: testAccAWSAPIGatewayIntegrationConfigUpdateContentHandling(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAPIGatewayIntegrationExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "type", "HTTP"),
					resource.TestCheckResourceAttr(resourceName, "integration_http_method", "GET"),
					resource.TestCheckResourceAttr(resourceName, "uri", "https://www.google.de"),
					resource.TestCheckResourceAttr(resourceName, "passthrough_behavior", "WHEN_NO_MATCH"),
					resource.TestCheckResourceAttr(resourceName, "content_handling", "CONVERT_TO_BINARY"),
					resource.TestCheckResourceAttr(resourceName, "credentials", ""),
					resource.TestCheckResourceAttr(resourceName, "request_parameters.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "request_parameters.integration.request.header.X-Authorization", "'static'"),
					resource.TestCheckResourceAttr(resourceName, "request_parameters.integration.request.header.X-Foo", "'Bar'"),
					resource.TestCheckResourceAttr(resourceName, "request_templates.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "request_templates.application/json", ""),
					resource.TestCheckResourceAttr(resourceName, "request_templates.application/xml", "#set($inputRoot = $input.path('$'))\n{ }"),
				),
			},
			{
				Config: testAccAWSAPIGatewayIntegrationConfigRemoveContentHandling(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAPIGatewayIntegrationExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "type", "HTTP"),
					resource.TestCheckResourceAttr(resourceName, "integration_http_method", "GET"),
					resource.TestCheckResourceAttr(resourceName, "uri", "https://www.google.de"),
					resource.TestCheckResourceAttr(resourceName, "passthrough_behavior", "WHEN_NO_MATCH"),
					resource.TestCheckResourceAttr(resourceName, "content_handling", ""),
					resource.TestCheckResourceAttr(resourceName, "credentials", ""),
					resource.TestCheckResourceAttr(resourceName, "request_parameters.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "request_parameters.integration.request.header.X-Authorization", "'static'"),
					resource.TestCheckResourceAttr(resourceName, "request_parameters.integration.request.header.X-Foo", "'Bar'"),
					resource.TestCheckResourceAttr(resourceName, "request_templates.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "request_templates.application/json", ""),
					resource.TestCheckResourceAttr(resourceName, "request_templates.application/xml", "#set($inputRoot = $input.path('$'))\n{ }"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testAccAWSAPIGatewayIntegrationImportStateIdFunc(resourceName),
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAWSAPIGatewayIntegration_cache_key_parameters(t *testing.T) {
	var conf apigateway.Integration
	rName := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(7))
	resourceName := "aws_api_gateway_integration.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t); testAccAPIGatewayTypeEDGEPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, apigateway.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSAPIGatewayIntegrationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSAPIGatewayIntegrationConfigCacheKeyParameters(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAPIGatewayIntegrationExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "type", "HTTP"),
					resource.TestCheckResourceAttr(resourceName, "integration_http_method", "GET"),
					resource.TestCheckResourceAttr(resourceName, "uri", "https://www.google.de"),
					resource.TestCheckResourceAttr(resourceName, "passthrough_behavior", "WHEN_NO_MATCH"),
					resource.TestCheckResourceAttr(resourceName, "content_handling", "CONVERT_TO_TEXT"),
					resource.TestCheckResourceAttr(resourceName, "credentials", ""),
					resource.TestCheckResourceAttr(resourceName, "request_parameters.%", "3"),
					resource.TestCheckResourceAttr(resourceName, "request_parameters.integration.request.header.X-Authorization", "'static'"),
					resource.TestCheckResourceAttr(resourceName, "request_parameters.integration.request.header.X-Foo", "'Bar'"),
					resource.TestCheckResourceAttr(resourceName, "request_parameters.integration.request.path.param", "method.request.path.param"),
					resource.TestCheckResourceAttr(resourceName, "cache_key_parameters.#", "1"),
					resource.TestCheckTypeSetElemAttr(resourceName, "cache_key_parameters.*", "method.request.path.param"),
					resource.TestCheckResourceAttr(resourceName, "cache_namespace", "foobar"),
					resource.TestCheckResourceAttr(resourceName, "request_templates.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "request_templates.application/json", ""),
					resource.TestCheckResourceAttr(resourceName, "request_templates.application/xml", "#set($inputRoot = $input.path('$'))\n{ }"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testAccAWSAPIGatewayIntegrationImportStateIdFunc(resourceName),
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAWSAPIGatewayIntegration_integrationType(t *testing.T) {
	var conf apigateway.Integration
	rName := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(7))
	resourceName := "aws_api_gateway_integration.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t); testAccAPIGatewayTypeEDGEPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, apigateway.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSAPIGatewayIntegrationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSAPIGatewayIntegrationConfig_IntegrationTypeInternet(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAPIGatewayIntegrationExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "connection_type", "INTERNET"),
					resource.TestCheckResourceAttr(resourceName, "connection_id", ""),
				),
			},
			{
				Config: testAccAWSAPIGatewayIntegrationConfig_IntegrationTypeVpcLink(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAPIGatewayIntegrationExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "connection_type", "VPC_LINK"),
					resource.TestMatchResourceAttr(resourceName, "connection_id", regexp.MustCompile("^[0-9a-z]+$")),
				),
			},
			{
				Config: testAccAWSAPIGatewayIntegrationConfig_IntegrationTypeInternet(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAPIGatewayIntegrationExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "connection_type", "INTERNET"),
					resource.TestCheckResourceAttr(resourceName, "connection_id", ""),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testAccAWSAPIGatewayIntegrationImportStateIdFunc(resourceName),
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAWSAPIGatewayIntegration_TlsConfig_InsecureSkipVerification(t *testing.T) {
	var conf apigateway.Integration
	rName := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(7))
	resourceName := "aws_api_gateway_integration.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t); testAccAPIGatewayTypeEDGEPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, apigateway.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSAPIGatewayIntegrationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSAPIGatewayIntegrationConfig_TlsConfig_InsecureSkipVerification(rName, true),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAPIGatewayIntegrationExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "tls_config.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "tls_config.0.insecure_skip_verification", "true"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testAccAWSAPIGatewayIntegrationImportStateIdFunc(resourceName),
				ImportStateVerify: true,
			},
			{
				Config: testAccAWSAPIGatewayIntegrationConfig_TlsConfig_InsecureSkipVerification(rName, false),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAPIGatewayIntegrationExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "tls_config.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "tls_config.0.insecure_skip_verification", "false"),
				),
			},
		},
	})
}

func TestAccAWSAPIGatewayIntegration_disappears(t *testing.T) {
	var conf apigateway.Integration
	rName := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(7))
	resourceName := "aws_api_gateway_integration.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t); testAccAPIGatewayTypeEDGEPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, apigateway.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSAPIGatewayIntegrationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSAPIGatewayIntegrationConfig(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAPIGatewayIntegrationExists(resourceName, &conf),
					testAccCheckResourceDisappears(testAccProvider, resourceAwsApiGatewayIntegration(), resourceName),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccCheckAWSAPIGatewayIntegrationExists(n string, res *apigateway.Integration) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No API Gateway Method ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).apigatewayconn

		req := &apigateway.GetIntegrationInput{
			HttpMethod: aws.String("GET"),
			ResourceId: aws.String(s.RootModule().Resources["aws_api_gateway_resource.test"].Primary.ID),
			RestApiId:  aws.String(s.RootModule().Resources["aws_api_gateway_rest_api.test"].Primary.ID),
		}
		describe, err := conn.GetIntegration(req)
		if err != nil {
			return err
		}

		*res = *describe

		return nil
	}
}

func testAccCheckAWSAPIGatewayIntegrationDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).apigatewayconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_api_gateway_integration" {
			continue
		}

		req := &apigateway.GetIntegrationInput{
			HttpMethod: aws.String("GET"),
			ResourceId: aws.String(s.RootModule().Resources["aws_api_gateway_resource.test"].Primary.ID),
			RestApiId:  aws.String(s.RootModule().Resources["aws_api_gateway_rest_api.test"].Primary.ID),
		}
		_, err := conn.GetIntegration(req)

		if err == nil {
			return fmt.Errorf("API Gateway Method still exists")
		}

		aws2err, ok := err.(awserr.Error)
		if !ok {
			return err
		}
		if aws2err.Code() != "NotFoundException" {
			return err
		}

		return nil
	}

	return nil
}

func testAccAWSAPIGatewayIntegrationImportStateIdFunc(resourceName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("Not found: %s", resourceName)
		}

		return fmt.Sprintf("%s/%s/%s", rs.Primary.Attributes["rest_api_id"], rs.Primary.Attributes["resource_id"], rs.Primary.Attributes["http_method"]), nil
	}
}

func testAccAWSAPIGatewayIntegrationConfig(rName string) string {
	return fmt.Sprintf(`
resource "aws_api_gateway_rest_api" "test" {
  name = "%s"
}

resource "aws_api_gateway_resource" "test" {
  rest_api_id = aws_api_gateway_rest_api.test.id
  parent_id   = aws_api_gateway_rest_api.test.root_resource_id
  path_part   = "test"
}

resource "aws_api_gateway_method" "test" {
  rest_api_id   = aws_api_gateway_rest_api.test.id
  resource_id   = aws_api_gateway_resource.test.id
  http_method   = "GET"
  authorization = "NONE"

  request_models = {
    "application/json" = "Error"
  }
}

resource "aws_api_gateway_integration" "test" {
  rest_api_id = aws_api_gateway_rest_api.test.id
  resource_id = aws_api_gateway_resource.test.id
  http_method = aws_api_gateway_method.test.http_method

  request_templates = {
    "application/json" = ""
    "application/xml"  = "#set($inputRoot = $input.path('$'))\n{ }"
  }

  request_parameters = {
    "integration.request.header.X-Authorization" = "'static'"
    "integration.request.header.X-Foo"           = "'Bar'"
  }

  type                    = "HTTP"
  uri                     = "https://www.google.de"
  integration_http_method = "GET"
  passthrough_behavior    = "WHEN_NO_MATCH"
  content_handling        = "CONVERT_TO_TEXT"
}
`, rName)
}

func testAccAWSAPIGatewayIntegrationConfigUpdate(rName string) string {
	return fmt.Sprintf(`
resource "aws_api_gateway_rest_api" "test" {
  name = "%s"
}

resource "aws_api_gateway_resource" "test" {
  rest_api_id = aws_api_gateway_rest_api.test.id
  parent_id   = aws_api_gateway_rest_api.test.root_resource_id
  path_part   = "test"
}

resource "aws_api_gateway_method" "test" {
  rest_api_id   = aws_api_gateway_rest_api.test.id
  resource_id   = aws_api_gateway_resource.test.id
  http_method   = "GET"
  authorization = "NONE"

  request_models = {
    "application/json" = "Error"
  }
}

resource "aws_api_gateway_integration" "test" {
  rest_api_id = aws_api_gateway_rest_api.test.id
  resource_id = aws_api_gateway_resource.test.id
  http_method = aws_api_gateway_method.test.http_method

  request_templates = {
    "application/json" = "{'foobar': 'bar}"
    "text/html"        = "<html>Foo</html>"
  }

  request_parameters = {
    "integration.request.header.X-Authorization" = "'updated'"
    "integration.request.header.X-FooBar"        = "'Baz'"
  }

  type                    = "HTTP"
  uri                     = "https://www.google.de"
  integration_http_method = "GET"
  passthrough_behavior    = "WHEN_NO_MATCH"
  content_handling        = "CONVERT_TO_TEXT"
  timeout_milliseconds    = 2000
}
`, rName)
}

func testAccAWSAPIGatewayIntegrationConfigUpdateURI(rName string) string {
	return fmt.Sprintf(`
resource "aws_api_gateway_rest_api" "test" {
  name = "%s"
}

resource "aws_api_gateway_resource" "test" {
  rest_api_id = aws_api_gateway_rest_api.test.id
  parent_id   = aws_api_gateway_rest_api.test.root_resource_id
  path_part   = "test"
}

resource "aws_api_gateway_method" "test" {
  rest_api_id   = aws_api_gateway_rest_api.test.id
  resource_id   = aws_api_gateway_resource.test.id
  http_method   = "GET"
  authorization = "NONE"

  request_models = {
    "application/json" = "Error"
  }
}

resource "aws_api_gateway_integration" "test" {
  rest_api_id = aws_api_gateway_rest_api.test.id
  resource_id = aws_api_gateway_resource.test.id
  http_method = aws_api_gateway_method.test.http_method

  request_templates = {
    "application/json" = ""
    "application/xml"  = "#set($inputRoot = $input.path('$'))\n{ }"
  }

  request_parameters = {
    "integration.request.header.X-Authorization" = "'static'"
    "integration.request.header.X-Foo"           = "'Bar'"
  }

  type                    = "HTTP"
  uri                     = "https://www.google.de/updated"
  integration_http_method = "GET"
  passthrough_behavior    = "WHEN_NO_MATCH"
  content_handling        = "CONVERT_TO_TEXT"
  timeout_milliseconds    = 2000
}
`, rName)
}

func testAccAWSAPIGatewayIntegrationConfigUpdateContentHandling(rName string) string {
	return fmt.Sprintf(`
resource "aws_api_gateway_rest_api" "test" {
  name = "%s"
}

resource "aws_api_gateway_resource" "test" {
  rest_api_id = aws_api_gateway_rest_api.test.id
  parent_id   = aws_api_gateway_rest_api.test.root_resource_id
  path_part   = "test"
}

resource "aws_api_gateway_method" "test" {
  rest_api_id   = aws_api_gateway_rest_api.test.id
  resource_id   = aws_api_gateway_resource.test.id
  http_method   = "GET"
  authorization = "NONE"

  request_models = {
    "application/json" = "Error"
  }
}

resource "aws_api_gateway_integration" "test" {
  rest_api_id = aws_api_gateway_rest_api.test.id
  resource_id = aws_api_gateway_resource.test.id
  http_method = aws_api_gateway_method.test.http_method

  request_templates = {
    "application/json" = ""
    "application/xml"  = "#set($inputRoot = $input.path('$'))\n{ }"
  }

  request_parameters = {
    "integration.request.header.X-Authorization" = "'static'"
    "integration.request.header.X-Foo"           = "'Bar'"
  }

  type                    = "HTTP"
  uri                     = "https://www.google.de"
  integration_http_method = "GET"
  passthrough_behavior    = "WHEN_NO_MATCH"
  content_handling        = "CONVERT_TO_BINARY"
  timeout_milliseconds    = 2000
}
`, rName)
}

func testAccAWSAPIGatewayIntegrationConfigRemoveContentHandling(rName string) string {
	return fmt.Sprintf(`
resource "aws_api_gateway_rest_api" "test" {
  name = "%s"
}

resource "aws_api_gateway_resource" "test" {
  rest_api_id = aws_api_gateway_rest_api.test.id
  parent_id   = aws_api_gateway_rest_api.test.root_resource_id
  path_part   = "test"
}

resource "aws_api_gateway_method" "test" {
  rest_api_id   = aws_api_gateway_rest_api.test.id
  resource_id   = aws_api_gateway_resource.test.id
  http_method   = "GET"
  authorization = "NONE"

  request_models = {
    "application/json" = "Error"
  }
}

resource "aws_api_gateway_integration" "test" {
  rest_api_id = aws_api_gateway_rest_api.test.id
  resource_id = aws_api_gateway_resource.test.id
  http_method = aws_api_gateway_method.test.http_method

  request_templates = {
    "application/json" = ""
    "application/xml"  = "#set($inputRoot = $input.path('$'))\n{ }"
  }

  request_parameters = {
    "integration.request.header.X-Authorization" = "'static'"
    "integration.request.header.X-Foo"           = "'Bar'"
  }

  type                    = "HTTP"
  uri                     = "https://www.google.de"
  integration_http_method = "GET"
  passthrough_behavior    = "WHEN_NO_MATCH"
  timeout_milliseconds    = 2000
}
`, rName)
}

func testAccAWSAPIGatewayIntegrationConfigUpdateNoTemplates(rName string) string {
	return fmt.Sprintf(`
resource "aws_api_gateway_rest_api" "test" {
  name = "%s"
}

resource "aws_api_gateway_resource" "test" {
  rest_api_id = aws_api_gateway_rest_api.test.id
  parent_id   = aws_api_gateway_rest_api.test.root_resource_id
  path_part   = "test"
}

resource "aws_api_gateway_method" "test" {
  rest_api_id   = aws_api_gateway_rest_api.test.id
  resource_id   = aws_api_gateway_resource.test.id
  http_method   = "GET"
  authorization = "NONE"

  request_models = {
    "application/json" = "Error"
  }
}

resource "aws_api_gateway_integration" "test" {
  rest_api_id = aws_api_gateway_rest_api.test.id
  resource_id = aws_api_gateway_resource.test.id
  http_method = aws_api_gateway_method.test.http_method

  type                    = "HTTP"
  uri                     = "https://www.google.de"
  integration_http_method = "GET"
  passthrough_behavior    = "WHEN_NO_MATCH"
  content_handling        = "CONVERT_TO_TEXT"
  timeout_milliseconds    = 2000
}
`, rName)
}

func testAccAWSAPIGatewayIntegrationConfigCacheKeyParameters(rName string) string {
	return fmt.Sprintf(`
resource "aws_api_gateway_rest_api" "test" {
  name = "%s"
}

resource "aws_api_gateway_resource" "test" {
  rest_api_id = aws_api_gateway_rest_api.test.id
  parent_id   = aws_api_gateway_rest_api.test.root_resource_id
  path_part   = "{param}"
}

resource "aws_api_gateway_method" "test" {
  rest_api_id   = aws_api_gateway_rest_api.test.id
  resource_id   = aws_api_gateway_resource.test.id
  http_method   = "GET"
  authorization = "NONE"

  request_models = {
    "application/json" = "Error"
  }

  request_parameters = {
    "method.request.path.param" = true
  }
}

resource "aws_api_gateway_integration" "test" {
  rest_api_id = aws_api_gateway_rest_api.test.id
  resource_id = aws_api_gateway_resource.test.id
  http_method = aws_api_gateway_method.test.http_method

  request_templates = {
    "application/json" = ""
    "application/xml"  = "#set($inputRoot = $input.path('$'))\n{ }"
  }

  request_parameters = {
    "integration.request.header.X-Authorization" = "'static'"
    "integration.request.header.X-Foo"           = "'Bar'"
    "integration.request.path.param"             = "method.request.path.param"
  }

  cache_key_parameters = ["method.request.path.param"]
  cache_namespace      = "foobar"

  type                    = "HTTP"
  uri                     = "https://www.google.de"
  integration_http_method = "GET"
  passthrough_behavior    = "WHEN_NO_MATCH"
  content_handling        = "CONVERT_TO_TEXT"
  timeout_milliseconds    = 2000
}
`, rName)
}

func testAccAWSAPIGatewayIntegrationConfig_IntegrationTypeBase(rName string) string {
	return fmt.Sprintf(`
variable "name" {
  default = "%s"
}

data "aws_availability_zones" "test" {
  state = "available"

  filter {
    name   = "opt-in-status"
    values = ["opt-in-not-required"]
  }
}

resource "aws_vpc" "test" {
  cidr_block = "10.10.0.0/16"

  tags = {
    Name = var.name
  }
}

resource "aws_subnet" "test" {
  vpc_id            = aws_vpc.test.id
  cidr_block        = "10.10.0.0/24"
  availability_zone = data.aws_availability_zones.test.names[0]
}

resource "aws_api_gateway_rest_api" "test" {
  name = var.name
}

resource "aws_api_gateway_resource" "test" {
  rest_api_id = aws_api_gateway_rest_api.test.id
  parent_id   = aws_api_gateway_rest_api.test.root_resource_id
  path_part   = "test"
}

resource "aws_api_gateway_method" "test" {
  rest_api_id   = aws_api_gateway_rest_api.test.id
  resource_id   = aws_api_gateway_resource.test.id
  http_method   = "GET"
  authorization = "NONE"

  request_models = {
    "application/json" = "Error"
  }
}

resource "aws_lb" "test" {
  name               = var.name
  internal           = true
  load_balancer_type = "network"
  subnets            = [aws_subnet.test.id]
}

resource "aws_api_gateway_vpc_link" "test" {
  name        = var.name
  target_arns = [aws_lb.test.arn]
}
`, rName)
}

func testAccAWSAPIGatewayIntegrationConfig_IntegrationTypeVpcLink(rName string) string {
	return testAccAWSAPIGatewayIntegrationConfig_IntegrationTypeBase(rName) + `
resource "aws_api_gateway_integration" "test" {
  rest_api_id = aws_api_gateway_rest_api.test.id
  resource_id = aws_api_gateway_resource.test.id
  http_method = aws_api_gateway_method.test.http_method

  type                    = "HTTP"
  uri                     = "https://www.google.de"
  integration_http_method = "GET"
  passthrough_behavior    = "WHEN_NO_MATCH"
  content_handling        = "CONVERT_TO_TEXT"

  connection_type = "VPC_LINK"
  connection_id   = aws_api_gateway_vpc_link.test.id
}
`
}

func testAccAWSAPIGatewayIntegrationConfig_IntegrationTypeInternet(rName string) string {
	return testAccAWSAPIGatewayIntegrationConfig_IntegrationTypeBase(rName) + `
resource "aws_api_gateway_integration" "test" {
  rest_api_id = aws_api_gateway_rest_api.test.id
  resource_id = aws_api_gateway_resource.test.id
  http_method = aws_api_gateway_method.test.http_method

  type                    = "HTTP"
  uri                     = "https://www.google.de"
  integration_http_method = "GET"
  passthrough_behavior    = "WHEN_NO_MATCH"
  content_handling        = "CONVERT_TO_TEXT"
}
`
}

func testAccAWSAPIGatewayIntegrationConfig_TlsConfig_InsecureSkipVerification(rName string, insecureSkipVerification bool) string {
	return fmt.Sprintf(`
resource "aws_api_gateway_rest_api" "test" {
  name = %[1]q
}

resource "aws_api_gateway_resource" "test" {
  rest_api_id = aws_api_gateway_rest_api.test.id
  parent_id   = aws_api_gateway_rest_api.test.root_resource_id
  path_part   = "test"
}

resource "aws_api_gateway_method" "test" {
  rest_api_id   = aws_api_gateway_rest_api.test.id
  resource_id   = aws_api_gateway_resource.test.id
  http_method   = "GET"
  authorization = "NONE"

  request_models = {
    "application/json" = "Error"
  }

  request_parameters = {
    "method.request.path.param" = true
  }
}

resource "aws_api_gateway_integration" "test" {
  rest_api_id = aws_api_gateway_rest_api.test.id
  resource_id = aws_api_gateway_resource.test.id
  http_method = aws_api_gateway_method.test.http_method

  type                    = "HTTP"
  uri                     = "https://www.google.de"
  integration_http_method = "GET"
  passthrough_behavior    = "WHEN_NO_MATCH"
  content_handling        = "CONVERT_TO_TEXT"

  tls_config {
    insecure_skip_verification = %[2]t
  }
}
`, rName, insecureSkipVerification)
}
