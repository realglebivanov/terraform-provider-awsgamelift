package ag

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/service/globalaccelerator"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/realglebivanov/terraform-provider-awsgamelift/ag/internal/service/globalaccelerator/finder"
	"github.com/realglebivanov/terraform-provider-awsgamelift/ag/internal/tfresource"
)

func TestAccAwsGlobalAcceleratorListener_basic(t *testing.T) {
	resourceName := "aws_globalaccelerator_listener.example"
	rInt := acctest.RandInt()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t); testAccPreCheckGlobalAccelerator(t) },
		ErrorCheck:   testAccErrorCheck(t, globalaccelerator.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckGlobalAcceleratorListenerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGlobalAcceleratorListener_basic(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGlobalAcceleratorListenerExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "client_affinity", "NONE"),
					resource.TestCheckResourceAttr(resourceName, "protocol", "TCP"),
					resource.TestCheckResourceAttr(resourceName, "port_range.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "port_range.*", map[string]string{
						"from_port": "80",
						"to_port":   "81",
					}),
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

func TestAccAwsGlobalAcceleratorListener_disappears(t *testing.T) {
	resourceName := "aws_globalaccelerator_listener.example"
	rInt := acctest.RandInt()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t); testAccPreCheckGlobalAccelerator(t) },
		ErrorCheck:   testAccErrorCheck(t, globalaccelerator.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckGlobalAcceleratorListenerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGlobalAcceleratorListener_basic(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGlobalAcceleratorListenerExists(resourceName),
					testAccCheckResourceDisappears(testAccProvider, resourceAwsGlobalAcceleratorListener(), resourceName),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccAwsGlobalAcceleratorListener_update(t *testing.T) {
	resourceName := "aws_globalaccelerator_listener.example"
	rInt := acctest.RandInt()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t); testAccPreCheckGlobalAccelerator(t) },
		ErrorCheck:   testAccErrorCheck(t, globalaccelerator.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckGlobalAcceleratorListenerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGlobalAcceleratorListener_basic(rInt),
			},
			{
				Config: testAccGlobalAcceleratorListener_update(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGlobalAcceleratorListenerExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "client_affinity", "SOURCE_IP"),
					resource.TestCheckResourceAttr(resourceName, "protocol", "UDP"),
					resource.TestCheckResourceAttr(resourceName, "port_range.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "port_range.*", map[string]string{
						"from_port": "443",
						"to_port":   "444",
					}),
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

func testAccCheckGlobalAcceleratorListenerExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := testAccProvider.Meta().(*AWSClient).globalacceleratorconn

		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		_, err := finder.ListenerByARN(conn, rs.Primary.ID)

		if err != nil {
			return err
		}

		return nil
	}
}

func testAccCheckGlobalAcceleratorListenerDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).globalacceleratorconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_globalaccelerator_listener" {
			continue
		}

		_, err := finder.ListenerByARN(conn, rs.Primary.ID)

		if tfresource.NotFound(err) {
			continue
		}

		if err != nil {
			return err
		}

		return fmt.Errorf("Global Accelerator Accelerator %s still exists", rs.Primary.ID)
	}
	return nil
}

func testAccGlobalAcceleratorListener_basic(rInt int) string {
	return fmt.Sprintf(`
resource "aws_globalaccelerator_accelerator" "example" {
  name            = "tf-%d"
  ip_address_type = "IPV4"
  enabled         = false
}

resource "aws_globalaccelerator_listener" "example" {
  accelerator_arn = aws_globalaccelerator_accelerator.example.id
  protocol        = "TCP"

  port_range {
    from_port = 80
    to_port   = 81
  }
}
`, rInt)
}

func testAccGlobalAcceleratorListener_update(rInt int) string {
	return fmt.Sprintf(`
resource "aws_globalaccelerator_accelerator" "example" {
  name            = "tf-%d"
  ip_address_type = "IPV4"
  enabled         = false
}

resource "aws_globalaccelerator_listener" "example" {
  accelerator_arn = aws_globalaccelerator_accelerator.example.id
  client_affinity = "SOURCE_IP"
  protocol        = "UDP"

  port_range {
    from_port = 443
    to_port   = 444
  }
}
`, rInt)
}
