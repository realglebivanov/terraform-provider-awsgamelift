package ag

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/realglebivanov/terraform-provider-awsgamelift/ag/internal/keyvaluetags"
)

func resourceAwsIamSamlProvider() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsIamSamlProviderCreate,
		Read:   resourceAwsIamSamlProviderRead,
		Update: resourceAwsIamSamlProviderUpdate,
		Delete: resourceAwsIamSamlProviderDelete,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"valid_until": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringLenBetween(1, 128),
			},
			"saml_metadata_document": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringLenBetween(1000, 10000000),
			},
			"tags":     tagsSchema(),
			"tags_all": tagsSchemaComputed(),
		},

		CustomizeDiff: SetTagsDiff,
	}
}

func resourceAwsIamSamlProviderCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iamconn
	defaultTagsConfig := meta.(*AWSClient).DefaultTagsConfig
	tags := defaultTagsConfig.MergeTags(keyvaluetags.New(d.Get("tags").(map[string]interface{})))

	input := &iam.CreateSAMLProviderInput{
		Name:                 aws.String(d.Get("name").(string)),
		SAMLMetadataDocument: aws.String(d.Get("saml_metadata_document").(string)),
		Tags:                 tags.IgnoreAws().IamTags(),
	}

	out, err := conn.CreateSAMLProvider(input)
	if err != nil {
		return fmt.Errorf("error creating IAM SAML Provider: %w", err)
	}

	d.SetId(aws.StringValue(out.SAMLProviderArn))

	return resourceAwsIamSamlProviderRead(d, meta)
}

func resourceAwsIamSamlProviderRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iamconn
	defaultTagsConfig := meta.(*AWSClient).DefaultTagsConfig
	ignoreTagsConfig := meta.(*AWSClient).IgnoreTagsConfig

	input := &iam.GetSAMLProviderInput{
		SAMLProviderArn: aws.String(d.Id()),
	}
	out, err := conn.GetSAMLProvider(input)
	if err != nil {
		if isAWSErr(err, iam.ErrCodeNoSuchEntityException, "") {
			log.Printf("[WARN] IAM SAML Provider %q not found, removing from state.", d.Id())
			d.SetId("")
			return nil
		}
		return fmt.Errorf("error reading IAM SAML Provider (%q): %w", d.Id(), err)
	}

	d.Set("arn", d.Id())
	name, err := extractNameFromIAMSamlProviderArn(d.Id())
	if err != nil {
		return err
	}
	d.Set("name", name)
	d.Set("valid_until", out.ValidUntil.Format(time.RFC1123))
	d.Set("saml_metadata_document", out.SAMLMetadataDocument)

	tags := keyvaluetags.IamKeyValueTags(out.Tags).IgnoreAws().IgnoreConfig(ignoreTagsConfig)

	//lintignore:AWSR002
	if err := d.Set("tags", tags.RemoveDefaultConfig(defaultTagsConfig).Map()); err != nil {
		return fmt.Errorf("error setting tags: %w", err)
	}

	if err := d.Set("tags_all", tags.Map()); err != nil {
		return fmt.Errorf("error setting tags_all: %w", err)
	}

	return nil
}

func resourceAwsIamSamlProviderUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iamconn

	if d.HasChangesExcept("tags", "tags_all") {
		input := &iam.UpdateSAMLProviderInput{
			SAMLProviderArn:      aws.String(d.Id()),
			SAMLMetadataDocument: aws.String(d.Get("saml_metadata_document").(string)),
		}
		_, err := conn.UpdateSAMLProvider(input)
		if err != nil {
			return fmt.Errorf("error updating IAM SAML Provider (%q): %w", d.Id(), err)
		}
	}

	if d.HasChange("tags_all") {
		o, n := d.GetChange("tags_all")

		if err := keyvaluetags.IamSAMLProviderUpdateTags(conn, d.Id(), o, n); err != nil {
			return fmt.Errorf("error updating tags for IAM SAML Provider (%s): %w", d.Id(), err)
		}
	}

	return resourceAwsIamSamlProviderRead(d, meta)
}

func resourceAwsIamSamlProviderDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iamconn

	input := &iam.DeleteSAMLProviderInput{
		SAMLProviderArn: aws.String(d.Id()),
	}
	_, err := conn.DeleteSAMLProvider(input)
	if err != nil {
		if isAWSErr(err, iam.ErrCodeNoSuchEntityException, "") {
			return nil
		}
		return fmt.Errorf("error deleting IAM SAML Provider (%q): %w", d.Id(), err)
	}

	return nil
}

func extractNameFromIAMSamlProviderArn(samlArn string) (string, error) {
	parsedArn, err := arn.Parse(samlArn)
	if err != nil {
		return "", fmt.Errorf("Unable to extract name from a given ARN: %q", samlArn)
	}

	name := strings.TrimPrefix(parsedArn.Resource, "saml-provider/")

	return name, nil
}
