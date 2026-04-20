package module

import (
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/arifazola/red-john/enums"
	"github.com/arifazola/red-john/models"
)

func CommandRouter(commands []string, memoryStore *InMemoryStore, role string) (string, error) {	
	commandIsValid := validateCommand(commands)

	if !commandIsValid {
		return "", errors.New("Invalid Command")
	}

	switch commands[0] {
	case "SET":
		if role == enums.RoleFollower {
			return "", errors.New("READ_ONLY_SERVER")
		}

		copiedCommand := slices.Clone(commands)
		value := strings.Join(slices.Delete(copiedCommand, 0, 2), " ")
		item := models.Item{
			Value: value,
			ExpiresAt: time.Now().Add(1200 * time.Second).UnixNano(),
		}
		memoryStore.Set(commands[1], item)
		return "OK\n", nil
	case "GET":
		item, ok := memoryStore.Get(commands[1])

		if !ok {
			return "NOT_FOUND\n", nil
		}

		return item.Value + "\n", nil
	default:
		return "", errors.New("Invalid Command")
	}
}

func validateCommand(commands []string) bool {
	if len(commands) < 2 { return false }
    
    if commands[0] == "GET" && len(commands) != 2 { return false }
    if commands[0] == "SET" && len(commands) < 3 { return false }
    
    return true
}