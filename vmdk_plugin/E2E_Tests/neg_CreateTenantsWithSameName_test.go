//See the License for the specific language governing permissions and
// limitations under the License.

// This is a negative test to verify if we are able to create two tenants same name.
// Test will ssh to the esx and  create a tenant.
// After creating the tenant, test tries to again create a tenant with same name.
// Expected behavior: Two tenants with same name should not exist.

// Test assumes that SSH cert has been setup to enable password-less login

package e2e_test

import (
	"fmt"
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
1. Create a new tenant named tenant1_abc.
   Command: /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant create --name=tenant1_abc
2. Create another tenant with same name tenant1_abc
   Tenant creation will fail as we had already created tenant with same name
3. Do a list of tenants and check if we have only one tenant1_abc present.
   Command: /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant ls
*/
func TestCreateTwoTenantsWithSameName(t *testing.T) {
	log.Println("This is a negative test to create two tenants with same.")
	log.Println("Creating a tenant..")
	tenantName = "tenant1_" + strconv.FormatInt(time.Now().Unix(), 20)
	_, err := TestUtil.InvokeCommand(esx, ADMIN_CLI_TENANT+"create --name="+tenantName)
	if err != nil {
		log.Fatalf("Failed to create a tenant named: %s . Error: %v ", tenantName, err)
	} else {
		log.Printf("\n Successfully created a  tenant named - %s ", tenantName)
	}

	// again creating a tenant with the same name
	log.Println("Creating another tenant with same name - ", tenantName)
	TestUtil.InvokeCommand(esx, ADMIN_CLI_TENANT+"create --name="+tenantName)

	// checking if both tenants exist
	out, err := TestUtil.InvokeCommand(esx, ADMIN_CLI_TENANT+"ls 2>/dev/null")
	if err != nil {
		log.Fatalf("Failed to get list of tenants. Error: %v ", err)
	}
	fmt.Println("OUT: ", string(out))
	occurence := strings.Count(string(out), tenantName)
	fmt.Println("COUNT: ", strings.Count(string(out), tenantName))
	if occurence == 0 || occurence > 1 {
		log.Fatalf("Two tenants with same name [%s] exist.", tenantName)
	}
	log.Println("Pass: Test to create two tenants with same name finished.")
}
