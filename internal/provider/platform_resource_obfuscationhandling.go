package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/meshcloud/terraform-provider-meshstack/client"
)

const (
	obfuscatedValue = "mesh/hidden-secret"
)

// This function is necessary to handle obfuscated secrets for meshPlatforms.
// The meshPlatform API won't return secrets in plain text, but obfuscated values.
// As a result we keep those from the plan/state and re-apply them to the object read from the API.
//
// MUST NOT PASS ANY NIL VALUES
// MUST PASS compatible types
func handleObfuscatedSecrets(obfuscated *client.PlatformConfig, plain *client.PlatformConfig, d diag.Diagnostics) {
	if obfuscated == nil || plain == nil || obfuscated.Type != plain.Type {
		d.AddError(
			"Internal Error",
			"Could not handle obfuscated secrets due to invalid input parameters.",
		)
		return
	}

	switch obfuscated.Type {

	case "aks":
		if obfuscated.Aks != nil && obfuscated.Aks.Replication != nil && plain.Aks != nil && plain.Aks.Replication != nil {
			// access token
			if obfuscated.Aks.Replication.AccessToken != nil &&
				plain.Aks.Replication.AccessToken != nil &&
				*obfuscated.Aks.Replication.AccessToken == obfuscatedValue {
				obfuscated.Aks.Replication.AccessToken = plain.Aks.Replication.AccessToken
			}
			// SP client secret
			if obfuscated.Aks.Replication.ServicePrincipal != nil &&
				plain.Aks.Replication.ServicePrincipal != nil &&
				obfuscated.Aks.Replication.ServicePrincipal.CredentialsAuthClientSecret != nil &&
				plain.Aks.Replication.ServicePrincipal.CredentialsAuthClientSecret != nil &&
				*obfuscated.Aks.Replication.ServicePrincipal.CredentialsAuthClientSecret == obfuscatedValue {
				obfuscated.Aks.Replication.ServicePrincipal.CredentialsAuthClientSecret = plain.Aks.Replication.ServicePrincipal.CredentialsAuthClientSecret
			}
		}

	case "aws":
		if obfuscated.Aws != nil && obfuscated.Aws.Replication != nil && plain.Aws != nil && plain.Aws.Replication != nil {
			// replication access-config service-user secret key
			if obfuscated.Aws.Replication.AccessConfig != nil &&
				plain.Aws.Replication.AccessConfig != nil &&
				obfuscated.Aws.Replication.AccessConfig.ServiceUserConfig != nil &&
				plain.Aws.Replication.AccessConfig.ServiceUserConfig != nil &&
				obfuscated.Aws.Replication.AccessConfig.ServiceUserConfig.SecretKey != nil &&
				plain.Aws.Replication.AccessConfig.ServiceUserConfig.SecretKey != nil &&
				*obfuscated.Aws.Replication.AccessConfig.ServiceUserConfig.SecretKey == obfuscatedValue {
				obfuscated.Aws.Replication.AccessConfig.ServiceUserConfig.SecretKey = plain.Aws.Replication.AccessConfig.ServiceUserConfig.SecretKey
			}
			// replication AWS SSO token
			if obfuscated.Aws.Replication.AwsSso != nil &&
				plain.Aws.Replication.AwsSso != nil &&
				obfuscated.Aws.Replication.AwsSso.SsoAccessToken != nil &&
				plain.Aws.Replication.AwsSso.SsoAccessToken != nil &&
				*obfuscated.Aws.Replication.AwsSso.SsoAccessToken == obfuscatedValue {
				obfuscated.Aws.Replication.AwsSso.SsoAccessToken = plain.Aws.Replication.AwsSso.SsoAccessToken
			}
		}

	case "azure":
		if obfuscated.Azure != nil && obfuscated.Azure.Replication != nil && plain.Azure != nil && plain.Azure.Replication != nil {
			// replication SP client secret
			if obfuscated.Azure.Replication.ServicePrincipal != nil &&
				plain.Azure.Replication.ServicePrincipal != nil &&
				obfuscated.Azure.Replication.ServicePrincipal.CredentialsAuthClientSecret != nil &&
				plain.Azure.Replication.ServicePrincipal.CredentialsAuthClientSecret != nil &&
				*obfuscated.Azure.Replication.ServicePrincipal.CredentialsAuthClientSecret == obfuscatedValue {
				obfuscated.Azure.Replication.ServicePrincipal.CredentialsAuthClientSecret = plain.Azure.Replication.ServicePrincipal.CredentialsAuthClientSecret
			}
			// replication provisioning customer agreement SP client secret
			if obfuscated.Azure.Replication.Provisioning.CustomerAgreement != nil &&
				plain.Azure.Replication.Provisioning.CustomerAgreement != nil {
				if obfuscated.Azure.Replication.Provisioning.CustomerAgreement.SourceServicePrincipal != nil &&
					plain.Azure.Replication.Provisioning.CustomerAgreement.SourceServicePrincipal != nil &&
					obfuscated.Azure.Replication.Provisioning.CustomerAgreement.SourceServicePrincipal.CredentialsAuthClientSecret != nil &&
					plain.Azure.Replication.Provisioning.CustomerAgreement.SourceServicePrincipal.CredentialsAuthClientSecret != nil &&
					*obfuscated.Azure.Replication.Provisioning.CustomerAgreement.SourceServicePrincipal.CredentialsAuthClientSecret == obfuscatedValue {
					obfuscated.Azure.Replication.Provisioning.CustomerAgreement.SourceServicePrincipal.CredentialsAuthClientSecret = plain.Azure.Replication.Provisioning.CustomerAgreement.SourceServicePrincipal.CredentialsAuthClientSecret
				}
			}
		}

	case "azurerg":
		if obfuscated.AzureRg != nil && obfuscated.AzureRg.Replication != nil && plain.AzureRg != nil && plain.AzureRg.Replication != nil {
			// replication SP client secret
			if obfuscated.AzureRg.Replication.ServicePrincipal != nil &&
				plain.AzureRg.Replication.ServicePrincipal != nil &&
				obfuscated.AzureRg.Replication.ServicePrincipal.CredentialsAuthClientSecret != nil &&
				plain.AzureRg.Replication.ServicePrincipal.CredentialsAuthClientSecret != nil &&
				*obfuscated.AzureRg.Replication.ServicePrincipal.CredentialsAuthClientSecret == obfuscatedValue {
				obfuscated.AzureRg.Replication.ServicePrincipal.CredentialsAuthClientSecret = plain.AzureRg.Replication.ServicePrincipal.CredentialsAuthClientSecret
			}
		}

	case "kubernetes":
		if obfuscated.Kubernetes != nil && plain.Kubernetes != nil {
			// replication access token
			if obfuscated.Kubernetes.Replication != nil && plain.Kubernetes.Replication != nil {
				if obfuscated.Kubernetes.Replication.ClientConfig != nil &&
					plain.Kubernetes.Replication.ClientConfig != nil &&
					obfuscated.Kubernetes.Replication.ClientConfig.AccessToken != nil &&
					plain.Kubernetes.Replication.ClientConfig.AccessToken != nil &&
					*obfuscated.Kubernetes.Replication.ClientConfig.AccessToken == obfuscatedValue {
					obfuscated.Kubernetes.Replication.ClientConfig.AccessToken = plain.Kubernetes.Replication.ClientConfig.AccessToken
				}
			}
			// metering access token
			if obfuscated.Kubernetes.Metering != nil && plain.Kubernetes.Metering != nil {
				if obfuscated.Kubernetes.Metering.ClientConfig != nil &&
					plain.Kubernetes.Metering.ClientConfig != nil &&
					obfuscated.Kubernetes.Metering.ClientConfig.AccessToken != nil &&
					plain.Kubernetes.Metering.ClientConfig.AccessToken != nil &&
					*obfuscated.Kubernetes.Metering.ClientConfig.AccessToken == obfuscatedValue {
					obfuscated.Kubernetes.Metering.ClientConfig.AccessToken = plain.Kubernetes.Metering.ClientConfig.AccessToken
				}
			}
		}

	case "gcp":
		if obfuscated.Gcp != nil && obfuscated.Gcp.Replication != nil && plain.Gcp != nil && plain.Gcp.Replication != nil {
			// service account credentials
			if obfuscated.Gcp.Replication.ServiceAccountConfig != nil &&
				plain.Gcp.Replication.ServiceAccountConfig != nil &&
				obfuscated.Gcp.Replication.ServiceAccountConfig.ServiceAccountCredentialsConfig != nil &&
				plain.Gcp.Replication.ServiceAccountConfig.ServiceAccountCredentialsConfig != nil &&
				obfuscated.Gcp.Replication.ServiceAccountConfig.ServiceAccountCredentialsConfig.ServiceAccountCredentialsB64 != nil &&
				plain.Gcp.Replication.ServiceAccountConfig.ServiceAccountCredentialsConfig.ServiceAccountCredentialsB64 != nil &&
				*obfuscated.Gcp.Replication.ServiceAccountConfig.ServiceAccountCredentialsConfig.ServiceAccountCredentialsB64 == obfuscatedValue {
				obfuscated.Gcp.Replication.ServiceAccountConfig.ServiceAccountCredentialsConfig = plain.Gcp.Replication.ServiceAccountConfig.ServiceAccountCredentialsConfig
			}
		}

	case "openshift":
		if obfuscated.OpenShift != nil && plain.OpenShift != nil {
			// replication access token
			if obfuscated.OpenShift.Replication != nil && plain.OpenShift.Replication != nil {
				if obfuscated.OpenShift.Replication.ClientConfig != nil &&
					plain.OpenShift.Replication.ClientConfig != nil &&
					obfuscated.OpenShift.Replication.ClientConfig.AccessToken != nil &&
					plain.OpenShift.Replication.ClientConfig.AccessToken != nil &&
					*obfuscated.OpenShift.Replication.ClientConfig.AccessToken == obfuscatedValue {
					obfuscated.OpenShift.Replication.ClientConfig.AccessToken = plain.OpenShift.Replication.ClientConfig.AccessToken
				}
			}
			// metering access token
			if obfuscated.OpenShift.Metering != nil && plain.OpenShift.Metering != nil {
				if obfuscated.OpenShift.Metering.ClientConfig != nil &&
					plain.OpenShift.Metering.ClientConfig != nil &&
					obfuscated.OpenShift.Metering.ClientConfig.AccessToken != nil &&
					plain.OpenShift.Metering.ClientConfig.AccessToken != nil &&
					*obfuscated.OpenShift.Metering.ClientConfig.AccessToken == obfuscatedValue {
					obfuscated.OpenShift.Metering.ClientConfig.AccessToken = plain.OpenShift.Metering.ClientConfig.AccessToken
				}
			}
		}

	}
}
