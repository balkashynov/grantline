# Real provider schemas, mocked resource creation. No Atlas credentials/network.
mock_provider "mongodbatlas" {
  mock_resource "mongodbatlas_access_list_api_key" {
    defaults = { ip_address = "" }
  }
  mock_resource "mongodbatlas_service_account_access_list_entry" {
    defaults = { ip_address = "" }
  }
  mock_resource "mongodbatlas_api_key" {
    defaults = {
      api_key_id  = "333333333333333333333333"
      public_key  = "SYNTHETIC_PUBLIC_KEY"
      private_key = "SYNTHETIC_SECRET_SENTINEL"
    }
  }
  mock_resource "mongodbatlas_service_account" {
    defaults = {
      client_id = "mdb_sa_id_444444444444444444444444"
    }
  }
}
run "overlap" {
  command = apply
  assert {
    condition     = mongodbatlas_api_key.legacy.org_id == mongodbatlas_service_account.replacement.org_id
    error_message = "The two synthetic identities must share an organization."
  }
}
