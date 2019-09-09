package main

import (
	"bufio"
	"crypto/sha256"
	"fmt"
	"golang.org/x/xerrors"
	"io"
	"os"
	pathutil "path"
	"time"

	"github.com/drone/drone-cache-lib/archive/util"
	"github.com/drone/drone-cache-lib/cache"
	"github.com/drone/drone-cache-lib/storage"
	log "github.com/sirupsen/logrus"
)

// Plugin structure
type Plugin struct {
	Filename     string
	Path         string
	FallbackPath string
	FlushPath    string
	Mode         string
	FlushAge     int
	Mount        []string

	Storage storage.Storage

	UseFileHash bool
	FileHashPath string
}

const (
	// RestoreMode for resotre mode string
	RestoreMode = "restore"
	// RebuildMode for rebuild mode string
	RebuildMode = "rebuild"
	// FlushMode for flush mode string
	FlushMode = "flush"
)

// Exec runs the plugin
func (p *Plugin) Exec() error {
	var err error

	at, err := util.FromFilename(p.Filename)

	if err != nil {
		return err
	}

	c := cache.New(p.Storage, at)

	path := pathutil.Join(p.Path, p.Filename)
	fallbackPath := pathutil.Join(p.FallbackPath, p.Filename)

	if p.Mode == RebuildMode {
		if p.UseFileHash && p.FileHashPath != ""{
			if path,err = p.getPathFromHashedFile();err != nil{
				return  err
			}
		}

		log.Infof("Rebuilding cache at %s", path)
		err = c.Rebuild(p.Mount, path)

		if err == nil {
			log.Infof("Cache rebuilt")
		}
	}

	if p.Mode == RestoreMode {
		if p.UseFileHash && p.FileHashPath != ""{
			if path,err = p.getPathFromHashedFile();err != nil{
				return  err
			}
		}

		log.Infof("Restoring cache at %s", path)
		err = c.Restore(path, fallbackPath)

		if err == nil {
			log.Info("Cache restored")
		}
	}

	if p.Mode == FlushMode {
		log.Infof("Flushing cache items older than %d days at %s", p.FlushAge, path)
		f := cache.NewFlusher(p.Storage, genIsExpired(p.FlushAge))
		err = f.Flush(p.FlushPath)

		if err == nil {
			log.Info("Cache flushed")
		}
	}

	return err
}

func (p *Plugin) getPathFromHashedFile()(string,error){
	hashed_filename,err := getHashedFile(p.FileHashPath)
	if err != nil{
		return "",err
	}
	return pathutil.Join(p.Path,hashed_filename),nil
}

func genIsExpired(age int) cache.DirtyFunc {
	return func(file storage.FileEntry) bool {
		// Check if older than "age" days
		return file.LastModified.Before(time.Now().AddDate(0, 0, age*-1))
	}
}

func getHashedFile(filepath string)(string,error){
	f,err := os.Open(filepath)
	if err != nil{
		return "",xerrors.Errorf("file open error: %w",err)
	}
	defer f.Close()
	h := sha256.New()
	if _,err := io.Copy(h,f);err != nil{
		return "",xerrors.Errorf("failed file hash: %w",err)
	}
	return fmt.Sprintf("%x",h.Sum(nil)),nil
}