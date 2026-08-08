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

package actions

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/saichler/l8nasfile/go/types/files"
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8web"
	"github.com/saichler/l8utils/go/utils/web"
)

const (
	ServiceName = "Actions"
	ServiceType = "ActionService"
	ServiceArea = byte(0)
)

type ActionService struct {
	sla *ifs.ServiceLevelAgreement
}

func Activate(vnic ifs.IVNic) {
	sla := ifs.NewServiceLevelAgreement(&ActionService{}, ServiceName, ServiceArea, false, nil)
	ws := web.New(ServiceName, ServiceArea, 0)
	ws.AddEndpoint(&files.Action{}, ifs.POST, &files.ActionResponse{})
	sla.SetWebService(ws)
	vnic.Resources().Services().Activate(sla, vnic)
}

func (this *ActionService) Activate(sla *ifs.ServiceLevelAgreement, vnic ifs.IVNic) error {
	vnic.Resources().Registry().Register(&files.File{})
	vnic.Resources().Registry().Register(&files.FileList{})
	vnic.Resources().Registry().Register(&files.Action{})
	vnic.Resources().Registry().Register(&files.ActionResponse{})
	vnic.Resources().Registry().Register(&l8web.L8Empty{})
	this.sla = sla
	return nil
}

func (this *ActionService) DeActivate() error {
	return nil
}

func respond(msg string, isError bool) ifs.IElements {
	fmt.Println(msg)
	return object.New(nil, &files.ActionResponse{Msg: msg, IsError: isError})
}

func resolvePath(f *files.File) (string, error) {
	if f == nil {
		return "", errors.New("file is nil")
	}
	raw := f.Path + "/" + f.Name
	if strings.HasPrefix(raw, "//") {
		raw = raw[1:]
	}
	cleaned := filepath.Clean(raw)
	return cleaned, nil
}

func (this *ActionService) Post(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	ac, ok := pb.Element().(*files.Action)
	if ok {
		fmt.Println("Doing: ", ac.Action.String())
		switch ac.Action {
		case files.ActionType_ACTION_TYPE_COPY:
			return doCopy(ac)
		case files.ActionType_ACTION_TYPE_CUT:
			return doCut(ac)
		case files.ActionType_ACTION_TYPE_DELETE:
			return doDelete(ac)
		case files.ActionType_ACTION_TYPE_RENAME:
			return doRename(ac)
		case files.ActionType_ACTION_TYPE_NEW_FOLDER:
			return doNewFolder(ac)
		}
	}
	return object.New(nil, &l8web.L8Empty{})
}

func doCopy(ac *files.Action) ifs.IElements {
	srcPath, err := resolvePath(ac.Source)
	if err != nil {
		return respond(err.Error(), true)
	}
	dstPath, err := resolvePath(ac.Target)
	if err != nil {
		return respond(err.Error(), true)
	}

	srcInfo, err := os.Stat(srcPath)
	if err != nil {
		return respond("source does not exist: "+srcPath, true)
	}

	if srcInfo.IsDir() {
		err = copyDir(srcPath, dstPath)
	} else {
		err = copyFile(srcPath, dstPath)
	}
	if err != nil {
		return respond(err.Error(), true)
	}
	return respond("copied "+srcPath+" to "+dstPath, false)
}

func doCut(ac *files.Action) ifs.IElements {
	srcPath, err := resolvePath(ac.Source)
	if err != nil {
		return respond(err.Error(), true)
	}
	dstPath, err := resolvePath(ac.Target)
	if err != nil {
		return respond(err.Error(), true)
	}

	if _, err := os.Stat(srcPath); err != nil {
		return respond("source does not exist: "+srcPath, true)
	}

	if err := os.Rename(srcPath, dstPath); err != nil {
		return respond(err.Error(), true)
	}
	return respond("moved "+srcPath+" to "+dstPath, false)
}

func doDelete(ac *files.Action) ifs.IElements {
	srcPath, err := resolvePath(ac.Source)
	if err != nil {
		return respond(err.Error(), true)
	}

	if _, err := os.Stat(srcPath); err != nil {
		return respond("path does not exist: "+srcPath, true)
	}

	if err := os.RemoveAll(srcPath); err != nil {
		return respond(err.Error(), true)
	}
	return respond("deleted "+srcPath, false)
}

func doRename(ac *files.Action) ifs.IElements {
	srcPath, err := resolvePath(ac.Source)
	if err != nil {
		return respond(err.Error(), true)
	}
	dstPath, err := resolvePath(ac.Target)
	if err != nil {
		return respond(err.Error(), true)
	}

	if _, err := os.Stat(srcPath); err != nil {
		return respond("source does not exist: "+srcPath, true)
	}

	if err := os.Rename(srcPath, dstPath); err != nil {
		return respond(err.Error(), true)
	}
	return respond("renamed "+srcPath+" to "+dstPath, false)
}

func doNewFolder(ac *files.Action) ifs.IElements {
	folderPath, err := resolvePath(ac.Source)
	if err != nil {
		return respond(err.Error(), true)
	}

	if err := os.MkdirAll(folderPath, 0755); err != nil {
		return respond(err.Error(), true)
	}
	return respond("created folder "+folderPath, false)
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}

	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(dst, relPath)

		if d.IsDir() {
			return os.MkdirAll(targetPath, 0755)
		}
		return copyFile(path, targetPath)
	})
}

func (this *ActionService) Put(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return nil
}

func (this *ActionService) Patch(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return nil
}

func (this *ActionService) Delete(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return nil
}

func (this *ActionService) Get(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return nil
}
func (this *ActionService) GetCopy(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return nil
}
func (this *ActionService) Failed(pb ifs.IElements, vnic ifs.IVNic, msg *ifs.Message) ifs.IElements {
	return nil
}
func (this *ActionService) TransactionConfig() ifs.ITransactionConfig {
	return nil
}

func (this *ActionService) WebService() ifs.IWebService {
	return this.sla.WebService()
}

func DownloadHandler(w http.ResponseWriter, r *http.Request, resources ifs.IResources) {
	filePath := r.URL.Query().Get("path")
	if filePath == "" {
		http.Error(w, "Missing path parameter", http.StatusBadRequest)
		return
	}

	cleanPath := filepath.Clean(filePath)

	fileInfo, err := os.Stat(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "File not found", http.StatusNotFound)
		} else {
			http.Error(w, "Error accessing file", http.StatusInternalServerError)
		}
		return
	}

	if fileInfo.IsDir() {
		http.Error(w, "Cannot download a directory", http.StatusBadRequest)
		return
	}

	file, err := os.Open(cleanPath)
	if err != nil {
		http.Error(w, "Error opening file", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	fileName := filepath.Base(cleanPath)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename*=UTF-8''%s", encodeRFC5987(fileName)))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))

	_, err = io.Copy(w, file)
	if err != nil {
		resources.Logger().Error("Error streaming file: ", err)
	}
}

func encodeRFC5987(s string) string {
	encoded := url.QueryEscape(s)
	encoded = strings.ReplaceAll(encoded, "+", "%20")
	return encoded
}
