//go:build idena_memory_ipfs

package ipfs

import (
	"bytes"
	"context"
	"io"
	"os"
	"sync"

	"github.com/idena-network/idena-go/common"
	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multihash"
	"github.com/pkg/errors"
)

const CidLength = 36

type DataType = uint32

const (
	Block      DataType = 1
	Flip       DataType = 2
	Profile    DataType = 3
	TxReceipt  DataType = 4
	CustomData DataType = 5
)

var (
	EmptyCid  cid.Cid
	MinCid    [CidLength]byte
	MaxCid    [CidLength]byte
	TooBigErr = errors.New("ipfs data is too big")
)

func init() {
	empty, _ := cid.Decode("bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku")
	EmptyCid = empty

	for i := range MaxCid {
		MaxCid[i] = 0xFF
	}
}

type Proxy interface {
	Add(data []byte, pin bool) (cid.Cid, error)
	Get(key []byte, dataType DataType) ([]byte, error)
	LoadTo(key []byte, to io.Writer, ctx context.Context, onLoading func(size, loaded int64)) error
	Pin(key []byte) error
	Unpin(key []byte) error
	Cid(data []byte) (cid.Cid, error)
	Port() int
	PeerId() string
	AddFile(absPath string, data io.ReadCloser, fi os.FileInfo) (cid.Cid, error)
	ShouldPin(dataType DataType) bool
	GetWithSizeLimit(key []byte, dataType DataType, size int64) ([]byte, error)
	GC() (ctx context.Context, cancel context.CancelFunc)
}

func NewMemoryIpfsProxy() Proxy {
	return &memoryIpfs{
		values: make(map[cid.Cid][]byte),
	}
}

type memoryIpfs struct {
	mu     sync.RWMutex
	values map[cid.Cid][]byte
}

func (i *memoryIpfs) ShouldPin(dataType DataType) bool {
	return true
}

func (i *memoryIpfs) LoadTo(key []byte, to io.Writer, ctx context.Context, onLoading func(size, loaded int64)) error {
	data, err := i.Get(key, Block)
	if err != nil {
		return err
	}
	if onLoading != nil {
		onLoading(int64(len(data)), 0)
	}
	reader := bytes.NewReader(data)
	buf := make([]byte, 32*1024)
	var loaded int64
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		n, readErr := reader.Read(buf)
		if n > 0 {
			written, writeErr := to.Write(buf[:n])
			loaded += int64(written)
			if onLoading != nil {
				onLoading(int64(len(data)), loaded)
			}
			if writeErr != nil {
				return writeErr
			}
			if written != n {
				return io.ErrShortWrite
			}
		}
		if readErr == io.EOF {
			return nil
		}
		if readErr != nil {
			return readErr
		}
	}
}

func (i *memoryIpfs) AddFile(absPath string, data io.ReadCloser, fi os.FileInfo) (cid.Cid, error) {
	defer data.Close()
	content, err := io.ReadAll(data)
	if err != nil {
		return cid.Cid{}, err
	}
	return i.Add(content, true)
}

func (i *memoryIpfs) Unpin(key []byte) error {
	return nil
}

func (i *memoryIpfs) Add(data []byte, pin bool) (cid.Cid, error) {
	c, err := i.Cid(data)
	if err != nil {
		return cid.Cid{}, err
	}
	if c == EmptyCid {
		return c, nil
	}
	copied := append([]byte(nil), data...)
	i.mu.Lock()
	i.values[c] = copied
	i.mu.Unlock()
	return c, nil
}

func (i *memoryIpfs) Get(key []byte, dataType DataType) ([]byte, error) {
	return i.GetWithSizeLimit(key, dataType, 0)
}

func (i *memoryIpfs) GetWithSizeLimit(key []byte, dataType DataType, size int64) ([]byte, error) {
	if len(key) == 0 {
		return []byte{}, nil
	}
	c, err := cid.Cast(key)
	if err != nil {
		return nil, err
	}
	if c == EmptyCid {
		return []byte{}, nil
	}

	i.mu.RLock()
	value, ok := i.values[c]
	i.mu.RUnlock()
	if !ok {
		return nil, errors.New("not found")
	}

	maxSize := memoryMaxSize(dataType)
	if size > 0 && (maxSize <= 0 || size < maxSize) {
		maxSize = size
	}
	if maxSize > 0 && int64(len(value)) > maxSize {
		return nil, TooBigErr
	}

	return append([]byte(nil), value...), nil
}

func (*memoryIpfs) Pin(key []byte) error {
	return nil
}

func (*memoryIpfs) PeerId() string {
	return ""
}

func (*memoryIpfs) Port() int {
	return 0
}

func (*memoryIpfs) Cid(data []byte) (cid.Cid, error) {
	if len(data) == 0 {
		return EmptyCid, nil
	}
	prefix := cid.Prefix{
		Codec:    cid.Raw,
		MhLength: -1,
		MhType:   multihash.SHA2_256,
		Version:  1,
	}
	return prefix.Sum(data)
}

func (*memoryIpfs) GC() (context.Context, context.CancelFunc) {
	return context.WithCancel(context.Background())
}

func memoryMaxSize(dataType DataType) int64 {
	switch dataType {
	case Flip:
		return common.MaxFlipSize
	case Profile:
		return common.MaxProfileSize
	case CustomData:
		return common.MaxCustomDataSize
	default:
		return -1
	}
}
