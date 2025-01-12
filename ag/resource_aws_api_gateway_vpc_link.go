package ag

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/realglebivanov/terraform-provider-awsgamelift/ag/internal/keyvaluetags"
)

const (
	// Maximum amount of time for VpcLink to become available
	apigatewayVpcLinkAvailableTimeout = 20 * time.Minute
	// Maximum amount of time for VpcLink to delete
	apigatewayVpcLinkDeleteTimeout = 20 * time.Minute
)

func resourceAwsApiGatewayVpcLink() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsApiGatewayVpcLinkCreate,
		Read:   resourceAwsApiGatewayVpcLinkRead,
		Update: resourceAwsApiGatewayVpcLinkUpdate,
		Delete: resourceAwsApiGatewayVpcLinkDelete,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"target_arns": {
				Type:     schema.TypeList,
				MaxItems: 1,
				Required: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"tags":     tagsSchema(),
			"tags_all": tagsSchemaComputed(),
		},

		CustomizeDiff: SetTagsDiff,
	}
}

func resourceAwsApiGatewayVpcLinkCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigatewayconn
	defaultTagsConfig := meta.(*AWSClient).DefaultTagsConfig
	tags := defaultTagsConfig.MergeTags(keyvaluetags.New(d.Get("tags").(map[string]interface{})))

	input := &apigateway.CreateVpcLinkInput{
		Name:       aws.String(d.Get("name").(string)),
		TargetArns: expandStringList(d.Get("target_arns").([]interface{})),
		Tags:       tags.IgnoreAws().ApigatewayTags(),
	}
	if v, ok := d.GetOk("description"); ok {
		input.Description = aws.String(v.(string))
	}

	resp, err := conn.CreateVpcLink(input)
	if err != nil {
		return err
	}

	d.SetId(aws.StringValue(resp.Id))

	stateConf := &resource.StateChangeConf{
		Pending:    []string{apigateway.VpcLinkStatusPending},
		Target:     []string{apigateway.VpcLinkStatusAvailable},
		Refresh:    apigatewayVpcLinkRefreshStatusFunc(conn, *resp.Id),
		Timeout:    apigatewayVpcLinkAvailableTimeout,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("error waiting for APIGateway Vpc Link status to be \"%s\": %w", apigateway.VpcLinkStatusAvailable, err)
	}

	return resourceAwsApiGatewayVpcLinkRead(d, meta)
}

func resourceAwsApiGatewayVpcLinkRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigatewayconn
	defaultTagsConfig := meta.(*AWSClient).DefaultTagsConfig
	ignoreTagsConfig := meta.(*AWSClient).IgnoreTagsConfig

	input := &apigateway.GetVpcLinkInput{
		VpcLinkId: aws.String(d.Id()),
	}

	resp, err := conn.GetVpcLink(input)
	if err != nil {
		if isAWSErr(err, apigateway.ErrCodeNotFoundException, "") {
			log.Printf("[WARN] VPC Link %s not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}

	tags := keyvaluetags.ApigatewayKeyValueTags(resp.Tags).IgnoreAws().IgnoreConfig(ignoreTagsConfig)

	//lintignore:AWSR002
	if err := d.Set("tags", tags.RemoveDefaultConfig(defaultTagsConfig).Map()); err != nil {
		return fmt.Errorf("error setting tags: %w", err)
	}

	if err := d.Set("tags_all", tags.Map()); err != nil {
		return fmt.Errorf("error setting tags_all: %w", err)
	}

	arn := arn.ARN{
		Partition: meta.(*AWSClient).partition,
		Service:   "apigateway",
		Region:    meta.(*AWSClient).region,
		Resource:  fmt.Sprintf("/vpclinks/%s", d.Id()),
	}.String()
	d.Set("arn", arn)

	d.Set("name", resp.Name)
	d.Set("description", resp.Description)
	d.Set("target_arns", flattenStringList(resp.TargetArns))
	return nil
}

func resourceAwsApiGatewayVpcLinkUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigatewayconn

	operations := make([]*apigateway.PatchOperation, 0)

	if d.HasChange("name") {
		operations = append(operations, &apigateway.PatchOperation{
			Op:    aws.String("replace"),
			Path:  aws.String("/name"),
			Value: aws.String(d.Get("name").(string)),
		})
	}

	if d.HasChange("description") {
		operations = append(operations, &apigateway.PatchOperation{
			Op:    aws.String("replace"),
			Path:  aws.String("/description"),
			Value: aws.String(d.Get("description").(string)),
		})
	}

	if d.HasChange("tags_all") {
		o, n := d.GetChange("tags_all")
		if err := keyvaluetags.ApigatewayUpdateTags(conn, d.Get("arn").(string), o, n); err != nil {
			return fmt.Errorf("error updating tags: %s", err)
		}
	}

	input := &apigateway.UpdateVpcLinkInput{
		VpcLinkId:       aws.String(d.Id()),
		PatchOperations: operations,
	}

	_, err := conn.UpdateVpcLink(input)
	if err != nil {
		if isAWSErr(err, apigateway.ErrCodeNotFoundException, "") {
			log.Printf("[WARN] VPC Link %s not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{apigateway.VpcLinkStatusPending},
		Target:     []string{apigateway.VpcLinkStatusAvailable},
		Refresh:    apigatewayVpcLinkRefreshStatusFunc(conn, d.Id()),
		Timeout:    apigatewayVpcLinkAvailableTimeout,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for APIGateway Vpc Link status to be \"%s\": %s", apigateway.VpcLinkStatusAvailable, err)
	}

	return resourceAwsApiGatewayVpcLinkRead(d, meta)
}

func resourceAwsApiGatewayVpcLinkDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigatewayconn

	input := &apigateway.DeleteVpcLinkInput{
		VpcLinkId: aws.String(d.Id()),
	}

	_, err := conn.DeleteVpcLink(input)

	if isAWSErr(err, apigateway.ErrCodeNotFoundException, "") {
		return nil
	}

	if err != nil {
		return fmt.Errorf("error deleting API Gateway VPC Link (%s): %s", d.Id(), err)
	}

	if err := waitForApiGatewayVpcLinkDeletion(conn, d.Id()); err != nil {
		return fmt.Errorf("error waiting for API Gateway VPC Link (%s) deletion: %s", d.Id(), err)
	}

	return nil
}

func apigatewayVpcLinkRefreshStatusFunc(conn *apigateway.APIGateway, vl string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		input := &apigateway.GetVpcLinkInput{
			VpcLinkId: aws.String(vl),
		}
		resp, err := conn.GetVpcLink(input)
		if err != nil {
			return nil, "failed", err
		}
		return resp, *resp.Status, nil
	}
}

func waitForApiGatewayVpcLinkDeletion(conn *apigateway.APIGateway, vpcLinkID string) error {
	stateConf := resource.StateChangeConf{
		Pending: []string{apigateway.VpcLinkStatusPending,
			apigateway.VpcLinkStatusAvailable,
			apigateway.VpcLinkStatusDeleting},
		Target:     []string{""},
		Timeout:    apigatewayVpcLinkDeleteTimeout,
		MinTimeout: 1 * time.Second,
		Refresh: func() (interface{}, string, error) {
			resp, err := conn.GetVpcLink(&apigateway.GetVpcLinkInput{
				VpcLinkId: aws.String(vpcLinkID),
			})

			if isAWSErr(err, apigateway.ErrCodeNotFoundException, "") {
				return 1, "", nil
			}

			if err != nil {
				return nil, apigateway.VpcLinkStatusFailed, err
			}

			return resp, aws.StringValue(resp.Status), nil
		},
	}

	_, err := stateConf.WaitForState()

	return err
}
