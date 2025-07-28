package processor

import (
	"archive/zip"
	"file-archiver/internal/task"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Processor struct {
	store         task.Store
	queue         chan string
	sem           chan struct{}
	onceStartSync sync.Once
}

func New(store task.Store, maxConcurrent int) *Processor {
	return &Processor{
		store: store,
		queue: make(chan string, 100),
		sem:   make(chan struct{}, maxConcurrent),
	}
}

func (p *Processor) Start() {
	p.onceStartSync.Do(func() {
		go p.loop()
	})
}

func (p *Processor) loop() {
	for id := range p.queue {
		p.sem <- struct{}{}
		go func(taskID string) {
			defer func() { <-p.sem }()
			p.handle(taskID)
		}(id)
	}
}

func (p *Processor) Enqueue(id string) {
	select {
	case p.queue <- id:
	default:
	}
}

func (p *Processor) handle(id string) {
	_ = p.store.Update(id, func(t *task.Task) {
		t.Status = task.StatusProcessing
	})

	tPtr, err := p.store.Get(id)
	if err != nil {
		return
	}

	workDir := filepath.Join(os.TempDir(), "file-archiver", id)
	_ = os.MkdirAll(workDir, 0o755)

	var files []string
	for idx := range tPtr.Items {
		itm := &tPtr.Items[idx]
		filename := fmt.Sprintf("item-%d%s", idx, filepath.Ext(itm.URL))
		dest := filepath.Join(workDir, filename)
		if err := downloadFile(itm.URL, dest); err != nil {
			itm.Status = task.ItemError
			itm.ErrMsg = err.Error()
		} else {
			itm.Status = task.ItemOK
			files = append(files, dest)
		}
	}

	if len(files) > 0 {
		zipPath := filepath.Join(workDir, id+".zip")
		if err := createZip(zipPath, files); err != nil {
			_ = p.store.Update(id, func(t *task.Task) {
				t.Status = task.StatusError
				t.ErrMsg = err.Error()
			})
			return
		}
		_ = p.store.Update(id, func(t *task.Task) {
			t.Status = task.StatusDone
			t.ResultPath = zipPath
		})
	} else {
		_ = p.store.Update(id, func(t *task.Task) {
			t.Status = task.StatusError
			t.ErrMsg = "no files to archive"
		})
	}
}

func downloadFile(srcURL, dest string) error {
	client := http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(srcURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http status %d", resp.StatusCode)
	}
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, resp.Body)
	return err
}

func createZip(zipPath string, files []string) error {
	zipFile, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	zw := zip.NewWriter(zipFile)
	for _, fpath := range files {
		if err := addFileToZip(zw, fpath); err != nil {
			zw.Close()
			return err
		}
	}
	return zw.Close()
}

func addFileToZip(zw *zip.Writer, fpath string) error {
	file, err := os.Open(fpath)
	if err != nil {
		return err
	}
	defer file.Close()

	fi, err := file.Stat()
	if err != nil {
		return err
	}

	header, err := zip.FileInfoHeader(fi)
	if err != nil {
		return err
	}
	header.Name = filepath.Base(fpath)
	header.Method = zip.Deflate

	w, err := zw.CreateHeader(header)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, file)
	return err
}
