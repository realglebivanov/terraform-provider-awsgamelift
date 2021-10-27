package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/aws-sdk-go-base/tfawserr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	tfec2 "github.com/realglebivanov/terraform-provider-aws-gamelift/aws/internal/service/ec2"
	"github.com/realglebivanov/terraform-provider-aws-gamelift/aws/internal/service/ec2/finder"
)

func TestAccAWSNetworkInterfaceSGAttachment_basic(t *testing.T) {
	networkInterfaceResourceName := "aws_network_interface.test"
	securityGroupResourceName := "aws_security_group.test"
	resourceName := "aws_network_interface_sg_attachment.test"
	rName := acctest.RandomWithPrefix("tf-acc-test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, ec2.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSNetworkInterfaceSGAttachmentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAwsNetworkInterfaceSGAttachmentConfig(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSNetworkInterfaceSGAttachmentExists(resourceName),
					resource.TestCheckResourceAttrPair(resourceName, "network_interface_id", networkInterfaceResourceName, "id"),
					resource.TestCheckResourceAttrPair(resourceName, "security_group_id", securityGroupResourceName, "id"),
				),
			},
		},
	})
}

func TestAccAWSNetworkInterfaceSGAttachment_disappears(t *testing.T) {
	resourceName := "aws_network_interface_sg_attachment.test"
	rName := acctest.RandomWithPrefix("tf-acc-test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, ec2.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSNetworkInterfaceSGAttachmentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAwsNetworkInterfaceSGAttachmentConfig(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSNetworkInterfaceSGAttachmentExists(resourceName),
					testAccCheckResourceDisappears(testAccProvider, resourceAwsNetworkInterfaceSGAttachment(), resourceName),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccAWSNetworkInterfaceSGAttachment_Instance(t *testing.T) {
	instanceResourceName := "aws_instance.test"
	securityGroupResourceName := "aws_security_group.test"
	resourceName := "aws_network_interface_sg_attachment.test"
	rName := acctest.RandomWithPrefix("tf-acc-test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, ec2.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSNetworkInterfaceSGAttachmentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAwsNetworkInterfaceSGAttachmentConfigViaInstance(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSNetworkInterfaceSGAttachmentExists(resourceName),
					resource.TestCheckResourceAttrPair(resourceName, "network_interface_id", instanceResourceName, "primary_network_interface_id"),
					resource.TestCheckResourceAttrPair(resourceName, "security_group_id", securityGroupResourceName, "id"),
				),
			},
		},
	})
}

func TestAccAWSNetworkInterfaceSGAttachment_DataSource(t *testing.T) {
	instanceDataSourceName := "data.aws_instance.test"
	securityGroupResourceName := "aws_security_group.test"
	resourceName := "aws_network_interface_sg_attachment.test"
	rName := acctest.RandomWithPrefix("tf-acc-test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, ec2.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSNetworkInterfaceSGAttachmentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAwsNetworkInterfaceSGAttachmentConfigViaDataSource(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSNetworkInterfaceSGAttachmentExists(resourceName),
					resource.TestCheckResourceAttrPair(resourceName, "network_interface_id", instanceDataSourceName, "network_interface_id"),
					resource.TestCheckResourceAttrPair(resourceName, "security_group_id", securityGroupResourceName, "id"),
				),
			},
		},
	})
}

func TestAccAWSNetworkInterfaceSGAttachment_Multiple(t *testing.T) {
	networkInterfaceResourceName := "aws_network_interface.test"
	securityGroupResourceName1 := "aws_security_group.test.0"
	securityGroupResourceName2 := "aws_security_group.test.1"
	securityGroupResourceName3 := "aws_security_group.test.2"
	securityGroupResourceName4 := "aws_security_group.test.3"
	resourceName1 := "aws_network_interface_sg_attachment.test.0"
	resourceName2 := "aws_network_interface_sg_attachment.test.1"
	resourceName3 := "aws_network_interface_sg_attachment.test.2"
	resourceName4 := "aws_network_interface_sg_attachment.test.3"
	rName := acctest.RandomWithPrefix("tf-acc-test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, ec2.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSNetworkInterfaceSGAttachmentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAwsNetworkInterfaceSGAttachmentConfigMultiple(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSNetworkInterfaceSGAttachmentExists(resourceName1),
					resource.TestCheckResourceAttrPair(resourceName1, "network_interface_id", networkInterfaceResourceName, "id"),
					resource.TestCheckResourceAttrPair(resourceName1, "security_group_id", securityGroupResourceName1, "id"),
					testAccCheckAWSNetworkInterfaceSGAttachmentExists(resourceName2),
					resource.TestCheckResourceAttrPair(resourceName2, "network_interface_id", networkInterfaceResourceName, "id"),
					resource.TestCheckResourceAttrPair(resourceName2, "security_group_id", securityGroupResourceName2, "id"),
					testAccCheckAWSNetworkInterfaceSGAttachmentExists(resourceName3),
					resource.TestCheckResourceAttrPair(resourceName3, "network_interface_id", networkInterfaceResourceName, "id"),
					resource.TestCheckResourceAttrPair(resourceName3, "security_group_id", securityGroupResourceName3, "id"),
					testAccCheckAWSNetworkInterfaceSGAttachmentExists(resourceName4),
					resource.TestCheckResourceAttrPair(resourceName4, "network_interface_id", networkInterfaceResourceName, "id"),
					resource.TestCheckResourceAttrPair(resourceName4, "security_group_id", securityGroupResourceName4, "id"),
				),
			},
		},
	})
}

func testAccCheckAWSNetworkInterfaceSGAttachmentExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("Not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No resource ID set: %s", resourceName)
		}

		conn := testAccProvider.Meta().(*AWSClient).ec2conn
		networkInterfaceID := rs.Primary.Attributes["network_interface_id"]
		securityGroupID := rs.Primary.Attributes["security_group_id"]

		groupIdentifier, err := finder.NetworkInterfaceSecurityGroup(conn, networkInterfaceID, securityGroupID)

		if err != nil {
			return fmt.Errorf("error reading EC2 Network Interface Security Group Attachment (%s): %s", rs.Primary.ID, err)
		}

		if groupIdentifier == nil {
			return fmt.Errorf("Security Group ID (%s) not attached to ENI (%s)", securityGroupID, networkInterfaceID)
		}

		return nil
	}
}

func testAccCheckAWSNetworkInterfaceSGAttachmentDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).ec2conn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_network_interface_sg_attachment" {
			continue
		}

		networkInterfaceID := rs.Primary.Attributes["network_interface_id"]
		securityGroupID := rs.Primary.Attributes["security_group_id"]

		groupIdentifier, err := finder.NetworkInterfaceSecurityGroup(conn, networkInterfaceID, securityGroupID)

		if tfawserr.ErrCodeEquals(err, tfec2.ErrCodeInvalidNetworkInterfaceIDNotFound) {
			continue
		}

		if err != nil {
			return fmt.Errorf("error reading EC2 Network Interface Security Group Attachment (%s): %s", rs.Primary.ID, err)
		}

		if groupIdentifier != nil {
			return fmt.Errorf("Security Group ID (%s) still attached to ENI (%s)", securityGroupID, networkInterfaceID)
		}
	}

	return nil
}

func testAccAwsNetworkInterfaceSGAttachmentConfig(rName string) string {
	return fmt.Sprintf(`
resource "aws_vpc" "test" {
  cidr_block = "172.16.0.0/16"

  tags = {
    Name = %q
  }
}

resource "aws_subnet" "test" {
  cidr_block = "172.16.10.0/24"
  vpc_id     = aws_vpc.test.id

  tags = {
    Name = %q
  }
}

resource "aws_security_group" "test" {
  name   = %q
  vpc_id = aws_vpc.test.id
}

resource "aws_network_interface" "test" {
  subnet_id = aws_subnet.test.id

  tags = {
    Name = %q
  }
}

resource "aws_network_interface_sg_attachment" "test" {
  network_interface_id = aws_network_interface.test.id
  security_group_id    = aws_security_group.test.id
}
`, rName, rName, rName, rName)
}

func testAccAwsNetworkInterfaceSGAttachmentConfigViaInstance(rName string) string {
	return fmt.Sprintf(`
data "aws_ami" "ami" {
  most_recent = true

  filter {
    name   = "name"
    values = ["amzn-ami-hvm-*"]
  }

  owners = ["amazon"]
}

resource "aws_instance" "test" {
  instance_type = "t2.micro"
  ami           = data.aws_ami.ami.id

  tags = {
    Name = %q
  }
}

resource "aws_security_group" "test" {
  name = %q
}

resource "aws_network_interface_sg_attachment" "test" {
  network_interface_id = aws_instance.test.primary_network_interface_id
  security_group_id    = aws_security_group.test.id
}
`, rName, rName)
}

func testAccAwsNetworkInterfaceSGAttachmentConfigViaDataSource(rName string) string {
	return fmt.Sprintf(`
data "aws_ami" "ami" {
  most_recent = true

  filter {
    name   = "name"
    values = ["amzn-ami-hvm-*"]
  }

  owners = ["amazon"]
}

resource "aws_instance" "test" {
  instance_type = "t2.micro"
  ami           = data.aws_ami.ami.id

  tags = {
    Name = %q
  }
}

data "aws_instance" "test" {
  instance_id = aws_instance.test.id
}

resource "aws_security_group" "test" {
  name = %q
}

resource "aws_network_interface_sg_attachment" "test" {
  security_group_id    = aws_security_group.test.id
  network_interface_id = data.aws_instance.test.network_interface_id
}
`, rName, rName)
}

func testAccAwsNetworkInterfaceSGAttachmentConfigMultiple(rName string) string {
	return fmt.Sprintf(`
data "aws_availability_zones" "available" {
  state = "available"

  filter {
    name   = "opt-in-status"
    values = ["opt-in-not-required"]
  }
}

data "aws_subnet" "test" {
  availability_zone = data.aws_availability_zones.available.names[0]
  default_for_az    = "true"
}

resource "aws_network_interface" "test" {
  subnet_id = data.aws_subnet.test.id

  tags = {
    Name = %q
  }
}

resource "aws_security_group" "test" {
  count = 4
  name  = "%s-${count.index}"
}

resource "aws_network_interface_sg_attachment" "test" {
  count                = 4
  network_interface_id = aws_network_interface.test.id
  security_group_id    = aws_security_group.test.*.id[count.index]
}
`, rName, rName)
}
