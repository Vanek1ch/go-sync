package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type JSONSyncManager struct {
	SyncF     *SyncFolders
	JSONSFile *JSONSyncFile
}

type JSONSyncFile struct {
	SyncType        string     `json:"sync_type"`
	FirstFolder     string     `json:"first_folder"`
	LastFolder      string     `json:"last_folder"`
	ElemList        []*Element `json:"elem_list"`
	FirstFolderHash string     `json:"first_folder_hash"`
	LastFolderHash  string     `json:"last_folder_hash"`
	LastModified    string     `json:"last_modified"`
	errElems        *ErrorElements
}

type Element struct {
	FullName     string     `json:"full_name"`
	Name         string     `json:"name"`
	Extension    string     `json:"extension"`
	Route        string     `json:"route"`
	LastModified string     `json:"last_modified"`
	Size         int64      `json:"size"`
	Elems        []*Element `json:"elems,omitempty"`
}

type ErrorElements struct {
	errs []error
	mu   sync.RWMutex
}

func (m *JSONSyncManager) JSONSyncUpdater(folders *SyncFolders) error {

	newSyncFile := &JSONSyncFile{
		SyncType:        "solo",
		FirstFolder:     folders.FirstFolder,
		LastFolder:      folders.LastFolder,
		ElemList:        nil,
		FirstFolderHash: "example_hash",
		LastFolderHash:  "example_hash",
		LastModified:    time.Now().Format(time.RFC3339),
	}

	newErrorElems := &ErrorElements{}

	m.JSONSFile = newSyncFile
	newSyncFile.errElems = newErrorElems
	m.SyncF = folders

	rootElement, err := m.elemChecker(folders.FirstFolder)
	if err != nil {
		return err
	}

	newSyncFile.ElemList = []*Element{rootElement}

	jsonData, err := json.MarshalIndent(newSyncFile, "", "  ")
	if err != nil {
		return errors.New("error marshalling JSON Sync File")
	}

	if err := os.WriteFile(filepath.Join(folders.FirstFolder, "JSONSync.json"), jsonData, 0644); err != nil {
		return errors.New("error writing JSON Sync File")
	}

	fmt.Println("Файл JSONSync.JSON успешно создан, приступаю к копированию файлов...")

	m.CopyFile(m.JSONSFile.ElemList[0])

	if len(m.JSONSFile.errElems.errs) > 0 {
		for _, errs := range m.JSONSFile.errElems.errs {
			fmt.Println(errs)
		}
	}

	return nil
}

func (m *JSONSyncManager) elemChecker(path string) (*Element, error) {
	dirInfo, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	currentElement := &Element{
		FullName:     dirInfo.Name(),
		Name:         dirInfo.Name(),
		Extension:    "",
		Route:        path,
		LastModified: dirInfo.ModTime().Format(time.RFC3339),
		Size:         dirInfo.Size(),
		Elems:        make([]*Element, 0),
	}

	var wg sync.WaitGroup
	dirChan := make(chan *Element)

	for _, entry := range entries {
		if entry.IsDir() {
			wg.Add(1)
			go func(e os.DirEntry) {
				defer wg.Done()
				elem, err := m.elemChecker(filepath.Join(path, e.Name()))
				if err != nil {
					m.JSONSFile.errElems.mu.Lock()
					defer m.JSONSFile.errElems.mu.Unlock()
					m.JSONSFile.errElems.errs = append(m.JSONSFile.errElems.errs, err)
					return
				}
				dirChan <- elem
			}(entry)
		} else {
			info, err := entry.Info()
			if err != nil {
				return nil, err
			}

			fileName, fileExtension, _ := strings.Cut(info.Name(), ".")
			if fileName == "JSONSync" {
				continue
			}
			fileElem := &Element{
				FullName:     info.Name(),
				Name:         fileName,
				Extension:    fileExtension,
				Route:        filepath.Join(path, info.Name()),
				LastModified: info.ModTime().Format(time.RFC3339),
				Size:         info.Size(),
			}
			currentElement.Elems = append(currentElement.Elems, fileElem)
		}
	}

	go func() {
		wg.Wait()
		close(dirChan)
	}()

	for elem := range dirChan {
		currentElement.Elems = append(currentElement.Elems, elem)
	}

	return currentElement, nil
}

func (m *JSONSyncManager) CopyFile(element *Element) {
	hostFolder := m.SyncF.FirstFolder
	_, elemRouteWithoutHostFolder, _ := strings.Cut(element.Route, hostFolder)
	dst := path.Join(m.SyncF.LastFolder, elemRouteWithoutHostFolder)

	for _, elem := range element.Elems {
		src := path.Join(elem.Route)
		sfi, err := os.Stat(src)
		if err != nil {
			m.JSONSFile.errElems.mu.Lock()
			m.JSONSFile.errElems.errs = append(m.JSONSFile.errElems.errs, err)
			m.JSONSFile.errElems.mu.Unlock()
			continue
		}

		if sfi.IsDir() {
			relPath, err := filepath.Rel(hostFolder, elem.Route)
			if err != nil {
				m.JSONSFile.errElems.mu.Lock()
				m.JSONSFile.errElems.errs = append(m.JSONSFile.errElems.errs, err)
				m.JSONSFile.errElems.mu.Unlock()
				continue
			}

			dstDir := filepath.Join(dst, relPath)
			if err := os.MkdirAll(dstDir, 0755); err != nil {
				m.JSONSFile.errElems.mu.Lock()
				m.JSONSFile.errElems.errs = append(m.JSONSFile.errElems.errs, err)
				m.JSONSFile.errElems.mu.Unlock()
				continue
			}

		} else if sfi.Mode().IsRegular() { // Если это обычный файл
			dstFile := filepath.Join(dst, filepath.Base(src))

			dfi, err := os.Stat(dstFile)
			if err == nil {
				if !dfi.Mode().IsRegular() {
					m.JSONSFile.errElems.mu.Lock()
					m.JSONSFile.errElems.errs = append(m.JSONSFile.errElems.errs, err)
					m.JSONSFile.errElems.mu.Unlock()
					continue
				}
				if os.SameFile(sfi, dfi) {
					continue
				}
			} else if !os.IsNotExist(err) {
				m.JSONSFile.errElems.mu.Lock()
				m.JSONSFile.errElems.errs = append(m.JSONSFile.errElems.errs, err)
				m.JSONSFile.errElems.mu.Unlock()
				continue
			}

			if err := copyFileContents(src, dstFile); err != nil {
				m.JSONSFile.errElems.mu.Lock()
				m.JSONSFile.errElems.errs = append(m.JSONSFile.errElems.errs, err)
				m.JSONSFile.errElems.mu.Unlock()
				continue
			}
		} else {
			m.JSONSFile.errElems.mu.Lock()
			m.JSONSFile.errElems.errs = append(m.JSONSFile.errElems.errs, err)
			m.JSONSFile.errElems.mu.Unlock()
			continue
		}

	}

}
func copyFileContents(src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		cerr := out.Close()
		if err == nil {
			err = cerr
		}
	}()
	if _, err = io.Copy(out, in); err != nil {
		return err
	}
	err = out.Sync()
	return err
}

/*func removeFiles(dir string) error {
	return nil
}
*/
