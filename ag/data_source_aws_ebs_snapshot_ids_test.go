package ag

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceAwsEbsSnapshotIds_basic(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:   func() { testAccPreCheck(t) },
		ErrorCheck: testAccErrorCheck(t, ec2.EndpointsID),
		Providers:  testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceAwsEbsSnapshotIdsConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsEbsSnapshotDataSourceID("data.aws_ebs_snapshot_ids.test"),
				),
			},
		},
	})
}

func TestAccDataSourceAwsEbsSnapshotIds_sorted(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:   func() { testAccPreCheck(t) },
		ErrorCheck: testAccErrorCheck(t, ec2.EndpointsID),
		Providers:  testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceAwsEbsSnapshotIdsConfig_sorted1(rName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("aws_ebs_snapshot.a", "id"),
					resource.TestCheckResourceAttrSet("aws_ebs_snapshot.b", "id"),
				),
			},
			{
				Config: testAccDataSourceAwsEbsSnapshotIdsConfig_sorted2(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsEbsSnapshotDataSourceID("data.aws_ebs_snapshot_ids.test"),
					resource.TestCheckResourceAttr("data.aws_ebs_snapshot_ids.test", "ids.#", "2"),
					resource.TestCheckResourceAttrPair(
						"data.aws_ebs_snapshot_ids.test", "ids.0",
						"aws_ebs_snapshot.b", "id"),
					resource.TestCheckResourceAttrPair(
						"data.aws_ebs_snapshot_ids.test", "ids.1",
						"aws_ebs_snapshot.a", "id"),
				),
			},
		},
	})
}

func TestAccDataSourceAwsEbsSnapshotIds_empty(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:   func() { testAccPreCheck(t) },
		ErrorCheck: testAccErrorCheck(t, ec2.EndpointsID),
		Providers:  testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceAwsEbsSnapshotIdsConfig_empty,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsEbsSnapshotDataSourceID("data.aws_ebs_snapshot_ids.empty"),
					resource.TestCheckResourceAttr("data.aws_ebs_snapshot_ids.empty", "ids.#", "0"),
				),
			},
		},
	})
}

func testAccDataSourceAwsEbsSnapshotIdsConfig_basic() string {
	return composeConfig(testAccAvailableAZsNoOptInConfig(), `
resource "aws_ebs_volume" "test" {
  availability_zone = data.aws_availability_zones.available.names[0]
  size              = 1
}

resource "aws_ebs_snapshot" "test" {
  volume_id = aws_ebs_volume.test.id
}

data "aws_ebs_snapshot_ids" "test" {
  owners = ["self"]
}
`)
}

func testAccDataSourceAwsEbsSnapshotIdsConfig_sorted1(rName string) string {
	return composeConfig(testAccAvailableAZsNoOptInConfig(), fmt.Sprintf(`
resource "aws_ebs_volume" "test" {
  availability_zone = data.aws_availability_zones.available.names[0]
  size              = 1

  count = 2
}

resource "aws_ebs_snapshot" "a" {
  volume_id   = aws_ebs_volume.test.*.id[0]
  description = %[1]q
}

resource "aws_ebs_snapshot" "b" {
  volume_id   = aws_ebs_volume.test.*.id[1]
  description = %[1]q

  # We want to ensure that 'aws_ebs_snapshot.a.creation_date' is less than
  # 'aws_ebs_snapshot.b.creation_date'/ so that we can ensure that the
  # snapshots are being sorted correctly.
  depends_on = [aws_ebs_snapshot.a]
}
`, rName))
}

func testAccDataSourceAwsEbsSnapshotIdsConfig_sorted2(rName string) string {
	return composeConfig(testAccDataSourceAwsEbsSnapshotIdsConfig_sorted1(rName), fmt.Sprintf(`
data "aws_ebs_snapshot_ids" "test" {
  owners = ["self"]

  filter {
    name   = "description"
    values = [%q]
  }
}
`, rName))
}

const testAccDataSourceAwsEbsSnapshotIdsConfig_empty = `
data "aws_ebs_snapshot_ids" "empty" {
  owners = ["000000000000"]
}
`
