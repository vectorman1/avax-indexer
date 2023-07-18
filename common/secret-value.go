package common

type SecretValue string

func (s SecretValue) String() string {
	return "********"
}
