package ag

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceAwsAutoscalingLifecycleHook() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsAutoscalingLifecycleHookPut,
		Read:   resourceAwsAutoscalingLifecycleHookRead,
		Update: resourceAwsAutoscalingLifecycleHookPut,
		Delete: resourceAwsAutoscalingLifecycleHookDelete,

		Importer: &schema.ResourceImporter{
			State: resourceAwsAutoscalingLifecycleHookImport,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"autoscaling_group_name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"default_result": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"heartbeat_timeout": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"lifecycle_transition": {
				Type:     schema.TypeString,
				Required: true,
			},
			"notification_metadata": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"notification_target_arn": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"role_arn": {
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func resourceAwsAutoscalingLifecycleHookPutOp(conn *autoscaling.AutoScaling, params *autoscaling.PutLifecycleHookInput) error {
	log.Printf("[DEBUG] AutoScaling PutLifecyleHook: %s", params)
	err := resource.Retry(5*time.Minute, func() *resource.RetryError {
		_, err := conn.PutLifecycleHook(params)

		if err != nil {
			if awsErr, ok := err.(awserr.Error); ok {
				if strings.Contains(awsErr.Message(), "Unable to publish test message to notification target") {
					return resource.RetryableError(fmt.Errorf("Retrying AWS AutoScaling Lifecycle Hook: %s", awsErr))
				}
			}
			return resource.NonRetryableError(fmt.Errorf("Error putting lifecycle hook: %s", err))
		}
		return nil
	})
	if isResourceTimeoutError(err) {
		_, err = conn.PutLifecycleHook(params)
	}
	if err != nil {
		return fmt.Errorf("Error putting autoscaling lifecycle hook: %s", err)
	}
	return nil
}

func resourceAwsAutoscalingLifecycleHookPut(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).autoscalingconn
	params := getAwsAutoscalingPutLifecycleHookInput(d)

	if err := resourceAwsAutoscalingLifecycleHookPutOp(conn, &params); err != nil {
		return err
	}

	d.SetId(d.Get("name").(string))

	return resourceAwsAutoscalingLifecycleHookRead(d, meta)
}

func resourceAwsAutoscalingLifecycleHookRead(d *schema.ResourceData, meta interface{}) error {
	p, err := getAwsAutoscalingLifecycleHook(d, meta)
	if err != nil {
		return err
	}
	if p == nil {
		log.Printf("[WARN] Autoscaling Lifecycle Hook (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	log.Printf("[DEBUG] Read Lifecycle Hook: ASG: %s, SH: %s, Obj: %#v", d.Get("autoscaling_group_name"), d.Get("name"), p)

	d.Set("default_result", p.DefaultResult)
	d.Set("heartbeat_timeout", p.HeartbeatTimeout)
	d.Set("lifecycle_transition", p.LifecycleTransition)
	d.Set("notification_metadata", p.NotificationMetadata)
	d.Set("notification_target_arn", p.NotificationTargetARN)
	d.Set("name", p.LifecycleHookName)
	d.Set("role_arn", p.RoleARN)

	return nil
}

func resourceAwsAutoscalingLifecycleHookDelete(d *schema.ResourceData, meta interface{}) error {
	autoscalingconn := meta.(*AWSClient).autoscalingconn
	p, err := getAwsAutoscalingLifecycleHook(d, meta)
	if err != nil {
		return err
	}
	if p == nil {
		return nil
	}

	params := autoscaling.DeleteLifecycleHookInput{
		AutoScalingGroupName: aws.String(d.Get("autoscaling_group_name").(string)),
		LifecycleHookName:    aws.String(d.Get("name").(string)),
	}
	if _, err := autoscalingconn.DeleteLifecycleHook(&params); err != nil {
		return fmt.Errorf("Autoscaling Lifecycle Hook: %s", err)
	}

	return nil
}

func getAwsAutoscalingPutLifecycleHookInput(d *schema.ResourceData) autoscaling.PutLifecycleHookInput {
	var params = autoscaling.PutLifecycleHookInput{
		AutoScalingGroupName: aws.String(d.Get("autoscaling_group_name").(string)),
		LifecycleHookName:    aws.String(d.Get("name").(string)),
	}

	if v, ok := d.GetOk("default_result"); ok {
		params.DefaultResult = aws.String(v.(string))
	}

	if v, ok := d.GetOk("heartbeat_timeout"); ok {
		params.HeartbeatTimeout = aws.Int64(int64(v.(int)))
	}

	if v, ok := d.GetOk("lifecycle_transition"); ok {
		params.LifecycleTransition = aws.String(v.(string))
	}

	if v, ok := d.GetOk("notification_metadata"); ok {
		params.NotificationMetadata = aws.String(v.(string))
	}

	if v, ok := d.GetOk("notification_target_arn"); ok {
		params.NotificationTargetARN = aws.String(v.(string))
	}

	if v, ok := d.GetOk("role_arn"); ok {
		params.RoleARN = aws.String(v.(string))
	}

	return params
}

func getAwsAutoscalingLifecycleHook(d *schema.ResourceData, meta interface{}) (*autoscaling.LifecycleHook, error) {
	autoscalingconn := meta.(*AWSClient).autoscalingconn

	params := autoscaling.DescribeLifecycleHooksInput{
		AutoScalingGroupName: aws.String(d.Get("autoscaling_group_name").(string)),
		LifecycleHookNames:   []*string{aws.String(d.Get("name").(string))},
	}

	log.Printf("[DEBUG] AutoScaling Lifecycle Hook Describe Params: %#v", params)
	resp, err := autoscalingconn.DescribeLifecycleHooks(&params)
	if err != nil {
		return nil, fmt.Errorf("Error retrieving lifecycle hooks: %s", err)
	}

	// find lifecycle hooks
	name := d.Get("name")
	for idx, sp := range resp.LifecycleHooks {
		if sp == nil {
			continue
		}

		if aws.StringValue(sp.LifecycleHookName) == name {
			return resp.LifecycleHooks[idx], nil
		}
	}

	// lifecycle hook not found
	return nil, nil
}

func resourceAwsAutoscalingLifecycleHookImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	idParts := strings.SplitN(d.Id(), "/", 2)
	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		return nil, fmt.Errorf("unexpected format (%q), expected <asg-name>/<lifecycle-hook-name>", d.Id())
	}

	asgName := idParts[0]
	lifecycleHookName := idParts[1]

	d.Set("name", lifecycleHookName)
	d.Set("autoscaling_group_name", asgName)
	d.SetId(lifecycleHookName)

	return []*schema.ResourceData{d}, nil
}
