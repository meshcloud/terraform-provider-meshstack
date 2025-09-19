resource "meshstack_landingzone" "example" {
  metadata = {
    name = "my-landing-zone-identifier"
    tags = {
      "confidentiality" = ["internal"],
      "environment"     = ["dev", "qa", "test"],
    }
  }
  spec = {
    display_name                  = "My Landing Zone's Display Name"
    description                   = "My Landing Zone Description"
    automate_deletion_approval    = false
    automate_deletion_replication = false
    info_link                     = "https://example.com/info-about-aws-landing-zone"
    platform_ref = {
      uuid = "4af5864a-da15-11ed-98c8-0242ac190003"
      kind = "meshPlatform"
    }
    platform_properties = {
      aws = {
        aws_target_org_unit_id = "ou-lpzq-kmf17bec"
        aws_enroll_account     = true
        aws_lambda_arn         = "arn:aws:lambda:us-east-1:123456789012:function:MyLambdaFunction"
        aws_role_mappings = [
          {
            project_role_ref = {
              name = "reader"
            }
            platform_role = "project-reader"
            policies = [
              "arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess",
            ]
          }
        ]
      }
    }
  }
}
