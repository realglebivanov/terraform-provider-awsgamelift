package ag

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/codepipeline"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/realglebivanov/terraform-provider-awsgamelift/ag/internal/keyvaluetags"
	iamwaiter "github.com/realglebivanov/terraform-provider-awsgamelift/ag/internal/service/iam/waiter"
)

const (
	CodePipelineProviderGitHub = "GitHub"

	CodePipelineGitHubActionConfigurationOAuthToken = "OAuthToken"
)

func resourceAwsCodePipeline() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsCodePipelineCreate,
		Read:   resourceAwsCodePipelineRead,
		Update: resourceAwsCodePipelineUpdate,
		Delete: resourceAwsCodePipelineDelete,
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

			"role_arn": {
				Type:     schema.TypeString,
				Required: true,
			},
			"artifact_store": {
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"location": {
							Type:     schema.TypeString,
							Required: true,
						},
						"type": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validation.StringInSlice(codepipeline.ArtifactStoreType_Values(), false),
						},
						"encryption_key": {
							Type:     schema.TypeList,
							MaxItems: 1,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"id": {
										Type:     schema.TypeString,
										Required: true,
									},
									"type": {
										Type:         schema.TypeString,
										Required:     true,
										ValidateFunc: validation.StringInSlice(codepipeline.EncryptionKeyType_Values(), false),
									},
								},
							},
						},
						"region": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
					},
				},
			},
			"stage": {
				Type:     schema.TypeList,
				MinItems: 2,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"action": {
							Type:     schema.TypeList,
							Required: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"configuration": {
										Type:             schema.TypeMap,
										Optional:         true,
										Elem:             &schema.Schema{Type: schema.TypeString},
										DiffSuppressFunc: suppressCodePipelineStageActionConfiguration,
									},
									"category": {
										Type:         schema.TypeString,
										Required:     true,
										ValidateFunc: validation.StringInSlice(codepipeline.ActionCategory_Values(), false),
									},
									"owner": {
										Type:         schema.TypeString,
										Required:     true,
										ValidateFunc: validation.StringInSlice(codepipeline.ActionOwner_Values(), false),
									},
									"provider": {
										Type:             schema.TypeString,
										Required:         true,
										ValidateDiagFunc: resourceAwsCodePipelineValidateActionProvider,
									},
									"version": {
										Type:     schema.TypeString,
										Required: true,
									},
									"input_artifacts": {
										Type:     schema.TypeList,
										Optional: true,
										Elem:     &schema.Schema{Type: schema.TypeString},
									},
									"output_artifacts": {
										Type:     schema.TypeList,
										Optional: true,
										Elem:     &schema.Schema{Type: schema.TypeString},
									},
									"name": {
										Type:     schema.TypeString,
										Required: true,
									},
									"role_arn": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"run_order": {
										Type:     schema.TypeInt,
										Optional: true,
										Computed: true,
									},
									"region": {
										Type:     schema.TypeString,
										Optional: true,
										Computed: true,
									},
									"namespace": {
										Type:     schema.TypeString,
										Optional: true,
									},
								},
							},
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

func resourceAwsCodePipelineCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).codepipelineconn
	defaultTagsConfig := meta.(*AWSClient).DefaultTagsConfig
	tags := defaultTagsConfig.MergeTags(keyvaluetags.New(d.Get("tags").(map[string]interface{})))

	pipeline, err := expandAwsCodePipeline(d)
	if err != nil {
		return err
	}
	params := &codepipeline.CreatePipelineInput{
		Pipeline: pipeline,
		Tags:     tags.IgnoreAws().CodepipelineTags(),
	}

	var resp *codepipeline.CreatePipelineOutput
	err = resource.Retry(iamwaiter.PropagationTimeout, func() *resource.RetryError {
		var err error

		resp, err = conn.CreatePipeline(params)

		if isAWSErr(err, codepipeline.ErrCodeInvalidStructureException, "not authorized") {
			return resource.RetryableError(err)
		}

		if err != nil {
			return resource.NonRetryableError(err)
		}

		return nil
	})
	if isResourceTimeoutError(err) {
		resp, err = conn.CreatePipeline(params)
	}
	if err != nil {
		return fmt.Errorf("Error creating CodePipeline: %w", err)
	}
	if resp.Pipeline == nil {
		return fmt.Errorf("Error creating CodePipeline: invalid response from AWS")
	}

	d.SetId(aws.StringValue(resp.Pipeline.Name))

	return resourceAwsCodePipelineRead(d, meta)
}

func expandAwsCodePipeline(d *schema.ResourceData) (*codepipeline.PipelineDeclaration, error) {
	pipeline := codepipeline.PipelineDeclaration{
		Name:    aws.String(d.Get("name").(string)),
		RoleArn: aws.String(d.Get("role_arn").(string)),
		Stages:  expandAwsCodePipelineStages(d),
	}

	pipelineArtifactStores, err := expandAwsCodePipelineArtifactStores(d.Get("artifact_store").(*schema.Set).List())
	if err != nil {
		return nil, err
	}
	if len(pipelineArtifactStores) == 1 {
		for _, v := range pipelineArtifactStores {
			pipeline.ArtifactStore = v
		}
	} else {
		pipeline.ArtifactStores = pipelineArtifactStores
	}

	return &pipeline, nil
}

func expandAwsCodePipelineArtifactStores(configs []interface{}) (map[string]*codepipeline.ArtifactStore, error) {
	if len(configs) == 0 {
		return nil, nil
	}

	regions := make([]string, 0, len(configs))
	pipelineArtifactStores := make(map[string]*codepipeline.ArtifactStore)
	for _, config := range configs {
		region, store := expandAwsCodePipelineArtifactStoreData(config.(map[string]interface{}))
		regions = append(regions, region)
		pipelineArtifactStores[region] = store
	}

	if len(regions) == 1 {
		if regions[0] != "" {
			return nil, errors.New("region cannot be set for a single-region CodePipeline")
		}
	} else {
		for _, v := range regions {
			if v == "" {
				return nil, errors.New("region must be set for a cross-region CodePipeline")
			}
		}
		if len(configs) != len(pipelineArtifactStores) {
			return nil, errors.New("only one Artifact Store can be defined per region for a cross-region CodePipeline")
		}
	}

	return pipelineArtifactStores, nil
}

func expandAwsCodePipelineArtifactStoreData(data map[string]interface{}) (string, *codepipeline.ArtifactStore) {
	pipelineArtifactStore := codepipeline.ArtifactStore{
		Location: aws.String(data["location"].(string)),
		Type:     aws.String(data["type"].(string)),
	}
	tek := data["encryption_key"].([]interface{})
	if len(tek) > 0 {
		vk := tek[0].(map[string]interface{})
		ek := codepipeline.EncryptionKey{
			Type: aws.String(vk["type"].(string)),
			Id:   aws.String(vk["id"].(string)),
		}
		pipelineArtifactStore.EncryptionKey = &ek
	}

	return data["region"].(string), &pipelineArtifactStore
}

func flattenAwsCodePipelineArtifactStore(artifactStore *codepipeline.ArtifactStore) []interface{} {
	if artifactStore == nil {
		return []interface{}{}
	}

	values := map[string]interface{}{}
	values["type"] = aws.StringValue(artifactStore.Type)
	values["location"] = aws.StringValue(artifactStore.Location)
	if artifactStore.EncryptionKey != nil {
		as := map[string]interface{}{
			"id":   aws.StringValue(artifactStore.EncryptionKey.Id),
			"type": aws.StringValue(artifactStore.EncryptionKey.Type),
		}
		values["encryption_key"] = []interface{}{as}
	}
	return []interface{}{values}
}

func flattenAwsCodePipelineArtifactStores(artifactStores map[string]*codepipeline.ArtifactStore) []interface{} {
	values := []interface{}{}
	for region, artifactStore := range artifactStores {
		store := flattenAwsCodePipelineArtifactStore(artifactStore)[0].(map[string]interface{})
		store["region"] = region
		values = append(values, store)
	}
	return values
}

func expandAwsCodePipelineStages(d *schema.ResourceData) []*codepipeline.StageDeclaration {
	stages := d.Get("stage").([]interface{})
	pipelineStages := []*codepipeline.StageDeclaration{}

	for _, stage := range stages {
		data := stage.(map[string]interface{})
		a := data["action"].([]interface{})
		actions := expandAwsCodePipelineActions(a)
		pipelineStages = append(pipelineStages, &codepipeline.StageDeclaration{
			Name:    aws.String(data["name"].(string)),
			Actions: actions,
		})
	}
	return pipelineStages
}

func flattenAwsCodePipelineStages(stages []*codepipeline.StageDeclaration, d *schema.ResourceData) []interface{} {
	stagesList := []interface{}{}
	for si, stage := range stages {
		values := map[string]interface{}{}
		values["name"] = aws.StringValue(stage.Name)
		values["action"] = flattenAwsCodePipelineStageActions(si, stage.Actions, d)
		stagesList = append(stagesList, values)
	}
	return stagesList
}

func expandAwsCodePipelineActions(a []interface{}) []*codepipeline.ActionDeclaration {
	actions := []*codepipeline.ActionDeclaration{}
	for _, config := range a {
		data := config.(map[string]interface{})

		conf := expandStringMap(data["configuration"].(map[string]interface{}))

		action := codepipeline.ActionDeclaration{
			ActionTypeId: &codepipeline.ActionTypeId{
				Category: aws.String(data["category"].(string)),
				Owner:    aws.String(data["owner"].(string)),

				Provider: aws.String(data["provider"].(string)),
				Version:  aws.String(data["version"].(string)),
			},
			Name:          aws.String(data["name"].(string)),
			Configuration: conf,
		}

		oa := data["output_artifacts"].([]interface{})
		if len(oa) > 0 {
			outputArtifacts := expandAwsCodePipelineActionsOutputArtifacts(oa)
			action.OutputArtifacts = outputArtifacts

		}
		ia := data["input_artifacts"].([]interface{})
		if len(ia) > 0 {
			inputArtifacts := expandAwsCodePipelineActionsInputArtifacts(ia)
			action.InputArtifacts = inputArtifacts

		}
		ra := data["role_arn"].(string)
		if ra != "" {
			action.RoleArn = aws.String(ra)
		}
		ro := data["run_order"].(int)
		if ro > 0 {
			action.RunOrder = aws.Int64(int64(ro))
		}
		r := data["region"].(string)
		if r != "" {
			action.Region = aws.String(r)
		}
		ns := data["namespace"].(string)
		if len(ns) > 0 {
			action.Namespace = aws.String(ns)
		}
		actions = append(actions, &action)
	}
	return actions
}

func flattenAwsCodePipelineStageActions(si int, actions []*codepipeline.ActionDeclaration, d *schema.ResourceData) []interface{} {
	actionsList := []interface{}{}
	for ai, action := range actions {
		values := map[string]interface{}{
			"category": aws.StringValue(action.ActionTypeId.Category),
			"owner":    aws.StringValue(action.ActionTypeId.Owner),
			"provider": aws.StringValue(action.ActionTypeId.Provider),
			"version":  aws.StringValue(action.ActionTypeId.Version),
			"name":     aws.StringValue(action.Name),
		}
		if action.Configuration != nil {
			config := aws.StringValueMap(action.Configuration)

			actionProvider := aws.StringValue(action.ActionTypeId.Provider)
			if actionProvider == CodePipelineProviderGitHub {
				if _, ok := config[CodePipelineGitHubActionConfigurationOAuthToken]; ok {
					// The AWS API returns "****" for the OAuthToken value. Pull the value from the configuration.
					addr := fmt.Sprintf("stage.%d.action.%d.configuration.OAuthToken", si, ai)
					config[CodePipelineGitHubActionConfigurationOAuthToken] = d.Get(addr).(string)
				}
			}

			values["configuration"] = config
		}

		if len(action.OutputArtifacts) > 0 {
			values["output_artifacts"] = flattenAwsCodePipelineActionsOutputArtifacts(action.OutputArtifacts)
		}

		if len(action.InputArtifacts) > 0 {
			values["input_artifacts"] = flattenAwsCodePipelineActionsInputArtifacts(action.InputArtifacts)
		}

		if action.RoleArn != nil {
			values["role_arn"] = aws.StringValue(action.RoleArn)
		}

		if action.RunOrder != nil {
			values["run_order"] = int(aws.Int64Value(action.RunOrder))
		}

		if action.Region != nil {
			values["region"] = aws.StringValue(action.Region)
		}

		if action.Namespace != nil {
			values["namespace"] = aws.StringValue(action.Namespace)
		}

		actionsList = append(actionsList, values)
	}
	return actionsList
}

func expandAwsCodePipelineActionsOutputArtifacts(s []interface{}) []*codepipeline.OutputArtifact {
	outputArtifacts := []*codepipeline.OutputArtifact{}
	for _, artifact := range s {
		if artifact == nil {
			continue
		}
		outputArtifacts = append(outputArtifacts, &codepipeline.OutputArtifact{
			Name: aws.String(artifact.(string)),
		})
	}
	return outputArtifacts
}

func flattenAwsCodePipelineActionsOutputArtifacts(artifacts []*codepipeline.OutputArtifact) []string {
	values := []string{}
	for _, artifact := range artifacts {
		values = append(values, aws.StringValue(artifact.Name))
	}
	return values
}

func expandAwsCodePipelineActionsInputArtifacts(s []interface{}) []*codepipeline.InputArtifact {
	outputArtifacts := []*codepipeline.InputArtifact{}
	for _, artifact := range s {
		if artifact == nil {
			continue
		}
		outputArtifacts = append(outputArtifacts, &codepipeline.InputArtifact{
			Name: aws.String(artifact.(string)),
		})
	}
	return outputArtifacts
}

func flattenAwsCodePipelineActionsInputArtifacts(artifacts []*codepipeline.InputArtifact) []string {
	values := []string{}
	for _, artifact := range artifacts {
		values = append(values, aws.StringValue(artifact.Name))
	}
	return values
}

func resourceAwsCodePipelineRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).codepipelineconn
	defaultTagsConfig := meta.(*AWSClient).DefaultTagsConfig
	ignoreTagsConfig := meta.(*AWSClient).IgnoreTagsConfig

	resp, err := conn.GetPipeline(&codepipeline.GetPipelineInput{
		Name: aws.String(d.Id()),
	})

	if isAWSErr(err, codepipeline.ErrCodePipelineNotFoundException, "") {
		log.Printf("[WARN] CodePipeline (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err != nil {
		return fmt.Errorf("error reading CodePipeline: %w", err)
	}

	metadata := resp.Metadata
	pipeline := resp.Pipeline

	if pipeline.ArtifactStore != nil {
		if err := d.Set("artifact_store", flattenAwsCodePipelineArtifactStore(pipeline.ArtifactStore)); err != nil {
			return err
		}
	} else if pipeline.ArtifactStores != nil {
		if err := d.Set("artifact_store", flattenAwsCodePipelineArtifactStores(pipeline.ArtifactStores)); err != nil {
			return err
		}
	}

	if err := d.Set("stage", flattenAwsCodePipelineStages(pipeline.Stages, d)); err != nil {
		return err
	}

	arn := aws.StringValue(metadata.PipelineArn)
	d.Set("arn", arn)
	d.Set("name", pipeline.Name)
	d.Set("role_arn", pipeline.RoleArn)

	tags, err := keyvaluetags.CodepipelineListTags(conn, arn)

	if err != nil {
		return fmt.Errorf("error listing tags for CodePipeline (%s): %w", arn, err)
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

func resourceAwsCodePipelineUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).codepipelineconn

	pipeline, err := expandAwsCodePipeline(d)
	if err != nil {
		return err
	}
	params := &codepipeline.UpdatePipelineInput{
		Pipeline: pipeline,
	}
	_, err = conn.UpdatePipeline(params)

	if err != nil {
		return fmt.Errorf("[ERROR] Error updating CodePipeline (%s): %w", d.Id(), err)
	}

	arn := d.Get("arn").(string)
	if d.HasChange("tags_all") {
		o, n := d.GetChange("tags_all")

		if err := keyvaluetags.CodepipelineUpdateTags(conn, arn, o, n); err != nil {
			return fmt.Errorf("error updating CodePipeline (%s) tags: %w", arn, err)
		}
	}

	return resourceAwsCodePipelineRead(d, meta)
}

func resourceAwsCodePipelineDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).codepipelineconn

	_, err := conn.DeletePipeline(&codepipeline.DeletePipelineInput{
		Name: aws.String(d.Id()),
	})

	if isAWSErr(err, codepipeline.ErrCodePipelineNotFoundException, "") {
		return nil
	}

	if err != nil {
		return fmt.Errorf("error deleting CodePipeline (%s): %w", d.Id(), err)
	}

	return err
}

func resourceAwsCodePipelineValidateActionProvider(i interface{}, path cty.Path) diag.Diagnostics {
	v, ok := i.(string)
	if !ok {
		return diag.Errorf("expected type to be string")
	}

	if v == CodePipelineProviderGitHub {
		return diag.Diagnostics{
			diag.Diagnostic{
				Severity: diag.Warning,
				Summary:  "The CodePipeline GitHub version 1 action provider is deprecated.",
				Detail:   "Use a GitHub version 2 action (with a CodeStar Connection `aws_codestarconnections_connection`) instead. See https://docs.aws.amazon.com/codepipeline/latest/userguide/update-github-action-connections.html",
			},
		}
	}

	return nil
}

func suppressCodePipelineStageActionConfiguration(k, old, new string, d *schema.ResourceData) bool {
	parts := strings.Split(k, ".")
	parts = parts[:len(parts)-2]
	providerAddr := strings.Join(append(parts, "provider"), ".")
	provider := d.Get(providerAddr).(string)

	if provider == CodePipelineProviderGitHub && strings.HasSuffix(k, CodePipelineGitHubActionConfigurationOAuthToken) {
		hash := hashCodePipelineGitHubToken(new)
		return old == hash
	}

	return false
}

const codePipelineGitHubTokenHashPrefix = "hash-"

func hashCodePipelineGitHubToken(token string) string {
	// Without this check, the value was getting encoded twice
	if strings.HasPrefix(token, codePipelineGitHubTokenHashPrefix) {
		return token
	}
	sum := sha256.Sum256([]byte(token))
	return codePipelineGitHubTokenHashPrefix + hex.EncodeToString(sum[:])
}
