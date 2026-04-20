package module

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

func LogCommand(command string) {
	f, err := os.OpenFile("wal.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

	if err != nil {
		fmt.Println("Error opening wal.log: ", err)
		return
	}

	defer f.Close()
	expiredDataNano := time.Now().Add(1200 * time.Second).UnixNano()
	f.WriteString(command + " EXP_AT " + strconv.FormatInt(expiredDataNano, 10) + "\n")
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