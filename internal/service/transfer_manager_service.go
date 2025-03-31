package service

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ensingerphilipp/premiumizearr-nova/internal/config"
	"github.com/ensingerphilipp/premiumizearr-nova/internal/progress_downloader"
	"github.com/ensingerphilipp/premiumizearr-nova/internal/utils"
	"github.com/ensingerphilipp/premiumizearr-nova/pkg/premiumizeme"
	log "github.com/sirupsen/logrus"
)

type DownloadDetails struct {
	Added              time.Time
	Name               string
	ProgressDownloader *progress_downloader.WriteCounter
}

type TransferManagerService struct {
	premiumizemeClient *premiumizeme.Premiumizeme
	arrsManager        *ArrsManagerService
	config             *config.Config
	lastUpdated        int64
	transfers          []premiumizeme.Transfer
	runningTask        bool
	downloadListMutex  *sync.Mutex
	downloadList       map[string]*DownloadDetails
	status             string
	downloadsFolderID  string
}

// Handle
func (t TransferManagerService) New() TransferManagerService {
	t.premiumizemeClient = nil
	t.arrsManager = nil
	t.config = nil
	t.lastUpdated = time.Now().Unix()
	t.transfers = make([]premiumizeme.Transfer, 0)
	t.runningTask = false
	t.downloadListMutex = &sync.Mutex{}
	t.downloadList = make(map[string]*DownloadDetails, 0)
	t.status = ""
	t.downloadsFolderID = ""
	return t
}

func (t *TransferManagerService) Init(pme *premiumizeme.Premiumizeme, arrsManager *ArrsManagerService, config *config.Config) {
	t.premiumizemeClient = pme
	t.arrsManager = arrsManager
	t.config = config
	t.CleanUpUnzipDirPeriod()
}

func (t *TransferManagerService) CleanUpUnzipDirPeriod() {
	log.Info("Cleaning unzip directory - deleting files older than 4 days")

	unzipBase, err := t.config.GetUnzipBaseLocation()
	if err != nil {
		log.Errorf("Error getting unzip base location: %s", err.Error())
		return
	}

	// Define the threshold for deletion: 4 days
	threshold := time.Now().AddDate(0, 0, -4)

	err = filepath.Walk(unzipBase, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Warnf("Error accessing path %s: %s", path, err.Error())
			return nil // Continue processing other files/directories
		}

		// Skip the base directory itself
		if path == unzipBase {
			return nil
		}

		// Check if the file/directory is older than 4 days
		if info.ModTime().Before(threshold) {
			log.Infof("Deleting %s (last modified: %s)", path, info.ModTime())

			// Remove the directory/file
			err = os.RemoveAll(path)
			if err != nil {
				log.Errorf("Error deleting %s: %s", path, err.Error())
			}
		}
		return nil
	})

	if err != nil {
		log.Errorf("Error cleaning unzip directory: %s", err.Error())
	}
}

func (t *TransferManagerService) CleanUpUnzipDir() {
	log.Info("Cleaning unzip directory")

	unzipBase, err := t.config.GetUnzipBaseLocation()
	if err != nil {
		log.Errorf("Error getting unzip base location: %s", err.Error())
		return
	}

	err = utils.RemoveContents(unzipBase)
	if err != nil {
		log.Errorf("Error cleaning unzip directory: %s", err.Error())
		return
	}

}

func (manager *TransferManagerService) ConfigUpdatedCallback(currentConfig config.Config, newConfig config.Config) {
	// Todo Change to Downloads-Directory and rmeove Zip Download functionality
	if currentConfig.UnzipDirectory != newConfig.UnzipDirectory {
		log.Trace("Inside ConfigUpdatedCallback")
		manager.CleanUpUnzipDir()
	}
}

func (manager *TransferManagerService) Run(interval time.Duration) {
	manager.downloadsFolderID = utils.GetDownloadsFolderIDFromPremiumizeme(manager.premiumizemeClient)
	for {
		manager.runningTask = true
		manager.TaskUpdateTransfersList()
		manager.TaskCheckPremiumizeDownloadsFolder()
		manager.runningTask = false
		manager.lastUpdated = time.Now().Unix()
		time.Sleep(interval)
	}
}

func (manager *TransferManagerService) GetDownloads() map[string]*DownloadDetails {
	return manager.downloadList
}

func (manager *TransferManagerService) GetTransfers() *[]premiumizeme.Transfer {
	return &manager.transfers
}
func (manager *TransferManagerService) GetStatus() string {
	return manager.status
}

func (manager *TransferManagerService) TaskUpdateTransfersList() {
	log.Debug("Running Task UpdateTransfersList")
	transfers, err := manager.premiumizemeClient.GetTransfers()
	if err != nil {
		log.Errorf("Error getting transfers: %s", err.Error())
		return
	}
	manager.updateTransfers(transfers)

	log.Tracef("Checking %d transfers against %d Arr clients", len(transfers), len(manager.arrsManager.GetArrs()))
	for _, transfer := range transfers {
		found := false
		for _, arr := range manager.arrsManager.GetArrs() {
			if found {
				break
			}
			if transfer.Status == "error" {
				log.Tracef("Checking errored transfer %s against %s history", transfer.Name, arr.GetArrName())
				arrID, contains := arr.HistoryContains(transfer.Name)
				if !contains {
					log.Tracef("%s history doesn't contain %s", arr.GetArrName(), transfer.Name)
					continue
				}
				log.Tracef("Found %s in %s history", transfer.Name, arr.GetArrName())
				found = true
				log.Debugf("Processing transfer that has errored: %s", transfer.Name)
				go arr.HandleErrorTransfer(&transfer, arrID, manager.premiumizemeClient)

			}
		}
	}
}

func (manager *TransferManagerService) TaskCheckPremiumizeDownloadsFolder() {
	log.Debug("Running Task CheckPremiumizeDownloadsFolder")

	items, err := manager.premiumizemeClient.ListFolder(manager.downloadsFolderID)
	if err != nil {
		log.Errorf("Error listing downloads folder: %s", err.Error())
		return
	}

	for _, item := range items {
		if manager.countDownloads() < manager.config.SimultaneousDownloads {
			log.Debugf("Processing completed item: %s", item.Name)
			//TODO Remove Zip capability
			var zip bool = false
			if zip == true {
				manager.HandleFinishedItemZip(item, manager.config.DownloadsDirectory)
			} else {
				manager.HandleFinishedItem(item, manager.config.DownloadsDirectory)
			}
		} else {
			log.Debugf("Not processing any more transfers, %d are running and cap is %d", manager.countDownloads(), manager.config.SimultaneousDownloads)
			break
		}
	}
}

func (manager *TransferManagerService) updateTransfers(transfers []premiumizeme.Transfer) {
	manager.transfers = transfers
}

func (manager *TransferManagerService) addDownload(item *premiumizeme.Item) {
	manager.downloadListMutex.Lock()
	defer manager.downloadListMutex.Unlock()

	manager.downloadList[item.Name] = &DownloadDetails{
		Added:              time.Now(),
		Name:               item.Name,
		ProgressDownloader: progress_downloader.NewWriteCounter(),
	}
}

func (manager *TransferManagerService) countDownloads() int {
	manager.downloadListMutex.Lock()
	defer manager.downloadListMutex.Unlock()

	return len(manager.downloadList)
}

func (manager *TransferManagerService) removeDownload(name string) {
	manager.downloadListMutex.Lock()
	defer manager.downloadListMutex.Unlock()

	delete(manager.downloadList, name)
}

func (manager *TransferManagerService) downloadExists(itemName string) bool {
	manager.downloadListMutex.Lock()
	defer manager.downloadListMutex.Unlock()

	for _, dl := range manager.downloadList {
		if dl.Name == itemName {
			return true
		}
	}

	return false
}

func (manager *TransferManagerService) HandleFinishedItem(item premiumizeme.Item, downloadDirectory string) {
	if manager.downloadExists(item.Name) {
		log.Tracef("Transfer %s is already downloading", item.Name)
		return
	}

	if item.Type == "file" {
		log.Tracef("Handling Item Type File in finished Transfer %s", item.Name)

		// Create Folder with Item Name
		// Move Item into Folder

	}

	//TODO Implement download of single Files
	/* 	if item.Type == "file" {
		log.Debugf("Item is type single file", item.Name)
		//manager.HandleFinishedItemZip(item, downloadDirectory)
		manager.addDownload(&item)
		go func() {
			defer manager.removeDownload(item.Name)
			link, err := manager.premiumizemeClient.GenerateFileLink(item.ID)

			if err != nil {
				log.Debugf("File Link Generation err: %s", err)
			}

			var savePath = path.Join(downloadDirectory, (item.Name + "/"))
			log.Trace("Downloading to: ", savePath)
			err = os.Mkdir(savePath, os.ModePerm)
			if err != nil {
				log.Errorf("Could not create save path: %s", err)
				//		manager.removeDownload(item.Name)
				//		return fmt.Errorf("error creating save path: %w", err)
			}

			var fileSavePath = path.Join(savePath, item.Name)
			log.Trace("Downloading to: ", fileSavePath)
			err = progress_downloader.DownloadFile(link, fileSavePath, manager.downloadList[item.Name].ProgressDownloader)

			if err != nil {
				log.Errorf("error downloading file %s: %s", item.Name, err)
				return
			}
			//Remove download entry from downloads map
			//manager.removeDownload(item.Name)
		}()

	} */

	//Adding of the Root-Parent-Folder of the Transfer prevents the transfer from being downloaded multiple times
	//TODO Needs to be adjusted so the LockItem is not visible in the downloadList
	if item.Type != "folder" {
		log.Errorf("Item Type mismatch when trying to handle finished Transfer %s | %s", item.Name, item.Type)
		return
	}

	manager.addDownload(&item)
	go func() {
		defer manager.removeDownload(item.Name)
		err := manager.downloadFolderRecursively(item, downloadDirectory)
		if err != nil {
			log.Errorf("Error downloading item %s: %s", item.Name, err)
			manager.removeDownload(item.Name)
			return
		}

		err = manager.premiumizemeClient.DeleteFolder(item.ID)
		if err != nil {
			manager.removeDownload(item.Name)
			log.Errorf("Error deleting folder on premiumize.me: %s", err)
			return
		}

		//Remove download entry from downloads map
		//manager.removeDownload(item.Name)
	}()
}

func (manager *TransferManagerService) downloadFolderRecursively(item premiumizeme.Item, downloadDirectory string) error {
	items, err := manager.premiumizemeClient.ListFolder(item.ID)
	if err != nil {
		return fmt.Errorf("error listing folder items: %w", err)
	}
	savePath := path.Join(downloadDirectory, (item.Name + "/"))
	log.Trace("Downloading to: ", savePath)
	err = os.Mkdir(savePath, os.ModePerm)
	if err != nil {
		log.Errorf("Could not create save path: %s", err)
		//		manager.removeDownload(item.Name)
		//		return fmt.Errorf("error creating save path: %w", err)
	}

	for _, item := range items {
		if manager.downloadExists(item.Name) {
			log.Tracef("Transfer %s is already downloading", item.Name)
			return nil
		}
		if item.Type == "file" {
			manager.addDownload(&item)
			link, err := manager.premiumizemeClient.GenerateFileLink(item.ID)
			if err != nil {
				log.Debugf("File Link Generation err: %s", err)
			}
			var fileSavePath = path.Join(savePath, item.Name)
			log.Trace("Downloading to: ", fileSavePath)
			err = progress_downloader.DownloadFile(link, fileSavePath, manager.downloadList[item.Name].ProgressDownloader)
			if err != nil {
				return fmt.Errorf("error downloading file %s: %w", item.Name, err)
			}
			manager.removeDownload(item.Name)
		} else if item.Type == "folder" {
			err = manager.downloadFolderRecursively(item, savePath)
			if err != nil {
				return fmt.Errorf("error downloading folder %s: %w", item.Name, err)
			}
		}
	}
	return nil
}

// Returns when the download has been added to the list
func (manager *TransferManagerService) HandleFinishedItemZip(item premiumizeme.Item, downloadDirectory string) {
	if manager.downloadExists(item.Name) {
		log.Tracef("Transfer %s is already downloading", item.Name)
		return
	}

	manager.addDownload(&item)

	go func() {
		log.Debug("Downloading: ", item.Name)
		log.Tracef("%+v", item)
		var link string
		var err error
		if item.Type == "file" {
			link, err = manager.premiumizemeClient.GenerateZippedFileLink(item.ID)
		} else if item.Type == "folder" {
			link, err = manager.premiumizemeClient.GenerateZippedFolderLink(item.ID)
		} else {
			log.Errorf("Item is not of type 'file' or 'folder' !! Can't download %s", item.Name)
			return
		}
		if err != nil {
			log.Error("Error generating download link: %s", err)
			manager.removeDownload(item.Name)
			return
		}
		log.Trace("Downloading from: ", link)

		splitString := strings.Split(link, "/")

		tempDir, err := manager.config.GetNewUnzipLocation(item.ID)
		if err != nil {
			log.Errorf("Could not create temp dir: %s", err)
			manager.removeDownload(item.Name)
			return
		}

		savePath := path.Join(tempDir, (item.Name + "/"))
		log.Trace("Creating DownloadDirectory: ", savePath)
		err = os.Mkdir(savePath, os.ModePerm)
		if err != nil {
			log.Errorf("Could not create save path: %s", err)
			//		manager.removeDownload(item.Name)
			//		return fmt.Errorf("error creating save path: %w", err)
		}

		var fileSavePath string = path.Join(savePath, splitString[len(splitString)-1])
		log.Trace("Downloading to: ", fileSavePath)

		err = progress_downloader.DownloadFile(link, fileSavePath, manager.downloadList[item.Name].ProgressDownloader)

		if err != nil {
			log.Errorf("Could not download file: %s", err)
			manager.removeDownload(item.Name)
			return
		}

		log.Tracef("Unzipping %s to %s", savePath, downloadDirectory)
		err = utils.Unzip(savePath, downloadDirectory)
		if err != nil {
			log.Errorf("Could not unzip file: %s", err)
			manager.removeDownload(item.Name)
			return
		}

		log.Tracef("Removing zip %s from system", savePath)
		err = os.RemoveAll(savePath)
		if err != nil {
			manager.removeDownload(item.Name)
			log.Errorf("Could not remove zip: %s", err)
			return
		}

		err = manager.premiumizemeClient.DeleteFolder(item.ID)
		if err != nil {
			manager.removeDownload(item.Name)
			log.Error("Error deleting folder on premiumize.me: %s", err)
			return
		}

		//Remove download entry from downloads map
		manager.removeDownload(item.Name)
	}()
}
