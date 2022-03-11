package fakes

import (
	"sync"

	phpstart "github.com/paketo-buildpacks/php-start"
)

type ProcMgr struct {
	AddCall struct {
		mutex     sync.Mutex
		CallCount int
		Receives  struct {
			Name string
			Proc phpstart.Proc
		}
		Stub func(string, phpstart.Proc)
	}
	WriteFileCall struct {
		mutex     sync.Mutex
		CallCount int
		Receives  struct {
			Path string
		}
		Returns struct {
			Error error
		}
		Stub func(string) error
	}
}

func (f *ProcMgr) Add(param1 string, param2 phpstart.Proc) {
	f.AddCall.mutex.Lock()
	defer f.AddCall.mutex.Unlock()
	f.AddCall.CallCount++
	f.AddCall.Receives.Name = param1
	f.AddCall.Receives.Proc = param2
	if f.AddCall.Stub != nil {
		f.AddCall.Stub(param1, param2)
	}
}
func (f *ProcMgr) WriteFile(param1 string) error {
	f.WriteFileCall.mutex.Lock()
	defer f.WriteFileCall.mutex.Unlock()
	f.WriteFileCall.CallCount++
	f.WriteFileCall.Receives.Path = param1
	if f.WriteFileCall.Stub != nil {
		return f.WriteFileCall.Stub(param1)
	}
	return f.WriteFileCall.Returns.Error
}
