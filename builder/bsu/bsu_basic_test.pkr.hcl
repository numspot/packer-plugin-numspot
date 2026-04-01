{
	"builders": [{
		"type": "numspot-bsu",
		"vm_type": "ns-eco7-2c2r",
		"source_image": "ami-52b3214f",
		"ssh_username": "outscale",
		"image_name": "packer-test",
		"associate_public_ip_address": true,
		"force_deregister": true
	}]
}
