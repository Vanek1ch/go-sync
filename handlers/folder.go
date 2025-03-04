package handlers

import (
	"fmt"
	"os"
	prErr "proj/errors"
)

type SyncFolders struct {
	FirstFolder string
	LastFolder  string
	SyncType    bool
}

// sync type true == solo mode, false duo mode.
func folderPicker(firstFolder, secondFolder string, syncType bool) *SyncFolders {
	canEnter := CanEnterFolder(firstFolder, secondFolder)
	if !canEnter {
		fmt.Println(prErr.ErrorFolder)
		return nil
	}
	if syncType {
		newSync := &SyncFolders{firstFolder, secondFolder, true}
		return newSync
	}
	fmt.Println(prErr.ErrorSyncTypeMissing)
	return nil
}

func CanEnterFolder(folder ...string) bool {
	for _, f := range folder {
		if err := os.Chdir(f); err != nil {
			return false
		}
	}
	return true
}
