//go:build acceptance

package bsu

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/hashicorp/packer-plugin-sdk/acctest"
)

var errBadExitCode = errors.New("bad exit code")

func getTestTemplate() string {
	subnetID := os.Getenv("NUMSPOT_SUBNET_ID")
	return fmt.Sprintf(`{
	"builders": [{
		"type": "numspot-bsu",
		"vm_type": "ns-eco7-2c2r",
		"source_image": "ami-52b3214f",
		"subnet_id": "%s",
		"ssh_username": "outscale",
		"image_name": "packer-test-acc",
		"associate_public_ip_address": true,
		"force_deregister": true
	}]
}`, subnetID)
}

func getTestTemplateWithProductCode() string {
	subnetID := os.Getenv("NUMSPOT_SUBNET_ID")
	return fmt.Sprintf(`{
	"builders": [{
		"type": "numspot-bsu",
		"vm_type": "ns-eco7-2c2r",
		"source_image": "ami-52b3214f",
		"subnet_id": "%s",
		"ssh_username": "outscale",
		"image_name": "packer-test-acc-product",
		"product_codes": ["0001"],
		"associate_public_ip_address": true,
		"force_deregister": true
	}]
}`, subnetID)
}

func TestAccBuilder_basic(t *testing.T) {
	testCase := &acctest.PluginTestCase{
		Name:     "bsu_basic_test",
		Template: getTestTemplate(),
		Check: func(buildCommand *exec.Cmd, logfile string) error {
			if buildCommand.ProcessState != nil {
				if buildCommand.ProcessState.ExitCode() != 0 {
					return fmt.Errorf("%w. Logfile: %s", errBadExitCode, logfile)
				}
			}
			return nil
		},
	}
	acctest.TestPlugin(t, testCase)
}

func TestAccBuilder_GoodProductCode(t *testing.T) {
	testCase := &acctest.PluginTestCase{
		Name:     "bsu_product_code_test",
		Template: getTestTemplateWithProductCode(),
		Check: func(buildCommand *exec.Cmd, logfile string) error {
			if buildCommand.ProcessState != nil {
				if buildCommand.ProcessState.ExitCode() != 0 {
					return fmt.Errorf("%w. Logfile: %s", errBadExitCode, logfile)
				}
			}
			return nil
		},
	}
	acctest.TestPlugin(t, testCase)
}

const testBuilderAccWithGoodProductCode = `
{
	"builders": [{
		"type": "numspot-bsu",
		"vm_type": "ns-eco7-2c2r",
		"source_image": "ami-52b3214f",
		"ssh_username": "outscale",
		"image_name": "packer-test-acc-product",
		"product_codes": ["0001"],
		"associate_public_ip_address": true,
		"force_deregister": true
	}]
}
`
