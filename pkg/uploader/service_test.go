package uploader

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/build-tanker/shipper/pkg/appcontext"
	"github.com/build-tanker/shipper/pkg/config"
	"github.com/build-tanker/shipper/pkg/logger"
	"github.com/stretchr/testify/assert"
)

var testContext *appcontext.AppContext
var testBuffer string
var b bytes.Buffer

func NewTestContext() *appcontext.AppContext {
	if testContext == nil {
		conf := config.NewConfig([]string{"$HOME"})
		log := logger.NewLogger(conf, &b)
		testContext = appcontext.NewAppContext(conf, log)
	}
	return testContext
}

type MockClient struct {
	TestState string
}

func (m *MockClient) ChangeState(newState string) {
	m.TestState = newState
}

func (m MockClient) GetAccessKey(server string) (string, error) {
	if m.TestState == "AccessKeyOK" {
		return "", nil
	}

	if m.TestState == "AccessKeyFailure" {
		return "", errors.New("AccessKeyFailure")
	}

	return "", nil
}

func (m MockClient) DeleteAccessKey(server, accessKey string) error {
	return nil
}

func (m MockClient) GetUploadURL(server, accessKey, bundle string) (string, error) {
	return "", nil
}

func (m MockClient) UploadFile(string, string) error {
	return nil
}

func (m MockClient) ConfirmFileUpload(string, string) error {
	return nil
}

type MockFileSystem struct {
	TestState string
	TestLog   string
}

func (m MockFileSystem) ReadCompleteFileFromDisk(path string) ([]byte, error) {
	return []byte(""), nil
}

func (m MockFileSystem) WriteCompleteFileToDisk(path string, data []byte, permissions os.FileMode) error {
	testBuffer = fmt.Sprintln("path", path, "data", string(data))
	return nil
}

func (m MockFileSystem) DeleteFileFromDisk(path string) error {
	testBuffer = fmt.Sprintln("delete", path)
	return nil
}

func (m *MockFileSystem) GetTestLog() string {
	return m.TestLog
}

func NewTestService() *service {
	ctx := NewTestContext()
	client := &MockClient{}
	fs := &MockFileSystem{}
	return &service{
		ctx:    ctx,
		client: client,
		fs:     fs,
	}
}

func TestServiceInstall(t *testing.T) {
	s := NewTestService()

	s.ctx.GetConfig().AccessKey = "testAccessKey"
	s.ctx.GetConfig().Server = "testServer"
	err := s.Install("http://localhost:8000")
	assert.Equal(t, "uploader:service Non empty config already present", err.Error())

	s.ctx.GetConfig().AccessKey = ""
	s.ctx.GetConfig().Server = ""
	err = s.Install("")
	assert.Equal(t, "uploader:service Server flag missing", err.Error())

	mc := s.client.(*MockClient)
	mc.ChangeState("AccessKeyFailure")

	err = s.Install("http://localhost:8000")
	assert.Equal(t, "uploader:service Could not get Access Key: AccessKeyFailure", err.Error())

}

func TestServiceUninstall(t *testing.T) {
	s := NewTestService()

	s.ctx.GetConfig().AccessKey = ""
	s.ctx.GetConfig().Server = ""
	err := s.Uninstall()
	assert.Equal(t, "uploader:service No config file found", err.Error())

	s.ctx.GetConfig().AccessKey = "testAccessKey"
	s.ctx.GetConfig().Server = "testServer"
	err = s.Uninstall()
	assert.Nil(t, err)
}

func TestServiceUpload(t *testing.T) {
	s := NewTestService()

	s.ctx.GetConfig().AccessKey = ""
	s.ctx.GetConfig().Server = ""
	err := s.Upload("testBundle", "testFile")
	assert.Equal(t, "uploader:service Need to install shipper first", err.Error())

	s.ctx.GetConfig().AccessKey = "testAccessKey"
	s.ctx.GetConfig().Server = "testServer"
	err = s.Upload("", "testFile")
	assert.Equal(t, "uploader:service BundleID missing", err.Error())

	err = s.Upload("testBundle", "")
	assert.Equal(t, "uploader:service File path is missing", err.Error())
}

func TestServiceWriteConfigFile(t *testing.T) {
	s := NewTestService()
	err := s.writeConfigFile("testServer", "testAccessKey")
	assert.Nil(t, err)
	assert.Equal(t, true, strings.Contains(testBuffer, ".shipper.toml data [application]\nserver = \"testServer\"\naccessKey = \"testAccessKey\"\n"))
}

func TestServiceDeleteConfigFile(t *testing.T) {
	s := NewTestService()
	err := s.deleteConfigFile()
	assert.Nil(t, err)
	assert.Equal(t, true, strings.Contains(testBuffer, "delete") && strings.Contains(testBuffer, "/.shipper.toml\n"))
}
