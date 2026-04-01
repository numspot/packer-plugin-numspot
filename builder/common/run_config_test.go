package common

import (
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/packer-plugin-sdk/communicator"
	"github.com/hashicorp/packer-plugin-sdk/template/config"
)

func testConfig() *RunConfig {
	return &RunConfig{
		SourceImage: "abcd",
		VmType:      "m1.small",
		Comm: communicator.Config{
			SSH: communicator.SSH{
				SSHUsername: "foo",
			},
		},
	}
}

func testConfigFilter() *RunConfig {
	cfg := testConfig()
	cfg.SourceImage = ""
	cfg.SourceImageFilter = ImageFilterOptions{}
	return cfg
}

func TestRunConfigPrepare(t *testing.T) {
	c := testConfig()
	err := c.Prepare(nil)
	if len(err) > 0 {
		t.Fatalf("err: %s", err)
	}
}

func TestRunConfigPrepare_VmType(t *testing.T) {
	c := testConfig()
	c.VmType = ""
	if err := c.Prepare(nil); len(err) != 1 {
		t.Fatalf("Should error if an vm_type is not specified")
	}
}

func TestRunConfigPrepare_SourceImage(t *testing.T) {
	c := testConfig()
	c.SourceImage = ""
	if err := c.Prepare(nil); len(err) != 2 {
		t.Fatalf("Should error if a source_image (or source_image_filter) is not specified")
	}
}

func TestRunConfigPrepare_SourceImageFilterBlank(t *testing.T) {
	c := testConfigFilter()
	if err := c.Prepare(nil); len(err) != 2 {
		t.Fatalf(
			"Should error if source_ami_filter is empty or not specified (and source_ami is not specified)",
		)
	}
}

func TestRunConfigPrepare_SourceImageFilterOwnersBlank(t *testing.T) {
	c := testConfigFilter()
	const filterKeyName = "name"
	const filterValue = "foo"
	c.SourceImageFilter = ImageFilterOptions{
		NameValueFilter: config.NameValueFilter{
			Filters: map[string]string{filterKeyName: filterValue},
		},
	}
	if err := c.Prepare(nil); len(err) != 1 {
		t.Fatalf("Should error if Owners is not specified)")
	}
}

func TestRunConfigPrepare_SourceImageFilterGood(t *testing.T) {
	c := testConfigFilter()
	owner := "123"
	const filterKeyName = "name"
	const filterValue = "foo"
	goodFilter := ImageFilterOptions{
		Owners: []string{owner},
		NameValueFilter: config.NameValueFilter{
			Filters: map[string]string{filterKeyName: filterValue},
		},
	}
	c.SourceImageFilter = goodFilter
	if err := c.Prepare(nil); len(err) != 0 {
		t.Fatalf("err: %s", err)
	}
}

func TestRunConfigPrepare_EnableT2UnlimitedGood(t *testing.T) {
	c := testConfig()
	c.VmType = "t2.micro"
	c.EnableT2Unlimited = true
	err := c.Prepare(nil)
	if len(err) > 0 {
		t.Fatalf("err: %s", err)
	}
}

func TestRunConfigPrepare_EnableT2UnlimitedBadVmType(t *testing.T) {
	c := testConfig()
	c.VmType = "m5.large"
	c.EnableT2Unlimited = true
	err := c.Prepare(nil)
	if len(err) != 1 {
		t.Fatalf("Should error if T2 Unlimited is enabled with non-T2 vm_type")
	}
}

func TestRunConfigPrepare_SSHPort(t *testing.T) {
	c := testConfig()
	c.Comm.SSHPort = 0
	if err := c.Prepare(nil); len(err) != 0 {
		t.Fatalf("err: %s", err)
	}

	if c.Comm.SSHPort != 22 {
		t.Fatalf("invalid value: %d", c.Comm.SSHPort)
	}

	c.Comm.SSHPort = 44
	if err := c.Prepare(nil); len(err) != 0 {
		t.Fatalf("err: %s", err)
	}

	if c.Comm.SSHPort != 44 {
		t.Fatalf("invalid value: %d", c.Comm.SSHPort)
	}
}

func TestRunConfigPrepare_UserData(t *testing.T) {
	c := testConfig()
	tf, err := os.CreateTemp("", "packer")
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	defer func() { _ = os.Remove(tf.Name()) }()
	defer func() { _ = tf.Close() }()

	c.UserData = "foo"
	c.UserDataFile = tf.Name()
	if err := c.Prepare(nil); len(err) != 1 {
		t.Fatalf("Should error if user_data string and user_data_file have both been specified")
	}
}

func TestRunConfigPrepare_UserDataFile(t *testing.T) {
	c := testConfig()
	if err := c.Prepare(nil); len(err) != 0 {
		t.Fatalf("err: %s", err)
	}

	c.UserDataFile = "idontexistidontthink"
	if err := c.Prepare(nil); len(err) != 1 {
		t.Fatalf("Should error if the file specified by user_data_file does not exist")
	}

	tf, err := os.CreateTemp("", "packer")
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	defer func() { _ = os.Remove(tf.Name()) }()
	defer func() { _ = tf.Close() }()

	c.UserDataFile = tf.Name()
	if err := c.Prepare(nil); len(err) != 0 {
		t.Fatalf("err: %s", err)
	}
}

func TestRunConfigPrepare_TemporaryKeyPairName(t *testing.T) {
	c := testConfig()
	c.Comm.SSHTemporaryKeyPairName = ""
	if err := c.Prepare(nil); len(err) != 0 {
		t.Fatalf("err: %s", err)
	}

	if c.Comm.SSHTemporaryKeyPairName == "" {
		t.Fatal("keypair name is empty")
	}

	r := regexp.MustCompile(`\Apk-\d+\z`)
	if !r.MatchString(c.Comm.SSHTemporaryKeyPairName) {
		t.Fatal("keypair name is not valid")
	}

	c.Comm.SSHTemporaryKeyPairName = "ssh-key-123"
	if err := c.Prepare(nil); len(err) != 0 {
		t.Fatalf("err: %s", err)
	}

	if c.Comm.SSHTemporaryKeyPairName != "ssh-key-123" {
		t.Fatal("keypair name does not match")
	}
}
