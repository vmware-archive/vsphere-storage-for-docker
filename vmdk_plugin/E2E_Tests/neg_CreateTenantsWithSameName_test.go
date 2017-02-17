//See the License for the specific language governing permissions and
// limitations under the License.

// This is a negative test to verify if we are able to create two tenants same name.
// Test will ssh to the esx and  create a tenant.
// After creating the tenant, test tries to again create a tenant with same name.
// Expected behavior: Two tenants with same name should not exist.

// Test assumes that SSH cert has been setup to enable password-less login

package e2e_test

import (
	TestUtil "github.com/vmware/docker-volume-vsphere/vmdk_plugin/utils/test_util"
	"log"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

var (
	esx              string = os.Getenv("ESX")
	tenantName       string = ""
	ADMIN_CLI_TENANT string = "/usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant "
)

/*
Test steps:
1. Create a new tenant.
   Command: /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant create --name=<tenantName>
2. Create another tenant with same name.
   Tenant creation will fail as we had already created tenant with same name
3. Do a list of tenants and check if we have only one with same name  present.
   Command: /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant ls
*/
func TestCreateTwoTenantsWithSameName(t *testing.T) {
	log.Println("This is a negative test to create two tenants with same name.")
	executeTest(t)
	deleteTenant(esx, tenantName)
}

func executeTest(t *testing.T) {
	log.Println("Creating a tenant..")
	tenantName = "tenant1_" + strconv.FormatInt(time.Now().Unix(), 20)
	out, err := TestUtil.InvokeCommand(esx, ADMIN_CLI_TENANT+"create --name="+tenantName)
	if err != nil {
		log.Fatalf("Failed to create a tenant named: %s . Error: %v ", tenantName, err)
	} else {
		log.Printf("\n Successfully created a  tenant named - %s ", tenantName)
	}
	log.Println("Creating another tenant with the same name - ", tenantName)
	out, err = TestUtil.InvokeCommand(esx, ADMIN_CLI_TENANT+"create --name="+tenantName)

	if !strings.Contains(string(out), tenantName+" already exists") {
		log.Fatalf("Failing the test as two tenants with same name get created. Tenant name - [%s]. Out: [%s]", tenantName, string(out))
	}

	// Checking if both tenants exist by using tenant ls command
	out, err = TestUtil.InvokeCommand(esx, ADMIN_CLI_TENANT+"ls 2>/dev/null")
	if err != nil {
		log.Fatalf("Failed to get list of tenants. Error: %v ", err)
	}
	occurence := strings.Count(string(out), tenantName)
	if occurence > 1 {
		log.Fatalf("Two tenants with same name [%s] exist.", tenantName)
	}
	log.Println("Test to create two tenants with same name finished successfully.")
}

// following method deletes a tenant.
// It will be deleted after PR # 914 is approved as there is already a method in that
// PR which does the same thing
func deleteTenant(ipAddr string, tenantName string) {
	cmd := ADMIN_CLI_TENANT + "rm --name=" + tenantName
	_, err := TestUtil.InvokeCommand(ipAddr, cmd)
	if err != nil {
		log.Fatalf("Failed to delete the tenant - [%s]. Error - %v ", tenantName, err)
	} else {
		log.Println("Successfully deleted the tenant - ", tenantName)
	}

}
