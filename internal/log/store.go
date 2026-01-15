package log

import (
	"bufio"
	"encoding/binary"
	"os"
	"sync"
)

var enc = binary.BigEndian

const lenWidth = 8

type store struct {
	*os.File
	mu   sync.Mutex
	buf  *bufio.Writer
	size uint64
}

func newStore(f *os.File) (*store, error) {
	fi, err := os.Stat(f.Name())
	if err != nil {
		return nil, err
	}
	size := uint64(fi.Size())
	return &store{
		File: f,
		buf:  bufio.NewWriter(f),
		size: size,
	}, nil
}

func (s *store) Append(p []byte) (n uint64, pos uint64, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	pos = s.size
	// first the length of the actual data is written
	if err = binary.Write(s.buf, enc, uint64(len(p))); err != nil {
		return 0, 0, err
	}
	// now the actual data is appended to the length written before
	w, err := s.buf.Write(p)
	if err != nil {
		return 0, 0, err
	}
	w += lenWidth
	s.size += uint64(w)
	return uint64(w), pos, nil
}

func (s *store) Read(pos uint64) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.buf.Flush(); err != nil {
		return nil, err
	}
	// the length of the data to be read will be
	//  written at the end of the byte
	size := make([]byte, lenWidth)
	if _, err := s.File.ReadAt(size, int64(pos)); err != nil {
		return nil, err
	}
	// creating a byte with length of the data
	b := make([]byte, enc.Uint64(size))
	// the actual data starts after the
	// pos data is finished ie, at (pos + 8 bits)
	if _, err := s.File.ReadAt(b, int64(pos+lenWidth)); err != nil {
		return nil, err
	}
	return b, nil
}

func (s *store) ReadAt(p []byte, offset int64) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.buf.Flush(); err != nil {
		return 0, nil
	}
	return s.File.ReadAt(p, offset)
}

func (s *store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.buf.Flush(); err != nil {
		return err
	}
	return s.File.Close()
}
