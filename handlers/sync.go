package handlers

import (
	"encoding/json"
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
	ElemList        []*Element `json:"elem_list"` // Используем указатели для эффективности
	FirstFolderHash string     `json:"first_folder_hash"`
	LastFolderHash  string     `json:"last_folder_hash"`
	LastModified    string     `json:"last_modified"`
	mu              sync.RWMutex
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

func (m *JSONSyncManager) JSONSyncUpdater(folders *SyncFolders) error {
	rootElement, err := m.elemChecker(folders.FirstFolder)
	if err != nil {
		return fmt.Errorf("ошибка при обходе директории: %w", err)
	}

	newSyncFile := &JSONSyncFile{
		SyncType:        "solo",
		FirstFolder:     folders.FirstFolder,
		LastFolder:      folders.LastFolder,
		ElemList:        []*Element{rootElement},
		FirstFolderHash: "example_hash",
		LastFolderHash:  "example_hash",
		LastModified:    time.Now().Format(time.RFC3339),
	}

	m.JSONSFile = newSyncFile
	m.SyncF = folders

	jsonData, err := json.MarshalIndent(newSyncFile, "", "  ")
	if err != nil {
		return fmt.Errorf("ошибка при сериализации JSON: %w", err)
	}

	if err := os.WriteFile(filepath.Join(folders.FirstFolder, "JSONSync.json"), jsonData, 0644); err != nil {
		return fmt.Errorf("ошибка при записи файла: %w", err)
	}

	fmt.Println("Файл JSONSync.JSON успешно создан, приступаю к копированию файлов...")

	err = m.CopyFile(m.JSONSFile.ElemList[0])
	if err != nil {
		fmt.Printf("Ошибка копирования файла: %v\n", err)
		return err
	}

	return nil
}

func (m *JSONSyncManager) elemChecker(path string) (*Element, error) {
	dirInfo, err := os.Stat(path)
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

	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	var wg sync.WaitGroup
	dirChan := make(chan *Element)
	errChan := make(chan error)

	for _, entry := range entries {
		if entry.IsDir() {
			wg.Add(1)
			go func(e os.DirEntry) {
				defer wg.Done()
				elem, err := m.elemChecker(filepath.Join(path, e.Name()))
				if err != nil {
					errChan <- fmt.Errorf("директория %s: %w", e.Name(), err)
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
		close(errChan)
	}()

	for elem := range dirChan {
		currentElement.Elems = append(currentElement.Elems, elem)
	}

	if len(errChan) > 0 {
		for err := range errChan {
			return nil, err
		}
	}

	return currentElement, nil
}

func (m *JSONSyncManager) CopyFile(element *Element) (err error) {
	hostFolder := m.SyncF.FirstFolder
	_, elemRouteWithoutHostFolder, _ := strings.Cut(element.Route, hostFolder)
	dst := path.Join(m.SyncF.LastFolder, elemRouteWithoutHostFolder)

	var wg sync.WaitGroup
	errChan := make(chan error)

	for _, elem := range element.Elems {
		src := path.Join(elem.Route)
		sfi, err := os.Stat(src)
		if err != nil {
			return fmt.Errorf("failed to stat source: %w", err)
		}

		if sfi.IsDir() {
			relPath, err := filepath.Rel(hostFolder, elem.Route)
			if err != nil {
				return fmt.Errorf("failed to get relative path: %w", err)
			}

			dstDir := filepath.Join(dst, relPath)
			if err := os.MkdirAll(dstDir, 0755); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}

			wg.Add(1)
			go func(e *Element) {
				defer wg.Done()
				if err := m.CopyFile(e); err != nil {
					errChan <- err
				}
			}(elem)
		} else if sfi.Mode().IsRegular() { // Если это обычный файл
			dstFile := filepath.Join(dst, filepath.Base(src))

			dfi, err := os.Stat(dstFile)
			if err == nil {
				if !dfi.Mode().IsRegular() {
					return fmt.Errorf("destination is not a regular file: %s", dstFile)
				}
				if os.SameFile(sfi, dfi) {
					continue
				}
			} else if !os.IsNotExist(err) {
				return fmt.Errorf("failed to stat destination: %w", err)
			}

			if err := copyFileContents(src, dstFile); err != nil {
				return fmt.Errorf("failed to copy file contents: %w", err)
			}
		} else {
			return fmt.Errorf("unsupported file type: %s (%q)", src, sfi.Mode().String())
		}

	}

	go func() {
		wg.Wait()
		close(errChan)
	}()
	return <-errChan

}
func copyFileContents(src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return
	}
	defer func() {
		cerr := out.Close()
		if err == nil {
			err = cerr
		}
	}()
	if _, err = io.Copy(out, in); err != nil {
		return
	}
	err = out.Sync()
	return
}

func removeFiles(dir string) error {
	return nil
}
