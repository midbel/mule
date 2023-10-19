package mule

type Cache interface {
	Get(string) ([]byte, error)
}