package ag

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/docdb"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/realglebivanov/terraform-provider-awsgamelift/ag/internal/keyvaluetags"
)

func resourceAwsDocDBSubnetGroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsDocDBSubnetGroupCreate,
		Read:   resourceAwsDocDBSubnetGroupRead,
		Update: resourceAwsDocDBSubnetGroupUpdate,
		Delete: resourceAwsDocDBSubnetGroupDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ConflictsWith: []string{"name_prefix"},
				ValidateFunc:  validateDocDBSubnetGroupName,
			},
			"name_prefix": {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ConflictsWith: []string{"name"},
				ValidateFunc:  validateDocDBSubnetGroupNamePrefix,
			},

			"description": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "Managed by Terraform",
			},

			"subnet_ids": {
				Type:     schema.TypeSet,
				Required: true,
				MinItems: 1,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"tags":     tagsSchema(),
			"tags_all": tagsSchemaComputed(),
		},

		CustomizeDiff: SetTagsDiff,
	}
}

func resourceAwsDocDBSubnetGroupCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).docdbconn
	defaultTagsConfig := meta.(*AWSClient).DefaultTagsConfig
	tags := defaultTagsConfig.MergeTags(keyvaluetags.New(d.Get("tags").(map[string]interface{})))

	subnetIds := expandStringSet(d.Get("subnet_ids").(*schema.Set))

	var groupName string
	if v, ok := d.GetOk("name"); ok {
		groupName = v.(string)
	} else if v, ok := d.GetOk("name_prefix"); ok {
		groupName = resource.PrefixedUniqueId(v.(string))
	} else {
		groupName = resource.UniqueId()
	}

	createOpts := docdb.CreateDBSubnetGroupInput{
		DBSubnetGroupName:        aws.String(groupName),
		DBSubnetGroupDescription: aws.String(d.Get("description").(string)),
		SubnetIds:                subnetIds,
		Tags:                     tags.IgnoreAws().DocdbTags(),
	}

	log.Printf("[DEBUG] Create DocDB Subnet Group: %#v", createOpts)
	_, err := conn.CreateDBSubnetGroup(&createOpts)
	if err != nil {
		return fmt.Errorf("error creating DocDB Subnet Group: %s", err)
	}

	d.SetId(groupName)

	return resourceAwsDocDBSubnetGroupRead(d, meta)
}

func resourceAwsDocDBSubnetGroupRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).docdbconn
	defaultTagsConfig := meta.(*AWSClient).DefaultTagsConfig
	ignoreTagsConfig := meta.(*AWSClient).IgnoreTagsConfig

	describeOpts := docdb.DescribeDBSubnetGroupsInput{
		DBSubnetGroupName: aws.String(d.Id()),
	}

	var subnetGroups []*docdb.DBSubnetGroup
	if err := conn.DescribeDBSubnetGroupsPages(&describeOpts, func(resp *docdb.DescribeDBSubnetGroupsOutput, lastPage bool) bool {
		subnetGroups = append(subnetGroups, resp.DBSubnetGroups...)
		return !lastPage
	}); err != nil {
		if isAWSErr(err, docdb.ErrCodeDBSubnetGroupNotFoundFault, "") {
			log.Printf("[WARN] DocDB Subnet Group (%s) not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return fmt.Errorf("error reading DocDB Subnet Group (%s) parameters: %s", d.Id(), err)
	}

	if len(subnetGroups) != 1 ||
		*subnetGroups[0].DBSubnetGroupName != d.Id() {
		return fmt.Errorf("unable to find DocDB Subnet Group: %s, removing from state", d.Id())
	}

	subnetGroup := subnetGroups[0]
	d.Set("name", subnetGroup.DBSubnetGroupName)
	d.Set("description", subnetGroup.DBSubnetGroupDescription)
	d.Set("arn", subnetGroup.DBSubnetGroupArn)

	subnets := make([]string, 0, len(subnetGroup.Subnets))
	for _, s := range subnetGroup.Subnets {
		subnets = append(subnets, aws.StringValue(s.SubnetIdentifier))
	}
	if err := d.Set("subnet_ids", subnets); err != nil {
		return fmt.Errorf("error setting subnet_ids: %s", err)
	}

	tags, err := keyvaluetags.DocdbListTags(conn, d.Get("arn").(string))

	if err != nil {
		return fmt.Errorf("error listing tags for DocumentDB Subnet Group (%s): %s", d.Get("arn").(string), err)
	}

	tags = tags.IgnoreAws().IgnoreConfig(ignoreTagsConfig)

	//lintignore:AWSR002
	if err := d.Set("tags", tags.RemoveDefaultConfig(defaultTagsConfig).Map()); err != nil {
		return fmt.Errorf("error setting tags: %w", err)
	}

	if err := d.Set("tags_all", tags.Map()); err != nil {
		return fmt.Errorf("error setting tags_all: %w", err)
	}

	return nil
}

func resourceAwsDocDBSubnetGroupUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).docdbconn

	if d.HasChanges("subnet_ids", "description") {
		_, n := d.GetChange("subnet_ids")
		if n == nil {
			n = new(schema.Set)
		}
		sIds := expandStringSet(n.(*schema.Set))

		_, err := conn.ModifyDBSubnetGroup(&docdb.ModifyDBSubnetGroupInput{
			DBSubnetGroupName:        aws.String(d.Id()),
			DBSubnetGroupDescription: aws.String(d.Get("description").(string)),
			SubnetIds:                sIds,
		})

		if err != nil {
			return fmt.Errorf("error modify DocDB Subnet Group (%s) parameters: %s", d.Id(), err)
		}
	}

	if d.HasChange("tags_all") {
		o, n := d.GetChange("tags_all")

		if err := keyvaluetags.DocdbUpdateTags(conn, d.Get("arn").(string), o, n); err != nil {
			return fmt.Errorf("error updating DocumentDB Subnet Group (%s) tags: %s", d.Get("arn").(string), err)
		}
	}

	return resourceAwsDocDBSubnetGroupRead(d, meta)
}

func resourceAwsDocDBSubnetGroupDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).docdbconn

	delOpts := docdb.DeleteDBSubnetGroupInput{
		DBSubnetGroupName: aws.String(d.Id()),
	}

	log.Printf("[DEBUG] Deleting DocDB Subnet Group: %s", d.Id())

	_, err := conn.DeleteDBSubnetGroup(&delOpts)
	if err != nil {
		if isAWSErr(err, docdb.ErrCodeDBSubnetGroupNotFoundFault, "") {
			return nil
		}
		return fmt.Errorf("error deleting DocDB Subnet Group (%s): %s", d.Id(), err)
	}

	return waitForDocDBSubnetGroupDeletion(conn, d.Id())
}

func waitForDocDBSubnetGroupDeletion(conn *docdb.DocDB, name string) error {
	params := &docdb.DescribeDBSubnetGroupsInput{
		DBSubnetGroupName: aws.String(name),
	}

	err := resource.Retry(10*time.Minute, func() *resource.RetryError {
		_, err := conn.DescribeDBSubnetGroups(params)

		if isAWSErr(err, docdb.ErrCodeDBSubnetGroupNotFoundFault, "") {
			return nil
		}

		if err != nil {
			return resource.NonRetryableError(err)
		}

		return resource.RetryableError(fmt.Errorf("DocDB Subnet Group (%s) still exists", name))
	})
	if isResourceTimeoutError(err) {
		_, err = conn.DescribeDBSubnetGroups(params)
		if isAWSErr(err, docdb.ErrCodeDBSubnetGroupNotFoundFault, "") {
			return nil
		}
	}
	if err != nil {
		return fmt.Errorf("Error deleting DocDB subnet group: %s", err)
	}
	return nil
}
