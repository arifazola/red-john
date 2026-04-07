package storeInterface

type Store interface {
	Get(key string) (string, bool)
	Set(key, value string)
	Delete(key string)
}