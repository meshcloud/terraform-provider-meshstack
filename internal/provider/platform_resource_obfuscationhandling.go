package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/meshcloud/terraform-provider-meshstack/client"
)

// This function is necessary to handle obfuscated secrets for meshPlatforms.
// The meshPlatform API won't return secrets in plain text, but obfuscated values.
// As a result we keep those from the plan/state and re-apply them to the object read from the API.
//
// MUST NOT PASS ANY NIL VALUES
// MUST PASS compatible types
func handleObfuscatedSecrets(target *client.PlatformConfig, input *client.PlatformConfig, d diag.Diagnostics) {
	if target == nil || input == nil || target.Type != input.Type {
		d.AddError(
			"Internal Error",
			"Could not handle obfuscated secrets due to invalid input parameters.",
		)
		return
	}

	switch target.Type {

	case "aks":
		if target.Aks != nil && input.Aks != nil {
			if target.Aks.Replication != nil && input.Aks.Replication != nil {
				// replication access token - only restore plaintext if it was obfuscated (nil) from API
				if target.Aks.Replication.AccessToken.Plaintext == nil && input.Aks.Replication.AccessToken.Plaintext != nil {
					target.Aks.Replication.AccessToken.Plaintext = input.Aks.Replication.AccessToken.Plaintext
				}
				// SP client secret - only restore plaintext if it was obfuscated (nil) from API
				if target.Aks.Replication.ServicePrincipal.Auth.Credential != nil &&
					input.Aks.Replication.ServicePrincipal.Auth.Credential != nil &&
					target.Aks.Replication.ServicePrincipal.Auth.Credential.Plaintext == nil &&
					input.Aks.Replication.ServicePrincipal.Auth.Credential.Plaintext != nil {
					target.Aks.Replication.ServicePrincipal.Auth.Credential.Plaintext = input.Aks.Replication.ServicePrincipal.Auth.Credential.Plaintext
				}
			}
			// metering access token - only restore plaintext if it was obfuscated (nil) from API
			if target.Aks.Metering != nil && input.Aks.Metering != nil &&
				target.Aks.Metering.ClientConfig.AccessToken.Plaintext == nil && input.Aks.Metering.ClientConfig.AccessToken.Plaintext != nil {
				target.Aks.Metering.ClientConfig.AccessToken.Plaintext = input.Aks.Metering.ClientConfig.AccessToken.Plaintext
			}
		}

	case "aws":
		if target.Aws != nil && input.Aws != nil {
			if target.Aws.Replication != nil && input.Aws.Replication != nil {
				// replication access-config service-user secret key - only restore plaintext if it was obfuscated (nil) from API
				if target.Aws.Replication.AccessConfig.Auth.Credential != nil &&
					input.Aws.Replication.AccessConfig.Auth.Credential != nil &&
					target.Aws.Replication.AccessConfig.Auth.Credential.SecretKey.Plaintext == nil &&
					input.Aws.Replication.AccessConfig.Auth.Credential.SecretKey.Plaintext != nil {
					target.Aws.Replication.AccessConfig.Auth.Credential.SecretKey.Plaintext = input.Aws.Replication.AccessConfig.Auth.Credential.SecretKey.Plaintext
				}
				// replication AWS SSO token - only restore plaintext if it was obfuscated (nil) from API
				if target.Aws.Replication.AwsSso != nil &&
					input.Aws.Replication.AwsSso != nil &&
					target.Aws.Replication.AwsSso.SsoAccessToken.Plaintext == nil &&
					input.Aws.Replication.AwsSso.SsoAccessToken.Plaintext != nil {
					target.Aws.Replication.AwsSso.SsoAccessToken.Plaintext = input.Aws.Replication.AwsSso.SsoAccessToken.Plaintext
				}
			}
			// metering access-config service-user secret key - only restore plaintext if it was obfuscated (nil) from API
			if target.Aws.Metering != nil && input.Aws.Metering != nil &&
				target.Aws.Metering.AccessConfig.Auth.Credential != nil &&
				input.Aws.Metering.AccessConfig.Auth.Credential != nil &&
				target.Aws.Metering.AccessConfig.Auth.Credential.SecretKey.Plaintext == nil &&
				input.Aws.Metering.AccessConfig.Auth.Credential.SecretKey.Plaintext != nil {
				target.Aws.Metering.AccessConfig.Auth.Credential.SecretKey.Plaintext = input.Aws.Metering.AccessConfig.Auth.Credential.SecretKey.Plaintext
			}
		}

	case "azure":
		if target.Azure != nil && input.Azure != nil {
			if target.Azure.Replication != nil && input.Azure.Replication != nil {
				// replication SP client secret - only restore plaintext if it was obfuscated (nil) from API
				if target.Azure.Replication.ServicePrincipal.Auth.Credential != nil &&
					input.Azure.Replication.ServicePrincipal.Auth.Credential != nil &&
					target.Azure.Replication.ServicePrincipal.Auth.Credential.Plaintext == nil &&
					input.Azure.Replication.ServicePrincipal.Auth.Credential.Plaintext != nil {
					target.Azure.Replication.ServicePrincipal.Auth.Credential.Plaintext = input.Azure.Replication.ServicePrincipal.Auth.Credential.Plaintext
				}
				// replication provisioning customer agreement SP client secret - only restore plaintext if it was obfuscated (nil) from API
				if target.Azure.Replication.Provisioning != nil &&
					input.Azure.Replication.Provisioning != nil &&
					target.Azure.Replication.Provisioning.CustomerAgreement != nil &&
					input.Azure.Replication.Provisioning.CustomerAgreement != nil &&
					target.Azure.Replication.Provisioning.CustomerAgreement.SourceServicePrincipal.Auth.Credential != nil &&
					input.Azure.Replication.Provisioning.CustomerAgreement.SourceServicePrincipal.Auth.Credential != nil &&
					target.Azure.Replication.Provisioning.CustomerAgreement.SourceServicePrincipal.Auth.Credential.Plaintext == nil &&
					input.Azure.Replication.Provisioning.CustomerAgreement.SourceServicePrincipal.Auth.Credential.Plaintext != nil {
					target.Azure.Replication.Provisioning.CustomerAgreement.SourceServicePrincipal.Auth.Credential.Plaintext = input.Azure.Replication.Provisioning.CustomerAgreement.SourceServicePrincipal.Auth.Credential.Plaintext
				}
			}
			// metering SP client secret - only restore plaintext if it was obfuscated (nil) from API
			if target.Azure.Metering != nil && input.Azure.Metering != nil {
				if target.Azure.Metering.ServicePrincipal.Auth.Credential != nil &&
					input.Azure.Metering.ServicePrincipal.Auth.Credential != nil &&
					target.Azure.Metering.ServicePrincipal.Auth.Credential.Plaintext == nil &&
					input.Azure.Metering.ServicePrincipal.Auth.Credential.Plaintext != nil {
					target.Azure.Metering.ServicePrincipal.Auth.Credential.Plaintext = input.Azure.Metering.ServicePrincipal.Auth.Credential.Plaintext
				}
			}
		}

	case "azurerg":
		if target.AzureRg != nil && input.AzureRg != nil {
			// replication SP client secret - only restore plaintext if it was obfuscated (nil) from API
			if target.AzureRg.Replication != nil && input.AzureRg.Replication != nil &&
				target.AzureRg.Replication.ServicePrincipal.Auth.Credential != nil &&
				input.AzureRg.Replication.ServicePrincipal.Auth.Credential != nil &&
				target.AzureRg.Replication.ServicePrincipal.Auth.Credential.Plaintext == nil &&
				input.AzureRg.Replication.ServicePrincipal.Auth.Credential.Plaintext != nil {
				target.AzureRg.Replication.ServicePrincipal.Auth.Credential.Plaintext = input.AzureRg.Replication.ServicePrincipal.Auth.Credential.Plaintext
			}
		}

	case "kubernetes":
		if target.Kubernetes != nil && input.Kubernetes != nil {
			// replication access token - only restore plaintext if it was obfuscated (nil) from API
			if target.Kubernetes.Replication != nil && input.Kubernetes.Replication != nil &&
				target.Kubernetes.Replication.ClientConfig.AccessToken.Plaintext == nil && input.Kubernetes.Replication.ClientConfig.AccessToken.Plaintext != nil {
				target.Kubernetes.Replication.ClientConfig.AccessToken.Plaintext = input.Kubernetes.Replication.ClientConfig.AccessToken.Plaintext
			}
			// metering access token - only restore plaintext if it was obfuscated (nil) from API
			if target.Kubernetes.Metering != nil && input.Kubernetes.Metering != nil &&
				target.Kubernetes.Metering.ClientConfig.AccessToken.Plaintext == nil && input.Kubernetes.Metering.ClientConfig.AccessToken.Plaintext != nil {
				target.Kubernetes.Metering.ClientConfig.AccessToken.Plaintext = input.Kubernetes.Metering.ClientConfig.AccessToken.Plaintext
			}
		}

	case "gcp":
		if target.Gcp != nil && input.Gcp != nil {
			// replication service account credentials - only restore plaintext if it was obfuscated (nil) from API
			if target.Gcp.Replication != nil && input.Gcp.Replication != nil &&
				target.Gcp.Replication.ServiceAccount.Credential != nil &&
				input.Gcp.Replication.ServiceAccount.Credential != nil &&
				target.Gcp.Replication.ServiceAccount.Credential.Plaintext == nil &&
				input.Gcp.Replication.ServiceAccount.Credential.Plaintext != nil {
				target.Gcp.Replication.ServiceAccount.Credential.Plaintext = input.Gcp.Replication.ServiceAccount.Credential.Plaintext
			}
			// metering service account credentials - only restore plaintext if it was obfuscated (nil) from API
			if target.Gcp.Metering != nil && input.Gcp.Metering != nil &&
				target.Gcp.Metering.ServiceAccount.Credential != nil &&
				input.Gcp.Metering.ServiceAccount.Credential != nil &&
				target.Gcp.Metering.ServiceAccount.Credential.Plaintext == nil &&
				input.Gcp.Metering.ServiceAccount.Credential.Plaintext != nil {
				target.Gcp.Metering.ServiceAccount.Credential.Plaintext = input.Gcp.Metering.ServiceAccount.Credential.Plaintext
			}
		}

	case "openshift":
		if target.OpenShift != nil && input.OpenShift != nil {
			// replication access token - only restore plaintext if it was obfuscated (nil) from API
			if target.OpenShift.Replication != nil && input.OpenShift.Replication != nil &&
				target.OpenShift.Replication.ClientConfig.AccessToken.Plaintext == nil &&
				input.OpenShift.Replication.ClientConfig.AccessToken.Plaintext != nil {
				target.OpenShift.Replication.ClientConfig.AccessToken.Plaintext = input.OpenShift.Replication.ClientConfig.AccessToken.Plaintext
			}
			// metering access token - only restore plaintext if it was obfuscated (nil) from API
			if target.OpenShift.Metering != nil && input.OpenShift.Metering != nil &&
				target.OpenShift.Metering.ClientConfig.AccessToken.Plaintext == nil &&
				input.OpenShift.Metering.ClientConfig.AccessToken.Plaintext != nil {
				target.OpenShift.Metering.ClientConfig.AccessToken.Plaintext = input.OpenShift.Metering.ClientConfig.AccessToken.Plaintext
			}
		}
	}
}
