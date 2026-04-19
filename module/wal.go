package module

import (
	"fmt"
	"os"
)

func LogCommand(command string) {
	f, err := os.OpenFile("wal.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

	if err != nil {
		fmt.Println("Error opening wal.log: ", err)
		return
	}

	defer f.Close()
	f.WriteString(command + "\n")
	f.Sync()
}

func ClearLog() bool{
	err := os.Truncate("wal.log", 0)

	if err != nil {
		fmt.Println("Error truncate wal.log", err)
		return false
	}
	
	return true
}