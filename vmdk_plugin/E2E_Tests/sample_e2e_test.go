package e2e_test

import "testing"
import "os/exec"
import "fmt"
import "os"
import "strings"

//TODO: This is a sample testcase and will be removed after finishing the review.

func TestSomething(t *testing.T) {
	var err error
	var out []byte

	out, err = exec.Command("/usr/bin/ssh", strings.Split(os.Getenv("SSH_KEY_OPT")," ")[0], strings.Split(os.Getenv("SSH_KEY_OPT")," ")[1], "-q", "-kTax", "-o StrictHostKeyChecking=no", "root@"+os.Getenv("ESX"), "/usr/lib/vmware/vmdkops/bin/vmdkops_admin.py", "ls").CombinedOutput()
	fmt.Printf("\nerr=>%s.....\nout=>%s", err, out)

	out, err = exec.Command("/usr/bin/ssh", strings.Split(os.Getenv("SSH_KEY_OPT")," ")[0], strings.Split(os.Getenv("SSH_KEY_OPT")," ")[1], "-q", "-kTax", "-o StrictHostKeyChecking=no", "root@"+os.Getenv("VM1"), "ifconfig").CombinedOutput()
	fmt.Printf("\nerr=>%s .....\nout=>%s", err, out)
}
