package ag

import (
	"fmt"
	"log"
	"regexp"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/waf"
	"github.com/hashicorp/aws-sdk-go-base/tfawserr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/realglebivanov/terraform-provider-awsgamelift/ag/internal/keyvaluetags"
)

func resourceAwsWafWebAcl() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsWafWebAclCreate,
		Read:   resourceAwsWafWebAclRead,
		Update: resourceAwsWafWebAclUpdate,
		Delete: resourceAwsWafWebAclDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"default_action": {
				Type:     schema.TypeList,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"metric_name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringMatch(regexp.MustCompile(`^[0-9A-Za-z]+$`), "must contain only alphanumeric characters"),
			},
			"logging_configuration": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"log_destination": {
							Type:     schema.TypeString,
							Required: true,
						},
						"redacted_fields": {
							Type:     schema.TypeList,
							Optional: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"field_to_match": {
										Type:     schema.TypeSet,
										Required: true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"data": {
													Type:     schema.TypeString,
													Optional: true,
												},
												"type": {
													Type:     schema.TypeString,
													Required: true,
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			"rules": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"action": {
							Type:     schema.TypeList,
							Optional: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"type": {
										Type:     schema.TypeString,
										Required: true,
									},
								},
							},
						},
						"override_action": {
							Type:     schema.TypeList,
							Optional: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"type": {
										Type:     schema.TypeString,
										Required: true,
									},
								},
							},
						},
						"priority": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"type": {
							Type:         schema.TypeString,
							Optional:     true,
							Default:      waf.WafRuleTypeRegular,
							ValidateFunc: validation.StringInSlice(waf.WafRuleType_Values(), false),
						},
						"rule_id": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"tags":     tagsSchema(),
			"tags_all": tagsSchemaComputed(),
		},

		CustomizeDiff: SetTagsDiff,
	}
}

func resourceAwsWafWebAclCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafconn
	defaultTagsConfig := meta.(*AWSClient).DefaultTagsConfig
	tags := defaultTagsConfig.MergeTags(keyvaluetags.New(d.Get("tags").(map[string]interface{})))

	wr := newWafRetryer(conn)
	out, err := wr.RetryWithToken(func(token *string) (interface{}, error) {
		params := &waf.CreateWebACLInput{
			ChangeToken:   token,
			DefaultAction: expandWafAction(d.Get("default_action").([]interface{})),
			MetricName:    aws.String(d.Get("metric_name").(string)),
			Name:          aws.String(d.Get("name").(string)),
		}

		if len(tags) > 0 {
			params.Tags = tags.IgnoreAws().WafTags()
		}

		return conn.CreateWebACL(params)
	})

	if err != nil {
		return fmt.Errorf("error creating WAF Web ACL (%s): %w", d.Get("name").(string), err)
	}

	resp := out.(*waf.CreateWebACLOutput)
	d.SetId(aws.StringValue(resp.WebACL.WebACLId))

	arn := arn.ARN{
		Partition: meta.(*AWSClient).partition,
		Service:   "waf",
		AccountID: meta.(*AWSClient).accountid,
		Resource:  fmt.Sprintf("webacl/%s", d.Id()),
	}.String()

	loggingConfiguration := d.Get("logging_configuration").([]interface{})
	if len(loggingConfiguration) == 1 {
		input := &waf.PutLoggingConfigurationInput{
			LoggingConfiguration: expandWAFLoggingConfiguration(loggingConfiguration, arn),
		}

		if _, err := conn.PutLoggingConfiguration(input); err != nil {
			return fmt.Errorf("error putting WAF Web ACL (%s) Logging Configuration: %w", d.Id(), err)
		}
	}

	rules := d.Get("rules").(*schema.Set).List()
	if len(rules) > 0 {
		wr := newWafRetryer(conn)
		_, err := wr.RetryWithToken(func(token *string) (interface{}, error) {
			req := &waf.UpdateWebACLInput{
				ChangeToken:   token,
				DefaultAction: expandWafAction(d.Get("default_action").([]interface{})),
				Updates:       diffWafWebAclRules([]interface{}{}, rules),
				WebACLId:      aws.String(d.Id()),
			}
			return conn.UpdateWebACL(req)
		})

		if err != nil {
			return fmt.Errorf("error updating WAF Web ACL (%s): %w", d.Id(), err)
		}
	}

	return resourceAwsWafWebAclRead(d, meta)
}

func resourceAwsWafWebAclRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafconn
	defaultTagsConfig := meta.(*AWSClient).DefaultTagsConfig
	ignoreTagsConfig := meta.(*AWSClient).IgnoreTagsConfig

	params := &waf.GetWebACLInput{
		WebACLId: aws.String(d.Id()),
	}

	resp, err := conn.GetWebACL(params)
	if !d.IsNewResource() && tfawserr.ErrCodeEquals(err, waf.ErrCodeNonexistentItemException) {
		log.Printf("[WARN] WAF Web ACL (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err != nil {
		return fmt.Errorf("error reading WAF Web ACL (%s): %w", d.Id(), err)
	}

	if resp == nil || resp.WebACL == nil {
		if d.IsNewResource() {
			return fmt.Errorf("error reading WAF Web ACL (%s): not found", d.Id())
		}

		log.Printf("[WARN] WAF Web ACL (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	d.Set("arn", resp.WebACL.WebACLArn)
	arn := aws.StringValue(resp.WebACL.WebACLArn)

	if err := d.Set("default_action", flattenWafAction(resp.WebACL.DefaultAction)); err != nil {
		return fmt.Errorf("error setting default_action: %w", err)
	}
	d.Set("name", resp.WebACL.Name)
	d.Set("metric_name", resp.WebACL.MetricName)

	tags, err := keyvaluetags.WafListTags(conn, arn)
	if err != nil {
		return fmt.Errorf("error listing tags for WAF Web ACL (%s): %w", arn, err)
	}

	tags = tags.IgnoreAws().IgnoreConfig(ignoreTagsConfig)

	//lintignore:AWSR002
	if err := d.Set("tags", tags.RemoveDefaultConfig(defaultTagsConfig).Map()); err != nil {
		return fmt.Errorf("error setting tags: %w", err)
	}

	if err := d.Set("tags_all", tags.Map()); err != nil {
		return fmt.Errorf("error setting tags_all: %w", err)
	}

	if err := d.Set("rules", flattenWafWebAclRules(resp.WebACL.Rules)); err != nil {
		return fmt.Errorf("error setting rules: %w", err)
	}

	getLoggingConfigurationInput := &waf.GetLoggingConfigurationInput{
		ResourceArn: aws.String(arn),
	}
	loggingConfiguration := []interface{}{}

	getLoggingConfigurationOutput, err := conn.GetLoggingConfiguration(getLoggingConfigurationInput)

	if err != nil && !tfawserr.ErrCodeEquals(err, waf.ErrCodeNonexistentItemException) {
		return fmt.Errorf("error reading WAF Web ACL (%s) Logging Configuration: %w", d.Id(), err)
	}

	if getLoggingConfigurationOutput != nil {
		loggingConfiguration = flattenWAFLoggingConfiguration(getLoggingConfigurationOutput.LoggingConfiguration)
	}

	if err := d.Set("logging_configuration", loggingConfiguration); err != nil {
		return fmt.Errorf("error setting logging_configuration: %w", err)
	}

	return nil
}

func resourceAwsWafWebAclUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafconn

	if d.HasChanges("default_action", "rules") {
		o, n := d.GetChange("rules")
		oldR, newR := o.(*schema.Set).List(), n.(*schema.Set).List()

		wr := newWafRetryer(conn)
		_, err := wr.RetryWithToken(func(token *string) (interface{}, error) {
			req := &waf.UpdateWebACLInput{
				ChangeToken:   token,
				DefaultAction: expandWafAction(d.Get("default_action").([]interface{})),
				Updates:       diffWafWebAclRules(oldR, newR),
				WebACLId:      aws.String(d.Id()),
			}
			return conn.UpdateWebACL(req)
		})
		if err != nil {
			return fmt.Errorf("error updating WAF Web ACL (%s): %w", d.Id(), err)
		}
	}

	if d.HasChange("logging_configuration") {
		loggingConfiguration := d.Get("logging_configuration").([]interface{})

		if len(loggingConfiguration) == 1 {
			input := &waf.PutLoggingConfigurationInput{
				LoggingConfiguration: expandWAFLoggingConfiguration(loggingConfiguration, d.Get("arn").(string)),
			}

			if _, err := conn.PutLoggingConfiguration(input); err != nil {
				return fmt.Errorf("error updating WAF Web ACL (%s) Logging Configuration: %w", d.Id(), err)
			}
		} else {
			input := &waf.DeleteLoggingConfigurationInput{
				ResourceArn: aws.String(d.Get("arn").(string)),
			}

			if _, err := conn.DeleteLoggingConfiguration(input); err != nil {
				return fmt.Errorf("error deleting WAF Web ACL (%s) Logging Configuration: %w", d.Id(), err)
			}
		}

	}

	if d.HasChange("tags_all") {
		o, n := d.GetChange("tags_all")

		if err := keyvaluetags.WafUpdateTags(conn, d.Get("arn").(string), o, n); err != nil {
			return fmt.Errorf("error updating WAF Web ACL (%s) tags: %w", d.Id(), err)
		}
	}

	return resourceAwsWafWebAclRead(d, meta)
}

func resourceAwsWafWebAclDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafconn

	// First, need to delete all rules
	rules := d.Get("rules").(*schema.Set).List()
	if len(rules) > 0 {
		wr := newWafRetryer(conn)
		_, err := wr.RetryWithToken(func(token *string) (interface{}, error) {
			req := &waf.UpdateWebACLInput{
				ChangeToken:   token,
				DefaultAction: expandWafAction(d.Get("default_action").([]interface{})),
				Updates:       diffWafWebAclRules(rules, []interface{}{}),
				WebACLId:      aws.String(d.Id()),
			}
			return conn.UpdateWebACL(req)
		})
		if err != nil {
			return fmt.Errorf("error removing WAF Web ACL (%s) rules: %w", d.Id(), err)
		}
	}

	wr := newWafRetryer(conn)
	_, err := wr.RetryWithToken(func(token *string) (interface{}, error) {
		req := &waf.DeleteWebACLInput{
			ChangeToken: token,
			WebACLId:    aws.String(d.Id()),
		}

		return conn.DeleteWebACL(req)
	})

	if err != nil {
		if tfawserr.ErrCodeEquals(err, waf.ErrCodeNonexistentItemException) {
			return nil
		}
		return fmt.Errorf("error deleting WAF Web ACL (%s): %w", d.Id(), err)
	}

	return nil
}

func expandWAFLoggingConfiguration(l []interface{}, resourceARN string) *waf.LoggingConfiguration {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	m := l[0].(map[string]interface{})

	loggingConfiguration := &waf.LoggingConfiguration{
		LogDestinationConfigs: []*string{
			aws.String(m["log_destination"].(string)),
		},
		RedactedFields: expandWAFRedactedFields(m["redacted_fields"].([]interface{})),
		ResourceArn:    aws.String(resourceARN),
	}

	return loggingConfiguration
}

func expandWAFRedactedFields(l []interface{}) []*waf.FieldToMatch {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	m := l[0].(map[string]interface{})

	if m["field_to_match"] == nil {
		return nil
	}

	redactedFields := make([]*waf.FieldToMatch, 0)

	for _, fieldToMatch := range m["field_to_match"].(*schema.Set).List() {
		if fieldToMatch == nil {
			continue
		}

		redactedFields = append(redactedFields, expandFieldToMatch(fieldToMatch.(map[string]interface{})))
	}

	return redactedFields
}

func flattenWAFLoggingConfiguration(loggingConfiguration *waf.LoggingConfiguration) []interface{} {
	if loggingConfiguration == nil {
		return []interface{}{}
	}

	m := map[string]interface{}{
		"log_destination": "",
		"redacted_fields": flattenWAFRedactedFields(loggingConfiguration.RedactedFields),
	}

	if len(loggingConfiguration.LogDestinationConfigs) > 0 {
		m["log_destination"] = aws.StringValue(loggingConfiguration.LogDestinationConfigs[0])
	}

	return []interface{}{m}
}

func flattenWAFRedactedFields(fieldToMatches []*waf.FieldToMatch) []interface{} {
	if len(fieldToMatches) == 0 {
		return []interface{}{}
	}

	fieldToMatchResource := &schema.Resource{
		Schema: map[string]*schema.Schema{
			"data": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"type": {
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
	l := make([]interface{}, len(fieldToMatches))

	for i, fieldToMatch := range fieldToMatches {
		l[i] = flattenFieldToMatch(fieldToMatch)[0]
	}

	m := map[string]interface{}{
		"field_to_match": schema.NewSet(schema.HashResource(fieldToMatchResource), l),
	}

	return []interface{}{m}
}
