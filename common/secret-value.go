package common

// SecretValue is a string that is protected from being logged
type SecretValue string

// String returns a string representation of the secret value
func (s SecretValue) String() string {
	return "********"
}
