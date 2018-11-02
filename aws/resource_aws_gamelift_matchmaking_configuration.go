package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/gamelift"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceAwsGameliftMatchmakingConfiguration() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsGameliftMatchmakingConfigurationCreate,
		Read:   resourceAwsGameliftMatchmakingConfigurationRead,
		Update: resourceAwsGameliftMatchmakingConfigurationUpdate,
		Delete: resourceAwsGameliftMatchmakingConfigurationDelete,

		Schema: map[string]*schema.Schema{
			"acceptance_required": {
				Type:     schema.TypeBool,
				Required: true,
			},
			"acceptance_timeout_seconds": {
				Type:         schema.TypeInt,
				Optional:     true,
				ValidateFunc: validation.IntBetween(1, 600),
			},
			"additional_player_count": {
				Type:         schema.TypeInt,
				Optional:     true,
				ValidateFunc: validation.IntAtLeast(0),
			},
			"custom_event_data": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringLenBetween(0, 256),
			},
			"description": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringLenBetween(1, 1024),
			},
			"game_property": {
				Type:     schema.TypeSet,
				Optional: true,
				MaxItems: 16,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"key": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validation.StringLenBetween(1, 32),
						},
						"value": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validation.StringLenBetween(1, 64),
						},
					},
				},
			},
			"game_session_data": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringLenBetween(1, 4096),
			},
			"game_session_queue_arns": {
				Type:     schema.TypeList,
				Required: true,
				MinItems: 1,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringLenBetween(1, 128),
			},
			"notification_target": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringLenBetween(0, 300),
			},
			"request_timeout_seconds": {
				Type:         schema.TypeInt,
				Required:     true,
				ValidateFunc: validation.IntBetween(1, 43200),
			},
			"rule_set_name": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringLenBetween(1, 128),
			},
		},
	}
}

func resourceAwsGameliftMatchmakingConfigurationCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).gameliftconn

	input := gamelift.CreateMatchmakingConfigurationInput{
		AcceptanceRequired:   aws.Bool(d.Get("acceptance_required").(bool)),
		GameSessionQueueArns: aws.StringSlice(d.Get("game_session_queue_arns").([]string)),
		Name:                 aws.String(d.Get("name").(string)),
		RequestTimeoutSeconds: aws.Int64(d.Get("request_timeout_seconds").(int64)),
		RuleSetName:           aws.String(d.Get("rule_set_name").(string)),
	}
	if v, ok := d.GetOk("acceptance_timeout_seconds"); ok {
		input.AcceptanceTimeoutSeconds = aws.Int64(v.(int64))
	}
	if v, ok := d.GetOk("additional_player_count"); ok {
		input.AdditionalPlayerCount = aws.Int64(v.(int64))
	}
	if v, ok := d.GetOk("custom_event_data"); ok {
		input.CustomEventData = aws.String(v.(string))
	}
	if v, ok := d.GetOk("description"); ok {
		input.Description = aws.String(v.(string))
	}
	if v, ok := d.GetOk("game_property"); ok {
		input.GameProperties = expandGameliftGameProperties(v.([]interface{}))
	}
	if v, ok := d.GetOk("game_session_data"); ok {
		input.GameSessionData = aws.String(v.(string))
	}
	if v, ok := d.GetOk("notification_target"); ok {
		input.NotificationTarget = aws.String(v.(string))
	}
	log.Printf("[INFO] Creating Gamelift Matchmaking Configuration: %s", input)
	out, err := conn.CreateMatchmakingConfiguration(&input)
	if err != nil {
		return err
	}

	d.SetId(*out.Configuration.Name)

	return resourceAwsGameliftMatchmakingConfigurationRead(d, meta)
}

func resourceAwsGameliftMatchmakingConfigurationRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).gameliftconn

	log.Printf("[INFO] Reading Gamelift Matchmaking Configuration: %s", d.Id())
	limit := int64(1)
	out, err := conn.DescribeMatchmakingConfigurations(&gamelift.DescribeMatchmakingConfigurationsInput{
		Names: aws.StringSlice([]string{d.Get("name").(string)}),
		Limit: &limit,
	})
	if err != nil {
		if isAWSErr(err, gamelift.ErrCodeNotFoundException, "") {
			log.Printf("[WARN] Gamelift MatchmakingConfiguration (%s) not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}
	configurations := out.Configurations

	if len(configurations) < 1 {
		log.Printf("[WARN] Gamelift Matchmaking Configuration (%s) not found, removing from state", d.Get("name"))
		d.SetId("")
		return nil
	}
	if len(configurations) != 1 {
		return fmt.Errorf("expected exactly 1 Gamelift Matchmaking Configuration, found %d under %q",
			len(configurations), d.Get("name"))
	}
	c := configurations[0]

	d.Set("acceptance_required", c.AcceptanceRequired)
	d.Set("acceptance_timeout_seconds", c.AcceptanceTimeoutSeconds)
	d.Set("additional_player_count", c.AdditionalPlayerCount)
	d.Set("custom_event_data", c.CustomEventData)
	d.Set("description", c.Description)
	if err := d.Set("game_property", flattenGameliftGameProperties(c.GameProperties)); err != nil {
		return fmt.Errorf("error setting game_properties: %s", err)
	}
	d.Set("game_session_data", c.GameSessionData)
	d.Set("game_session_queue_arns", c.GameSessionQueueArns)
	d.Set("notification_target", c.NotificationTarget)
	d.Set("request_timeout_seconds", c.RequestTimeoutSeconds)
	d.Set("rule_set_name", c.RuleSetName)

	return nil
}

func resourceAwsGameliftMatchmakingConfigurationUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).gameliftconn

	log.Printf("[INFO] Updating Gamelift Matchmaking Configuration: %s", d.Id())
	input := gamelift.UpdateMatchmakingConfigurationInput{
		AcceptanceRequired:   aws.Bool(d.Get("acceptance_required").(bool)),
		GameSessionQueueArns: aws.StringSlice(d.Get("game_session_queue_arns").([]string)),
		Name:                 aws.String(d.Get("name").(string)),
		RequestTimeoutSeconds: aws.Int64(d.Get("request_timeout_seconds").(int64)),
		RuleSetName:           aws.String(d.Get("rule_set_name").(string)),
	}
	if v, ok := d.GetOk("acceptance_timeout_seconds"); ok {
		input.AcceptanceTimeoutSeconds = aws.Int64(v.(int64))
	}
	if v, ok := d.GetOk("additional_player_count"); ok {
		input.AdditionalPlayerCount = aws.Int64(v.(int64))
	}
	if v, ok := d.GetOk("custom_event_data"); ok {
		input.CustomEventData = aws.String(v.(string))
	}
	if v, ok := d.GetOk("description"); ok {
		input.Description = aws.String(v.(string))
	}
	if v, ok := d.GetOk("game_property"); ok {
		input.GameProperties = expandGameliftGameProperties(v.([]interface{}))
	}
	if v, ok := d.GetOk("game_session_data"); ok {
		input.GameSessionData = aws.String(v.(string))
	}
	if v, ok := d.GetOk("notification_target"); ok {
		input.NotificationTarget = aws.String(v.(string))
	}

	_, err := conn.UpdateMatchmakingConfiguration(&input)
	if err != nil {
		return err
	}

	return resourceAwsGameliftMatchmakingConfigurationRead(d, meta)
}

func resourceAwsGameliftMatchmakingConfigurationDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).gameliftconn

	log.Printf("[INFO] Deleting Gamelift Matchmaking Configuration: %s", d.Id())
	_, err := conn.DeleteMatchmakingConfiguration(&gamelift.DeleteMatchmakingConfigurationInput{
		Name: aws.String(d.Get("name").(string)),
	})
	if isAWSErr(err, gamelift.ErrCodeNotFoundException, "") {
		return nil
	}
	if err != nil {
		return err
	}

	return nil
}

func flattenGameliftGameProperties(gameProperties []*gamelift.GameProperty) []interface{} {
	gamePropertyResources := []interface{}{}
	for _, gameProperty := range gameProperties {
		gamePropertyResources = append(
			gamePropertyResources,
			map[string]interface{}{
				"key":   aws.String(*gameProperty.Key),
				"value": aws.String(*gameProperty.Value),
			})
	}
	return gamePropertyResources
}

func expandGameliftGameProperties(gamePropertyResources []interface{}) []*gamelift.GameProperty {
	var gameProperties []*gamelift.GameProperty
	for _, gamePropertyResource := range gamePropertyResources {
		m := gamePropertyResource.(map[string]interface{})
		gameProperties = append(
			gameProperties,
			&gamelift.GameProperty{Key: aws.String(m["key"].(string)), Value: aws.String(m["value"].(string))})
	}
	return gameProperties
}
