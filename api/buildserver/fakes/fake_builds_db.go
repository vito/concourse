// This file was generated by counterfeiter
package fakes

import (
	"sync"

	"github.com/concourse/atc/api/buildserver"
	"github.com/concourse/atc/builds"
)

type FakeBuildsDB struct {
	CreateOneOffBuildStub        func() (builds.Build, error)
	createOneOffBuildMutex       sync.RWMutex
	createOneOffBuildArgsForCall []struct{}
	createOneOffBuildReturns struct {
		result1 builds.Build
		result2 error
	}
	AbortBuildStub        func(buildID int) (string, error)
	abortBuildMutex       sync.RWMutex
	abortBuildArgsForCall []struct {
		buildID int
	}
	abortBuildReturns struct {
		result1 string
		result2 error
	}
}

func (fake *FakeBuildsDB) CreateOneOffBuild() (builds.Build, error) {
	fake.createOneOffBuildMutex.Lock()
	fake.createOneOffBuildArgsForCall = append(fake.createOneOffBuildArgsForCall, struct{}{})
	fake.createOneOffBuildMutex.Unlock()
	if fake.CreateOneOffBuildStub != nil {
		return fake.CreateOneOffBuildStub()
	} else {
		return fake.createOneOffBuildReturns.result1, fake.createOneOffBuildReturns.result2
	}
}

func (fake *FakeBuildsDB) CreateOneOffBuildCallCount() int {
	fake.createOneOffBuildMutex.RLock()
	defer fake.createOneOffBuildMutex.RUnlock()
	return len(fake.createOneOffBuildArgsForCall)
}

func (fake *FakeBuildsDB) CreateOneOffBuildReturns(result1 builds.Build, result2 error) {
	fake.CreateOneOffBuildStub = nil
	fake.createOneOffBuildReturns = struct {
		result1 builds.Build
		result2 error
	}{result1, result2}
}

func (fake *FakeBuildsDB) AbortBuild(buildID int) (string, error) {
	fake.abortBuildMutex.Lock()
	fake.abortBuildArgsForCall = append(fake.abortBuildArgsForCall, struct {
		buildID int
	}{buildID})
	fake.abortBuildMutex.Unlock()
	if fake.AbortBuildStub != nil {
		return fake.AbortBuildStub(buildID)
	} else {
		return fake.abortBuildReturns.result1, fake.abortBuildReturns.result2
	}
}

func (fake *FakeBuildsDB) AbortBuildCallCount() int {
	fake.abortBuildMutex.RLock()
	defer fake.abortBuildMutex.RUnlock()
	return len(fake.abortBuildArgsForCall)
}

func (fake *FakeBuildsDB) AbortBuildArgsForCall(i int) int {
	fake.abortBuildMutex.RLock()
	defer fake.abortBuildMutex.RUnlock()
	return fake.abortBuildArgsForCall[i].buildID
}

func (fake *FakeBuildsDB) AbortBuildReturns(result1 string, result2 error) {
	fake.AbortBuildStub = nil
	fake.abortBuildReturns = struct {
		result1 string
		result2 error
	}{result1, result2}
}

var _ buildserver.BuildsDB = new(FakeBuildsDB)
