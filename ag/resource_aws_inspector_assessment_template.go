package ag

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	// "github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/inspector"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/realglebivanov/terraform-provider-awsgamelift/ag/internal/keyvaluetags"
)

func resourceAWSInspectorAssessmentTemplate() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsInspectorAssessmentTemplateCreate,
		Read:   resourceAwsInspectorAssessmentTemplateRead,
		Update: resourceAwsInspectorAssessmentTemplateUpdate,
		Delete: resourceAwsInspectorAssessmentTemplateDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"target_arn": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"duration": {
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},
			"rules_package_arns": {
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
				Required: true,
				ForceNew: true,
			},
			"tags":     tagsSchema(),
			"tags_all": tagsSchemaComputed(),
		},

		CustomizeDiff: SetTagsDiff,
	}
}

func resourceAwsInspectorAssessmentTemplateCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).inspectorconn
	defaultTagsConfig := meta.(*AWSClient).DefaultTagsConfig
	tags := defaultTagsConfig.MergeTags(keyvaluetags.New(d.Get("tags").(map[string]interface{})))

	req := &inspector.CreateAssessmentTemplateInput{
		AssessmentTargetArn:    aws.String(d.Get("target_arn").(string)),
		AssessmentTemplateName: aws.String(d.Get("name").(string)),
		DurationInSeconds:      aws.Int64(int64(d.Get("duration").(int))),
		RulesPackageArns:       expandStringSet(d.Get("rules_package_arns").(*schema.Set)),
	}

	log.Printf("[DEBUG] Creating Inspector assessment template: %s", req)
	resp, err := conn.CreateAssessmentTemplate(req)
	if err != nil {
		return fmt.Errorf("error creating Inspector assessment template: %s", err)
	}

	d.SetId(aws.StringValue(resp.AssessmentTemplateArn))

	if len(tags) > 0 {
		if err := keyvaluetags.InspectorUpdateTags(conn, d.Id(), nil, tags); err != nil {
			return fmt.Errorf("error adding Inspector assessment template (%s) tags: %s", d.Id(), err)
		}
	}

	return resourceAwsInspectorAssessmentTemplateRead(d, meta)
}

func resourceAwsInspectorAssessmentTemplateRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).inspectorconn
	defaultTagsConfig := meta.(*AWSClient).DefaultTagsConfig
	ignoreTagsConfig := meta.(*AWSClient).IgnoreTagsConfig

	resp, err := conn.DescribeAssessmentTemplates(&inspector.DescribeAssessmentTemplatesInput{
		AssessmentTemplateArns: aws.StringSlice([]string{d.Id()}),
	})
	if err != nil {
		return fmt.Errorf("error reading Inspector assessment template (%s): %s", d.Id(), err)
	}

	if resp.AssessmentTemplates == nil || len(resp.AssessmentTemplates) == 0 {
		log.Printf("[WARN] Inspector assessment template (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	template := resp.AssessmentTemplates[0]

	arn := aws.StringValue(template.Arn)
	d.Set("arn", arn)
	d.Set("duration", template.DurationInSeconds)
	d.Set("name", template.Name)
	d.Set("target_arn", template.AssessmentTargetArn)

	if err := d.Set("rules_package_arns", flattenStringSet(template.RulesPackageArns)); err != nil {
		return fmt.Errorf("error setting rules_package_arns: %s", err)
	}

	tags, err := keyvaluetags.InspectorListTags(conn, arn)

	if err != nil {
		return fmt.Errorf("error listing tags for Inspector assessment template (%s): %s", arn, err)
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

func resourceAwsInspectorAssessmentTemplateUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).inspectorconn

	if d.HasChange("tags_all") {
		o, n := d.GetChange("tags_all")

		if err := keyvaluetags.InspectorUpdateTags(conn, d.Id(), o, n); err != nil {
			return fmt.Errorf("error updating Inspector assessment template (%s) tags: %s", d.Id(), err)
		}
	}

	return resourceAwsInspectorAssessmentTemplateRead(d, meta)
}

func resourceAwsInspectorAssessmentTemplateDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).inspectorconn

	_, err := conn.DeleteAssessmentTemplate(&inspector.DeleteAssessmentTemplateInput{
		AssessmentTemplateArn: aws.String(d.Id()),
	})
	if err != nil {
		return fmt.Errorf("error deleting Inspector assessment template (%s): %s", d.Id(), err)
	}

	return nil
}
