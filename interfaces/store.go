package interfaces

import "github.com/arifazola/red-john/models"

type Store interface {
	Get(key string) (models.Item, bool)
	Set(key string, value models.Item)
	Delete(key string)
	GetAll() (map[string]models.Item, bool)
}