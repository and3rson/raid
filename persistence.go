package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
)

type Persistence[T interface{}] struct {
	path string
	Data T
}

func NewPersistence[T interface{}](data T, path string) (*Persistence[T], error) {
	p := &Persistence[T]{
		path,
		data,
	}
	if err := p.Load(); err != nil {
		return nil, err
	}

	return p, nil
}

func (p *Persistence[T]) open(flag int) (*os.File, error) {
	if err := os.MkdirAll(path.Dir(p.path), 0o755); err != nil {
		return nil, fmt.Errorf("persistence: create directories: %w", err)
	}

	f, err := os.OpenFile(p.path, flag, 0o644)
	if err != nil {
		return nil, fmt.Errorf("persistence: open database: %w", err)
	}

	return f, nil
}

func (p *Persistence[T]) Load() error {
	var (
		f   *os.File
		err error
	)

	if f, err = p.open(os.O_RDONLY | os.O_CREATE); err != nil {
		return fmt.Errorf("persistence: open database for read: %w", err)
	}

	dec := json.NewDecoder(f)
	if err = dec.Decode(&p.Data); err != nil {
		if err.Error() != "EOF" {
			return fmt.Errorf("persistence: decode database: %w", err)
		}
	}

	return nil
}

func (p *Persistence[T]) Save() error {
	var (
		f    *os.File
		err  error
		data []byte
	)

	if f, err = p.open(os.O_WRONLY | os.O_CREATE | os.O_TRUNC); err != nil {
		return fmt.Errorf("persistence: open database for write: %w", err)
	}

	if data, err = json.MarshalIndent(&p.Data, "", "    "); err != nil {
		return fmt.Errorf("persistence: encode database: %w", err)
	}

	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("persistence: write database: %w", err)
	}

	return nil
}
