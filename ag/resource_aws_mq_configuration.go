package ag

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/mq"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/customdiff"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/realglebivanov/terraform-provider-awsgamelift/ag/internal/keyvaluetags"
)

func resourceAwsMqConfiguration() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsMqConfigurationCreate,
		Read:   resourceAwsMqConfigurationRead,
		Update: resourceAwsMqConfigurationUpdate,
		Delete: resourceAwsMqConfigurationDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		CustomizeDiff: customdiff.Sequence(
			func(_ context.Context, diff *schema.ResourceDiff, v interface{}) error {
				if diff.HasChange("description") {
					return diff.SetNewComputed("latest_revision")
				}
				if diff.HasChange("data") {
					o, n := diff.GetChange("data")
					os := o.(string)
					ns := n.(string)
					if !suppressXMLEquivalentConfig("data", os, ns, nil) {
						return diff.SetNewComputed("latest_revision")
					}
				}
				return nil
			},
			SetTagsDiff,
		),

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"authentication_strategy": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validation.StringInSlice(mq.AuthenticationStrategy_Values(), true),
			},
			"data": {
				Type:             schema.TypeString,
				Required:         true,
				DiffSuppressFunc: suppressXMLEquivalentConfig,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"engine_type": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice(mq.EngineType_Values(), true),
			},
			"engine_version": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"latest_revision": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"tags":     tagsSchema(),
			"tags_all": tagsSchemaComputed(),
		},
	}
}

func resourceAwsMqConfigurationCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).mqconn
	defaultTagsConfig := meta.(*AWSClient).DefaultTagsConfig
	tags := defaultTagsConfig.MergeTags(keyvaluetags.New(d.Get("tags").(map[string]interface{})))

	input := mq.CreateConfigurationRequest{
		EngineType:    aws.String(d.Get("engine_type").(string)),
		EngineVersion: aws.String(d.Get("engine_version").(string)),
		Name:          aws.String(d.Get("name").(string)),
	}

	if v, ok := d.GetOk("authentication_strategy"); ok {
		input.AuthenticationStrategy = aws.String(v.(string))
	}
	if len(tags) > 0 {
		input.Tags = tags.IgnoreAws().MqTags()
	}

	log.Printf("[INFO] Creating MQ Configuration: %s", input)
	out, err := conn.CreateConfiguration(&input)
	if err != nil {
		return err
	}

	d.SetId(aws.StringValue(out.Id))
	d.Set("arn", out.Arn)

	return resourceAwsMqConfigurationUpdate(d, meta)
}

func resourceAwsMqConfigurationRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).mqconn
	defaultTagsConfig := meta.(*AWSClient).DefaultTagsConfig
	ignoreTagsConfig := meta.(*AWSClient).IgnoreTagsConfig

	log.Printf("[INFO] Reading MQ Configuration %s", d.Id())
	out, err := conn.DescribeConfiguration(&mq.DescribeConfigurationInput{
		ConfigurationId: aws.String(d.Id()),
	})
	if err != nil {
		if isAWSErr(err, mq.ErrCodeNotFoundException, "") {
			log.Printf("[WARN] MQ Configuration %q not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}

	d.Set("arn", out.Arn)
	d.Set("authentication_strategy", out.AuthenticationStrategy)
	d.Set("description", out.LatestRevision.Description)
	d.Set("engine_type", out.EngineType)
	d.Set("engine_version", out.EngineVersion)
	d.Set("latest_revision", out.LatestRevision.Revision)
	d.Set("name", out.Name)

	rOut, err := conn.DescribeConfigurationRevision(&mq.DescribeConfigurationRevisionInput{
		ConfigurationId:       aws.String(d.Id()),
		ConfigurationRevision: aws.String(fmt.Sprintf("%d", *out.LatestRevision.Revision)),
	})
	if err != nil {
		return err
	}

	b, err := base64.StdEncoding.DecodeString(*rOut.Data)
	if err != nil {
		return err
	}

	d.Set("data", string(b))

	tags := keyvaluetags.MqKeyValueTags(out.Tags).IgnoreAws().IgnoreConfig(ignoreTagsConfig)

	//lintignore:AWSR002
	if err := d.Set("tags", tags.RemoveDefaultConfig(defaultTagsConfig).Map()); err != nil {
		return fmt.Errorf("error setting tags: %w", err)
	}

	if err := d.Set("tags_all", tags.Map()); err != nil {
		return fmt.Errorf("error setting tags_all: %w", err)
	}

	return nil
}

func resourceAwsMqConfigurationUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).mqconn

	if d.HasChanges("data", "description") {
		rawData := d.Get("data").(string)
		data := base64.StdEncoding.EncodeToString([]byte(rawData))

		input := mq.UpdateConfigurationRequest{
			ConfigurationId: aws.String(d.Id()),
			Data:            aws.String(data),
		}
		if v, ok := d.GetOk("description"); ok {
			input.Description = aws.String(v.(string))
		}

		log.Printf("[INFO] Updating MQ Configuration %s: %s", d.Id(), input)
		_, err := conn.UpdateConfiguration(&input)
		if err != nil {
			return err
		}
	}

	if d.HasChange("tags_all") {
		o, n := d.GetChange("tags_all")

		if err := keyvaluetags.MqUpdateTags(conn, d.Get("arn").(string), o, n); err != nil {
			return fmt.Errorf("error updating MQ Broker (%s) tags: %s", d.Get("arn").(string), err)
		}
	}

	return resourceAwsMqConfigurationRead(d, meta)
}

func resourceAwsMqConfigurationDelete(d *schema.ResourceData, meta interface{}) error {
	// TODO: Delete is not available in the API

	return nil
}

func suppressXMLEquivalentConfig(k, old, new string, d *schema.ResourceData) bool {
	os, err := canonicalXML(old)
	if err != nil {
		log.Printf("[ERR] Error getting cannonicalXML from state (%s): %s", k, err)
		return false
	}
	ns, err := canonicalXML(new)
	if err != nil {
		log.Printf("[ERR] Error getting cannonicalXML from config (%s): %s", k, err)
		return false
	}

	return os == ns
}
