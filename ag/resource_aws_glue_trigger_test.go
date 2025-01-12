package ag

import (
	"fmt"
	"log"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/glue"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/realglebivanov/terraform-provider-awsgamelift/ag/internal/service/glue/finder"
)

func init() {
	resource.AddTestSweepers("aws_glue_trigger", &resource.Sweeper{
		Name: "aws_glue_trigger",
		F:    testSweepGlueTriggers,
	})
}

func testSweepGlueTriggers(region string) error {
	client, err := sharedClientForRegion(region)
	if err != nil {
		return fmt.Errorf("error getting client: %s", err)
	}
	conn := client.(*AWSClient).glueconn

	input := &glue.GetTriggersInput{}
	err = conn.GetTriggersPages(input, func(page *glue.GetTriggersOutput, lastPage bool) bool {
		if page == nil || len(page.Triggers) == 0 {
			log.Printf("[INFO] No Glue Triggers to sweep")
			return false
		}
		for _, trigger := range page.Triggers {
			name := aws.StringValue(trigger.Name)

			log.Printf("[INFO] Deleting Glue Trigger: %s", name)
			r := resourceAwsGlueTrigger()
			d := r.Data(nil)
			d.SetId(name)
			err := r.Delete(d, client)
			if err != nil {
				log.Printf("[ERROR] Failed to delete Glue Trigger %s: %s", name, err)
			}
		}
		return !lastPage
	})
	if err != nil {
		if testSweepSkipSweepError(err) {
			log.Printf("[WARN] Skipping Glue Trigger sweep for %s: %s", region, err)
			return nil
		}
		return fmt.Errorf("Error retrieving Glue Triggers: %s", err)
	}

	return nil
}

func TestAccAWSGlueTrigger_basic(t *testing.T) {
	var trigger glue.Trigger

	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "aws_glue_trigger.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, glue.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSGlueTriggerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSGlueTriggerConfig_OnDemand(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSGlueTriggerExists(resourceName, &trigger),
					resource.TestCheckResourceAttr(resourceName, "actions.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "actions.0.job_name", rName),
					resource.TestCheckResourceAttr(resourceName, "actions.0.notification_property.#", "0"),
					testAccCheckResourceAttrRegionalARN(resourceName, "arn", "glue", fmt.Sprintf("trigger/%s", rName)),
					resource.TestCheckResourceAttr(resourceName, "description", ""),
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "predicate.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "schedule", ""),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "type", "ON_DEMAND"),
					resource.TestCheckResourceAttr(resourceName, "workflow_name", ""),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"enabled"},
			},
		},
	})
}

func TestAccAWSGlueTrigger_Crawler(t *testing.T) {
	var trigger glue.Trigger

	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "aws_glue_trigger.test_trigger"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, glue.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSGlueTriggerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSGlueTriggerConfig_Crawler(rName, "SUCCEEDED"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSGlueTriggerExists(resourceName, &trigger),
					resource.TestCheckResourceAttr(resourceName, "actions.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "actions.0.crawler_name", rName),
					resource.TestCheckResourceAttr(resourceName, "predicate.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "predicate.0.conditions.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "predicate.0.conditions.0.crawler_name", fmt.Sprintf("%scrawl2", rName)),
					resource.TestCheckResourceAttr(resourceName, "predicate.0.conditions.0.crawl_state", "SUCCEEDED"),
					resource.TestCheckResourceAttr(resourceName, "type", "CONDITIONAL"),
				),
			},
			{
				Config: testAccAWSGlueTriggerConfig_Crawler(rName, "FAILED"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSGlueTriggerExists(resourceName, &trigger),
					resource.TestCheckResourceAttr(resourceName, "actions.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "actions.0.crawler_name", rName),
					resource.TestCheckResourceAttr(resourceName, "predicate.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "predicate.0.conditions.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "predicate.0.conditions.0.crawler_name", fmt.Sprintf("%scrawl2", rName)),
					resource.TestCheckResourceAttr(resourceName, "predicate.0.conditions.0.crawl_state", "FAILED"),
					resource.TestCheckResourceAttr(resourceName, "type", "CONDITIONAL"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"enabled"},
			},
		},
	})
}

func TestAccAWSGlueTrigger_Description(t *testing.T) {
	var trigger glue.Trigger

	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "aws_glue_trigger.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, glue.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSGlueTriggerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSGlueTriggerConfig_Description(rName, "description1"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSGlueTriggerExists(resourceName, &trigger),
					resource.TestCheckResourceAttr(resourceName, "description", "description1"),
				),
			},
			{
				Config: testAccAWSGlueTriggerConfig_Description(rName, "description2"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSGlueTriggerExists(resourceName, &trigger),
					resource.TestCheckResourceAttr(resourceName, "description", "description2"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"enabled"},
			},
		},
	})
}

func TestAccAWSGlueTrigger_Enabled(t *testing.T) {
	var trigger glue.Trigger

	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "aws_glue_trigger.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, glue.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSGlueTriggerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSGlueTriggerConfig_Enabled(rName, true),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSGlueTriggerExists(resourceName, &trigger),
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
				),
			},
			{
				Config: testAccAWSGlueTriggerConfig_Enabled(rName, false),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSGlueTriggerExists(resourceName, &trigger),
					resource.TestCheckResourceAttr(resourceName, "enabled", "false"),
				),
			},
			{
				Config: testAccAWSGlueTriggerConfig_Enabled(rName, true),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSGlueTriggerExists(resourceName, &trigger),
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"enabled"},
			},
		},
	})
}

func TestAccAWSGlueTrigger_Predicate(t *testing.T) {
	var trigger glue.Trigger

	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "aws_glue_trigger.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, glue.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSGlueTriggerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSGlueTriggerConfig_Predicate(rName, "SUCCEEDED"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSGlueTriggerExists(resourceName, &trigger),
					resource.TestCheckResourceAttr(resourceName, "predicate.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "predicate.0.conditions.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "predicate.0.conditions.0.job_name", rName),
					resource.TestCheckResourceAttr(resourceName, "predicate.0.conditions.0.state", "SUCCEEDED"),
					resource.TestCheckResourceAttr(resourceName, "type", "CONDITIONAL"),
				),
			},
			{
				Config: testAccAWSGlueTriggerConfig_Predicate(rName, "FAILED"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSGlueTriggerExists(resourceName, &trigger),
					resource.TestCheckResourceAttr(resourceName, "predicate.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "predicate.0.conditions.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "predicate.0.conditions.0.job_name", rName),
					resource.TestCheckResourceAttr(resourceName, "predicate.0.conditions.0.state", "FAILED"),
					resource.TestCheckResourceAttr(resourceName, "type", "CONDITIONAL"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"enabled"},
			},
		},
	})
}

func TestAccAWSGlueTrigger_Schedule(t *testing.T) {
	var trigger glue.Trigger

	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "aws_glue_trigger.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, glue.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSGlueTriggerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSGlueTriggerConfig_Schedule(rName, "cron(1 2 * * ? *)"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSGlueTriggerExists(resourceName, &trigger),
					resource.TestCheckResourceAttr(resourceName, "schedule", "cron(1 2 * * ? *)"),
				),
			},
			{
				Config: testAccAWSGlueTriggerConfig_Schedule(rName, "cron(2 3 * * ? *)"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSGlueTriggerExists(resourceName, &trigger),
					resource.TestCheckResourceAttr(resourceName, "schedule", "cron(2 3 * * ? *)"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"enabled"},
			},
		},
	})
}

func TestAccAWSGlueTrigger_Tags(t *testing.T) {
	var trigger1, trigger2, trigger3 glue.Trigger

	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "aws_glue_trigger.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, glue.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSGlueTriggerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSGlueTriggerConfigTags1(rName, "key1", "value1"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSGlueTriggerExists(resourceName, &trigger1),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "tags.key1", "value1"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"enabled"},
			},
			{
				Config: testAccAWSGlueTriggerConfigTags2(rName, "key1", "value1updated", "key2", "value2"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSGlueTriggerExists(resourceName, &trigger2),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "tags.key1", "value1updated"),
					resource.TestCheckResourceAttr(resourceName, "tags.key2", "value2"),
				),
			},
			{
				Config: testAccAWSGlueTriggerConfigTags1(rName, "key2", "value2"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSGlueTriggerExists(resourceName, &trigger3),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "tags.key2", "value2"),
				),
			},
		},
	})
}

func TestAccAWSGlueTrigger_WorkflowName(t *testing.T) {
	var trigger glue.Trigger

	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "aws_glue_trigger.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, glue.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSGlueTriggerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSGlueTriggerConfig_WorkflowName(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSGlueTriggerExists(resourceName, &trigger),
					resource.TestCheckResourceAttr(resourceName, "workflow_name", rName),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"enabled"},
			},
		},
	})
}

func TestAccAWSGlueTrigger_actions_notify(t *testing.T) {
	var trigger glue.Trigger

	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "aws_glue_trigger.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, glue.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSGlueTriggerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSGlueTriggerConfigActionsNotification(rName, 1),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSGlueTriggerExists(resourceName, &trigger),
					resource.TestCheckResourceAttr(resourceName, "actions.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "actions.0.job_name", rName),
					resource.TestCheckResourceAttr(resourceName, "actions.0.notification_property.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "actions.0.notification_property.0.notify_delay_after", "1"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"enabled"},
			},
			{
				Config: testAccAWSGlueTriggerConfigActionsNotification(rName, 2),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSGlueTriggerExists(resourceName, &trigger),
					resource.TestCheckResourceAttr(resourceName, "actions.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "actions.0.job_name", rName),
					resource.TestCheckResourceAttr(resourceName, "actions.0.notification_property.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "actions.0.notification_property.0.notify_delay_after", "2"),
				),
			},
			{
				Config: testAccAWSGlueTriggerConfigActionsNotification(rName, 1),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSGlueTriggerExists(resourceName, &trigger),
					resource.TestCheckResourceAttr(resourceName, "actions.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "actions.0.job_name", rName),
					resource.TestCheckResourceAttr(resourceName, "actions.0.notification_property.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "actions.0.notification_property.0.notify_delay_after", "1"),
				),
			},
		},
	})
}

func TestAccAWSGlueTrigger_actions_securityConfig(t *testing.T) {
	var trigger glue.Trigger

	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "aws_glue_trigger.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, glue.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSGlueTriggerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSGlueTriggerConfigActionsSecurityConfiguration(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSGlueTriggerExists(resourceName, &trigger),
					resource.TestCheckResourceAttr(resourceName, "actions.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "actions.0.job_name", rName),
					resource.TestCheckResourceAttr(resourceName, "actions.0.security_configuration", rName),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"enabled"},
			},
		},
	})
}

func TestAccAWSGlueTrigger_onDemandDisable(t *testing.T) {
	var trigger glue.Trigger

	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "aws_glue_trigger.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, glue.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSGlueTriggerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSGlueTriggerConfig_OnDemand(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSGlueTriggerExists(resourceName, &trigger),
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "type", "ON_DEMAND"),
				),
			},
			{
				Config: testAccAWSGlueTriggerConfig_OnDemandEnabled(rName, false),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSGlueTriggerExists(resourceName, &trigger),
					resource.TestCheckResourceAttr(resourceName, "enabled", "false"),
					resource.TestCheckResourceAttr(resourceName, "type", "ON_DEMAND"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"enabled"},
			},
			{
				Config: testAccAWSGlueTriggerConfig_OnDemandEnabled(rName, true),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSGlueTriggerExists(resourceName, &trigger),
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "type", "ON_DEMAND"),
				),
			},
		},
	})
}

func TestAccAWSGlueTrigger_disappears(t *testing.T) {
	var trigger glue.Trigger

	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "aws_glue_trigger.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, glue.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSGlueTriggerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSGlueTriggerConfig_OnDemand(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSGlueTriggerExists(resourceName, &trigger),
					testAccCheckResourceDisappears(testAccProvider, resourceAwsGlueTrigger(), resourceName),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccCheckAWSGlueTriggerExists(resourceName string, trigger *glue.Trigger) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("Not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Glue Trigger ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).glueconn

		output, err := finder.TriggerByName(conn, rs.Primary.ID)
		if err != nil {
			return err
		}

		if output.Trigger == nil {
			return fmt.Errorf("Glue Trigger (%s) not found", rs.Primary.ID)
		}

		if aws.StringValue(output.Trigger.Name) == rs.Primary.ID {
			*trigger = *output.Trigger
			return nil
		}

		return fmt.Errorf("Glue Trigger (%s) not found", rs.Primary.ID)
	}
}

func testAccCheckAWSGlueTriggerDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_glue_trigger" {
			continue
		}

		conn := testAccProvider.Meta().(*AWSClient).glueconn

		output, err := finder.TriggerByName(conn, rs.Primary.ID)

		if err != nil {
			if isAWSErr(err, glue.ErrCodeEntityNotFoundException, "") {
				return nil
			}

		}

		trigger := output.Trigger
		if trigger != nil && aws.StringValue(trigger.Name) == rs.Primary.ID {
			return fmt.Errorf("Glue Trigger %s still exists", rs.Primary.ID)
		}

		return err
	}

	return nil
}

func testAccAWSGlueTriggerConfig_Description(rName, description string) string {
	return composeConfig(testAccAWSGlueJobConfig_Required(rName), fmt.Sprintf(`
resource "aws_glue_trigger" "test" {
  description = %[1]q
  name        = %[2]q
  type        = "ON_DEMAND"

  actions {
    job_name = aws_glue_job.test.name
  }
}
`, description, rName))
}

func testAccAWSGlueTriggerConfig_Enabled(rName string, enabled bool) string {
	return composeConfig(testAccAWSGlueJobConfig_Required(rName), fmt.Sprintf(`
resource "aws_glue_trigger" "test" {
  enabled  = %[1]t
  name     = %[2]q
  schedule = "cron(15 12 * * ? *)"
  type     = "SCHEDULED"

  actions {
    job_name = aws_glue_job.test.name
  }
}
`, enabled, rName))
}

func testAccAWSGlueTriggerConfig_OnDemand(rName string) string {
	return composeConfig(testAccAWSGlueJobConfig_Required(rName), fmt.Sprintf(`
resource "aws_glue_trigger" "test" {
  name = %[1]q
  type = "ON_DEMAND"

  actions {
    job_name = aws_glue_job.test.name
  }
}
`, rName))
}

func testAccAWSGlueTriggerConfig_OnDemandEnabled(rName string, enabled bool) string {
	return composeConfig(testAccAWSGlueJobConfig_Required(rName), fmt.Sprintf(`
resource "aws_glue_trigger" "test" {
  name    = %[1]q
  type    = "ON_DEMAND"
  enabled = %[2]t

  actions {
    job_name = aws_glue_job.test.name
  }
}
`, rName, enabled))
}

func testAccAWSGlueTriggerConfig_Predicate(rName, state string) string {
	return composeConfig(testAccAWSGlueJobConfig_Required(rName), fmt.Sprintf(`
resource "aws_glue_job" "test2" {
  name     = "%[1]s2"
  role_arn = aws_iam_role.test.arn

  command {
    script_location = "testscriptlocation"
  }

  depends_on = [aws_iam_role_policy_attachment.test]
}

resource "aws_glue_trigger" "test" {
  name = %[1]q
  type = "CONDITIONAL"

  actions {
    job_name = aws_glue_job.test2.name
  }

  predicate {
    conditions {
      job_name = aws_glue_job.test.name
      state    = %[2]q
    }
  }
}
`, rName, state))
}

func testAccAWSGlueTriggerConfig_Crawler(rName, state string) string {
	return composeConfig(testAccGlueCrawlerConfig_S3Target(rName, "s3://test_bucket"), fmt.Sprintf(`
resource "aws_glue_crawler" "test2" {
  depends_on = [aws_iam_role_policy_attachment.test-AWSGlueServiceRole]

  database_name = aws_glue_catalog_database.test.name
  name          = "%[1]scrawl2"
  role          = aws_iam_role.test.name

  s3_target {
    path = "s3://test_bucket"
  }
}

resource "aws_glue_trigger" "test_trigger" {
  name = %[1]q
  type = "CONDITIONAL"

  actions {
    crawler_name = aws_glue_crawler.test.name
  }

  predicate {
    conditions {
      crawler_name = aws_glue_crawler.test2.name
      crawl_state  = %[2]q
    }
  }
}
`, rName, state))
}

func testAccAWSGlueTriggerConfig_Schedule(rName, schedule string) string {
	return composeConfig(testAccAWSGlueJobConfig_Required(rName), fmt.Sprintf(`
resource "aws_glue_trigger" "test" {
  name     = %[1]q
  schedule = %[2]q
  type     = "SCHEDULED"

  actions {
    job_name = aws_glue_job.test.name
  }
}
`, rName, schedule))
}

func testAccAWSGlueTriggerConfigTags1(rName, tagKey1, tagValue1 string) string {
	return composeConfig(testAccAWSGlueJobConfig_Required(rName), fmt.Sprintf(`
resource "aws_glue_trigger" "test" {
  name = %[1]q
  type = "ON_DEMAND"

  actions {
    job_name = aws_glue_job.test.name
  }

  tags = {
    %[2]q = %[3]q
  }
}
`, rName, tagKey1, tagValue1))
}

func testAccAWSGlueTriggerConfigTags2(rName, tagKey1, tagValue1, tagKey2, tagValue2 string) string {
	return composeConfig(testAccAWSGlueJobConfig_Required(rName), fmt.Sprintf(`
resource "aws_glue_trigger" "test" {
  name = %[1]q
  type = "ON_DEMAND"

  actions {
    job_name = aws_glue_job.test.name
  }

  tags = {
    %[2]q = %[3]q
    %[4]q = %[5]q
  }
}
`, rName, tagKey1, tagValue1, tagKey2, tagValue2))
}

func testAccAWSGlueTriggerConfig_WorkflowName(rName string) string {
	return composeConfig(testAccAWSGlueJobConfig_Required(rName), fmt.Sprintf(`
resource "aws_glue_workflow" test {
  name = %[1]q
}

resource "aws_glue_trigger" "test" {
  name          = %[1]q
  type          = "ON_DEMAND"
  workflow_name = aws_glue_workflow.test.name

  actions {
    job_name = aws_glue_job.test.name
  }
}
`, rName))
}

func testAccAWSGlueTriggerConfigActionsNotification(rName string, delay int) string {
	return composeConfig(testAccAWSGlueJobConfig_Required(rName), fmt.Sprintf(`
resource "aws_glue_trigger" "test" {
  name = %[1]q
  type = "ON_DEMAND"

  actions {
    job_name = aws_glue_job.test.name

    notification_property {
      notify_delay_after = %[2]d
    }
  }
}
`, rName, delay))
}

func testAccAWSGlueTriggerConfigActionsSecurityConfiguration(rName string) string {
	return composeConfig(testAccAWSGlueJobConfig_Required(rName), fmt.Sprintf(`
resource "aws_glue_security_configuration" "test" {
  name = %[1]q

  encryption_configuration {
    cloudwatch_encryption {
      cloudwatch_encryption_mode = "DISABLED"
    }

    job_bookmarks_encryption {
      job_bookmarks_encryption_mode = "DISABLED"
    }

    s3_encryption {
      s3_encryption_mode = "DISABLED"
    }
  }
}

resource "aws_glue_trigger" "test" {
  name = %[1]q
  type = "ON_DEMAND"

  actions {
    job_name               = aws_glue_job.test.name
    security_configuration = aws_glue_security_configuration.test.name
  }
}
`, rName))
}
