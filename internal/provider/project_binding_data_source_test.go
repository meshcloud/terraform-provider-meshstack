package provider

import (
	"testing"

	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/testconfig"
)

func TestAccProjectUserBindingDataSource(t *testing.T) {
	testAccProjectBindingDataSource(t, "user", seededUsername)
}

func TestAccProjectGroupBindingDataSource(t *testing.T) {
	testAccProjectBindingDataSource(t, "group", seededGroupName)
}

func testAccProjectBindingDataSource(t *testing.T, subjectKind, subjectName string) {
	t.Helper()
	bindingConfig, bindingAddr, _, _ := testconfig.ProjectBinding(t, subjectKind, subjectName)
	testAccBindingDataSource(t, "project_"+subjectKind+"_binding", bindingConfig, bindingAddr, "Project Reader", "Project Admin")
}
