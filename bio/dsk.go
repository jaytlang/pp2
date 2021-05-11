package bio

import (
	"errors"
	"time"
)

type Disk interface {
	Get(key string) (string, error)
	Put(key string, value string) error
	Acquire(lockk string)
	Release(lockk string) error
	Renew(lockk string) error
}

// Fake disk

type MockDisk struct {
	kv map[string]string
}

func (m *MockDisk) Get(key string) (string, error) {
	if m.kv["lock_"+key] != "1" {
		return "", errors.New("lock not held")
	}
	return m.kv[key], nil
}

func (m *MockDisk) Put(key string, value string) error {
	if m.kv["lock_"+key] != "1" {
		return errors.New("lock not held")
	}
	m.kv[key] = value
	return nil
}

func (m *MockDisk) Acquire(lockk string) {
	for m.kv["lock_"+lockk] == "1" {
		// This disk is single threaded, if this
		// happens we have a serious problem
		time.Sleep(500 * time.Millisecond)
	}
	m.kv["lock_"+lockk] = "1"
}

func (m *MockDisk) Release(lockk string) error {
	if m.kv["lock_"+lockk] == "0" {
		return errors.New("lock not held")
	}
	m.kv["lock_"+lockk] = "0"
	return nil
}

func (m *MockDisk) Renew(lockk string) error {
	if m.kv["lock_"+lockk] == "0" {
		// This disk is single threaded, if this
		// happens we have a serious problem
		return errors.New("lock not held")
	}
	return nil
}
