package ncloud

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/NaverCloudPlatform/ncloud-sdk-go-v2/ncloud"
	"github.com/NaverCloudPlatform/ncloud-sdk-go-v2/services/sourcecommit"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func init() {
	RegisterResource("ncloud_sourcecommit_repository", resourceNcloudSourceCommitRepository())
}

func resourceNcloudSourceCommitRepository() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceNcloudSourceCommitRepositoryCreate,
		ReadContext:   resourceNcloudSourceCommitRepositoryRead,
		UpdateContext: resourceNcloudSourceCommitRepositoryUpdate,
		DeleteContext: resourceNcloudSourceCommitRepositoryDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(DefaultCreateTimeout),
			Update: schema.DefaultTimeout(DefaultCreateTimeout),
			Delete: schema.DefaultTimeout(DefaultTimeout),
		},
		Schema: map[string]*schema.Schema{
			"id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: ToDiagFunc(validation.StringLenBetween(1, 100)),
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"creator": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"git_https": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"git_ssh": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"filesafer": {
				Type:     schema.TypeBool,
				Optional: true,
			},
		},
	}
}

func resourceNcloudSourceCommitRepositoryCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	reqParams := &sourcecommit.CreateRepository{
		Name:        ncloud.String(d.Get("name").(string)),
		Description: StringPtrOrNil(d.GetOk("description")),
	}

	if fileSafer, ok := d.GetOk("filesafer"); ok {
		reqParams.Linked = &sourcecommit.CreateRepositoryLinked{
			FileSafer: BoolPtrOrNil(fileSafer, ok),
		}
	}

	logCommonRequest("resourceNcloudSourceCommitRepositoryCreate", reqParams)
	resp, err := config.Client.sourcecommit.V2Api.RepositoryCreate(ctx, reqParams)

	var diags diag.Diagnostics

	if err != nil {
		logErrorResponse("resourceNcloudSourceCommitRepositoryCreate", err, reqParams)

		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Fail to create repository",
			Detail:   err.Error(),
		})
		return diags
	}

	name := ncloud.StringValue(reqParams.Name)

	logResponse("resourceNcloudSourceCommitRepositoryCreate", resp)

	if err := waitForSourceCommitRepositoryActive(ctx, d, config, name); err != nil {

		name := d.Get("name").(string)

		diags := append(diag.FromErr(err), diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Unable to search repository",
			Detail:   fmt.Sprintf("Unable to search repository - detail , name : (%s)", name),
		})
		return diags
	}

	return resourceNcloudSourceCommitRepositoryRead(ctx, d, meta)
}

func resourceNcloudSourceCommitRepositoryRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)
	name := ncloud.String(d.Get("name").(string))
	id := ncloud.String(d.Id())

	var repository *sourcecommit.GetRepositoryDetailResponseResult
	var err error

	if *name == "" {
		repository, err = getRepositoryById(ctx, config, *id)
	} else {
		repository, err = getRepository(ctx, config, *name)
	}

	logCommonRequest("test-log-common", name)
	var diags diag.Diagnostics

	if err != nil {
		logErrorResponse("resourceNcloudSourceCommitRepositoryRead", err, *name)
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Unable to search repository",
			Detail:   fmt.Sprintf("Unable to search repository - detail repository : %s", *name),
		})
		return diags
	}

	if repository == nil {
		d.SetId("")
		return nil
	}

	d.SetId(strconv.Itoa(*repository.Id))
	d.Set("id", strconv.Itoa(*repository.Id))
	d.Set("name", repository.Name)
	d.Set("description", repository.Description)
	d.Set("creator", repository.Created.User)
	d.Set("git_https", repository.Git.Https)
	d.Set("git_ssh", repository.Git.Ssh)
	d.Set("filesafer", repository.Linked.FileSafer)

	return nil
}

func resourceNcloudSourceCommitRepositoryUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	reqParams := &sourcecommit.ChangeRepository{
		Description: StringPtrOrNil(d.GetOk("description")),
	}

	if fileSafer, ok := d.GetOk("filesafer"); ok {
		reqParams.Linked = &sourcecommit.CreateRepositoryLinked{
			FileSafer: BoolPtrOrNil(fileSafer, ok),
		}
	}

	name := ncloud.String(d.Get("name").(string))

	_, err := config.Client.sourcecommit.V2Api.RepositoryUpdate(ctx, reqParams, name)

	if err != nil {
		logErrorResponse("resourceNcloudSourceCommitRepositoryUpdate", err, *name)
		return diag.FromErr(err)
	}

	return resourceNcloudSourceCommitRepositoryRead(ctx, d, meta)
}

func resourceNcloudSourceCommitRepositoryDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	name := ncloud.String(d.Get("name").(string))

	if err := waitForSourceCommitRepositoryActive(ctx, d, config, *name); err != nil {
		return diag.FromErr(err)
	}

	logCommonRequest("resourceNcloudSourceCommitRepositoryDelete", *name)

	if _, err := config.Client.sourcecommit.V2Api.RepositoryDelete(ctx, name); err != nil {
		logErrorResponse("resourceNcloudSourceCommitRepositoryDelete", err, *name)
		return diag.FromErr(err)
	}

	if err := waitForSourceCommitRepositoryDeletion(ctx, d, config); err != nil {
		return diag.FromErr(err)
	}
	return nil
}

func waitForSourceCommitRepositoryDeletion(ctx context.Context, d *schema.ResourceData, config *ProviderConfig) error {
	stateConf := &resource.StateChangeConf{
		Pending: []string{"PENDING"},
		Target:  []string{"RESOLVE"},
		Refresh: func() (result interface{}, state string, err error) {
			repository, err := getRepositoryById(ctx, config, d.Id())
			if err != nil {
				return nil, "", err
			}
			if repository == nil {
				return d.Id(), "RESOLVE", nil
			}
			return repository, "PENDING", nil
		},
		Timeout:    d.Timeout(schema.TimeoutDelete),
		MinTimeout: 3 * time.Second,
		Delay:      2 * time.Second,
	}

	if _, err := stateConf.WaitForStateContext(ctx); err != nil {
		return fmt.Errorf("Error waiting for SourceCommit Repository id : (%s) to become terminating: %s", d.Id(), err)
	}
	return nil
}

func waitForSourceCommitRepositoryActive(ctx context.Context, d *schema.ResourceData, config *ProviderConfig, name string) error {

	stateConf := &resource.StateChangeConf{
		Pending: []string{"PENDING"},
		Target:  []string{"RESOLVE"},
		Refresh: func() (result interface{}, state string, err error) {
			repository, err := getRepository(ctx, config, name)
			if err != nil {
				return nil, "", fmt.Errorf("Repository response error , name : (%s) to become activating: %s", name, err)
			}
			if repository == nil {
				return name, "NULL", nil
			}

			if ncloud.StringValue(repository.Name) == name {
				return repository, "RESOLVE", nil
			}

			return nil, "PENDING", err
		},
		Timeout:    d.Timeout(schema.TimeoutCreate),
		MinTimeout: 3 * time.Second,
		Delay:      2 * time.Second,
	}
	if _, err := stateConf.WaitForStateContext(ctx); err != nil {
		return fmt.Errorf("error waiting for SourceCommit Repository id : (%s) to become activating: %s", name, err)
	}
	return nil
}

func getRepository(ctx context.Context, config *ProviderConfig, name string) (*sourcecommit.GetRepositoryDetailResponseResult, error) {
	logCommonRequest("getRepository", name)
	resp, err := config.Client.sourcecommit.V2Api.RepositoryDetailGet(ctx, &name)

	if err != nil {
		logErrorResponse("getRepository", err, name)
		return nil, err
	}

	return resp.Result, nil
}

func getRepositoryList(ctx context.Context, config *ProviderConfig) ([]*sourcecommit.GetRepositoryListResponseResultRepository, error) {
	logCommonRequest("getRepositoryList", "")
	resp, err := config.Client.sourcecommit.V2Api.RepositoryListGet(ctx)

	if err != nil {
		logErrorResponse("getRepositoryList", err, "")
		return nil, err
	}

	return resp.Result.Repository, nil
}

func getRepositoryById(ctx context.Context, config *ProviderConfig, id string) (*sourcecommit.GetRepositoryDetailResponseResult, error) {

	logCommonRequest("getRepositoryById", id)
	repositories, err := getRepositoryList(ctx, config)
	if err != nil {
		return nil, err
	}

	for _, repository := range repositories {
		if strconv.Itoa(*repository.Id) == id {
			return getRepository(ctx, config, *repository.Name)
		}
	}
	logErrorResponse("getRepositoryById No Such Repository Id", err, id)

	return nil, nil
}
