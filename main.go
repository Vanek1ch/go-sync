package main

import (
	"proj/handlers"
)

func main() {

	newFolder := &handlers.SyncFolders{FirstFolder: "D:\\exampleRoute1", LastFolder: "D:\\exampleRoute2"}
	newSyncManager := handlers.JSONSyncManager{}

	err := newSyncManager.JSONSyncUpdater(newFolder)
	if err != nil {
		return
	}

	//eventMainLoop()
}
