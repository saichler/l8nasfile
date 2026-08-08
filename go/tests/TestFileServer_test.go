/*
 * © 2025 Sharon Aicler (saichler@gmail.com)
 *
 * Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
 * You may obtain a copy of the License at:
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package tests

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/saichler/l8nasfile/go/nas/server"
	"github.com/saichler/l8nasfile/go/types/files"
	"github.com/saichler/l8types/go/types/l8api"
	"github.com/saichler/l8utils/go/utils/ipsegment"
	"github.com/saichler/l8utils/go/utils/shared"
	"github.com/saichler/l8web/go/web/client"
)

var testRC *client.RestClient
var testDir string

func setup(t *testing.T) {
	t.Helper()
	if testRC != nil {
		return
	}

	var err error
	testDir, err = os.MkdirTemp("", "nasfile-test-*")
	if err != nil {
		t.Fatal("failed to create temp dir:", err)
	}

	os.CopyFS("./web", os.DirFS("../nas/web/web"))

	go server.Start()

	resources := shared.ResourcesOf("Client", 0, 0)
	vnetPort := int(resources.SysConfig().VnetPort)
	webPort := int(resources.SysConfig().WebConfig.WebPort)

	waitForServer(t, webPort, 30)

	resources = shared.ResourcesOf("Client", uint32(vnetPort), 0)
	resources.Registry().Register(files.Action{})
	resources.Registry().Register(files.ActionResponse{})
	resources.Registry().Register(files.File{})
	resources.Registry().Register(files.FileList{})
	domain, private, public := resources.Certificate()
	clientConfig := &client.RestClientConfig{
		Host:          ipsegment.MachineIP,
		Port:          webPort,
		Https:         true,
		TokenRequired: true,
		CertDomain:    domain,
		CertPrivate:   private,
		CertPublic:    public,
		Prefix:        "/files/",
		AuthInfo: &client.RestAuthInfo{
			NeedAuth:   true,
			BodyType:   "AuthUser",
			UserField:  "User",
			PassField:  "Pass",
			RespType:   "AuthToken",
			TokenField: "Token",
			AuthPath:   "/auth",
		},
	}
	testRC, err = client.NewRestClient(clientConfig, resources)
	if err != nil {
		t.Fatal("failed to create rest client:", err)
	}
	resources.Registry().Register(&files.FileList{})
	resources.Registry().Register(&l8api.AuthToken{})
	err = testRC.Auth("admin", "admin")
	if err != nil {
		t.Fatal("failed to authenticate:", err)
	}
}

func waitForServer(t *testing.T, port int, maxSeconds int) {
	t.Helper()
	httpClient := &http.Client{
		Timeout: 2 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	baseURL := fmt.Sprintf("https://%s:%d", ipsegment.MachineIP, port)
	// Wait for HTTPS to be reachable
	for i := 0; i < maxSeconds; i++ {
		resp, err := httpClient.Get(baseURL + "/files/")
		if err == nil {
			resp.Body.Close()
			// Server is up; now wait for API endpoints (registered via async multicast)
			apiURL := baseURL + "/files/0/Files"
			for j := 0; j < maxSeconds; j++ {
				resp2, err2 := httpClient.Post(apiURL, "application/json", nil)
				if err2 == nil {
					resp2.Body.Close()
					if resp2.StatusCode != http.StatusNotFound {
						return
					}
				}
				time.Sleep(time.Second)
			}
			t.Fatal("API endpoints not registered within", maxSeconds, "seconds")
			return
		}
		time.Sleep(time.Second)
	}
	t.Fatal("server did not start within", maxSeconds, "seconds")
}

func TestListFiles(t *testing.T) {
	setup(t)

	f := &files.File{
		Name:        filepath.Base(testDir),
		Path:        filepath.Dir(testDir),
		IsDirectory: true,
	}

	resp, err := testRC.POST("0/Files", "FileList", "", "", f)
	if err != nil {
		t.Fatal("POST Files failed:", err)
	}

	fl, ok := resp.(*files.FileList)
	if !ok {
		t.Fatal("response is not *FileList")
	}
	if fl == nil {
		t.Fatal("file list response is nil")
	}
}

func TestNewFolder(t *testing.T) {
	setup(t)

	folderName := "test-newfolder"
	actn := &files.Action{
		Action: files.ActionType_ACTION_TYPE_NEW_FOLDER,
		Source: &files.File{Path: testDir, Name: folderName},
	}

	resp, err := testRC.POST("0/Actions", "ActionResponse", "", "", actn)
	if err != nil {
		t.Fatal("POST Actions (newFolder) failed:", err)
	}

	ar, ok := resp.(*files.ActionResponse)
	if !ok {
		t.Fatal("response is not *ActionResponse")
	}
	if ar.IsError {
		t.Fatal("newFolder returned error:", ar.Msg)
	}

	newPath := filepath.Join(testDir, folderName)
	info, err := os.Stat(newPath)
	if err != nil {
		t.Fatal("new folder does not exist:", err)
	}
	if !info.IsDir() {
		t.Fatal("new folder is not a directory")
	}
}

func TestCopyFile(t *testing.T) {
	setup(t)

	srcFile := filepath.Join(testDir, "copy-src.txt")
	content := []byte("copy test content")
	if err := os.WriteFile(srcFile, content, 0644); err != nil {
		t.Fatal("failed to create source file:", err)
	}

	dstFile := filepath.Join(testDir, "copy-dst.txt")
	actn := &files.Action{
		Action: files.ActionType_ACTION_TYPE_COPY,
		Source: &files.File{Path: testDir, Name: "copy-src.txt"},
		Target: &files.File{Path: testDir, Name: "copy-dst.txt"},
	}

	resp, err := testRC.POST("0/Actions", "ActionResponse", "", "", actn)
	if err != nil {
		t.Fatal("POST Actions (copy) failed:", err)
	}

	ar := resp.(*files.ActionResponse)
	if ar.IsError {
		t.Fatal("copy returned error:", ar.Msg)
	}

	if _, err := os.Stat(srcFile); err != nil {
		t.Fatal("source file should still exist after copy:", err)
	}
	dstContent, err := os.ReadFile(dstFile)
	if err != nil {
		t.Fatal("destination file does not exist:", err)
	}
	if string(dstContent) != string(content) {
		t.Fatalf("destination content mismatch: got %q, want %q", dstContent, content)
	}
}

func TestCopyDirectory(t *testing.T) {
	setup(t)

	srcDir := filepath.Join(testDir, "copy-dir-src")
	os.MkdirAll(srcDir, 0755)
	os.WriteFile(filepath.Join(srcDir, "inside.txt"), []byte("nested"), 0644)

	actn := &files.Action{
		Action: files.ActionType_ACTION_TYPE_COPY,
		Source: &files.File{Path: testDir, Name: "copy-dir-src", IsDirectory: true},
		Target: &files.File{Path: testDir, Name: "copy-dir-dst", IsDirectory: true},
	}

	resp, err := testRC.POST("0/Actions", "ActionResponse", "", "", actn)
	if err != nil {
		t.Fatal("POST Actions (copy dir) failed:", err)
	}

	ar := resp.(*files.ActionResponse)
	if ar.IsError {
		t.Fatal("copy dir returned error:", ar.Msg)
	}

	data, err := os.ReadFile(filepath.Join(testDir, "copy-dir-dst", "inside.txt"))
	if err != nil {
		t.Fatal("nested file not found in copied dir:", err)
	}
	if string(data) != "nested" {
		t.Fatal("nested file content mismatch")
	}
}

func TestCutFile(t *testing.T) {
	setup(t)

	srcFile := filepath.Join(testDir, "cut-src.txt")
	if err := os.WriteFile(srcFile, []byte("cut test"), 0644); err != nil {
		t.Fatal("failed to create source file:", err)
	}

	dstFile := filepath.Join(testDir, "cut-dst.txt")
	actn := &files.Action{
		Action: files.ActionType_ACTION_TYPE_CUT,
		Source: &files.File{Path: testDir, Name: "cut-src.txt"},
		Target: &files.File{Path: testDir, Name: "cut-dst.txt"},
	}

	resp, err := testRC.POST("0/Actions", "ActionResponse", "", "", actn)
	if err != nil {
		t.Fatal("POST Actions (cut) failed:", err)
	}

	ar := resp.(*files.ActionResponse)
	if ar.IsError {
		t.Fatal("cut returned error:", ar.Msg)
	}

	if _, err := os.Stat(srcFile); !os.IsNotExist(err) {
		t.Fatal("source file should not exist after cut")
	}
	data, err := os.ReadFile(dstFile)
	if err != nil {
		t.Fatal("destination file does not exist:", err)
	}
	if string(data) != "cut test" {
		t.Fatal("destination content mismatch")
	}
}

func TestRenameFile(t *testing.T) {
	setup(t)

	srcFile := filepath.Join(testDir, "rename-src.txt")
	if err := os.WriteFile(srcFile, []byte("rename test"), 0644); err != nil {
		t.Fatal("failed to create source file:", err)
	}

	actn := &files.Action{
		Action: files.ActionType_ACTION_TYPE_RENAME,
		Source: &files.File{Path: testDir, Name: "rename-src.txt"},
		Target: &files.File{Path: testDir, Name: "rename-dst.txt"},
	}

	resp, err := testRC.POST("0/Actions", "ActionResponse", "", "", actn)
	if err != nil {
		t.Fatal("POST Actions (rename) failed:", err)
	}

	ar := resp.(*files.ActionResponse)
	if ar.IsError {
		t.Fatal("rename returned error:", ar.Msg)
	}

	if _, err := os.Stat(srcFile); !os.IsNotExist(err) {
		t.Fatal("source should not exist after rename")
	}
	data, err := os.ReadFile(filepath.Join(testDir, "rename-dst.txt"))
	if err != nil {
		t.Fatal("renamed file does not exist:", err)
	}
	if string(data) != "rename test" {
		t.Fatal("renamed file content mismatch")
	}
}

func TestDeleteFile(t *testing.T) {
	setup(t)

	delFile := filepath.Join(testDir, "delete-me.txt")
	if err := os.WriteFile(delFile, []byte("delete test"), 0644); err != nil {
		t.Fatal("failed to create file:", err)
	}

	actn := &files.Action{
		Action: files.ActionType_ACTION_TYPE_DELETE,
		Source: &files.File{Path: testDir, Name: "delete-me.txt"},
	}

	resp, err := testRC.POST("0/Actions", "ActionResponse", "", "", actn)
	if err != nil {
		t.Fatal("POST Actions (delete) failed:", err)
	}

	ar := resp.(*files.ActionResponse)
	if ar.IsError {
		t.Fatal("delete returned error:", ar.Msg)
	}

	if _, err := os.Stat(delFile); !os.IsNotExist(err) {
		t.Fatal("file should not exist after delete")
	}
}

func TestDeleteDirectory(t *testing.T) {
	setup(t)

	delDir := filepath.Join(testDir, "delete-dir")
	os.MkdirAll(delDir, 0755)
	os.WriteFile(filepath.Join(delDir, "child.txt"), []byte("child"), 0644)

	actn := &files.Action{
		Action: files.ActionType_ACTION_TYPE_DELETE,
		Source: &files.File{Path: testDir, Name: "delete-dir", IsDirectory: true},
	}

	resp, err := testRC.POST("0/Actions", "ActionResponse", "", "", actn)
	if err != nil {
		t.Fatal("POST Actions (delete dir) failed:", err)
	}

	ar := resp.(*files.ActionResponse)
	if ar.IsError {
		t.Fatal("delete dir returned error:", ar.Msg)
	}

	if _, err := os.Stat(delDir); !os.IsNotExist(err) {
		t.Fatal("directory should not exist after delete")
	}
}

func TestListFilesAfterOperations(t *testing.T) {
	setup(t)

	os.WriteFile(filepath.Join(testDir, "list-test-a.txt"), []byte("a"), 0644)
	os.WriteFile(filepath.Join(testDir, "list-test-b.txt"), []byte("b"), 0644)

	f := &files.File{
		Name:        filepath.Base(testDir),
		Path:        filepath.Dir(testDir),
		IsDirectory: true,
	}

	resp, err := testRC.POST("0/Files", "FileList", "", "", f)
	if err != nil {
		t.Fatal("POST Files failed:", err)
	}

	fl := resp.(*files.FileList)
	found := 0
	for _, entry := range fl.List {
		if entry.Name == "list-test-a.txt" || entry.Name == "list-test-b.txt" {
			found++
		}
	}
	if found != 2 {
		t.Fatalf("expected 2 list-test files, found %d", found)
	}
}
