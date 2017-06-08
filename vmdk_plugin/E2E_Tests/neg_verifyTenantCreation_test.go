// See the License for the specific language governing permissions and
// limitations under the License.

// This test covers two negative scenarios:
// 1. To verify that we are not able to add a non-existent vm to a tenant.
// 2. To verify that we are not able to create a new tenant with the same name as an existing tenant.

// Test will ssh to the esx and  try creating a tenant by passing non-existent vm
// with the tenant creation command. This operation will not succeed.
// Test will then try creating two tenants with the same name.
// We cannot have two tenants with the same name..

// Test assumes that SSH cert has been setup to enable password-less login

package e2e_test

import (
	TestUtil "github.com/vmware/docker-volume-vsphere/vmdk_plugin/utils/test_util"
	"log"
	"os"
	"strings"
	"testing"
)

var (
	host             string = os.Getenv("ESX")
	tenantName       string = ""
	ADMIN_CLI_TENANT string = "/usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant "
	vmName           string = ""
)

/*
Test steps:
1. Create a new tenant named by passing a non-existent vm.
   Command: /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant create --name=<tenant_name> --vm-list=<vmName>
2. Do a list of tenants and verify that tenant creation failed.
   Command: /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant ls
*/
func TestAddNonEsxtntVmToTenant(t *testing.T) {
	execAddNonExstntVmToTenantTest(t)
	testCleanup()
}

func execAddNonExstntVmToTenantTest(t *testing.T) {
	tenantName = TestUtil.GetNameWithTimeStamp("tenant1")
	vmName = TestUtil.GetNameWithTimeStamp("vm1")
	log.Printf("Creating a tenant named - %s with a non-existent vm - %s ", tenantName, vmName)
	out, err := TestUtil.InvokeCommand(host, ADMIN_CLI_TENANT+"create --name="+tenantName+" --vm-list="+vmName)
	if err != nil {
		log.Fatal("Failed to invoke tenant creation command. Error: ", err)
	}
	if strings.Contains(string(out), "Cannot find vm "+vmName) {
		log.Printf("\n As expected, error while creating a tenant named - %s by passing a non-existent vm - %s ", tenantName, vmName)
	}
	// check if tenant exist
	if TestUtil.CheckTenantExist(host, tenantName) {
		log.Fatalf("Tenant name [%s] does exist.", tenantName)
	}
}

/*
Test steps:
1. Create a new tenant.
   Command: /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant create --name=<tenantName>
2. Try creating another tenant with same name.
3. Do a list of tenants and check if we have only one with same name  present.
*/
func TestCreatingSameNameTenant(t *testing.T) {
	executeSameNameTenantTest(t)
	testCleanup()
}

func executeSameNameTenantTest(t *testing.T) {
	tenantName = TestUtil.GetNameWithTimeStamp("tenant1")
	out, err := TestUtil.InvokeCommand(host, ADMIN_CLI_TENANT+"create --name="+tenantName)
	if err != nil {
		log.Fatalf("Failed to create a tenant named: %s . Error: %v ", tenantName, err)
	} else {
		log.Printf("\n Successfully created a  tenant named - %s ", tenantName)
	}

	log.Println("Creating another tenant with the same name - ", tenantName)
	out, err = TestUtil.InvokeCommand(host, ADMIN_CLI_TENANT+"create --name="+tenantName)
	if !strings.Contains(string(out), tenantName+" already exists") {
		log.Fatalf("Failing the test as two tenants with same name get created. Tenant name - [%s]. Out: [%s]", tenantName, string(out))
	}
	// Checking if both tenants exist by using tenant ls command
	out, err = TestUtil.InvokeCommand(host, ADMIN_CLI_TENANT+"ls | cut -d ' ' -f 3")
	if err != nil {
		log.Fatalf("Failed to get list of tenants. Error: %v ", err)
	}
	occurence := strings.Count(string(out), tenantName)
	if occurence > 1 {
		log.Fatalf("Two tenants with same name [%s] exist.", tenantName)
	}
}

func testCleanup() {
	TestUtil.DeleteTenant(host, tenantName)
}
