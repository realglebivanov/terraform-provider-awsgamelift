package ag

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/realglebivanov/terraform-provider-awsgamelift/ag/internal/keyvaluetags"
)

func resourceAwsEbsSnapshotCopy() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsEbsSnapshotCopyCreate,
		Read:   resourceAwsEbsSnapshotCopyRead,
		Update: resourceAwsEbsSnapshotCopyUpdate,
		Delete: resourceAwsEbsSnapshotCopyDelete,

		CustomizeDiff: SetTagsDiff,

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"volume_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"owner_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"owner_alias": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"encrypted": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},
			"volume_size": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"kms_key_id": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"data_encryption_key_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"source_region": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"source_snapshot_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"tags":     tagsSchema(),
			"tags_all": tagsSchemaComputed(),
		},
	}
}

func resourceAwsEbsSnapshotCopyCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn
	defaultTagsConfig := meta.(*AWSClient).DefaultTagsConfig
	tags := defaultTagsConfig.MergeTags(keyvaluetags.New(d.Get("tags").(map[string]interface{})))

	request := &ec2.CopySnapshotInput{
		SourceRegion:      aws.String(d.Get("source_region").(string)),
		SourceSnapshotId:  aws.String(d.Get("source_snapshot_id").(string)),
		TagSpecifications: ec2TagSpecificationsFromKeyValueTags(tags, ec2.ResourceTypeSnapshot),
	}
	if v, ok := d.GetOk("description"); ok {
		request.Description = aws.String(v.(string))
	}
	if v, ok := d.GetOk("encrypted"); ok {
		request.Encrypted = aws.Bool(v.(bool))
	}
	if v, ok := d.GetOk("kms_key_id"); ok {
		request.KmsKeyId = aws.String(v.(string))
	}

	res, err := conn.CopySnapshot(request)
	if err != nil {
		return err
	}

	d.SetId(aws.StringValue(res.SnapshotId))

	err = resourceAwsEbsSnapshotCopyWaitForAvailable(d.Id(), conn)
	if err != nil {
		return err
	}

	return resourceAwsEbsSnapshotCopyRead(d, meta)
}

func resourceAwsEbsSnapshotCopyRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn
	defaultTagsConfig := meta.(*AWSClient).DefaultTagsConfig
	ignoreTagsConfig := meta.(*AWSClient).IgnoreTagsConfig

	req := &ec2.DescribeSnapshotsInput{
		SnapshotIds: []*string{aws.String(d.Id())},
	}
	res, err := conn.DescribeSnapshots(req)
	if isAWSErr(err, "InvalidSnapshot.NotFound", "") {
		log.Printf("Snapshot %q Not found - removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err != nil {
		return fmt.Errorf("error describing EC2 Snapshot (%s): %w", d.Id(), err)
	}

	snapshot := res.Snapshots[0]

	d.Set("description", snapshot.Description)
	d.Set("owner_id", snapshot.OwnerId)
	d.Set("encrypted", snapshot.Encrypted)
	d.Set("owner_alias", snapshot.OwnerAlias)
	d.Set("volume_id", snapshot.VolumeId)
	d.Set("data_encryption_key_id", snapshot.DataEncryptionKeyId)
	d.Set("kms_key_id", snapshot.KmsKeyId)
	d.Set("volume_size", snapshot.VolumeSize)

	tags := keyvaluetags.Ec2KeyValueTags(snapshot.Tags).IgnoreAws().IgnoreConfig(ignoreTagsConfig)

	//lintignore:AWSR002
	if err := d.Set("tags", tags.RemoveDefaultConfig(defaultTagsConfig).Map()); err != nil {
		return fmt.Errorf("error setting tags: %w", err)
	}

	if err := d.Set("tags_all", tags.Map()); err != nil {
		return fmt.Errorf("error setting tags_all: %w", err)
	}

	snapshotArn := arn.ARN{
		Partition: meta.(*AWSClient).partition,
		Region:    meta.(*AWSClient).region,
		Resource:  fmt.Sprintf("snapshot/%s", d.Id()),
		Service:   ec2.ServiceName,
	}.String()

	d.Set("arn", snapshotArn)

	return nil
}

func resourceAwsEbsSnapshotCopyDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn
	input := &ec2.DeleteSnapshotInput{
		SnapshotId: aws.String(d.Id()),
	}
	err := resource.Retry(5*time.Minute, func() *resource.RetryError {
		_, err := conn.DeleteSnapshot(input)
		if err == nil {
			return nil
		}

		if isAWSErr(err, "SnapshotInUse", "") {
			return resource.RetryableError(fmt.Errorf("EBS SnapshotInUse - trying again while it detaches"))
		}

		if isAWSErr(err, "InvalidSnapshot.NotFound", "") {
			return nil
		}

		return resource.NonRetryableError(err)
	})
	if isResourceTimeoutError(err) {
		_, err = conn.DeleteSnapshot(input)
		if isAWSErr(err, "InvalidSnapshot.NotFound", "") {
			return nil
		}
	}
	if err != nil {
		return fmt.Errorf("Error deleting EBS snapshot copy: %s", err)
	}
	return nil
}

func resourceAwsEbsSnapshotCopyUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	if d.HasChange("tags_all") {
		o, n := d.GetChange("tags_all")
		if err := keyvaluetags.Ec2UpdateTags(conn, d.Id(), o, n); err != nil {
			return fmt.Errorf("error updating tags: %s", err)
		}
	}

	return resourceAwsEbsSnapshotRead(d, meta)
}

func resourceAwsEbsSnapshotCopyWaitForAvailable(id string, conn *ec2.EC2) error {
	log.Printf("Waiting for Snapshot %s to become available...", id)

	req := &ec2.DescribeSnapshotsInput{
		SnapshotIds: []*string{aws.String(id)},
	}
	err := conn.WaitUntilSnapshotCompleted(req)
	return err
}
