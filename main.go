package main

import (
	"fmt"
	"go-sync/handlers"
	"runtime"
)

func main() {
	runtime.GOMAXPROCS(4)
	newFolder := &handlers.SyncFolders{FirstFolder: "C:\\Program Files\\Mozilla Firefox", LastFolder: "C:\\Program Files\\folder2"}
	newSyncManager := &handlers.JSONSyncManager{}

	err := newSyncManager.JSONSyncUpdater(newFolder)
	if err != nil {
		fmt.Print(err)
	}

	//eventMainLoop()
}
