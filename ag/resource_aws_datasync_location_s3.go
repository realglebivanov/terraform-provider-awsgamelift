package ag

import (
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/datasync"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/realglebivanov/terraform-provider-awsgamelift/ag/internal/keyvaluetags"
	iamwaiter "github.com/realglebivanov/terraform-provider-awsgamelift/ag/internal/service/iam/waiter"
)

func resourceAwsDataSyncLocationS3() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsDataSyncLocationS3Create,
		Read:   resourceAwsDataSyncLocationS3Read,
		Update: resourceAwsDataSyncLocationS3Update,
		Delete: resourceAwsDataSyncLocationS3Delete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"agent_arns": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"s3_bucket_arn": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.NoZeroValues,
			},
			"s3_config": {
				Type:     schema.TypeList,
				Required: true,
				ForceNew: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"bucket_access_role_arn": {
							Type:         schema.TypeString,
							Required:     true,
							ForceNew:     true,
							ValidateFunc: validation.NoZeroValues,
						},
					},
				},
			},
			"s3_storage_class": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice(datasync.S3StorageClass_Values(), false),
			},
			"subdirectory": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				// Ignore missing trailing slash
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					if new == "/" {
						return false
					}
					if strings.TrimSuffix(old, "/") == strings.TrimSuffix(new, "/") {
						return true
					}
					return false
				},
			},
			"tags":     tagsSchema(),
			"tags_all": tagsSchemaComputed(),
			"uri": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},

		CustomizeDiff: SetTagsDiff,
	}
}

func resourceAwsDataSyncLocationS3Create(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).datasyncconn
	defaultTagsConfig := meta.(*AWSClient).DefaultTagsConfig
	tags := defaultTagsConfig.MergeTags(keyvaluetags.New(d.Get("tags").(map[string]interface{})))

	input := &datasync.CreateLocationS3Input{
		S3BucketArn:  aws.String(d.Get("s3_bucket_arn").(string)),
		S3Config:     expandDataSyncS3Config(d.Get("s3_config").([]interface{})),
		Subdirectory: aws.String(d.Get("subdirectory").(string)),
		Tags:         tags.IgnoreAws().DatasyncTags(),
	}

	if v, ok := d.GetOk("agent_arns"); ok {
		input.AgentArns = expandStringSet(v.(*schema.Set))
	}

	if v, ok := d.GetOk("s3_storage_class"); ok {
		input.S3StorageClass = aws.String(v.(string))
	}

	log.Printf("[DEBUG] Creating DataSync Location S3: %s", input)

	var output *datasync.CreateLocationS3Output
	err := resource.Retry(iamwaiter.PropagationTimeout, func() *resource.RetryError {
		var err error
		output, err = conn.CreateLocationS3(input)

		// Retry for IAM eventual consistency on error:
		// InvalidRequestException: Unable to assume role. Reason: Access denied when calling sts:AssumeRole
		if isAWSErr(err, datasync.ErrCodeInvalidRequestException, "Unable to assume role") {
			return resource.RetryableError(err)
		}

		// Retry for IAM eventual consistency on error:
		// InvalidRequestException: DataSync location access test failed: could not perform s3:ListObjectsV2 on bucket
		if isAWSErr(err, datasync.ErrCodeInvalidRequestException, "access test failed") {
			return resource.RetryableError(err)
		}

		if err != nil {
			return resource.NonRetryableError(err)
		}

		return nil
	})

	if isResourceTimeoutError(err) {
		output, err = conn.CreateLocationS3(input)
	}

	if err != nil {
		return fmt.Errorf("error creating DataSync Location S3: %s", err)
	}

	d.SetId(aws.StringValue(output.LocationArn))

	return resourceAwsDataSyncLocationS3Read(d, meta)
}

func resourceAwsDataSyncLocationS3Read(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).datasyncconn
	defaultTagsConfig := meta.(*AWSClient).DefaultTagsConfig
	ignoreTagsConfig := meta.(*AWSClient).IgnoreTagsConfig

	input := &datasync.DescribeLocationS3Input{
		LocationArn: aws.String(d.Id()),
	}

	log.Printf("[DEBUG] Reading DataSync Location S3: %s", input)
	output, err := conn.DescribeLocationS3(input)

	if isAWSErr(err, "InvalidRequestException", "not found") {
		log.Printf("[WARN] DataSync Location S3 %q not found - removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err != nil {
		return fmt.Errorf("error reading DataSync Location S3 (%s): %s", d.Id(), err)
	}

	subdirectory, err := dataSyncParseLocationURI(aws.StringValue(output.LocationUri))

	if err != nil {
		return fmt.Errorf("error parsing Location S3 (%s) URI (%s): %s", d.Id(), aws.StringValue(output.LocationUri), err)
	}

	d.Set("agent_arns", flattenStringSet(output.AgentArns))
	d.Set("arn", output.LocationArn)
	if err := d.Set("s3_config", flattenDataSyncS3Config(output.S3Config)); err != nil {
		return fmt.Errorf("error setting s3_config: %s", err)
	}
	d.Set("s3_storage_class", output.S3StorageClass)
	d.Set("subdirectory", subdirectory)
	d.Set("uri", output.LocationUri)

	tags, err := keyvaluetags.DatasyncListTags(conn, d.Id())

	if err != nil {
		return fmt.Errorf("error listing tags for DataSync Location S3 (%s): %s", d.Id(), err)
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

func resourceAwsDataSyncLocationS3Update(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).datasyncconn

	if d.HasChange("tags_all") {
		o, n := d.GetChange("tags_all")

		if err := keyvaluetags.DatasyncUpdateTags(conn, d.Id(), o, n); err != nil {
			return fmt.Errorf("error updating DataSync Location S3 (%s) tags: %s", d.Id(), err)
		}
	}

	return resourceAwsDataSyncLocationS3Read(d, meta)
}

func resourceAwsDataSyncLocationS3Delete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).datasyncconn

	input := &datasync.DeleteLocationInput{
		LocationArn: aws.String(d.Id()),
	}

	log.Printf("[DEBUG] Deleting DataSync Location S3: %s", input)
	_, err := conn.DeleteLocation(input)

	if isAWSErr(err, "InvalidRequestException", "not found") {
		return nil
	}

	if err != nil {
		return fmt.Errorf("error deleting DataSync Location S3 (%s): %s", d.Id(), err)
	}

	return nil
}
