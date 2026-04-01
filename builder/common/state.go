package common

import (
	"fmt"

	openapi_types "github.com/oapi-codegen/runtime/types"
)

func parseSpaceID(spaceID string) (openapi_types.UUID, error) {
	var uuid openapi_types.UUID
	if err := uuid.UnmarshalText([]byte(spaceID)); err != nil {
		return uuid, fmt.Errorf("parsing space ID %q: %w", spaceID, err)
	}
	return uuid, nil
}
