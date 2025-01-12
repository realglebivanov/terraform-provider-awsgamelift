package ag

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go/service/efs"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccDataSourceAwsEfsFileSystem_id(t *testing.T) {
	dataSourceName := "data.aws_efs_file_system.test"
	resourceName := "aws_efs_file_system.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:   func() { testAccPreCheck(t) },
		ErrorCheck: testAccErrorCheck(t, efs.EndpointsID),
		Providers:  testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceAwsEfsFileSystemIDConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceAwsEfsFileSystemCheck(dataSourceName, resourceName),
					resource.TestCheckResourceAttrPair(dataSourceName, "arn", resourceName, "arn"),
					resource.TestCheckResourceAttrPair(dataSourceName, "performance_mode", resourceName, "performance_mode"),
					resource.TestCheckResourceAttrPair(dataSourceName, "creation_token", resourceName, "creation_token"),
					resource.TestCheckResourceAttrPair(dataSourceName, "encrypted", resourceName, "encrypted"),
					resource.TestCheckResourceAttrPair(dataSourceName, "kms_key_id", resourceName, "kms_key_id"),
					resource.TestCheckResourceAttrPair(dataSourceName, "tags", resourceName, "tags"),
					resource.TestCheckResourceAttrPair(dataSourceName, "dns_name", resourceName, "dns_name"),
					resource.TestCheckResourceAttrPair(dataSourceName, "provisioned_throughput_in_mibps", resourceName, "provisioned_throughput_in_mibps"),
					resource.TestCheckResourceAttrPair(dataSourceName, "throughput_mode", resourceName, "throughput_mode"),
					resource.TestCheckResourceAttrPair(dataSourceName, "lifecycle_policy", resourceName, "lifecycle_policy"),
					resource.TestMatchResourceAttr(dataSourceName, "size_in_bytes", regexp.MustCompile(`^\d+$`)),
				),
			},
		},
	})
}

func TestAccDataSourceAwsEfsFileSystem_name(t *testing.T) {
	dataSourceName := "data.aws_efs_file_system.test"
	resourceName := "aws_efs_file_system.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:   func() { testAccPreCheck(t) },
		ErrorCheck: testAccErrorCheck(t, efs.EndpointsID),
		Providers:  testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceAwsEfsFileSystemNameConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceAwsEfsFileSystemCheck(dataSourceName, resourceName),
					resource.TestCheckResourceAttrPair(dataSourceName, "arn", resourceName, "arn"),
					resource.TestCheckResourceAttrPair(dataSourceName, "performance_mode", resourceName, "performance_mode"),
					resource.TestCheckResourceAttrPair(dataSourceName, "creation_token", resourceName, "creation_token"),
					resource.TestCheckResourceAttrPair(dataSourceName, "encrypted", resourceName, "encrypted"),
					resource.TestCheckResourceAttrPair(dataSourceName, "kms_key_id", resourceName, "kms_key_id"),
					resource.TestCheckResourceAttrPair(dataSourceName, "tags", resourceName, "tags"),
					resource.TestCheckResourceAttrPair(dataSourceName, "dns_name", resourceName, "dns_name"),
					resource.TestCheckResourceAttrPair(dataSourceName, "provisioned_throughput_in_mibps", resourceName, "provisioned_throughput_in_mibps"),
					resource.TestCheckResourceAttrPair(dataSourceName, "throughput_mode", resourceName, "throughput_mode"),
					resource.TestCheckResourceAttrPair(dataSourceName, "lifecycle_policy", resourceName, "lifecycle_policy"),
					resource.TestMatchResourceAttr(dataSourceName, "size_in_bytes", regexp.MustCompile(`^\d+$`)),
				),
			},
		},
	})
}

func TestAccDataSourceAwsEfsFileSystem_availabilityZone(t *testing.T) {
	dataSourceName := "data.aws_efs_file_system.test"
	resourceName := "aws_efs_file_system.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:   func() { testAccPreCheck(t) },
		ErrorCheck: testAccErrorCheck(t, efs.EndpointsID),
		Providers:  testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceAwsEfsFileSystemAvailabilityZoneConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceAwsEfsFileSystemCheck(dataSourceName, resourceName),
					resource.TestCheckResourceAttrPair(dataSourceName, "availability_zone_id", resourceName, "availability_zone_id"),
					resource.TestCheckResourceAttrPair(dataSourceName, "availability_zone_name", resourceName, "availability_zone_name"),
				),
			},
		},
	})
}

func TestAccDataSourceAwsEfsFileSystem_NonExistent(t *testing.T) {

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:   func() { testAccPreCheck(t) },
		ErrorCheck: testAccErrorCheck(t, efs.EndpointsID),
		Providers:  testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      testAccDataSourceAwsEfsFileSystemIDConfig_NonExistent,
				ExpectError: regexp.MustCompile(`error reading EFS FileSystem`),
			},
		},
	})
}

func testAccDataSourceAwsEfsFileSystemCheck(dName, rName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[dName]
		if !ok {
			return fmt.Errorf("root module has no resource called %s", dName)
		}

		efsRs, ok := s.RootModule().Resources[rName]
		if !ok {
			return fmt.Errorf("can't find aws_efs_file_system.test in state")
		}

		attr := rs.Primary.Attributes

		if attr["creation_token"] != efsRs.Primary.Attributes["creation_token"] {
			return fmt.Errorf(
				"creation_token is %s; want %s",
				attr["creation_token"],
				efsRs.Primary.Attributes["creation_token"],
			)
		}

		if attr["id"] != efsRs.Primary.Attributes["id"] {
			return fmt.Errorf(
				"file_system_id is %s; want %s",
				attr["id"],
				efsRs.Primary.Attributes["id"],
			)
		}

		return nil
	}
}

const testAccDataSourceAwsEfsFileSystemIDConfig_NonExistent = `
data "aws_efs_file_system" "test" {
  file_system_id = "fs-nonexistent"
}
`

const testAccDataSourceAwsEfsFileSystemNameConfig = `
resource "aws_efs_file_system" "test" {}

data "aws_efs_file_system" "test" {
  creation_token = aws_efs_file_system.test.creation_token
}
`

const testAccDataSourceAwsEfsFileSystemIDConfig = `
resource "aws_efs_file_system" "test" {}

data "aws_efs_file_system" "test" {
  file_system_id = aws_efs_file_system.test.id
}
`

const testAccDataSourceAwsEfsFileSystemAvailabilityZoneConfig = `
data "aws_availability_zones" "available" {
  state = "available"

  filter {
    name   = "opt-in-status"
    values = ["opt-in-not-required"]
  }
}

resource "aws_efs_file_system" "test" {
  availability_zone_name = data.aws_availability_zones.available.names[0]
}

data "aws_efs_file_system" "test" {
  file_system_id = aws_efs_file_system.test.id
}
`
