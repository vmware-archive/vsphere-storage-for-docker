package e2e_test

import "testing"
import "os/exec"
import "os"
import "strings"

// TODO: This is a sample testcase and will be removed after finishing the review.

func TestSomething(t *testing.T) {

	exec.Command("/usr/bin/ssh", strings.Split(os.Getenv("SSH_KEY_OPT")," ")[0], strings.Split(os.Getenv("SSH_KEY_OPT")," ")[1], "-q", "-kTax", "-o StrictHostKeyChecking=no", "root@"+os.Getenv("ESX"), "/usr/lib/vmware/vmdkops/bin/vmdkops_admin.py", "ls").CombinedOutput()

	exec.Command("/usr/bin/ssh", strings.Split(os.Getenv("SSH_KEY_OPT")," ")[0], strings.Split(os.Getenv("SSH_KEY_OPT")," ")[1], "-q", "-kTax", "-o StrictHostKeyChecking=no", "root@"+os.Getenv("VM1"), "ifconfig").CombinedOutput()
}
