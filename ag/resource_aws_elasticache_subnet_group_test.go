package ag

import (
	"fmt"
	"log"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/elasticache"
	"github.com/hashicorp/aws-sdk-go-base/tfawserr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func init() {
	resource.AddTestSweepers("aws_elasticache_subnet_group", &resource.Sweeper{
		Name: "aws_elasticache_subnet_group",
		F:    testSweepElasticacheSubnetGroups,
		Dependencies: []string{
			"aws_elasticache_cluster",
			"aws_elasticache_replication_group",
		},
	})
}

func testSweepElasticacheSubnetGroups(region string) error {
	client, err := sharedClientForRegion(region)
	if err != nil {
		return fmt.Errorf("error getting client: %w", err)
	}
	conn := client.(*AWSClient).elasticacheconn

	err = conn.DescribeCacheSubnetGroupsPages(&elasticache.DescribeCacheSubnetGroupsInput{}, func(page *elasticache.DescribeCacheSubnetGroupsOutput, lastPage bool) bool {
		if len(page.CacheSubnetGroups) == 0 {
			log.Print("[DEBUG] No Elasticache Subnet Groups to sweep")
			return false
		}

		for _, subnetGroup := range page.CacheSubnetGroups {
			name := aws.StringValue(subnetGroup.CacheSubnetGroupName)

			log.Printf("[INFO] Deleting Elasticache Subnet Group: %s", name)
			_, err := conn.DeleteCacheSubnetGroup(&elasticache.DeleteCacheSubnetGroupInput{
				CacheSubnetGroupName: aws.String(name),
			})
			if err != nil {
				log.Printf("[ERROR] Failed to delete Elasticache Subnet Group (%s): %s", name, err)
			}
		}
		return !lastPage
	})
	if err != nil {
		if testSweepSkipSweepError(err) {
			log.Printf("[WARN] Skipping Elasticache Subnet Group sweep for %s: %s", region, err)
			return nil
		}
		return fmt.Errorf("Error retrieving Elasticache Subnet Groups: %w", err)
	}
	return nil
}

func TestAccAWSElasticacheSubnetGroup_basic(t *testing.T) {
	var csg elasticache.CacheSubnetGroup
	resourceName := "aws_elasticache_subnet_group.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, elasticache.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSElasticacheSubnetGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSElasticacheSubnetGroupConfig(acctest.RandInt()),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSElasticacheSubnetGroupExists(resourceName, &csg),
					resource.TestCheckResourceAttr(
						resourceName, "description", "Managed by Terraform"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"description"},
			},
		},
	})
}

func TestAccAWSElasticacheSubnetGroup_update(t *testing.T) {
	var csg elasticache.CacheSubnetGroup
	resourceName := "aws_elasticache_subnet_group.test"
	rInt := acctest.RandInt()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, elasticache.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSElasticacheSubnetGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSElasticacheSubnetGroupUpdateConfigPre(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSElasticacheSubnetGroupExists(resourceName, &csg),
					testAccCheckAWSElastiCacheSubnetGroupAttrs(&csg, resourceName, 1),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"description"},
			},
			{
				Config: testAccAWSElasticacheSubnetGroupUpdateConfigPost(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSElasticacheSubnetGroupExists(resourceName, &csg),
					testAccCheckAWSElastiCacheSubnetGroupAttrs(&csg, resourceName, 2),
				),
			},
		},
	})
}

func TestAccAWSElasticacheSubnetGroup_tags(t *testing.T) {
	var csg elasticache.CacheSubnetGroup
	resourceName := "aws_elasticache_subnet_group.test"
	rInt := acctest.RandInt()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, elasticache.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSElasticacheSubnetGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSElasticacheSubnetGroupTags1(rInt, "key1", "value1"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSElasticacheSubnetGroupExists(resourceName, &csg),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "tags.key1", "value1"),
					resource.TestCheckResourceAttr(resourceName, "tags_all.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "tags_all.key1", "value1"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"description"},
			},
			{
				Config: testAccAWSElasticacheSubnetGroupTags2(rInt, "key1", "value1updated", "key2", "value2"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSElasticacheSubnetGroupExists(resourceName, &csg),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "tags.key1", "value1updated"),
					resource.TestCheckResourceAttr(resourceName, "tags.key2", "value2"),
					resource.TestCheckResourceAttr(resourceName, "tags_all.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "tags_all.key1", "value1updated"),
					resource.TestCheckResourceAttr(resourceName, "tags_all.key2", "value2"),
				),
			},
			{
				Config: testAccAWSElasticacheSubnetGroupTags1(rInt, "key2", "value2"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSElasticacheSubnetGroupExists(resourceName, &csg),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "tags.key2", "value2"),
					resource.TestCheckResourceAttr(resourceName, "tags_all.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "tags_all.key2", "value2"),
				),
			},
		},
	})
}

func testAccCheckAWSElasticacheSubnetGroupDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).elasticacheconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_elasticache_subnet_group" {
			continue
		}
		res, err := conn.DescribeCacheSubnetGroups(&elasticache.DescribeCacheSubnetGroupsInput{
			CacheSubnetGroupName: aws.String(rs.Primary.ID),
		})
		if err != nil {
			// Verify the error is what we want
			if tfawserr.ErrCodeEquals(err, elasticache.ErrCodeCacheSubnetGroupNotFoundFault) {
				continue
			}
			return err
		}
		if len(res.CacheSubnetGroups) > 0 {
			return fmt.Errorf("still exist.")
		}
	}
	return nil
}

func testAccCheckAWSElasticacheSubnetGroupExists(n string, csg *elasticache.CacheSubnetGroup) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No cache subnet group ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).elasticacheconn
		resp, err := conn.DescribeCacheSubnetGroups(&elasticache.DescribeCacheSubnetGroupsInput{
			CacheSubnetGroupName: aws.String(rs.Primary.ID),
		})
		if err != nil {
			return fmt.Errorf("CacheSubnetGroup error: %w", err)
		}

		for _, c := range resp.CacheSubnetGroups {
			if rs.Primary.ID == *c.CacheSubnetGroupName {
				*csg = *c
			}
		}

		if csg == nil {
			return fmt.Errorf("cache subnet group not found")
		}
		return nil
	}
}

func testAccCheckAWSElastiCacheSubnetGroupAttrs(csg *elasticache.CacheSubnetGroup, n string, count int) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if len(csg.Subnets) != count {
			return fmt.Errorf("Bad cache subnet count, expected: %d, got: %d", count, len(csg.Subnets))
		}

		if rs.Primary.Attributes["description"] != *csg.CacheSubnetGroupDescription {
			return fmt.Errorf("Bad cache subnet description, expected: %s, got: %s", rs.Primary.Attributes["description"], *csg.CacheSubnetGroupDescription)
		}

		return nil
	}
}

func testAccAWSElasticacheSubnetGroupConfig(rInt int) string {
	return composeConfig(testAccAvailableAZsNoOptInConfig(), fmt.Sprintf(`
resource "aws_vpc" "foo" {
  cidr_block = "192.168.0.0/16"

  tags = {
    Name = "terraform-testacc-elasticache-subnet-group"
  }
}

resource "aws_subnet" "foo" {
  vpc_id            = aws_vpc.foo.id
  cidr_block        = "192.168.0.0/20"
  availability_zone = data.aws_availability_zones.available.names[0]

  tags = {
    Name = "tf-acc-elasticache-subnet-group"
  }
}

resource "aws_elasticache_subnet_group" "test" {
  # Including uppercase letters in this name to ensure
  # that we correctly handle the fact that the API
  # normalizes names to lowercase.
  name       = "tf-TEST-cache-subnet-%03d"
  subnet_ids = [aws_subnet.foo.id]
}
`, rInt))
}

func testAccAWSElasticacheSubnetGroupUpdateConfigPre(rInt int) string {
	return composeConfig(testAccAvailableAZsNoOptInConfig(), fmt.Sprintf(`
resource "aws_vpc" "foo" {
  cidr_block = "10.0.0.0/16"

  tags = {
    Name = "terraform-testacc-elasticache-subnet-group-update"
  }
}

resource "aws_subnet" "foo" {
  vpc_id            = aws_vpc.foo.id
  cidr_block        = "10.0.1.0/24"
  availability_zone = data.aws_availability_zones.available.names[0]

  tags = {
    Name = "tf-acc-elasticache-subnet-group-update-foo"
  }
}

resource "aws_elasticache_subnet_group" "test" {
  name        = "tf-test-cache-subnet-%03d"
  description = "tf-test-cache-subnet-group-descr"
  subnet_ids  = [aws_subnet.foo.id]
}
`, rInt))
}

func testAccAWSElasticacheSubnetGroupUpdateConfigPost(rInt int) string {
	return composeConfig(testAccAvailableAZsNoOptInConfig(), fmt.Sprintf(`
resource "aws_vpc" "foo" {
  cidr_block = "10.0.0.0/16"

  tags = {
    Name = "terraform-testacc-elasticache-subnet-group-update"
  }
}

resource "aws_subnet" "foo" {
  vpc_id            = aws_vpc.foo.id
  cidr_block        = "10.0.1.0/24"
  availability_zone = data.aws_availability_zones.available.names[0]

  tags = {
    Name = "tf-acc-elasticache-subnet-group-update-foo"
  }
}

resource "aws_subnet" "test" {
  vpc_id            = aws_vpc.foo.id
  cidr_block        = "10.0.2.0/24"
  availability_zone = data.aws_availability_zones.available.names[0]

  tags = {
    Name = "tf-acc-elasticache-subnet-group-update-test"
  }
}

resource "aws_elasticache_subnet_group" "test" {
  name        = "tf-test-cache-subnet-%03d"
  description = "tf-test-cache-subnet-group-descr-edited"
  subnet_ids = [
    aws_subnet.foo.id,
    aws_subnet.test.id,
  ]
}
`, rInt))
}

func testAccAWSElasticacheSubnetGroupTags1(rInt int, tag1Key, tag1Value string) string {
	return composeConfig(testAccAvailableAZsNoOptInConfig(), fmt.Sprintf(`
resource "aws_vpc" "foo" {
  cidr_block = "192.168.0.0/16"

  tags = {
    Name = "terraform-testacc-elasticache-subnet-group"
  }
}

resource "aws_subnet" "foo" {
  vpc_id            = aws_vpc.foo.id
  cidr_block        = "192.168.0.0/20"
  availability_zone = data.aws_availability_zones.available.names[0]

  tags = {
    Name = "tf-acc-elasticache-subnet-group"
  }
}

resource "aws_elasticache_subnet_group" "test" {
  # Including uppercase letters in this name to ensure
  # that we correctly handle the fact that the API
  # normalizes names to lowercase.
  name       = "tf-TEST-cache-subnet-%03d"
  subnet_ids = [aws_subnet.foo.id]

  tags = {
    %q = %q
  }
}
`, rInt, tag1Key, tag1Value))
}

func testAccAWSElasticacheSubnetGroupTags2(rInt int, tag1Key, tag1Value, tag2Key, tag2Value string) string {
	return composeConfig(testAccAvailableAZsNoOptInConfig(), fmt.Sprintf(`
resource "aws_vpc" "foo" {
  cidr_block = "192.168.0.0/16"

  tags = {
    Name = "terraform-testacc-elasticache-subnet-group"
  }
}

resource "aws_subnet" "foo" {
  vpc_id            = aws_vpc.foo.id
  cidr_block        = "192.168.0.0/20"
  availability_zone = data.aws_availability_zones.available.names[0]

  tags = {
    Name = "tf-acc-elasticache-subnet-group"
  }
}

resource "aws_elasticache_subnet_group" "test" {
  # Including uppercase letters in this name to ensure
  # that we correctly handle the fact that the API
  # normalizes names to lowercase.
  name       = "tf-TEST-cache-subnet-%03d"
  subnet_ids = [aws_subnet.foo.id]

  tags = {
    %q = %q
    %q = %q
  }
}
`, rInt, tag1Key, tag1Value, tag2Key, tag2Value))
}
