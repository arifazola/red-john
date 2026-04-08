package main_test

import (
	"fmt"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/arifazola/red-john/models"
	"github.com/arifazola/red-john/module"
)

func TestSetRace(t *testing.T) {
	store := module.NewInMemoryStore()

	var wg sync.WaitGroup
	wg.Add(200)

	for i := 0; i < 100; i++ {
		go func ()  {
			item1 := models.Item{
				Value: strconv.Itoa(i),
				ExpiresAt: 0,
			}
			store.Set("name", item1)
			defer wg.Done()
		}()
	}

	for i := 0; i < 100; i++ {
		go func ()  {
			time.Sleep(3 * time.Second)
			val, _ := store.Get("name")
			fmt.Println(val)
			defer wg.Done()
		}()
	}

	wg.Wait()
}