package module

import (
	"errors"
	"time"

	"github.com/arifazola/red-john/models"
)

func CommandRouter(commands []string, memoryStore *InMemoryStore) (string, error) {
	commandIsValid := validateCommand(commands)

	if !commandIsValid {
		return "", errors.New("Invalid Command")
	}

	switch commands[0] {
	case "SET":
		item := models.Item{
			Value: commands[2],
			ExpiresAt: time.Now().Add(1200 * time.Second).UnixNano(),
		}
		memoryStore.Set(commands[1], item)
		return "OK\n", nil
	case "GET":
		item, _ := memoryStore.Get(commands[1])
		return item.Value, nil
	default:
		return "", errors.New("Invalid Command")
	}
}

func validateCommand(commands []string) bool {
	if len(commands) < 2 { return false }
    
    if commands[0] == "GET" && len(commands) != 2 { return false }
    if commands[0] == "SET" && len(commands) != 3 { return false }
    
    return true
}