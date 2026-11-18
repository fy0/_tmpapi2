package repository

// Empty observation identifiers are NULL, not synthetic identifiers and not
// empty-string entries in the partial conversation indexes.
func gatewayLogOptionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
