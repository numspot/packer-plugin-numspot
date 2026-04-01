// Package common provides shared utilities for the numspot-plugin-packer builder.
package common

import "errors"

var (
	errClientIDRequired = errors.New(
		"client_id is required (or set NUMSPOT_CLIENT_ID env var)",
	)
	errClientSecretRequired = errors.New(
		"client_secret is required (or set NUMSPOT_CLIENT_SECRET env var)",
	)
	errSpaceIDRequired = errors.New(
		"space_id is required (or set NUMSPOT_SPACE_ID env var)",
	)
	errClientIDSecretBothSet = errors.New(
		"`client_id` and `client_secret` must both be either set or not set",
	)
	errUnexpectedKeyInImagesMap   = errors.New("unexpected type of key in Images map")
	errUnexpectedValueInImagesMap = errors.New("unexpected type for value in Images map")
	errDeviceNameRequired         = errors.New(
		"the `device_name` must be specified for every device in the block device mapping",
	)
	errImageNameRequired = errors.New("image_name must be specified")
	errImageNameLength   = errors.New(
		"image_name must be between 3 and 128 characters long",
	)
	errImageNameInvalidChars       = errors.New("image_name contains invalid characters")
	errNoImageFoundMatchingFilters = errors.New("no Image was found matching filters")
	errMultipleImagesFound         = errors.New(
		"your query returned more than one result. Please try a more specific search, or set most_recent to true",
	)
	errVMNoID                          = errors.New("VM has no Id")
	errVMNotFound                      = errors.New("VM not found")
	errCouldNotDetermineVMAddress      = errors.New("couldn't determine address for VM")
	errTimeoutWaitingForPassword       = errors.New("timeout waiting for password")
	errVMIDIsNil                       = errors.New("VM Id is nil")
	errPasswordWaitCancelled           = errors.New("retrieve password wait cancelled")
	errEncryptedPrivateKeyNotSupported = errors.New("encrypted private key isn't yet supported")
	errNoPrivateKeyReturned            = errors.New("error creating temporary keypair: no private key returned")
	errNoPublicIPIDReturned            = errors.New(
		"error creating temporary PublicIp: no Id returned",
	)
	errSourceImageIDIsNil            = errors.New("source image Id is nil")
	errNoVMReturnedFromCreate        = errors.New("no VM returned from CreateVms")
	errFindingSourceVM               = errors.New("error finding source VM")
	errNoSubnetsFoundMatchingFilters = errors.New("no Subnets were found matching filters")
	errSourceImageRequired           = errors.New(
		"a source_image or source_image_filter must be specified",
	)
	errSourceAMIFilterOwnerRequired = errors.New(
		"for security reasons, your source AMI filter must declare an owner",
	)
	errVMTypeRequired            = errors.New("an vm_type must be specified")
	errBlockDurationMultipleOf60 = errors.New("block_duration_minutes must be multiple of 60")
	errUserDataConflict          = errors.New(
		"only one of user_data or user_data_file can be specified",
	)
	errSecurityGroupIDConflict = errors.New(
		"only one of security_group_id or security_group_ids can be specified",
	)
	errShutdownBehaviorInvalid = errors.New(
		"shutdown_behavior only accepts 'stop' or 'terminate' values",
	)
	errSSHPrivateKeyRequiredForWinRM = errors.New(
		"ssh_private_key_file must be provided to retrieve the winrm password when using ssh_keypair_name",
	)
	errSSHPrivateKeyOrAgentRequired = errors.New(
		"ssh_private_key_file must be provided or ssh_agent_auth enabled when ssh_keypair_name is specified",
	)

	errStateTypeCastFailed        = errors.New("state type cast failed")
	errSourceImageTypeAssertion   = errors.New("source_image type assertion failed")
	errNoSecurityGroupIDReturned  = errors.New("error creating security group: no Id returned")
	errSecurityGroupVpcIdRequired = errors.New(
		"error creating security group: VpcId is required but not found. Please specify net_id or ensure a subnet is available",
	)
	errImageBootModeUnsupported = errors.New("image boot mode is not supported")
	// ErrInvalidRetryIntervals is returned when retry intervals are invalid.
	ErrInvalidRetryIntervals  = errors.New("invalid retry intervals (negative or initial < max)")
	errUnknownSSHInterface    = errors.New("unknown SSH interface type")
	errVMBootModeUnsupported  = errors.New("vm boot mode is not supported")
	errUserDataFileNotFound   = errors.New("user_data_file not found")
	errUnknownVMType          = errors.New("error determining main VM type")
	errT2UnlimitedNonT2       = errors.New("T2 Unlimited enabled with a non-T2 VM type")
	errImageRetrievalNoImages = errors.New("error retrieving details for Image: no images found")
	errMultipleVPCsMatched    = errors.New("exactly one VPC should match the filter")
	errMultipleSubnetsMatched = errors.New("multiple subnets matched filter")
	errImageNameConflict      = errors.New("name conflicts with an existing Image")
	errReadingUserDataFile    = errors.New("problem reading user data file")
	errInvalidRootDeviceType  = errors.New("source image has an invalid root device type")
	errTaggingSourceVM        = errors.New("error tagging source VM")
	errTaggingVolumes         = errors.New("error tagging volumes")
	errWrongShutdownBehavior  = errors.New("wrong value for the shutdown behavior")
	errImageNotFoundByID      = errors.New("no Image was found with the given Id")
)
