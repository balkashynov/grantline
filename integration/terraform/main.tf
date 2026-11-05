terraform {
  required_version = "= 1.13.5"
  required_providers {
    mongodbatlas = {
      source  = "mongodb/mongodbatlas"
      version = "= 2.21.0"
    }
  }
}
locals {
  org_id     = "111111111111111111111111"
  project_id = "222222222222222222222222"
}
resource "mongodbatlas_api_key" "legacy" {
  org_id      = local.org_id
  description = "Grantline synthetic integration fixture"
  role_names  = ["ORG_MEMBER"]
}
resource "mongodbatlas_service_account" "replacement" {
  org_id                 = local.org_id
  name                   = "Grantline synthetic integration fixture"
  description            = "Synthetic fixture; no Atlas resources created"
  roles                  = ["ORG_MEMBER"]
  without_initial_secret = true
}
resource "mongodbatlas_api_key_project_assignment" "legacy" {
  project_id = local.project_id
  api_key_id = mongodbatlas_api_key.legacy.api_key_id
  roles      = ["GROUP_READ_ONLY", "GROUP_DATA_ACCESS_READ_ONLY"]
}
resource "mongodbatlas_service_account_project_assignment" "replacement" {
  project_id = local.project_id
  client_id  = mongodbatlas_service_account.replacement.client_id
  roles      = ["GROUP_READ_ONLY"]
}
resource "mongodbatlas_access_list_api_key" "legacy" {
  org_id     = local.org_id
  api_key_id = mongodbatlas_api_key.legacy.api_key_id
  cidr_block = "192.0.2.0/24"
}
resource "mongodbatlas_service_account_access_list_entry" "replacement" {
  org_id     = local.org_id
  client_id  = mongodbatlas_service_account.replacement.client_id
  cidr_block = "192.0.2.0/25"
}
