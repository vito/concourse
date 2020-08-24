// Code generated by counterfeiter. DO NOT EDIT.
package dbfakes

import (
	"sync"
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/concourse/concourse/atc"
	"github.com/concourse/concourse/atc/db"
	"github.com/concourse/concourse/atc/db/lock"
)

type FakeResourceConfigScope struct {
	AcquireResourceCheckingLockStub        func(lager.Logger) (lock.Lock, bool, error)
	acquireResourceCheckingLockMutex       sync.RWMutex
	acquireResourceCheckingLockArgsForCall []struct {
		arg1 lager.Logger
	}
	acquireResourceCheckingLockReturns struct {
		result1 lock.Lock
		result2 bool
		result3 error
	}
	acquireResourceCheckingLockReturnsOnCall map[int]struct {
		result1 lock.Lock
		result2 bool
		result3 error
	}
	CheckErrorStub        func() error
	checkErrorMutex       sync.RWMutex
	checkErrorArgsForCall []struct {
	}
	checkErrorReturns struct {
		result1 error
	}
	checkErrorReturnsOnCall map[int]struct {
		result1 error
	}
	FindVersionStub        func(atc.Version) (db.ResourceConfigVersion, bool, error)
	findVersionMutex       sync.RWMutex
	findVersionArgsForCall []struct {
		arg1 atc.Version
	}
	findVersionReturns struct {
		result1 db.ResourceConfigVersion
		result2 bool
		result3 error
	}
	findVersionReturnsOnCall map[int]struct {
		result1 db.ResourceConfigVersion
		result2 bool
		result3 error
	}
	IDStub        func() int
	iDMutex       sync.RWMutex
	iDArgsForCall []struct {
	}
	iDReturns struct {
		result1 int
	}
	iDReturnsOnCall map[int]struct {
		result1 int
	}
	LastCheckEndTimeStub        func() (time.Time, error)
	lastCheckEndTimeMutex       sync.RWMutex
	lastCheckEndTimeArgsForCall []struct {
	}
	lastCheckEndTimeReturns struct {
		result1 time.Time
		result2 error
	}
	lastCheckEndTimeReturnsOnCall map[int]struct {
		result1 time.Time
		result2 error
	}
	LatestVersionStub        func() (db.ResourceConfigVersion, bool, error)
	latestVersionMutex       sync.RWMutex
	latestVersionArgsForCall []struct {
	}
	latestVersionReturns struct {
		result1 db.ResourceConfigVersion
		result2 bool
		result3 error
	}
	latestVersionReturnsOnCall map[int]struct {
		result1 db.ResourceConfigVersion
		result2 bool
		result3 error
	}
	ResourceStub        func() db.Resource
	resourceMutex       sync.RWMutex
	resourceArgsForCall []struct {
	}
	resourceReturns struct {
		result1 db.Resource
	}
	resourceReturnsOnCall map[int]struct {
		result1 db.Resource
	}
	ResourceConfigStub        func() db.ResourceConfig
	resourceConfigMutex       sync.RWMutex
	resourceConfigArgsForCall []struct {
	}
	resourceConfigReturns struct {
		result1 db.ResourceConfig
	}
	resourceConfigReturnsOnCall map[int]struct {
		result1 db.ResourceConfig
	}
	SaveVersionsStub        func(db.SpanContext, []atc.Version) error
	saveVersionsMutex       sync.RWMutex
	saveVersionsArgsForCall []struct {
		arg1 db.SpanContext
		arg2 []atc.Version
	}
	saveVersionsReturns struct {
		result1 error
	}
	saveVersionsReturnsOnCall map[int]struct {
		result1 error
	}
	SetCheckErrorStub        func(error) error
	setCheckErrorMutex       sync.RWMutex
	setCheckErrorArgsForCall []struct {
		arg1 error
	}
	setCheckErrorReturns struct {
		result1 error
	}
	setCheckErrorReturnsOnCall map[int]struct {
		result1 error
	}
	UpdateLastCheckEndTimeStub        func() (bool, error)
	updateLastCheckEndTimeMutex       sync.RWMutex
	updateLastCheckEndTimeArgsForCall []struct {
	}
	updateLastCheckEndTimeReturns struct {
		result1 bool
		result2 error
	}
	updateLastCheckEndTimeReturnsOnCall map[int]struct {
		result1 bool
		result2 error
	}
	UpdateLastCheckStartTimeStub        func() (bool, error)
	updateLastCheckStartTimeMutex       sync.RWMutex
	updateLastCheckStartTimeArgsForCall []struct {
	}
	updateLastCheckStartTimeReturns struct {
		result1 bool
		result2 error
	}
	updateLastCheckStartTimeReturnsOnCall map[int]struct {
		result1 bool
		result2 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeResourceConfigScope) AcquireResourceCheckingLock(arg1 lager.Logger) (lock.Lock, bool, error) {
	fake.acquireResourceCheckingLockMutex.Lock()
	ret, specificReturn := fake.acquireResourceCheckingLockReturnsOnCall[len(fake.acquireResourceCheckingLockArgsForCall)]
	fake.acquireResourceCheckingLockArgsForCall = append(fake.acquireResourceCheckingLockArgsForCall, struct {
		arg1 lager.Logger
	}{arg1})
	fake.recordInvocation("AcquireResourceCheckingLock", []interface{}{arg1})
	fake.acquireResourceCheckingLockMutex.Unlock()
	if fake.AcquireResourceCheckingLockStub != nil {
		return fake.AcquireResourceCheckingLockStub(arg1)
	}
	if specificReturn {
		return ret.result1, ret.result2, ret.result3
	}
	fakeReturns := fake.acquireResourceCheckingLockReturns
	return fakeReturns.result1, fakeReturns.result2, fakeReturns.result3
}

func (fake *FakeResourceConfigScope) AcquireResourceCheckingLockCallCount() int {
	fake.acquireResourceCheckingLockMutex.RLock()
	defer fake.acquireResourceCheckingLockMutex.RUnlock()
	return len(fake.acquireResourceCheckingLockArgsForCall)
}

func (fake *FakeResourceConfigScope) AcquireResourceCheckingLockCalls(stub func(lager.Logger) (lock.Lock, bool, error)) {
	fake.acquireResourceCheckingLockMutex.Lock()
	defer fake.acquireResourceCheckingLockMutex.Unlock()
	fake.AcquireResourceCheckingLockStub = stub
}

func (fake *FakeResourceConfigScope) AcquireResourceCheckingLockArgsForCall(i int) lager.Logger {
	fake.acquireResourceCheckingLockMutex.RLock()
	defer fake.acquireResourceCheckingLockMutex.RUnlock()
	argsForCall := fake.acquireResourceCheckingLockArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeResourceConfigScope) AcquireResourceCheckingLockReturns(result1 lock.Lock, result2 bool, result3 error) {
	fake.acquireResourceCheckingLockMutex.Lock()
	defer fake.acquireResourceCheckingLockMutex.Unlock()
	fake.AcquireResourceCheckingLockStub = nil
	fake.acquireResourceCheckingLockReturns = struct {
		result1 lock.Lock
		result2 bool
		result3 error
	}{result1, result2, result3}
}

func (fake *FakeResourceConfigScope) AcquireResourceCheckingLockReturnsOnCall(i int, result1 lock.Lock, result2 bool, result3 error) {
	fake.acquireResourceCheckingLockMutex.Lock()
	defer fake.acquireResourceCheckingLockMutex.Unlock()
	fake.AcquireResourceCheckingLockStub = nil
	if fake.acquireResourceCheckingLockReturnsOnCall == nil {
		fake.acquireResourceCheckingLockReturnsOnCall = make(map[int]struct {
			result1 lock.Lock
			result2 bool
			result3 error
		})
	}
	fake.acquireResourceCheckingLockReturnsOnCall[i] = struct {
		result1 lock.Lock
		result2 bool
		result3 error
	}{result1, result2, result3}
}

func (fake *FakeResourceConfigScope) CheckError() error {
	fake.checkErrorMutex.Lock()
	ret, specificReturn := fake.checkErrorReturnsOnCall[len(fake.checkErrorArgsForCall)]
	fake.checkErrorArgsForCall = append(fake.checkErrorArgsForCall, struct {
	}{})
	fake.recordInvocation("CheckError", []interface{}{})
	fake.checkErrorMutex.Unlock()
	if fake.CheckErrorStub != nil {
		return fake.CheckErrorStub()
	}
	if specificReturn {
		return ret.result1
	}
	fakeReturns := fake.checkErrorReturns
	return fakeReturns.result1
}

func (fake *FakeResourceConfigScope) CheckErrorCallCount() int {
	fake.checkErrorMutex.RLock()
	defer fake.checkErrorMutex.RUnlock()
	return len(fake.checkErrorArgsForCall)
}

func (fake *FakeResourceConfigScope) CheckErrorCalls(stub func() error) {
	fake.checkErrorMutex.Lock()
	defer fake.checkErrorMutex.Unlock()
	fake.CheckErrorStub = stub
}

func (fake *FakeResourceConfigScope) CheckErrorReturns(result1 error) {
	fake.checkErrorMutex.Lock()
	defer fake.checkErrorMutex.Unlock()
	fake.CheckErrorStub = nil
	fake.checkErrorReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeResourceConfigScope) CheckErrorReturnsOnCall(i int, result1 error) {
	fake.checkErrorMutex.Lock()
	defer fake.checkErrorMutex.Unlock()
	fake.CheckErrorStub = nil
	if fake.checkErrorReturnsOnCall == nil {
		fake.checkErrorReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.checkErrorReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeResourceConfigScope) FindVersion(arg1 atc.Version) (db.ResourceConfigVersion, bool, error) {
	fake.findVersionMutex.Lock()
	ret, specificReturn := fake.findVersionReturnsOnCall[len(fake.findVersionArgsForCall)]
	fake.findVersionArgsForCall = append(fake.findVersionArgsForCall, struct {
		arg1 atc.Version
	}{arg1})
	fake.recordInvocation("FindVersion", []interface{}{arg1})
	fake.findVersionMutex.Unlock()
	if fake.FindVersionStub != nil {
		return fake.FindVersionStub(arg1)
	}
	if specificReturn {
		return ret.result1, ret.result2, ret.result3
	}
	fakeReturns := fake.findVersionReturns
	return fakeReturns.result1, fakeReturns.result2, fakeReturns.result3
}

func (fake *FakeResourceConfigScope) FindVersionCallCount() int {
	fake.findVersionMutex.RLock()
	defer fake.findVersionMutex.RUnlock()
	return len(fake.findVersionArgsForCall)
}

func (fake *FakeResourceConfigScope) FindVersionCalls(stub func(atc.Version) (db.ResourceConfigVersion, bool, error)) {
	fake.findVersionMutex.Lock()
	defer fake.findVersionMutex.Unlock()
	fake.FindVersionStub = stub
}

func (fake *FakeResourceConfigScope) FindVersionArgsForCall(i int) atc.Version {
	fake.findVersionMutex.RLock()
	defer fake.findVersionMutex.RUnlock()
	argsForCall := fake.findVersionArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeResourceConfigScope) FindVersionReturns(result1 db.ResourceConfigVersion, result2 bool, result3 error) {
	fake.findVersionMutex.Lock()
	defer fake.findVersionMutex.Unlock()
	fake.FindVersionStub = nil
	fake.findVersionReturns = struct {
		result1 db.ResourceConfigVersion
		result2 bool
		result3 error
	}{result1, result2, result3}
}

func (fake *FakeResourceConfigScope) FindVersionReturnsOnCall(i int, result1 db.ResourceConfigVersion, result2 bool, result3 error) {
	fake.findVersionMutex.Lock()
	defer fake.findVersionMutex.Unlock()
	fake.FindVersionStub = nil
	if fake.findVersionReturnsOnCall == nil {
		fake.findVersionReturnsOnCall = make(map[int]struct {
			result1 db.ResourceConfigVersion
			result2 bool
			result3 error
		})
	}
	fake.findVersionReturnsOnCall[i] = struct {
		result1 db.ResourceConfigVersion
		result2 bool
		result3 error
	}{result1, result2, result3}
}

func (fake *FakeResourceConfigScope) ID() int {
	fake.iDMutex.Lock()
	ret, specificReturn := fake.iDReturnsOnCall[len(fake.iDArgsForCall)]
	fake.iDArgsForCall = append(fake.iDArgsForCall, struct {
	}{})
	fake.recordInvocation("ID", []interface{}{})
	fake.iDMutex.Unlock()
	if fake.IDStub != nil {
		return fake.IDStub()
	}
	if specificReturn {
		return ret.result1
	}
	fakeReturns := fake.iDReturns
	return fakeReturns.result1
}

func (fake *FakeResourceConfigScope) IDCallCount() int {
	fake.iDMutex.RLock()
	defer fake.iDMutex.RUnlock()
	return len(fake.iDArgsForCall)
}

func (fake *FakeResourceConfigScope) IDCalls(stub func() int) {
	fake.iDMutex.Lock()
	defer fake.iDMutex.Unlock()
	fake.IDStub = stub
}

func (fake *FakeResourceConfigScope) IDReturns(result1 int) {
	fake.iDMutex.Lock()
	defer fake.iDMutex.Unlock()
	fake.IDStub = nil
	fake.iDReturns = struct {
		result1 int
	}{result1}
}

func (fake *FakeResourceConfigScope) IDReturnsOnCall(i int, result1 int) {
	fake.iDMutex.Lock()
	defer fake.iDMutex.Unlock()
	fake.IDStub = nil
	if fake.iDReturnsOnCall == nil {
		fake.iDReturnsOnCall = make(map[int]struct {
			result1 int
		})
	}
	fake.iDReturnsOnCall[i] = struct {
		result1 int
	}{result1}
}

func (fake *FakeResourceConfigScope) LastCheckEndTime() (time.Time, error) {
	fake.lastCheckEndTimeMutex.Lock()
	ret, specificReturn := fake.lastCheckEndTimeReturnsOnCall[len(fake.lastCheckEndTimeArgsForCall)]
	fake.lastCheckEndTimeArgsForCall = append(fake.lastCheckEndTimeArgsForCall, struct {
	}{})
	fake.recordInvocation("LastCheckEndTime", []interface{}{})
	fake.lastCheckEndTimeMutex.Unlock()
	if fake.LastCheckEndTimeStub != nil {
		return fake.LastCheckEndTimeStub()
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	fakeReturns := fake.lastCheckEndTimeReturns
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeResourceConfigScope) LastCheckEndTimeCallCount() int {
	fake.lastCheckEndTimeMutex.RLock()
	defer fake.lastCheckEndTimeMutex.RUnlock()
	return len(fake.lastCheckEndTimeArgsForCall)
}

func (fake *FakeResourceConfigScope) LastCheckEndTimeCalls(stub func() (time.Time, error)) {
	fake.lastCheckEndTimeMutex.Lock()
	defer fake.lastCheckEndTimeMutex.Unlock()
	fake.LastCheckEndTimeStub = stub
}

func (fake *FakeResourceConfigScope) LastCheckEndTimeReturns(result1 time.Time, result2 error) {
	fake.lastCheckEndTimeMutex.Lock()
	defer fake.lastCheckEndTimeMutex.Unlock()
	fake.LastCheckEndTimeStub = nil
	fake.lastCheckEndTimeReturns = struct {
		result1 time.Time
		result2 error
	}{result1, result2}
}

func (fake *FakeResourceConfigScope) LastCheckEndTimeReturnsOnCall(i int, result1 time.Time, result2 error) {
	fake.lastCheckEndTimeMutex.Lock()
	defer fake.lastCheckEndTimeMutex.Unlock()
	fake.LastCheckEndTimeStub = nil
	if fake.lastCheckEndTimeReturnsOnCall == nil {
		fake.lastCheckEndTimeReturnsOnCall = make(map[int]struct {
			result1 time.Time
			result2 error
		})
	}
	fake.lastCheckEndTimeReturnsOnCall[i] = struct {
		result1 time.Time
		result2 error
	}{result1, result2}
}

func (fake *FakeResourceConfigScope) LatestVersion() (db.ResourceConfigVersion, bool, error) {
	fake.latestVersionMutex.Lock()
	ret, specificReturn := fake.latestVersionReturnsOnCall[len(fake.latestVersionArgsForCall)]
	fake.latestVersionArgsForCall = append(fake.latestVersionArgsForCall, struct {
	}{})
	fake.recordInvocation("LatestVersion", []interface{}{})
	fake.latestVersionMutex.Unlock()
	if fake.LatestVersionStub != nil {
		return fake.LatestVersionStub()
	}
	if specificReturn {
		return ret.result1, ret.result2, ret.result3
	}
	fakeReturns := fake.latestVersionReturns
	return fakeReturns.result1, fakeReturns.result2, fakeReturns.result3
}

func (fake *FakeResourceConfigScope) LatestVersionCallCount() int {
	fake.latestVersionMutex.RLock()
	defer fake.latestVersionMutex.RUnlock()
	return len(fake.latestVersionArgsForCall)
}

func (fake *FakeResourceConfigScope) LatestVersionCalls(stub func() (db.ResourceConfigVersion, bool, error)) {
	fake.latestVersionMutex.Lock()
	defer fake.latestVersionMutex.Unlock()
	fake.LatestVersionStub = stub
}

func (fake *FakeResourceConfigScope) LatestVersionReturns(result1 db.ResourceConfigVersion, result2 bool, result3 error) {
	fake.latestVersionMutex.Lock()
	defer fake.latestVersionMutex.Unlock()
	fake.LatestVersionStub = nil
	fake.latestVersionReturns = struct {
		result1 db.ResourceConfigVersion
		result2 bool
		result3 error
	}{result1, result2, result3}
}

func (fake *FakeResourceConfigScope) LatestVersionReturnsOnCall(i int, result1 db.ResourceConfigVersion, result2 bool, result3 error) {
	fake.latestVersionMutex.Lock()
	defer fake.latestVersionMutex.Unlock()
	fake.LatestVersionStub = nil
	if fake.latestVersionReturnsOnCall == nil {
		fake.latestVersionReturnsOnCall = make(map[int]struct {
			result1 db.ResourceConfigVersion
			result2 bool
			result3 error
		})
	}
	fake.latestVersionReturnsOnCall[i] = struct {
		result1 db.ResourceConfigVersion
		result2 bool
		result3 error
	}{result1, result2, result3}
}

func (fake *FakeResourceConfigScope) Resource() db.Resource {
	fake.resourceMutex.Lock()
	ret, specificReturn := fake.resourceReturnsOnCall[len(fake.resourceArgsForCall)]
	fake.resourceArgsForCall = append(fake.resourceArgsForCall, struct {
	}{})
	fake.recordInvocation("Resource", []interface{}{})
	fake.resourceMutex.Unlock()
	if fake.ResourceStub != nil {
		return fake.ResourceStub()
	}
	if specificReturn {
		return ret.result1
	}
	fakeReturns := fake.resourceReturns
	return fakeReturns.result1
}

func (fake *FakeResourceConfigScope) ResourceCallCount() int {
	fake.resourceMutex.RLock()
	defer fake.resourceMutex.RUnlock()
	return len(fake.resourceArgsForCall)
}

func (fake *FakeResourceConfigScope) ResourceCalls(stub func() db.Resource) {
	fake.resourceMutex.Lock()
	defer fake.resourceMutex.Unlock()
	fake.ResourceStub = stub
}

func (fake *FakeResourceConfigScope) ResourceReturns(result1 db.Resource) {
	fake.resourceMutex.Lock()
	defer fake.resourceMutex.Unlock()
	fake.ResourceStub = nil
	fake.resourceReturns = struct {
		result1 db.Resource
	}{result1}
}

func (fake *FakeResourceConfigScope) ResourceReturnsOnCall(i int, result1 db.Resource) {
	fake.resourceMutex.Lock()
	defer fake.resourceMutex.Unlock()
	fake.ResourceStub = nil
	if fake.resourceReturnsOnCall == nil {
		fake.resourceReturnsOnCall = make(map[int]struct {
			result1 db.Resource
		})
	}
	fake.resourceReturnsOnCall[i] = struct {
		result1 db.Resource
	}{result1}
}

func (fake *FakeResourceConfigScope) ResourceConfig() db.ResourceConfig {
	fake.resourceConfigMutex.Lock()
	ret, specificReturn := fake.resourceConfigReturnsOnCall[len(fake.resourceConfigArgsForCall)]
	fake.resourceConfigArgsForCall = append(fake.resourceConfigArgsForCall, struct {
	}{})
	fake.recordInvocation("ResourceConfig", []interface{}{})
	fake.resourceConfigMutex.Unlock()
	if fake.ResourceConfigStub != nil {
		return fake.ResourceConfigStub()
	}
	if specificReturn {
		return ret.result1
	}
	fakeReturns := fake.resourceConfigReturns
	return fakeReturns.result1
}

func (fake *FakeResourceConfigScope) ResourceConfigCallCount() int {
	fake.resourceConfigMutex.RLock()
	defer fake.resourceConfigMutex.RUnlock()
	return len(fake.resourceConfigArgsForCall)
}

func (fake *FakeResourceConfigScope) ResourceConfigCalls(stub func() db.ResourceConfig) {
	fake.resourceConfigMutex.Lock()
	defer fake.resourceConfigMutex.Unlock()
	fake.ResourceConfigStub = stub
}

func (fake *FakeResourceConfigScope) ResourceConfigReturns(result1 db.ResourceConfig) {
	fake.resourceConfigMutex.Lock()
	defer fake.resourceConfigMutex.Unlock()
	fake.ResourceConfigStub = nil
	fake.resourceConfigReturns = struct {
		result1 db.ResourceConfig
	}{result1}
}

func (fake *FakeResourceConfigScope) ResourceConfigReturnsOnCall(i int, result1 db.ResourceConfig) {
	fake.resourceConfigMutex.Lock()
	defer fake.resourceConfigMutex.Unlock()
	fake.ResourceConfigStub = nil
	if fake.resourceConfigReturnsOnCall == nil {
		fake.resourceConfigReturnsOnCall = make(map[int]struct {
			result1 db.ResourceConfig
		})
	}
	fake.resourceConfigReturnsOnCall[i] = struct {
		result1 db.ResourceConfig
	}{result1}
}

func (fake *FakeResourceConfigScope) SaveVersions(arg1 db.SpanContext, arg2 []atc.Version) error {
	var arg2Copy []atc.Version
	if arg2 != nil {
		arg2Copy = make([]atc.Version, len(arg2))
		copy(arg2Copy, arg2)
	}
	fake.saveVersionsMutex.Lock()
	ret, specificReturn := fake.saveVersionsReturnsOnCall[len(fake.saveVersionsArgsForCall)]
	fake.saveVersionsArgsForCall = append(fake.saveVersionsArgsForCall, struct {
		arg1 db.SpanContext
		arg2 []atc.Version
	}{arg1, arg2Copy})
	fake.recordInvocation("SaveVersions", []interface{}{arg1, arg2Copy})
	fake.saveVersionsMutex.Unlock()
	if fake.SaveVersionsStub != nil {
		return fake.SaveVersionsStub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1
	}
	fakeReturns := fake.saveVersionsReturns
	return fakeReturns.result1
}

func (fake *FakeResourceConfigScope) SaveVersionsCallCount() int {
	fake.saveVersionsMutex.RLock()
	defer fake.saveVersionsMutex.RUnlock()
	return len(fake.saveVersionsArgsForCall)
}

func (fake *FakeResourceConfigScope) SaveVersionsCalls(stub func(db.SpanContext, []atc.Version) error) {
	fake.saveVersionsMutex.Lock()
	defer fake.saveVersionsMutex.Unlock()
	fake.SaveVersionsStub = stub
}

func (fake *FakeResourceConfigScope) SaveVersionsArgsForCall(i int) (db.SpanContext, []atc.Version) {
	fake.saveVersionsMutex.RLock()
	defer fake.saveVersionsMutex.RUnlock()
	argsForCall := fake.saveVersionsArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeResourceConfigScope) SaveVersionsReturns(result1 error) {
	fake.saveVersionsMutex.Lock()
	defer fake.saveVersionsMutex.Unlock()
	fake.SaveVersionsStub = nil
	fake.saveVersionsReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeResourceConfigScope) SaveVersionsReturnsOnCall(i int, result1 error) {
	fake.saveVersionsMutex.Lock()
	defer fake.saveVersionsMutex.Unlock()
	fake.SaveVersionsStub = nil
	if fake.saveVersionsReturnsOnCall == nil {
		fake.saveVersionsReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.saveVersionsReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeResourceConfigScope) SetCheckError(arg1 error) error {
	fake.setCheckErrorMutex.Lock()
	ret, specificReturn := fake.setCheckErrorReturnsOnCall[len(fake.setCheckErrorArgsForCall)]
	fake.setCheckErrorArgsForCall = append(fake.setCheckErrorArgsForCall, struct {
		arg1 error
	}{arg1})
	fake.recordInvocation("SetCheckError", []interface{}{arg1})
	fake.setCheckErrorMutex.Unlock()
	if fake.SetCheckErrorStub != nil {
		return fake.SetCheckErrorStub(arg1)
	}
	if specificReturn {
		return ret.result1
	}
	fakeReturns := fake.setCheckErrorReturns
	return fakeReturns.result1
}

func (fake *FakeResourceConfigScope) SetCheckErrorCallCount() int {
	fake.setCheckErrorMutex.RLock()
	defer fake.setCheckErrorMutex.RUnlock()
	return len(fake.setCheckErrorArgsForCall)
}

func (fake *FakeResourceConfigScope) SetCheckErrorCalls(stub func(error) error) {
	fake.setCheckErrorMutex.Lock()
	defer fake.setCheckErrorMutex.Unlock()
	fake.SetCheckErrorStub = stub
}

func (fake *FakeResourceConfigScope) SetCheckErrorArgsForCall(i int) error {
	fake.setCheckErrorMutex.RLock()
	defer fake.setCheckErrorMutex.RUnlock()
	argsForCall := fake.setCheckErrorArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeResourceConfigScope) SetCheckErrorReturns(result1 error) {
	fake.setCheckErrorMutex.Lock()
	defer fake.setCheckErrorMutex.Unlock()
	fake.SetCheckErrorStub = nil
	fake.setCheckErrorReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeResourceConfigScope) SetCheckErrorReturnsOnCall(i int, result1 error) {
	fake.setCheckErrorMutex.Lock()
	defer fake.setCheckErrorMutex.Unlock()
	fake.SetCheckErrorStub = nil
	if fake.setCheckErrorReturnsOnCall == nil {
		fake.setCheckErrorReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.setCheckErrorReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeResourceConfigScope) UpdateLastCheckEndTime() (bool, error) {
	fake.updateLastCheckEndTimeMutex.Lock()
	ret, specificReturn := fake.updateLastCheckEndTimeReturnsOnCall[len(fake.updateLastCheckEndTimeArgsForCall)]
	fake.updateLastCheckEndTimeArgsForCall = append(fake.updateLastCheckEndTimeArgsForCall, struct {
	}{})
	fake.recordInvocation("UpdateLastCheckEndTime", []interface{}{})
	fake.updateLastCheckEndTimeMutex.Unlock()
	if fake.UpdateLastCheckEndTimeStub != nil {
		return fake.UpdateLastCheckEndTimeStub()
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	fakeReturns := fake.updateLastCheckEndTimeReturns
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeResourceConfigScope) UpdateLastCheckEndTimeCallCount() int {
	fake.updateLastCheckEndTimeMutex.RLock()
	defer fake.updateLastCheckEndTimeMutex.RUnlock()
	return len(fake.updateLastCheckEndTimeArgsForCall)
}

func (fake *FakeResourceConfigScope) UpdateLastCheckEndTimeCalls(stub func() (bool, error)) {
	fake.updateLastCheckEndTimeMutex.Lock()
	defer fake.updateLastCheckEndTimeMutex.Unlock()
	fake.UpdateLastCheckEndTimeStub = stub
}

func (fake *FakeResourceConfigScope) UpdateLastCheckEndTimeReturns(result1 bool, result2 error) {
	fake.updateLastCheckEndTimeMutex.Lock()
	defer fake.updateLastCheckEndTimeMutex.Unlock()
	fake.UpdateLastCheckEndTimeStub = nil
	fake.updateLastCheckEndTimeReturns = struct {
		result1 bool
		result2 error
	}{result1, result2}
}

func (fake *FakeResourceConfigScope) UpdateLastCheckEndTimeReturnsOnCall(i int, result1 bool, result2 error) {
	fake.updateLastCheckEndTimeMutex.Lock()
	defer fake.updateLastCheckEndTimeMutex.Unlock()
	fake.UpdateLastCheckEndTimeStub = nil
	if fake.updateLastCheckEndTimeReturnsOnCall == nil {
		fake.updateLastCheckEndTimeReturnsOnCall = make(map[int]struct {
			result1 bool
			result2 error
		})
	}
	fake.updateLastCheckEndTimeReturnsOnCall[i] = struct {
		result1 bool
		result2 error
	}{result1, result2}
}

func (fake *FakeResourceConfigScope) UpdateLastCheckStartTime() (bool, error) {
	fake.updateLastCheckStartTimeMutex.Lock()
	ret, specificReturn := fake.updateLastCheckStartTimeReturnsOnCall[len(fake.updateLastCheckStartTimeArgsForCall)]
	fake.updateLastCheckStartTimeArgsForCall = append(fake.updateLastCheckStartTimeArgsForCall, struct {
	}{})
	fake.recordInvocation("UpdateLastCheckStartTime", []interface{}{})
	fake.updateLastCheckStartTimeMutex.Unlock()
	if fake.UpdateLastCheckStartTimeStub != nil {
		return fake.UpdateLastCheckStartTimeStub()
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	fakeReturns := fake.updateLastCheckStartTimeReturns
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeResourceConfigScope) UpdateLastCheckStartTimeCallCount() int {
	fake.updateLastCheckStartTimeMutex.RLock()
	defer fake.updateLastCheckStartTimeMutex.RUnlock()
	return len(fake.updateLastCheckStartTimeArgsForCall)
}

func (fake *FakeResourceConfigScope) UpdateLastCheckStartTimeCalls(stub func() (bool, error)) {
	fake.updateLastCheckStartTimeMutex.Lock()
	defer fake.updateLastCheckStartTimeMutex.Unlock()
	fake.UpdateLastCheckStartTimeStub = stub
}

func (fake *FakeResourceConfigScope) UpdateLastCheckStartTimeReturns(result1 bool, result2 error) {
	fake.updateLastCheckStartTimeMutex.Lock()
	defer fake.updateLastCheckStartTimeMutex.Unlock()
	fake.UpdateLastCheckStartTimeStub = nil
	fake.updateLastCheckStartTimeReturns = struct {
		result1 bool
		result2 error
	}{result1, result2}
}

func (fake *FakeResourceConfigScope) UpdateLastCheckStartTimeReturnsOnCall(i int, result1 bool, result2 error) {
	fake.updateLastCheckStartTimeMutex.Lock()
	defer fake.updateLastCheckStartTimeMutex.Unlock()
	fake.UpdateLastCheckStartTimeStub = nil
	if fake.updateLastCheckStartTimeReturnsOnCall == nil {
		fake.updateLastCheckStartTimeReturnsOnCall = make(map[int]struct {
			result1 bool
			result2 error
		})
	}
	fake.updateLastCheckStartTimeReturnsOnCall[i] = struct {
		result1 bool
		result2 error
	}{result1, result2}
}

func (fake *FakeResourceConfigScope) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.acquireResourceCheckingLockMutex.RLock()
	defer fake.acquireResourceCheckingLockMutex.RUnlock()
	fake.checkErrorMutex.RLock()
	defer fake.checkErrorMutex.RUnlock()
	fake.findVersionMutex.RLock()
	defer fake.findVersionMutex.RUnlock()
	fake.iDMutex.RLock()
	defer fake.iDMutex.RUnlock()
	fake.lastCheckEndTimeMutex.RLock()
	defer fake.lastCheckEndTimeMutex.RUnlock()
	fake.latestVersionMutex.RLock()
	defer fake.latestVersionMutex.RUnlock()
	fake.resourceMutex.RLock()
	defer fake.resourceMutex.RUnlock()
	fake.resourceConfigMutex.RLock()
	defer fake.resourceConfigMutex.RUnlock()
	fake.saveVersionsMutex.RLock()
	defer fake.saveVersionsMutex.RUnlock()
	fake.setCheckErrorMutex.RLock()
	defer fake.setCheckErrorMutex.RUnlock()
	fake.updateLastCheckEndTimeMutex.RLock()
	defer fake.updateLastCheckEndTimeMutex.RUnlock()
	fake.updateLastCheckStartTimeMutex.RLock()
	defer fake.updateLastCheckStartTimeMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeResourceConfigScope) recordInvocation(key string, args []interface{}) {
	fake.invocationsMutex.Lock()
	defer fake.invocationsMutex.Unlock()
	if fake.invocations == nil {
		fake.invocations = map[string][][]interface{}{}
	}
	if fake.invocations[key] == nil {
		fake.invocations[key] = [][]interface{}{}
	}
	fake.invocations[key] = append(fake.invocations[key], args)
}

var _ db.ResourceConfigScope = new(FakeResourceConfigScope)
