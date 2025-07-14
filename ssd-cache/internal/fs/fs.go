package fs

type File interface {
	Pwrite(buf []byte) (currentPhysicalOffset int64, err error)
	Pread(fileOffset int64, buf []byte) (n int32, err error)
	TrimHead() (err error)
	Close()
}

type Page interface {
	Unmap() error
}
