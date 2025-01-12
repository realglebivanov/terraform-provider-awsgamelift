package ag

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/neptune"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/realglebivanov/terraform-provider-awsgamelift/ag/internal/keyvaluetags"
	iamwaiter "github.com/realglebivanov/terraform-provider-awsgamelift/ag/internal/service/iam/waiter"
)

const (

	// A constant for the supported CloudwatchLogsExports types
	// is not currently available in the AWS sdk-for-go
	// https://docs.aws.amazon.com/sdk-for-go/api/service/neptune/#pkg-constants
	CloudwatchLogsExportsAudit = "audit"

	neptuneDefaultPort = 8182
)

func resourceAwsNeptuneCluster() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsNeptuneClusterCreate,
		Read:   resourceAwsNeptuneClusterRead,
		Update: resourceAwsNeptuneClusterUpdate,
		Delete: resourceAwsNeptuneClusterDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(120 * time.Minute),
			Update: schema.DefaultTimeout(120 * time.Minute),
			Delete: schema.DefaultTimeout(120 * time.Minute),
		},

		Schema: map[string]*schema.Schema{

			// apply_immediately is used to determine when the update modifications
			// take place.
			"apply_immediately": {
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},

			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"availability_zones": {
				Type:     schema.TypeSet,
				MaxItems: 3,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
				ForceNew: true,
				Computed: true,
				Set:      schema.HashString,
			},

			"backup_retention_period": {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      1,
				ValidateFunc: validation.IntAtMost(35),
			},

			"cluster_identifier": {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ConflictsWith: []string{"cluster_identifier_prefix"},
				ValidateFunc:  validateNeptuneIdentifier,
			},

			"cluster_identifier_prefix": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ForceNew:     true,
				ValidateFunc: validateNeptuneIdentifierPrefix,
			},

			"cluster_members": {
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Computed: true,
				Set:      schema.HashString,
			},

			"cluster_resource_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"endpoint": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"enable_cloudwatch_logs_exports": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
					ValidateFunc: validation.StringInSlice([]string{
						CloudwatchLogsExportsAudit,
					}, false),
				},
				Set: schema.HashString,
			},

			"engine": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "neptune",
				ForceNew:     true,
				ValidateFunc: validateNeptuneEngine(),
			},

			"engine_version": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"final_snapshot_identifier": {
				Type:     schema.TypeString,
				Optional: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, es []error) {
					value := v.(string)
					if !regexp.MustCompile(`^[0-9A-Za-z-]+$`).MatchString(value) {
						es = append(es, fmt.Errorf(
							"only alphanumeric characters and hyphens allowed in %q", k))
					}
					if regexp.MustCompile(`--`).MatchString(value) {
						es = append(es, fmt.Errorf("%q cannot contain two consecutive hyphens", k))
					}
					if regexp.MustCompile(`-$`).MatchString(value) {
						es = append(es, fmt.Errorf("%q cannot end in a hyphen", k))
					}
					return
				},
			},

			"hosted_zone_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"iam_roles": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validateArn,
				},
				Set: schema.HashString,
			},

			"iam_database_authentication_enabled": {
				Type:     schema.TypeBool,
				Optional: true,
			},

			"kms_key_arn": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ForceNew:     true,
				ValidateFunc: validateArn,
			},

			"neptune_subnet_group_name": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"neptune_cluster_parameter_group_name": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "default.neptune1",
			},

			"port": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  neptuneDefaultPort,
				ForceNew: true,
			},

			"preferred_backup_window": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validateOnceADayWindowFormat,
			},

			"preferred_maintenance_window": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				StateFunc: func(val interface{}) string {
					if val == nil {
						return ""
					}
					return strings.ToLower(val.(string))
				},
				ValidateFunc: validateOnceAWeekWindowFormat,
			},

			"reader_endpoint": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"replication_source_identifier": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"storage_encrypted": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
				ForceNew: true,
			},

			"skip_final_snapshot": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"snapshot_identifier": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"tags":     tagsSchema(),
			"tags_all": tagsSchemaComputed(),

			"vpc_security_group_ids": {
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
			"deletion_protection": {
				Type:     schema.TypeBool,
				Optional: true,
			},
		},

		CustomizeDiff: SetTagsDiff,
	}
}

func resourceAwsNeptuneClusterCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).neptuneconn
	defaultTagsConfig := meta.(*AWSClient).DefaultTagsConfig
	tags := defaultTagsConfig.MergeTags(keyvaluetags.New(d.Get("tags").(map[string]interface{})))

	// Check if any of the parameters that require a cluster modification after creation are set
	clusterUpdate := false
	restoreDBClusterFromSnapshot := false
	if _, ok := d.GetOk("snapshot_identifier"); ok {
		restoreDBClusterFromSnapshot = true
	}

	if v, ok := d.GetOk("cluster_identifier"); ok {
		d.Set("cluster_identifier", v.(string))
	} else {
		if v, ok := d.GetOk("cluster_identifier_prefix"); ok {
			d.Set("cluster_identifier", resource.PrefixedUniqueId(v.(string)))
		} else {
			d.Set("cluster_identifier", resource.PrefixedUniqueId("tf-"))
		}
	}

	createDbClusterInput := &neptune.CreateDBClusterInput{
		DBClusterIdentifier: aws.String(d.Get("cluster_identifier").(string)),
		Engine:              aws.String(d.Get("engine").(string)),
		Port:                aws.Int64(int64(d.Get("port").(int))),
		StorageEncrypted:    aws.Bool(d.Get("storage_encrypted").(bool)),
		DeletionProtection:  aws.Bool(d.Get("deletion_protection").(bool)),
		Tags:                tags.IgnoreAws().NeptuneTags(),
	}
	restoreDBClusterFromSnapshotInput := &neptune.RestoreDBClusterFromSnapshotInput{
		DBClusterIdentifier: aws.String(d.Get("cluster_identifier").(string)),
		Engine:              aws.String(d.Get("engine").(string)),
		Port:                aws.Int64(int64(d.Get("port").(int))),
		SnapshotIdentifier:  aws.String(d.Get("snapshot_identifier").(string)),
		DeletionProtection:  aws.Bool(d.Get("deletion_protection").(bool)),
		Tags:                tags.IgnoreAws().NeptuneTags(),
	}

	if attr := d.Get("availability_zones").(*schema.Set); attr.Len() > 0 {
		createDbClusterInput.AvailabilityZones = expandStringSet(attr)
		restoreDBClusterFromSnapshotInput.AvailabilityZones = expandStringSet(attr)
	}

	if attr, ok := d.GetOk("backup_retention_period"); ok {
		createDbClusterInput.BackupRetentionPeriod = aws.Int64(int64(attr.(int)))
		if restoreDBClusterFromSnapshot {
			clusterUpdate = true
		}
	}

	if attr := d.Get("enable_cloudwatch_logs_exports").(*schema.Set); attr.Len() > 0 {
		createDbClusterInput.EnableCloudwatchLogsExports = expandStringSet(attr)
		restoreDBClusterFromSnapshotInput.EnableCloudwatchLogsExports = expandStringSet(attr)
	}

	if attr, ok := d.GetOk("engine_version"); ok {
		createDbClusterInput.EngineVersion = aws.String(attr.(string))
		restoreDBClusterFromSnapshotInput.EngineVersion = aws.String(attr.(string))
	}

	if attr, ok := d.GetOk("iam_database_authentication_enabled"); ok {
		createDbClusterInput.EnableIAMDatabaseAuthentication = aws.Bool(attr.(bool))
		restoreDBClusterFromSnapshotInput.EnableIAMDatabaseAuthentication = aws.Bool(attr.(bool))
	}

	if attr, ok := d.GetOk("kms_key_arn"); ok {
		createDbClusterInput.KmsKeyId = aws.String(attr.(string))
	}

	if attr, ok := d.GetOk("neptune_cluster_parameter_group_name"); ok {
		createDbClusterInput.DBClusterParameterGroupName = aws.String(attr.(string))
		if restoreDBClusterFromSnapshot {
			clusterUpdate = true
		}
	}

	if attr, ok := d.GetOk("neptune_subnet_group_name"); ok {
		createDbClusterInput.DBSubnetGroupName = aws.String(attr.(string))
		restoreDBClusterFromSnapshotInput.DBSubnetGroupName = aws.String(attr.(string))
	}

	if attr, ok := d.GetOk("preferred_backup_window"); ok {
		createDbClusterInput.PreferredBackupWindow = aws.String(attr.(string))
	}

	if attr, ok := d.GetOk("preferred_maintenance_window"); ok {
		createDbClusterInput.PreferredMaintenanceWindow = aws.String(attr.(string))
	}

	if attr, ok := d.GetOk("replication_source_identifier"); ok {
		createDbClusterInput.ReplicationSourceIdentifier = aws.String(attr.(string))
	}

	if attr := d.Get("vpc_security_group_ids").(*schema.Set); attr.Len() > 0 {
		createDbClusterInput.VpcSecurityGroupIds = expandStringSet(attr)
		if restoreDBClusterFromSnapshot {
			clusterUpdate = true
		}
		restoreDBClusterFromSnapshotInput.VpcSecurityGroupIds = expandStringSet(attr)
	}

	if restoreDBClusterFromSnapshot {
		log.Printf("[DEBUG] Neptune Cluster restore from snapshot configuration: %s", restoreDBClusterFromSnapshotInput)
	} else {
		log.Printf("[DEBUG] Neptune Cluster create options: %s", createDbClusterInput)
	}

	err := resource.Retry(iamwaiter.PropagationTimeout, func() *resource.RetryError {
		var err error
		if restoreDBClusterFromSnapshot {
			_, err = conn.RestoreDBClusterFromSnapshot(restoreDBClusterFromSnapshotInput)
		} else {
			_, err = conn.CreateDBCluster(createDbClusterInput)
		}
		if err != nil {
			if isAWSErr(err, "InvalidParameterValue", "IAM role ARN value is invalid or does not include the required permissions") {
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(err)
		}
		return nil
	})
	if isResourceTimeoutError(err) {
		if restoreDBClusterFromSnapshot {
			_, err = conn.RestoreDBClusterFromSnapshot(restoreDBClusterFromSnapshotInput)
		} else {
			_, err = conn.CreateDBCluster(createDbClusterInput)
		}
	}
	if err != nil {
		return fmt.Errorf("error creating Neptune Cluster: %s", err)
	}

	d.SetId(d.Get("cluster_identifier").(string))

	log.Printf("[INFO] Neptune Cluster ID: %s", d.Id())
	log.Println("[INFO] Waiting for Neptune Cluster to be available")

	stateConf := &resource.StateChangeConf{
		Pending:    resourceAwsNeptuneClusterCreatePendingStates,
		Target:     []string{"available"},
		Refresh:    resourceAwsNeptuneClusterStateRefreshFunc(d, meta),
		Timeout:    d.Timeout(schema.TimeoutCreate),
		MinTimeout: 10 * time.Second,
		Delay:      30 * time.Second,
	}

	// Wait, catching any errors
	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("error waiting for Neptune Cluster state to be \"available\": %s", err)
	}

	if v, ok := d.GetOk("iam_roles"); ok {
		for _, role := range v.(*schema.Set).List() {
			err := setIamRoleToNeptuneCluster(d.Id(), role.(string), conn)
			if err != nil {
				return err
			}
		}
	}

	if clusterUpdate {
		return resourceAwsNeptuneClusterUpdate(d, meta)
	}

	return resourceAwsNeptuneClusterRead(d, meta)

}

func resourceAwsNeptuneClusterRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).neptuneconn

	resp, err := conn.DescribeDBClusters(&neptune.DescribeDBClustersInput{
		DBClusterIdentifier: aws.String(d.Id()),
	})

	if err != nil {
		if isAWSErr(err, neptune.ErrCodeDBClusterNotFoundFault, "") {
			d.SetId("")
			log.Printf("[DEBUG] Neptune Cluster (%s) not found", d.Id())
			return nil
		}
		log.Printf("[DEBUG] Error describing Neptune Cluster (%s) when waiting: %s", d.Id(), err)
		return err
	}

	var dbc *neptune.DBCluster
	for _, v := range resp.DBClusters {
		if aws.StringValue(v.DBClusterIdentifier) == d.Id() {
			dbc = v
		}
	}

	if dbc == nil {
		log.Printf("[WARN] Neptune Cluster (%s) not found", d.Id())
		d.SetId("")
		return nil
	}

	return flattenAwsNeptuneClusterResource(d, meta, dbc)
}

func flattenAwsNeptuneClusterResource(d *schema.ResourceData, meta interface{}, dbc *neptune.DBCluster) error {
	conn := meta.(*AWSClient).neptuneconn
	defaultTagsConfig := meta.(*AWSClient).DefaultTagsConfig
	ignoreTagsConfig := meta.(*AWSClient).IgnoreTagsConfig

	if err := d.Set("availability_zones", aws.StringValueSlice(dbc.AvailabilityZones)); err != nil {
		return fmt.Errorf("Error saving AvailabilityZones to state for Neptune Cluster (%s): %s", d.Id(), err)
	}

	d.Set("backup_retention_period", dbc.BackupRetentionPeriod)
	d.Set("cluster_identifier", dbc.DBClusterIdentifier)
	d.Set("cluster_resource_id", dbc.DbClusterResourceId)

	if err := d.Set("enable_cloudwatch_logs_exports", aws.StringValueSlice(dbc.EnabledCloudwatchLogsExports)); err != nil {
		return fmt.Errorf("Error saving EnableCloudwatchLogsExports to state for Neptune Cluster (%s): %s", d.Id(), err)
	}

	d.Set("endpoint", dbc.Endpoint)
	d.Set("engine_version", dbc.EngineVersion)
	d.Set("engine", dbc.Engine)
	d.Set("hosted_zone_id", dbc.HostedZoneId)
	d.Set("iam_database_authentication_enabled", dbc.IAMDatabaseAuthenticationEnabled)
	d.Set("kms_key_arn", dbc.KmsKeyId)
	d.Set("neptune_cluster_parameter_group_name", dbc.DBClusterParameterGroup)
	d.Set("neptune_subnet_group_name", dbc.DBSubnetGroup)
	d.Set("port", dbc.Port)
	d.Set("preferred_backup_window", dbc.PreferredBackupWindow)
	d.Set("preferred_maintenance_window", dbc.PreferredMaintenanceWindow)
	d.Set("reader_endpoint", dbc.ReaderEndpoint)
	d.Set("replication_source_identifier", dbc.ReplicationSourceIdentifier)
	d.Set("storage_encrypted", dbc.StorageEncrypted)
	d.Set("deletion_protection", dbc.DeletionProtection)

	var sg []string
	for _, g := range dbc.VpcSecurityGroups {
		sg = append(sg, aws.StringValue(g.VpcSecurityGroupId))
	}
	if err := d.Set("vpc_security_group_ids", sg); err != nil {
		return fmt.Errorf("Error saving VPC Security Group IDs to state for Neptune Cluster (%s): %s", d.Id(), err)
	}

	var cm []string
	for _, m := range dbc.DBClusterMembers {
		cm = append(cm, aws.StringValue(m.DBInstanceIdentifier))
	}
	if err := d.Set("cluster_members", cm); err != nil {
		return fmt.Errorf("Error saving Neptune Cluster Members to state for Neptune Cluster (%s): %s", d.Id(), err)
	}

	var roles []string
	for _, r := range dbc.AssociatedRoles {
		roles = append(roles, aws.StringValue(r.RoleArn))
	}

	if err := d.Set("iam_roles", roles); err != nil {
		return fmt.Errorf("Error saving IAM Roles to state for Neptune Cluster (%s): %s", d.Id(), err)
	}

	arn := aws.StringValue(dbc.DBClusterArn)
	d.Set("arn", arn)

	tags, err := keyvaluetags.NeptuneListTags(conn, d.Get("arn").(string))

	if err != nil {
		return fmt.Errorf("error listing tags for Neptune Cluster (%s): %s", d.Get("arn").(string), err)
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

func resourceAwsNeptuneClusterUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).neptuneconn
	requestUpdate := false

	req := &neptune.ModifyDBClusterInput{
		ApplyImmediately:    aws.Bool(d.Get("apply_immediately").(bool)),
		DBClusterIdentifier: aws.String(d.Id()),
	}

	if d.HasChange("vpc_security_group_ids") {
		if attr := d.Get("vpc_security_group_ids").(*schema.Set); attr.Len() > 0 {
			req.VpcSecurityGroupIds = expandStringSet(attr)
		} else {
			req.VpcSecurityGroupIds = []*string{}
		}
		requestUpdate = true
	}

	if d.HasChange("enable_cloudwatch_logs_exports") {
		logs := &neptune.CloudwatchLogsExportConfiguration{}

		old, new := d.GetChange("enable_cloudwatch_logs_exports")

		disableLogTypes := old.(*schema.Set).Difference(new.(*schema.Set))

		if disableLogTypes.Len() > 0 {
			logs.SetDisableLogTypes(expandStringSet(disableLogTypes))
		}

		enableLogTypes := new.(*schema.Set).Difference(old.(*schema.Set))

		if enableLogTypes.Len() > 0 {
			logs.SetEnableLogTypes(expandStringSet(enableLogTypes))
		}

		req.CloudwatchLogsExportConfiguration = logs
		requestUpdate = true
	}

	if d.HasChange("preferred_backup_window") {
		req.PreferredBackupWindow = aws.String(d.Get("preferred_backup_window").(string))
		requestUpdate = true
	}

	if d.HasChange("preferred_maintenance_window") {
		req.PreferredMaintenanceWindow = aws.String(d.Get("preferred_maintenance_window").(string))
		requestUpdate = true
	}

	if d.HasChange("backup_retention_period") {
		req.BackupRetentionPeriod = aws.Int64(int64(d.Get("backup_retention_period").(int)))
		requestUpdate = true
	}

	if d.HasChange("neptune_cluster_parameter_group_name") {
		req.DBClusterParameterGroupName = aws.String(d.Get("neptune_cluster_parameter_group_name").(string))
		requestUpdate = true
	}

	if d.HasChange("iam_database_authentication_enabled") {
		req.EnableIAMDatabaseAuthentication = aws.Bool(d.Get("iam_database_authentication_enabled").(bool))
		requestUpdate = true
	}
	if d.HasChange("deletion_protection") {
		req.DeletionProtection = aws.Bool(d.Get("deletion_protection").(bool))
		requestUpdate = true
	}

	if requestUpdate {
		err := resource.Retry(5*time.Minute, func() *resource.RetryError {
			_, err := conn.ModifyDBCluster(req)
			if err != nil {
				if isAWSErr(err, "InvalidParameterValue", "IAM role ARN value is invalid or does not include the required permissions") {
					return resource.RetryableError(err)
				}
				if isAWSErr(err, neptune.ErrCodeInvalidDBClusterStateFault, "") {
					return resource.RetryableError(err)
				}
				return resource.NonRetryableError(err)
			}
			return nil
		})
		if isResourceTimeoutError(err) {
			_, err = conn.ModifyDBCluster(req)
		}
		if err != nil {
			return fmt.Errorf("Failed to modify Neptune Cluster (%s): %s", d.Id(), err)
		}

		stateConf := &resource.StateChangeConf{
			Pending:    resourceAwsNeptuneClusterUpdatePendingStates,
			Target:     []string{"available"},
			Refresh:    resourceAwsNeptuneClusterStateRefreshFunc(d, meta),
			Timeout:    d.Timeout(schema.TimeoutUpdate),
			MinTimeout: 10 * time.Second,
			Delay:      10 * time.Second,
		}

		log.Printf("[INFO] Waiting for Neptune Cluster (%s) to modify", d.Id())
		_, err = stateConf.WaitForState()
		if err != nil {
			return fmt.Errorf("error waiting for Neptune Cluster (%s) to modify: %s", d.Id(), err)
		}
	}

	if d.HasChange("iam_roles") {
		oraw, nraw := d.GetChange("iam_roles")
		if oraw == nil {
			oraw = new(schema.Set)
		}
		if nraw == nil {
			nraw = new(schema.Set)
		}

		os := oraw.(*schema.Set)
		ns := nraw.(*schema.Set)
		removeRoles := os.Difference(ns)
		enableRoles := ns.Difference(os)

		for _, role := range enableRoles.List() {
			err := setIamRoleToNeptuneCluster(d.Id(), role.(string), conn)
			if err != nil {
				return err
			}
		}

		for _, role := range removeRoles.List() {
			err := removeIamRoleFromNeptuneCluster(d.Id(), role.(string), conn)
			if err != nil {
				return err
			}
		}
	}

	if d.HasChange("tags_all") {
		o, n := d.GetChange("tags_all")

		if err := keyvaluetags.NeptuneUpdateTags(conn, d.Get("arn").(string), o, n); err != nil {
			return fmt.Errorf("error updating Neptune Cluster (%s) tags: %s", d.Get("arn").(string), err)
		}

	}

	return resourceAwsNeptuneClusterRead(d, meta)
}

func resourceAwsNeptuneClusterDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).neptuneconn
	log.Printf("[DEBUG] Destroying Neptune Cluster (%s)", d.Id())

	deleteOpts := neptune.DeleteDBClusterInput{
		DBClusterIdentifier: aws.String(d.Id()),
	}

	skipFinalSnapshot := d.Get("skip_final_snapshot").(bool)
	deleteOpts.SkipFinalSnapshot = aws.Bool(skipFinalSnapshot)

	if !skipFinalSnapshot {
		if name, present := d.GetOk("final_snapshot_identifier"); present {
			deleteOpts.FinalDBSnapshotIdentifier = aws.String(name.(string))
		} else {
			return fmt.Errorf("Neptune Cluster FinalSnapshotIdentifier is required when a final snapshot is required")
		}
	}

	log.Printf("[DEBUG] Neptune Cluster delete options: %s", deleteOpts)

	err := resource.Retry(1*time.Minute, func() *resource.RetryError {
		_, err := conn.DeleteDBCluster(&deleteOpts)
		if err != nil {
			if isAWSErr(err, neptune.ErrCodeInvalidDBClusterStateFault, "is not currently in the available state") {
				return resource.RetryableError(err)
			}
			if isAWSErr(err, neptune.ErrCodeDBClusterNotFoundFault, "") {
				return nil
			}
			return resource.NonRetryableError(err)
		}
		return nil
	})
	if isResourceTimeoutError(err) {
		_, err = conn.DeleteDBCluster(&deleteOpts)
	}
	if err != nil {
		return fmt.Errorf("Neptune Cluster cannot be deleted: %s", err)
	}

	stateConf := &resource.StateChangeConf{
		Pending:    resourceAwsNeptuneClusterDeletePendingStates,
		Target:     []string{"destroyed"},
		Refresh:    resourceAwsNeptuneClusterStateRefreshFunc(d, meta),
		Timeout:    d.Timeout(schema.TimeoutDelete),
		MinTimeout: 10 * time.Second,
		Delay:      30 * time.Second,
	}

	// Wait, catching any errors
	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error deleting Neptune Cluster (%s): %s", d.Id(), err)
	}

	return nil
}

func resourceAwsNeptuneClusterStateRefreshFunc(
	d *schema.ResourceData, meta interface{}) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		conn := meta.(*AWSClient).neptuneconn

		resp, err := conn.DescribeDBClusters(&neptune.DescribeDBClustersInput{
			DBClusterIdentifier: aws.String(d.Id()),
		})

		if err != nil {
			if isAWSErr(err, neptune.ErrCodeDBClusterNotFoundFault, "") {
				log.Printf("[DEBUG] Neptune Cluster (%s) not found", d.Id())
				return 42, "destroyed", nil
			}
			log.Printf("[DEBUG] Error on retrieving Neptune Cluster (%s) when waiting: %s", d.Id(), err)
			return nil, "", err
		}

		var dbc *neptune.DBCluster

		for _, v := range resp.DBClusters {
			if aws.StringValue(v.DBClusterIdentifier) == d.Id() {
				dbc = v
			}
		}

		if dbc == nil {
			return 42, "destroyed", nil
		}

		if dbc.Status != nil {
			log.Printf("[DEBUG] Neptune Cluster status (%s): %s", d.Id(), aws.StringValue(dbc.Status))
		}

		return dbc, aws.StringValue(dbc.Status), nil
	}
}

func setIamRoleToNeptuneCluster(clusterIdentifier string, roleArn string, conn *neptune.Neptune) error {
	params := &neptune.AddRoleToDBClusterInput{
		DBClusterIdentifier: aws.String(clusterIdentifier),
		RoleArn:             aws.String(roleArn),
	}
	_, err := conn.AddRoleToDBCluster(params)
	return err
}

func removeIamRoleFromNeptuneCluster(clusterIdentifier string, roleArn string, conn *neptune.Neptune) error {
	params := &neptune.RemoveRoleFromDBClusterInput{
		DBClusterIdentifier: aws.String(clusterIdentifier),
		RoleArn:             aws.String(roleArn),
	}
	_, err := conn.RemoveRoleFromDBCluster(params)
	return err
}

var resourceAwsNeptuneClusterCreatePendingStates = []string{
	"creating",
	"backing-up",
	"modifying",
	"preparing-data-migration",
	"migrating",
}

var resourceAwsNeptuneClusterUpdatePendingStates = []string{
	"backing-up",
	"modifying",
	"configuring-iam-database-auth",
}

var resourceAwsNeptuneClusterDeletePendingStates = []string{
	"available",
	"deleting",
	"backing-up",
	"modifying",
}
