// Code generated by MockGen. DO NOT EDIT.
// Source: pkg/argocd/coderepo.go

// Package argocd is a generated GoMock package.
package argocd

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
)

// MockCoderepoOperation is a mock of CoderepoOperation interface.
type MockCoderepoOperation struct {
	ctrl     *gomock.Controller
	recorder *MockCoderepoOperationMockRecorder
}

// MockCoderepoOperationMockRecorder is the mock recorder for MockCoderepoOperation.
type MockCoderepoOperationMockRecorder struct {
	mock *MockCoderepoOperation
}

// NewMockCoderepoOperation creates a new mock instance.
func NewMockCoderepoOperation(ctrl *gomock.Controller) *MockCoderepoOperation {
	mock := &MockCoderepoOperation{ctrl: ctrl}
	mock.recorder = &MockCoderepoOperationMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockCoderepoOperation) EXPECT() *MockCoderepoOperationMockRecorder {
	return m.recorder
}

// CreateRepository mocks base method.
func (m *MockCoderepoOperation) CreateRepository(repoName, key, repourl string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateRepository", repoName, key, repourl)
	ret0, _ := ret[0].(error)
	return ret0
}

// CreateRepository indicates an expected call of CreateRepository.
func (mr *MockCoderepoOperationMockRecorder) CreateRepository(repoName, key, repourl interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateRepository", reflect.TypeOf((*MockCoderepoOperation)(nil).CreateRepository), repoName, key, repourl)
}

// DeleteRepository mocks base method.
func (m *MockCoderepoOperation) DeleteRepository(repoURL string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteRepository", repoURL)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteRepository indicates an expected call of DeleteRepository.
func (mr *MockCoderepoOperationMockRecorder) DeleteRepository(repoURL interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteRepository", reflect.TypeOf((*MockCoderepoOperation)(nil).DeleteRepository), repoURL)
}

// GetRepositoryInfo mocks base method.
func (m *MockCoderepoOperation) GetRepositoryInfo(repoURL string) (*CodeRepoResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRepositoryInfo", repoURL)
	ret0, _ := ret[0].(*CodeRepoResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetRepositoryInfo indicates an expected call of GetRepositoryInfo.
func (mr *MockCoderepoOperationMockRecorder) GetRepositoryInfo(repoURL interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRepositoryInfo", reflect.TypeOf((*MockCoderepoOperation)(nil).GetRepositoryInfo), repoURL)
}

// UpdateRepository mocks base method.
func (m *MockCoderepoOperation) UpdateRepository(repoName, key, repourl string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateRepository", repoName, key, repourl)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateRepository indicates an expected call of UpdateRepository.
func (mr *MockCoderepoOperationMockRecorder) UpdateRepository(repoName, key, repourl interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateRepository", reflect.TypeOf((*MockCoderepoOperation)(nil).UpdateRepository), repoName, key, repourl)
}
