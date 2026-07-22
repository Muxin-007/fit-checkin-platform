package file

import (
	"io"
	"os"
	"sync"
)

const (
	// FileBlockSize 文件分块大小
	FileBlockSize = 10 * MiB
)

type File struct {
	uploadPool sync.Pool
}

func NewFile() *File {
	return &File{
		uploadPool: sync.Pool{
			New: func() any {
				return make([]byte, FileBlockSize)
			},
		},
	}
}

func (this *File) Upload(dataChan chan FileRes, f io.Reader, dest string, fileSize int64) {
	destFile, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		dataChan <- FileRes{Success: false, Finish: true, Err: err}
		return
	}
	defer func() {
		if cerr := destFile.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	if err = destFile.Truncate(fileSize); err != nil {
		dataChan <- FileRes{Success: false, Finish: true, Err: err}
		return
	}

	buf := this.uploadPool.Get().([]byte)
	defer this.uploadPool.Put(buf)

	_, err = io.CopyBuffer(destFile, f, buf)
	if err != nil {
		dataChan <- FileRes{Success: false, Finish: true, Err: err}
		return
	}

	if err = destFile.Sync(); err != nil {
		dataChan <- FileRes{Success: false, Finish: true, Err: err}
		return
	}

	dataChan <- FileRes{Success: true, Finish: true}
}

func (this *File) Download(src string) (io.ReadCloser, error) {
	srcFile, err := os.OpenFile(src, os.O_RDONLY, 0644)
	if err != nil {
		return nil, err
	}
	return srcFile, nil
}

// 这个是边读边写的实现
// func (this *File) Download(src string) (io.ReadCloser, error) {
// 	srcFile, err := os.OpenFile(src, os.O_RDONLY, 0644)
// 	if err != nil {
// 		return nil, err
// 	}
// 	pr, pw := io.Pipe()

// 	go func() {
// 		defer func() {
// 			srcFile.Close()
// 			pw.Close()
// 		}()
// 		for {
// 			buffer := make([]byte, FileBlockSize)
// 			n, err := srcFile.Read(buffer)
// 			if err != nil && err != io.EOF {
// 				pw.Write(buffer[:n])
// 				break
// 			}
// 			if n == 0 {
// 				pw.Write(buffer[:n])
// 				break
// 			} else {
// 				pw.Write(buffer[:n])
// 			}
// 		}
// 	}()
// 	return pr, nil
// }
