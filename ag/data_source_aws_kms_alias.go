package ag

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceAwsKmsAlias() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsKmsAliasRead,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateAwsKmsName,
			},
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"target_key_arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"target_key_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceAwsKmsAliasRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).kmsconn
	params := &kms.ListAliasesInput{}

	target := d.Get("name")
	var alias *kms.AliasListEntry
	log.Printf("[DEBUG] Reading KMS Alias: %s", params)
	err := conn.ListAliasesPages(params, func(page *kms.ListAliasesOutput, lastPage bool) bool {
		for _, entity := range page.Aliases {
			if aws.StringValue(entity.AliasName) == target {
				alias = entity
				return false
			}
		}

		return true
	})
	if err != nil {
		return fmt.Errorf("Error fetch KMS alias list: %w", err)
	}

	if alias == nil {
		return fmt.Errorf("No alias with name %q found in this region.", target)
	}

	d.SetId(aws.StringValue(alias.AliasArn))
	d.Set("arn", alias.AliasArn)

	// ListAliases can return an alias for an AWS service key (e.g.
	// alias/aws/rds) without a TargetKeyId if the alias has not yet been
	// used for the first time. In that situation, calling DescribeKey will
	// associate an actual key with the alias, and the next call to
	// ListAliases will have a TargetKeyId for the alias.
	//
	// For a simpler codepath, we always call DescribeKey with the alias
	// name to get the target key's ARN and Id direct from AWS.
	//
	// https://docs.aws.amazon.com/kms/latest/APIReference/API_ListAliases.html

	req := &kms.DescribeKeyInput{
		KeyId: aws.String(target.(string)),
	}
	resp, err := conn.DescribeKey(req)
	if err != nil {
		return fmt.Errorf("Error calling KMS DescribeKey: %w", err)
	}

	d.Set("target_key_arn", resp.KeyMetadata.Arn)
	d.Set("target_key_id", resp.KeyMetadata.KeyId)

	return nil
}
