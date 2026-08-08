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

package server

import (
	"encoding/base64"
	"net/http"
	"time"

	"github.com/saichler/l8utils/go/utils/ipsegment"

	"github.com/saichler/l8bus/go/overlay/vnet"
	"github.com/saichler/l8nasfile/go/nas/actions"
	"github.com/saichler/l8nasfile/go/nas/common"
	files2 "github.com/saichler/l8nasfile/go/nas/files"
	"github.com/saichler/l8nasfile/go/types/files"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/sec"
	"github.com/saichler/l8web/go/web/server"
)

func Start() {
	server.Timeout = 600
	r := common.CreateResources("vnet-nas", false)
	r.Logger().SetLogLevel(ifs.Info_Level)
	net := vnet.NewVNet(r)
	net.Start()

	r.Logger().Info("vnet started!")
	r.Logger().SetLogLevel(ifs.Error_Level)
	time.Sleep(time.Second)
	startWebServer(r)
}

func startWebServer(vnetResources ifs.IResources) {
	webPort := int(vnetResources.SysConfig().WebConfig.WebPort)
	nic := common.CreateVnic("web-nas", false, func(res ifs.IResources) {
		res.Registry().Register(&files.File{})
		res.Registry().Register(&files.FileList{})
		res.Registry().Register(&files.Action{})
		res.Registry().Register(&files.ActionResponse{})
	})

	files2.Activate(nic)
	actions.Activate(nic)

	domain, private, _ := nic.Resources().Certificate()
	if domain == "" || private == "" {
		d, p, _ := sec.CreateCertBundle()
		domainBytes, _ := base64.StdEncoding.DecodeString(d)
		privateBytes, _ := base64.StdEncoding.DecodeString(p)
		domain = string(domainBytes)
		private = string(privateBytes)
	}

	serverConfig := &server.RestServerConfig{
		Host:           ipsegment.MachineIP,
		Port:           webPort,
		Authentication: true,
		CertDomain:     domain,
		CertPrivate:    private,
		Prefix:         "/files/",
	}
	svr, err := server.NewRestServer(serverConfig)
	if err != nil {
		panic(err)
	}

	//Activate the webpoints service
	sla := ifs.NewServiceLevelAgreement(&server.WebService{}, ifs.WebService, 0, false, nil)
	sla.SetArgs(svr, nic)
	nic.Resources().Services().Activate(sla, nic)

	// Register download endpoint
	registerDownloadEndpoint(nic)

	nic.Resources().Logger().Info("Web Server Started!")

	svr.Start()
}

func registerDownloadEndpoint(vnic ifs.IVNic) {
	http.HandleFunc("/files/download", func(w http.ResponseWriter, r *http.Request) {
		// Check authentication
		bearer := r.Header.Get("Authorization")
		if bearer == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		_, ok := vnic.Resources().Security().ValidateToken(bearer, vnic)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		actions.DownloadHandler(w, r, vnic.Resources())
	})
}
