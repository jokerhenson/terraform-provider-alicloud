package alicloud

import (
	"fmt"
	"log"
	"strings"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/errors"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/bssopenapi"
	"github.com/aliyun/terraform-provider-alicloud/alicloud/connectivity"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
)

func resourceAlicloudCrEEInstance() *schema.Resource {
	return &schema.Resource{
		Create: resourceAlicloudCrEEInstanceCreate,
		Read:   resourceAlicloudCrEEInstanceRead,
		Update: resourceAlicloudCrEEInstanceUpdate,
		Delete: resourceAlicloudCrEEInstanceDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"payment_type": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice([]string{"Subscription"}, false),
				Default:      "Subscription",
			},
			"period": {
				Type:         schema.TypeInt,
				Optional:     true,
				ValidateFunc: validation.IntInSlice([]int{1, 2, 3, 6, 12, 24, 36, 48, 60}),
				Default:      12,
			},
			"renew_period": {
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
				Default:  0,
			},
			"renewal_status": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice([]string{"AutoRenewal", "ManualRenewal"}, false),
				Default:      "ManualRenewal",
			},
			"instance_type": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice([]string{"Basic", "Standard", "Advanced"}, false),
			},
			"instance_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"custom_oss_bucket": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"status": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"created_time": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"end_time": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAlicloudCrEEInstanceCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*connectivity.AliyunClient)

	request := bssopenapi.CreateCreateInstanceRequest()
	request.Scheme = "https"
	request.RegionId = "cn-hangzhou"
	request.ProductCode = "acr"
	request.ProductType = "acr_ee_public_cn"
	request.SubscriptionType = d.Get("payment_type").(string)
	request.Period = requests.NewInteger(d.Get("period").(int))
	if v, ok := d.GetOk("renew_period"); ok {
		request.RenewPeriod = requests.NewInteger(v.(int))
	}
	if v, ok := d.GetOk("renewal_status"); ok {
		request.RenewalStatus = v.(string)
	}

	parameters := []bssopenapi.CreateInstanceParameter{
		{
			Code:  "InstanceType",
			Value: d.Get("instance_type").(string),
		},
		{
			Code:  "InstanceName",
			Value: d.Get("instance_name").(string),
		},
		{
			Code:  "Region",
			Value: client.RegionId,
		},
	}
	if v, ok := d.GetOk("custom_oss_bucket"); ok {
		parameters = append(parameters, bssopenapi.CreateInstanceParameter{
			Code:  "DefaultOssBucket",
			Value: "false",
		})
		parameters = append(parameters, bssopenapi.CreateInstanceParameter{
			Code:  "InstanceStorageName",
			Value: v.(string),
		})
	} else {
		parameters = append(parameters, bssopenapi.CreateInstanceParameter{
			Code:  "DefaultOssBucket",
			Value: "true",
		})
	}
	request.Parameter = &parameters

	raw, err := client.WithBssopenapiClient(func(bssopenapiClient *bssopenapi.Client) (interface{}, error) {
		resp, errCreate := bssopenapiClient.CreateInstance(request)
		if errCreate != nil {
			// if account site is international, should route to  ap-southeast-1
			if serverErr, ok := errCreate.(*errors.ServerError); ok && serverErr.ErrorCode() == "NotApplicable" {
				request.RegionId = "ap-southeast-1"
				resp, errCreate = bssopenapiClient.CreateInstance(request)
			}
		}
		return resp, errCreate
	})
	if err != nil {
		return WrapErrorf(err, DefaultErrorMsg, "alicloud_cr_ee_instance", request.GetActionName(), AlibabaCloudSdkGoERROR)
	}
	addDebug(request.GetActionName(), raw)
	response, _ := raw.(*bssopenapi.CreateInstanceResponse)
	if !response.Success {
		return WrapErrorf(fmt.Errorf("%v", response), DefaultErrorMsg, "alicloud_cr_ee_instance", request.GetActionName(), AlibabaCloudSdkGoERROR)
	}
	d.SetId(fmt.Sprintf("%v", response.Data.InstanceId))

	return resourceAlicloudCrEEInstanceRead(d, meta)
}

func resourceAlicloudCrEEInstanceRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*connectivity.AliyunClient)
	crService := &CrService{client}

	resp, err := crService.DescribeCrEEInstance(d.Id())
	if err != nil {
		if NotFoundError(err) {
			log.Printf("[DEBUG] Resource alicloud_cr_ee_instance crService.DescribeCrEEInstance Failed!!! %s", err)
			d.SetId("")
			return nil
		}
		return WrapError(err)
	}
	d.Set("instance_name", resp.InstanceName)
	d.Set("instance_type", strings.TrimPrefix(resp.InstanceSpecification, "Enterprise_"))
	d.Set("status", resp.InstanceStatus)

	request := bssopenapi.CreateQueryAvailableInstancesRequest()
	request.Scheme = "https"
	request.RegionId = "cn-hangzhou"
	request.ProductCode = "acr"
	request.ProductType = "acr_ee_public_cn"
	request.InstanceIDs = resp.InstanceId
	raw, err := client.WithBssopenapiClient(func(bssopenapiClient *bssopenapi.Client) (interface{}, error) {
		resp, errQuery := bssopenapiClient.QueryAvailableInstances(request)
		if errQuery != nil {
			// if account site is international, should route to  ap-southeast-1
			if serverErr, ok := errQuery.(*errors.ServerError); ok && serverErr.ErrorCode() == "NotApplicable" {
				request.RegionId = "ap-southeast-1"
				resp, errQuery = bssopenapiClient.QueryAvailableInstances(request)
			}
		}
		return resp, errQuery
	})
	if err != nil {
		return WrapErrorf(err, DefaultErrorMsg, "alicloud_cr_ee_instance", request.GetActionName(), AlibabaCloudSdkGoERROR)
	}
	addDebug(request.GetActionName(), raw)
	response, _ := raw.(*bssopenapi.QueryAvailableInstancesResponse)
	if !response.Success {
		return WrapErrorf(fmt.Errorf("%v", response), DefaultErrorMsg, "alicloud_cr_ee_instance", request.GetActionName(), AlibabaCloudSdkGoERROR)
	}
	instance := response.Data.InstanceList[0]
	d.Set("payment_type", instance.SubscriptionType)
	d.Set("renewal_status", instance.RenewStatus)
	if instance.RenewalDurationUnit == "M" {
		d.Set("renew_period", instance.RenewalDuration)
	} else {
		d.Set("renew_period", instance.RenewalDuration*12)
	}
	d.Set("created_time", instance.CreateTime)
	d.Set("end_time", instance.EndTime)

	return nil
}

func resourceAlicloudCrEEInstanceUpdate(d *schema.ResourceData, meta interface{}) error {
	return resourceAlicloudCrEEInstanceRead(d, meta)
}

func resourceAlicloudCrEEInstanceDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[WARN] Cannot destroy resourceAlicloudCrEEInstance. Terraform will remove this resource from the state file, however resources may remain.")
	return nil
}
