// Copyright IBM Corp. 2022 All Rights Reserved.
// Licensed under the Mozilla Public License v2.0

package secretsmanager

import (
	"context"
	"fmt"
	"github.com/IBM-Cloud/bluemix-go/bmxerror"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/IBM-Cloud/terraform-provider-ibm/ibm/conns"
	"github.com/IBM-Cloud/terraform-provider-ibm/ibm/flex"
	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/secrets-manager-go-sdk/secretsmanagerv2"
)

func ResourceIbmSmPublicCertificate() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceIbmSmPublicCertificateCreate,
		ReadContext:   resourceIbmSmPublicCertificateRead,
		UpdateContext: resourceIbmSmPublicCertificateUpdate,
		DeleteContext: resourceIbmSmPublicCertificateDelete,
		Importer:      &schema.ResourceImporter{},

		Schema: map[string]*schema.Schema{
			"secret_type": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The secret type. Supported types are arbitrary, certificates (imported, public, and private), IAM credentials, key-value, and user credentials.",
			},
			"name": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "A human-readable name to assign to your secret.To protect your privacy, do not use personal data, such as your name or location, as a name for your secret.",
			},
			"description": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "An extended description of your secret.To protect your privacy, do not use personal data, such as your name or location, as a description for your secret group.",
			},
			"secret_group_id": &schema.Schema{
				Type:        schema.TypeString,
				ForceNew:    true,
				Optional:    true,
				Computed:    true,
				Description: "A v4 UUID identifier, or `default` secret group.",
			},
			"labels": &schema.Schema{
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Labels that you can use to search for secrets in your instance.Up to 30 labels can be created.",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"common_name": &schema.Schema{
				Type:        schema.TypeString,
				ForceNew:    true,
				Required:    true,
				Description: "The Common Name (AKA CN) represents the server name that is protected by the SSL certificate.",
			},
			"alt_names": &schema.Schema{
				Type:        schema.TypeList,
				ForceNew:    true,
				Optional:    true,
				Computed:    true,
				Description: "With the Subject Alternative Name field, you can specify additional host names to be protected by a single SSL certificate.",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"key_algorithm": &schema.Schema{
				Type:        schema.TypeString,
				ForceNew:    true,
				Optional:    true,
				Default:     "RSA2048",
				Description: "The identifier for the cryptographic algorithm to be used to generate the public key that is associated with the certificate.The algorithm that you select determines the encryption algorithm (`RSA` or `ECDSA`) and key size to be used to generate keys and sign certificates. For longer living certificates, it is recommended to use longer keys to provide more encryption protection. Allowed values:  RSA2048, RSA4096, EC256, EC384.",
			},
			"ca": &schema.Schema{
				Type:        schema.TypeString,
				ForceNew:    true,
				Required:    true,
				Description: "A human-readable unique name to assign to your configuration.To protect your privacy, do not use personal data, such as your name or location, as an name for your secret.",
			},
			"dns": &schema.Schema{
				Type:        schema.TypeString,
				ForceNew:    true,
				Required:    true,
				Description: "A human-readable unique name to assign to your configuration.To protect your privacy, do not use personal data, such as your name or location, as an name for your secret.",
			},
			"bundle_certs": &schema.Schema{
				Type:        schema.TypeBool,
				ForceNew:    true,
				Optional:    true,
				Default:     true,
				Description: "Determines whether your issued certificate is bundled with intermediate certificates. Set to `false` for the certificate file to contain only the issued certificate.",
			},
			"rotation": &schema.Schema{
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Computed:    true,
				Description: "Determines whether Secrets Manager rotates your secrets automatically.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"auto_rotate": &schema.Schema{
							Type:        schema.TypeBool,
							Optional:    true,
							Computed:    true,
							Description: "Determines whether Secrets Manager rotates your secret automatically.Default is `false`. If `auto_rotate` is set to `true` the service rotates your secret based on the defined interval.",
						},
						"interval": &schema.Schema{
							Type:        schema.TypeInt,
							Optional:    true,
							Computed:    true,
							Description: "The length of the secret rotation time interval.",
						},
						"unit": &schema.Schema{
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
							Description: "The units for the secret rotation time interval.",
						},
						"rotate_keys": &schema.Schema{
							Type:        schema.TypeBool,
							Optional:    true,
							Computed:    true,
							Description: "Determines whether Secrets Manager rotates the private key for your public certificate automatically.Default is `false`. If it is set to `true`, the service generates and stores a new private key for your rotated certificate.",
						},
					},
				},
			},
			"custom_metadata": &schema.Schema{
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "The secret metadata that a user can customize.",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"version_custom_metadata": &schema.Schema{
				Type:        schema.TypeMap,
				ForceNew:    true,
				Optional:    true,
				Description: "The secret version metadata that a user can customize.",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"created_by": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The unique identifier that is associated with the entity that created the secret.",
			},
			"created_at": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The date when a resource was created. The date format follows RFC 3339.",
			},
			"crn": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "A CRN that uniquely identifies an IBM Cloud resource.",
			},
			"downloaded": &schema.Schema{
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Indicates whether the secret data that is associated with a secret version was retrieved in a call to the service API.",
			},
			"secret_id": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "A v4 UUID identifier.",
			},
			"locks_total": &schema.Schema{
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The number of locks of the secret.",
			},
			"state": &schema.Schema{
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The secret state that is based on NIST SP 800-57. States are integers and correspond to the `Pre-activation = 0`, `Active = 1`,  `Suspended = 2`, `Deactivated = 3`, and `Destroyed = 5` values.",
			},
			"state_description": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "A text representation of the secret state.",
			},
			"updated_at": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The date when a resource was recently modified. The date format follows RFC 3339.",
			},
			"versions_total": &schema.Schema{
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The number of versions of the secret.",
			},
			"signing_algorithm": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The identifier for the cryptographic algorithm that was used by the issuing certificate authority to sign a certificate.",
			},
			"expiration_date": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The date a secret is expired. The date format follows RFC 3339.",
			},
			"issuance_info": &schema.Schema{
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Issuance information that is associated with your certificate.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"auto_rotated": &schema.Schema{
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "Indicates whether the issued certificate is configured with an automatic rotation policy.",
						},
						"challenges": &schema.Schema{
							Type:        schema.TypeList,
							Computed:    true,
							Description: "The set of challenges. It is returned only when ordering public certificates by using manual DNS configuration.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"domain": &schema.Schema{
										Type:        schema.TypeString,
										Computed:    true,
										Description: "The challenge domain.",
									},
									"expiration": &schema.Schema{
										Type:        schema.TypeString,
										Computed:    true,
										Description: "The challenge expiration date. The date format follows RFC 3339.",
									},
									"status": &schema.Schema{
										Type:        schema.TypeString,
										Computed:    true,
										Description: "The challenge status.",
									},
									"txt_record_name": &schema.Schema{
										Type:        schema.TypeString,
										Computed:    true,
										Description: "The TXT record name.",
									},
									"txt_record_value": &schema.Schema{
										Type:        schema.TypeString,
										Computed:    true,
										Description: "The TXT record value.",
									},
								},
							},
						},
						"dns_challenge_validation_time": &schema.Schema{
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The date that a user requests to validate DNS challenges for certificates that are ordered with a manual DNS provider. The date format follows RFC 3339.",
						},
						"error_code": &schema.Schema{
							Type:        schema.TypeString,
							Computed:    true,
							Description: "A code that identifies an issuance error.This field, along with `error_message`, is returned when Secrets Manager successfully processes your request, but the certificate authority is unable to issue a certificate.",
						},
						"error_message": &schema.Schema{
							Type:        schema.TypeString,
							Computed:    true,
							Description: "A human-readable message that provides details about the issuance error.",
						},
						"ordered_on": &schema.Schema{
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The date when the certificate is ordered. The date format follows RFC 3339.",
						},
						"state": &schema.Schema{
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The secret state that is based on NIST SP 800-57. States are integers and correspond to the `Pre-activation = 0`, `Active = 1`,  `Suspended = 2`, `Deactivated = 3`, and `Destroyed = 5` values.",
						},
						"state_description": &schema.Schema{
							Type:        schema.TypeString,
							Computed:    true,
							Description: "A text representation of the secret state.",
						},
					},
				},
			},
			"issuer": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The distinguished name that identifies the entity that signed and issued the certificate.",
			},
			"serial_number": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The unique serial number that was assigned to a certificate by the issuing certificate authority.",
			},
			"validity": &schema.Schema{
				Type:        schema.TypeList,
				Computed:    true,
				Description: "The date and time that the certificate validity period begins and ends.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"not_before": &schema.Schema{
							Type:        schema.TypeString,
							Required:    true,
							Description: "The date-time format follows RFC 3339.",
						},
						"not_after": &schema.Schema{
							Type:        schema.TypeString,
							Required:    true,
							Description: "The date-time format follows RFC 3339.",
						},
					},
				},
			},
			"certificate": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Sensitive:   true,
				Description: "The PEM-encoded contents of your certificate.",
			},
			"intermediate": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Sensitive:   true,
				Description: "(Optional) The PEM-encoded intermediate certificate to associate with the root certificate.",
			},
			"private_key": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Sensitive:   true,
				Description: "(Optional) The PEM-encoded private key to associate with the certificate.",
			},
		},
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(35 * time.Minute),
		},
	}
}

func resourceIbmSmPublicCertificateCreate(context context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	secretsManagerClient, err := meta.(conns.ClientSession).SecretsManagerV2()
	if err != nil {
		return diag.FromErr(err)
	}

	region := getRegion(secretsManagerClient, d)
	instanceId := d.Get("instance_id").(string)
	secretsManagerClient = getClientWithInstanceEndpoint(secretsManagerClient, instanceId, region, getEndpointType(secretsManagerClient, d))

	createSecretOptions := &secretsmanagerv2.CreateSecretOptions{}

	secretPrototypeModel, err := resourceIbmSmPublicCertificateMapToSecretPrototype(d)
	if err != nil {
		return diag.FromErr(err)
	}
	createSecretOptions.SetSecretPrototype(secretPrototypeModel)

	secretIntf, response, err := secretsManagerClient.CreateSecretWithContext(context, createSecretOptions)
	if err != nil {
		log.Printf("[DEBUG] CreateSecretWithContext failed %s\n%s", err, response)
		return diag.FromErr(fmt.Errorf("CreateSecretWithContext failed %s\n%s", err, response))
	}

	secret := secretIntf.(*secretsmanagerv2.PublicCertificate)
	d.SetId(fmt.Sprintf("%s/%s/%s", region, instanceId, *secret.ID))
	d.Set("secret_id", *secret.ID)

	_, err = waitForIbmSmPublicCertificateCreate(secretsManagerClient, d)
	if err != nil {
		return diag.FromErr(fmt.Errorf(
			"Error waiting for resource IbmSmPublicCertificate (%s) to be created: %s", d.Id(), err))
	}

	return resourceIbmSmPublicCertificateRead(context, d, meta)
}

func waitForIbmSmPublicCertificateCreate(secretsManagerClient *secretsmanagerv2.SecretsManagerV2, d *schema.ResourceData) (interface{}, error) {
	getSecretOptions := &secretsmanagerv2.GetSecretOptions{}

	id := strings.Split(d.Id(), "/")
	secretId := id[2]

	getSecretOptions.SetID(secretId)

	stateConf := &resource.StateChangeConf{
		Pending: []string{"pre_activation"},
		Target:  []string{"active"},
		Refresh: func() (interface{}, string, error) {
			stateObjIntf, response, err := secretsManagerClient.GetSecret(getSecretOptions)
			stateObj := stateObjIntf.(*secretsmanagerv2.PublicCertificate)
			if err != nil {
				if apiErr, ok := err.(bmxerror.RequestFailure); ok && apiErr.StatusCode() == 404 {
					return nil, "", fmt.Errorf("The instance %s does not exist anymore: %s\n%s", "getSecretOptions", err, response)
				}
				return nil, "", err
			}
			failStates := map[string]bool{"destroyed": true}
			if failStates[*stateObj.StateDescription] {
				return stateObj, *stateObj.StateDescription, fmt.Errorf("The instance %s failed: %s\n%s", "getSecretOptions", err, response)
			}
			return stateObj, *stateObj.StateDescription, nil
		},
		Timeout:    d.Timeout(schema.TimeoutCreate),
		Delay:      0 * time.Second,
		MinTimeout: 5 * time.Second,
	}

	return stateConf.WaitForState()
}

func resourceIbmSmPublicCertificateRead(context context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	secretsManagerClient, err := meta.(conns.ClientSession).SecretsManagerV2()
	if err != nil {
		return diag.FromErr(err)
	}

	id := strings.Split(d.Id(), "/")
	region := id[0]
	instanceId := id[1]
	secretId := id[2]
	secretsManagerClient = getClientWithInstanceEndpoint(secretsManagerClient, instanceId, region, getEndpointType(secretsManagerClient, d))

	getSecretOptions := &secretsmanagerv2.GetSecretOptions{}

	getSecretOptions.SetID(secretId)

	secretIntf, response, err := secretsManagerClient.GetSecretWithContext(context, getSecretOptions)
	if err != nil {
		if response != nil && response.StatusCode == 404 {
			d.SetId("")
			return nil
		}
		log.Printf("[DEBUG] GetSecretWithContext failed %s\n%s", err, response)
		return diag.FromErr(fmt.Errorf("GetSecretWithContext failed %s\n%s", err, response))
	}

	secret := secretIntf.(*secretsmanagerv2.PublicCertificate)

	if err = d.Set("secret_id", secretId); err != nil {
		return diag.FromErr(fmt.Errorf("Error setting secret_id: %s", err))
	}
	if err = d.Set("instance_id", instanceId); err != nil {
		return diag.FromErr(fmt.Errorf("Error setting instance_id: %s", err))
	}
	if err = d.Set("region", region); err != nil {
		return diag.FromErr(fmt.Errorf("Error setting region: %s", err))
	}
	if err = d.Set("created_by", secret.CreatedBy); err != nil {
		return diag.FromErr(fmt.Errorf("Error setting created_by: %s", err))
	}
	if err = d.Set("created_at", flex.DateTimeToString(secret.CreatedAt)); err != nil {
		return diag.FromErr(fmt.Errorf("Error setting created_at: %s", err))
	}
	if err = d.Set("crn", secret.Crn); err != nil {
		return diag.FromErr(fmt.Errorf("Error setting crn: %s", err))
	}
	if err = d.Set("downloaded", secret.Downloaded); err != nil {
		return diag.FromErr(fmt.Errorf("Error setting downloaded: %s", err))
	}
	if err = d.Set("locks_total", flex.IntValue(secret.LocksTotal)); err != nil {
		return diag.FromErr(fmt.Errorf("Error setting locks_total: %s", err))
	}
	if err = d.Set("name", secret.Name); err != nil {
		return diag.FromErr(fmt.Errorf("Error setting name: %s", err))
	}
	if err = d.Set("secret_group_id", secret.SecretGroupID); err != nil {
		return diag.FromErr(fmt.Errorf("Error setting secret_group_id: %s", err))
	}
	if err = d.Set("secret_type", secret.SecretType); err != nil {
		return diag.FromErr(fmt.Errorf("Error setting secret_type: %s", err))
	}
	if err = d.Set("state", flex.IntValue(secret.State)); err != nil {
		return diag.FromErr(fmt.Errorf("Error setting state: %s", err))
	}
	if err = d.Set("state_description", secret.StateDescription); err != nil {
		return diag.FromErr(fmt.Errorf("Error setting state_description: %s", err))
	}
	if err = d.Set("updated_at", flex.DateTimeToString(secret.UpdatedAt)); err != nil {
		return diag.FromErr(fmt.Errorf("Error setting updated_at: %s", err))
	}
	if err = d.Set("versions_total", flex.IntValue(secret.VersionsTotal)); err != nil {
		return diag.FromErr(fmt.Errorf("Error setting versions_total: %s", err))
	}
	if err = d.Set("common_name", secret.CommonName); err != nil {
		return diag.FromErr(fmt.Errorf("Error setting common_name: %s", err))
	}
	if secret.IssuanceInfo != nil {
		issuanceInfoMap, err := resourceIbmSmPublicCertificateCertificateIssuanceInfoToMap(secret.IssuanceInfo)
		if err != nil {
			return diag.FromErr(err)
		}
		if err = d.Set("issuance_info", []map[string]interface{}{issuanceInfoMap}); err != nil {
			return diag.FromErr(fmt.Errorf("Error setting issuance_info: %s", err))
		}
	}
	if err = d.Set("key_algorithm", secret.KeyAlgorithm); err != nil {
		return diag.FromErr(fmt.Errorf("Error setting key_algorithm: %s", err))
	}
	if err = d.Set("ca", secret.Ca); err != nil {
		return diag.FromErr(fmt.Errorf("Error setting ca: %s", err))
	}
	if err = d.Set("dns", secret.Dns); err != nil {
		return diag.FromErr(fmt.Errorf("Error setting dns: %s", err))
	}
	if err = d.Set("bundle_certs", secret.BundleCerts); err != nil {
		return diag.FromErr(fmt.Errorf("Error setting bundle_certs: %s", err))
	}
	rotationMap, err := resourceIbmSmPublicCertificateRotationPolicyToMap(secret.Rotation)
	if err != nil {
		return diag.FromErr(err)
	}
	if err = d.Set("rotation", []map[string]interface{}{rotationMap}); err != nil {
		return diag.FromErr(fmt.Errorf("Error setting rotation: %s", err))
	}
	if secret.CustomMetadata != nil {
		d.Set("custom_metadata", secret.CustomMetadata)
	}
	if err = d.Set("description", secret.Description); err != nil {
		return diag.FromErr(fmt.Errorf("Error setting description: %s", err))
	}
	if secret.Labels != nil {
		if err = d.Set("labels", secret.Labels); err != nil {
			return diag.FromErr(fmt.Errorf("Error setting labels: %s", err))
		}
	}
	if err = d.Set("signing_algorithm", secret.SigningAlgorithm); err != nil {
		return diag.FromErr(fmt.Errorf("Error setting signing_algorithm: %s", err))
	}
	if secret.AltNames != nil {
		if err = d.Set("alt_names", secret.AltNames); err != nil {
			return diag.FromErr(fmt.Errorf("Error setting alt_names: %s", err))
		}
	}
	if err = d.Set("expiration_date", flex.DateTimeToString(secret.ExpirationDate)); err != nil {
		return diag.FromErr(fmt.Errorf("Error setting expiration_date: %s", err))
	}
	if err = d.Set("issuer", secret.Issuer); err != nil {
		return diag.FromErr(fmt.Errorf("Error setting issuer: %s", err))
	}
	if err = d.Set("serial_number", secret.SerialNumber); err != nil {
		return diag.FromErr(fmt.Errorf("Error setting serial_number: %s", err))
	}
	if secret.Validity != nil {
		validityMap, err := resourceIbmSmPublicCertificateCertificateValidityToMap(secret.Validity)
		if err != nil {
			return diag.FromErr(err)
		}
		if err = d.Set("validity", []map[string]interface{}{validityMap}); err != nil {
			return diag.FromErr(fmt.Errorf("Error setting validity: %s", err))
		}
	}
	if err = d.Set("certificate", secret.Certificate); err != nil {
		return diag.FromErr(fmt.Errorf("Error setting certificate: %s", err))
	}
	if err = d.Set("intermediate", secret.Intermediate); err != nil {
		return diag.FromErr(fmt.Errorf("Error setting intermediate: %s", err))
	}
	if err = d.Set("private_key", secret.PrivateKey); err != nil {
		return diag.FromErr(fmt.Errorf("Error setting private_key: %s", err))
	}
	return nil
}

func resourceIbmSmPublicCertificateUpdate(context context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	secretsManagerClient, err := meta.(conns.ClientSession).SecretsManagerV2()
	if err != nil {
		return diag.FromErr(err)
	}

	id := strings.Split(d.Id(), "/")
	region := id[0]
	instanceId := id[1]
	secretId := id[2]
	secretsManagerClient = getClientWithInstanceEndpoint(secretsManagerClient, instanceId, region, getEndpointType(secretsManagerClient, d))

	updateSecretMetadataOptions := &secretsmanagerv2.UpdateSecretMetadataOptions{}

	updateSecretMetadataOptions.SetID(secretId)

	hasChange := false

	patchVals := &secretsmanagerv2.SecretMetadataPatch{}

	if d.HasChange("name") {
		patchVals.Name = core.StringPtr(d.Get("name").(string))
		hasChange = true
	}
	if d.HasChange("description") {
		patchVals.Description = core.StringPtr(d.Get("description").(string))
		hasChange = true
	}
	if d.HasChange("labels") {
		labels := d.Get("labels").([]interface{})
		labelsParsed := make([]string, len(labels))
		for i, v := range labels {
			labelsParsed[i] = fmt.Sprint(v)
		}
		patchVals.Labels = labelsParsed
		hasChange = true
	}
	if d.HasChange("custom_metadata") {
		patchVals.CustomMetadata = d.Get("custom_metadata").(map[string]interface{})
		hasChange = true
	}
	if d.HasChange("rotation") {
		RotationModel, err := resourceIbmSmPublicCertificateMapToRotationPolicy(d.Get("rotation").([]interface{})[0].(map[string]interface{}))
		if err != nil {
			log.Printf("[DEBUG] UpdateSecretMetadataWithContext failed: Reading Rotation parameter failed: %s", err)
			return diag.FromErr(fmt.Errorf("UpdateSecretMetadataWithContext failed: Reading Rotation parameter failed: %s", err))
		}
		patchVals.Rotation = RotationModel
		hasChange = true
	}

	if hasChange {
		updateSecretMetadataOptions.SecretMetadataPatch, _ = patchVals.AsPatch()
		_, response, err := secretsManagerClient.UpdateSecretMetadataWithContext(context, updateSecretMetadataOptions)
		if err != nil {
			log.Printf("[DEBUG] UpdateSecretMetadataWithContext failed %s\n%s", err, response)
			return diag.FromErr(fmt.Errorf("UpdateSecretMetadataWithContext failed %s\n%s", err, response))
		}
	}

	return resourceIbmSmPublicCertificateRead(context, d, meta)
}

func resourceIbmSmPublicCertificateDelete(context context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	secretsManagerClient, err := meta.(conns.ClientSession).SecretsManagerV2()
	if err != nil {
		return diag.FromErr(err)
	}

	id := strings.Split(d.Id(), "/")
	region := id[0]
	instanceId := id[1]
	secretId := id[2]
	secretsManagerClient = getClientWithInstanceEndpoint(secretsManagerClient, instanceId, region, getEndpointType(secretsManagerClient, d))

	deleteSecretOptions := &secretsmanagerv2.DeleteSecretOptions{}

	deleteSecretOptions.SetID(secretId)

	response, err := secretsManagerClient.DeleteSecretWithContext(context, deleteSecretOptions)
	if err != nil {
		log.Printf("[DEBUG] DeleteSecretWithContext failed %s\n%s", err, response)
		return diag.FromErr(fmt.Errorf("DeleteSecretWithContext failed %s\n%s", err, response))
	}

	d.SetId("")

	return nil
}

func resourceIbmSmPublicCertificateMapToSecretPrototype(d *schema.ResourceData) (secretsmanagerv2.SecretPrototypeIntf, error) {
	model := &secretsmanagerv2.PublicCertificatePrototype{}
	model.SecretType = core.StringPtr("public_cert")

	if _, ok := d.GetOk("name"); ok {
		model.Name = core.StringPtr(d.Get("name").(string))
	}
	if _, ok := d.GetOk("description"); ok {
		model.Description = core.StringPtr(d.Get("description").(string))
	}
	if _, ok := d.GetOk("secret_group_id"); ok {
		model.SecretGroupID = core.StringPtr(d.Get("secret_group_id").(string))
	}
	if _, ok := d.GetOk("labels"); ok {
		labels := d.Get("labels").([]interface{})
		labelsParsed := make([]string, len(labels))
		for i, v := range labels {
			labelsParsed[i] = fmt.Sprint(v)
		}
		model.Labels = labelsParsed
	}
	if _, ok := d.GetOk("common_name"); ok {
		model.CommonName = core.StringPtr(d.Get("common_name").(string))
	}
	if _, ok := d.GetOk("alt_names"); ok {
		altNames := d.Get("alt_names").([]interface{})
		altNamesParsed := make([]string, len(altNames))
		for i, v := range altNames {
			altNamesParsed[i] = fmt.Sprint(v)
		}
		model.AltNames = altNamesParsed
	}
	if _, ok := d.GetOk("key_algorithm"); ok {
		model.KeyAlgorithm = core.StringPtr(d.Get("key_algorithm").(string))
	}
	if _, ok := d.GetOk("ca"); ok {
		model.Ca = core.StringPtr(d.Get("ca").(string))
	}
	if _, ok := d.GetOk("dns"); ok {
		model.Dns = core.StringPtr(d.Get("dns").(string))
	}
	if _, ok := d.GetOk("bundle_certs"); ok {
		model.BundleCerts = core.BoolPtr(d.Get("bundle_certs").(bool))
	}
	if _, ok := d.GetOk("rotation"); ok {
		RotationModel, err := resourceIbmSmPublicCertificateMapToRotationPolicy(d.Get("rotation").([]interface{})[0].(map[string]interface{}))
		if err != nil {
			return model, err
		}
		model.Rotation = RotationModel
	}
	if _, ok := d.GetOk("custom_metadata"); ok {
		model.CustomMetadata = d.Get("custom_metadata").(map[string]interface{})
	}
	if _, ok := d.GetOk("version_custom_metadata"); ok {
		model.VersionCustomMetadata = d.Get("version_custom_metadata").(map[string]interface{})
	}
	return model, nil
}

func resourceIbmSmPublicCertificateMapToRotationPolicy(modelMap map[string]interface{}) (secretsmanagerv2.RotationPolicyIntf, error) {
	model := &secretsmanagerv2.RotationPolicy{}
	if modelMap["auto_rotate"] != nil {
		model.AutoRotate = core.BoolPtr(modelMap["auto_rotate"].(bool))
	}
	if modelMap["interval"] != nil {
		model.Interval = core.Int64Ptr(int64(modelMap["interval"].(int)))
	}
	if modelMap["unit"] != nil && modelMap["unit"].(string) != "" {
		model.Unit = core.StringPtr(modelMap["unit"].(string))
	}
	if modelMap["rotate_keys"] != nil {
		model.RotateKeys = core.BoolPtr(modelMap["rotate_keys"].(bool))
	}
	return model, nil
}

func resourceIbmSmPublicCertificateMapToCommonRotationPolicy(modelMap map[string]interface{}) (*secretsmanagerv2.CommonRotationPolicy, error) {
	model := &secretsmanagerv2.CommonRotationPolicy{}
	if modelMap["auto_rotate"] != nil {
		model.AutoRotate = core.BoolPtr(modelMap["auto_rotate"].(bool))
	}
	if modelMap["interval"] != nil {
		model.Interval = core.Int64Ptr(int64(modelMap["interval"].(int)))
	}
	if modelMap["unit"] != nil && modelMap["unit"].(string) != "" {
		model.Unit = core.StringPtr(modelMap["unit"].(string))
	}
	return model, nil
}

func resourceIbmSmPublicCertificateMapToPublicCertificateRotationPolicy(modelMap map[string]interface{}) (*secretsmanagerv2.PublicCertificateRotationPolicy, error) {
	model := &secretsmanagerv2.PublicCertificateRotationPolicy{}
	if modelMap["auto_rotate"] != nil {
		model.AutoRotate = core.BoolPtr(modelMap["auto_rotate"].(bool))
	}
	if modelMap["interval"] != nil {
		model.Interval = core.Int64Ptr(int64(modelMap["interval"].(int)))
	}
	if modelMap["unit"] != nil && modelMap["unit"].(string) != "" {
		model.Unit = core.StringPtr(modelMap["unit"].(string))
	}
	if modelMap["rotate_keys"] != nil {
		model.RotateKeys = core.BoolPtr(modelMap["rotate_keys"].(bool))
	}
	return model, nil
}

func resourceIbmSmPublicCertificateRotationPolicyToMap(modelIntf secretsmanagerv2.RotationPolicyIntf) (map[string]interface{}, error) {
	model := modelIntf.(*secretsmanagerv2.RotationPolicy)
	modelMap := make(map[string]interface{})

	if model.AutoRotate != nil {
		modelMap["auto_rotate"] = model.AutoRotate
	}
	if model.Interval != nil {
		modelMap["interval"] = flex.IntValue(model.Interval)
	}
	if model.Unit != nil {
		modelMap["unit"] = model.Unit
	}
	if model.RotateKeys != nil {
		modelMap["rotate_keys"] = model.RotateKeys
	}
	return modelMap, nil
}

func resourceIbmSmPublicCertificateCertificateIssuanceInfoToMap(model *secretsmanagerv2.CertificateIssuanceInfo) (map[string]interface{}, error) {
	modelMap := make(map[string]interface{})
	if model.AutoRotated != nil {
		modelMap["auto_rotated"] = model.AutoRotated
	}
	if model.Challenges != nil {
		challenges := []map[string]interface{}{}
		for _, challengesItem := range model.Challenges {
			challengesItemMap, err := resourceIbmSmPublicCertificateChallengeResourceToMap(&challengesItem)
			if err != nil {
				return modelMap, err
			}
			challenges = append(challenges, challengesItemMap)
		}
		modelMap["challenges"] = challenges
	}
	if model.DnsChallengeValidationTime != nil {
		modelMap["dns_challenge_validation_time"] = model.DnsChallengeValidationTime.String()
	}
	if model.ErrorCode != nil {
		modelMap["error_code"] = model.ErrorCode
	}
	if model.ErrorMessage != nil {
		modelMap["error_message"] = model.ErrorMessage
	}
	if model.OrderedOn != nil {
		modelMap["ordered_on"] = model.OrderedOn.String()
	}
	if model.State != nil {
		modelMap["state"] = flex.IntValue(model.State)
	}
	if model.StateDescription != nil {
		modelMap["state_description"] = model.StateDescription
	}
	return modelMap, nil
}

func resourceIbmSmPublicCertificateChallengeResourceToMap(model *secretsmanagerv2.ChallengeResource) (map[string]interface{}, error) {
	modelMap := make(map[string]interface{})
	if model.Domain != nil {
		modelMap["domain"] = model.Domain
	}
	if model.Expiration != nil {
		modelMap["expiration"] = model.Expiration.String()
	}
	if model.Status != nil {
		modelMap["status"] = model.Status
	}
	if model.TxtRecordName != nil {
		modelMap["txt_record_name"] = model.TxtRecordName
	}
	if model.TxtRecordValue != nil {
		modelMap["txt_record_value"] = model.TxtRecordValue
	}
	return modelMap, nil
}

func resourceIbmSmPublicCertificateCertificateValidityToMap(model *secretsmanagerv2.CertificateValidity) (map[string]interface{}, error) {
	modelMap := make(map[string]interface{})
	modelMap["not_before"] = model.NotBefore.String()
	modelMap["not_after"] = model.NotAfter.String()
	return modelMap, nil
}
