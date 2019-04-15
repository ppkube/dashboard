/*
Copyright 2019 The Tekton Authors
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
		http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package endpoints

import (
	"strconv"
	"net/http/httputil"
	"net/url"
	restful "github.com/emicklei/go-restful"
	logging "github.com/tektoncd/dashboard/pkg/logging"
	"github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	k8sclientset "k8s.io/client-go/kubernetes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/api/core/v1"
)

// extensionLabel - service with this extensionLabel is registered as extension
const extensionLabel =  "tekton-dashboard-extension=true"  
// urlKey - extension path is specified by the annotation with the urlKey 
const urlKey = "tekton-dashboard-endpoints"

// Resource - stores all types here that are reused throughout files
type Resource struct {
	PipelineClient versioned.Interface
	K8sClient      k8sclientset.Interface
}

// RegisterEndpoints - this registers our actual endpoints!
func (r Resource) RegisterEndpoints(container *restful.Container) {
	wsv1 := new(restful.WebService)
	wsv1.
		Path("/v1/namespaces").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	logging.Log.Info("Adding v1, and API for pipelines")

	wsv1.Route(wsv1.GET("/{namespace}/pipeline").To(r.getAllPipelines))
	wsv1.Route(wsv1.GET("/{namespace}/pipeline/{name}").To(r.getPipeline))

	wsv1.Route(wsv1.GET("/{namespace}/pipelinerun").To(r.getAllPipelineRuns))
	wsv1.Route(wsv1.GET("/{namespace}/pipelinerun/{name}").To(r.getPipelineRun))
	wsv1.Route(wsv1.PUT("/{namespace}/pipelinerun/{name}").To(r.updatePipelineRun))

	wsv1.Route(wsv1.GET("/{namespace}/pipelineresource").To(r.getAllPipelineResources))
	wsv1.Route(wsv1.GET("/{namespace}/pipelineresource/{name}").To(r.getPipelineResource))

	wsv1.Route(wsv1.GET("/{namespace}/task").To(r.getAllTasks))
	wsv1.Route(wsv1.GET("/{namespace}/task/{name}").To(r.getTask))

	wsv1.Route(wsv1.GET("/{namespace}/taskrun").To(r.getAllTaskRuns))
	wsv1.Route(wsv1.GET("/{namespace}/taskrun/{name}").To(r.getTaskRun))

	wsv1.Route(wsv1.GET("/{namespace}/log/{name}").To(r.getPodLog))

	wsv1.Route(wsv1.GET("/{namespace}/taskrunlog/{name}").To(r.getTaskRunLog))

	wsv1.Route(wsv1.GET("/{namespace}/pipelinerunlog/{name}").To(r.getPipelineRunLog))

	wsv1.Route(wsv1.GET("/{namespace}/credentials/").To(r.getAllCredentials))
	wsv1.Route(wsv1.GET("/{namespace}/credentials/{id}").To(r.getCredential))
	wsv1.Route(wsv1.POST("/{namespace}/credentials/").To(r.createCredential))
	wsv1.Route(wsv1.PUT("/{namespace}/credentials/{id}").To(r.updateCredential))
	wsv1.Route(wsv1.DELETE("/{namespace}/credentials/{id}").To(r.deleteCredential))

	container.Add(wsv1)
}

// RegisterWebsocket - this registers a websocket with which we can send log information to
func (r Resource) RegisterWebsocket(container *restful.Container) {
	logging.Log.Info("Adding API for websocket")
	wsv2 := new(restful.WebService)
	wsv2.
		Path("/v1/websocket").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)
	wsv2.Route(wsv2.GET("/logs").To(r.establishPipelineLogsWebsocket))
	wsv2.Route(wsv2.GET("/pipelineruns").To(r.establishPipelineRunsWebsocket))
	container.Add(wsv2)
}

// RegisterHealthProbes - this registers the /health endpoint
func (r Resource) RegisterHealthProbes(container *restful.Container) {
	logging.Log.Info("Adding API for health")
	wsv3 := new(restful.WebService)
	wsv3.
		Path("/health").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	wsv3.Route(wsv3.GET("/").To(r.checkHealth))

	container.Add(wsv3)
}

// RegisterReadinessProbes - this registers the /readiness endpoint
func (r Resource) RegisterReadinessProbes(container *restful.Container) {
	logging.Log.Info("Adding API for readiness")
	wsv4 := new(restful.WebService)
	wsv4.
		Path("/readiness").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	wsv4.Route(wsv4.GET("/").To(r.checkHealth))

	container.Add(wsv4)
}

// Back-end extension: Requests to the URL are passthrough to the Port of the Name service (extension)
// "label: tekton-dashboard-extension=true" in the service defines the extention
// "annotation: tekton-dashboard-endpoints=<URL>" spacifies the path for the extension
type Extension struct {
	Name string
	URL  string
	Port string
}

// RegisterExtension - this discovers the extensions and registers them as the REST API extension 
func (r Resource) RegisterExtension(container *restful.Container, namespace string) {
	logging.Log.Info("Adding API for extensions")
	svcs, err := r.K8sClient.CoreV1().Services(namespace).List(metav1.ListOptions{LabelSelector: extensionLabel})
	if err != nil {
		logging.Log.Errorf("no extension found: %s", err)
		return
	}
	ws := new(restful.WebService)
	ws.
		Path("/").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	for _, svc := range svcs.Items {
		for key, url := range svc.ObjectMeta.Annotations{
			if key == urlKey {
				logging.Log.Debugf("extension URL: %s", url)
                                ext := Extension { Name: svc.ObjectMeta.Name, URL: url, Port: getPort(svc) }
				ws.Route(ws.POST(url).To(ext.HandleExtension))
			}
		}
	}
	container.Add(ws)
}

// HandleExtension - this route request to the extention path to the extention service
func (ext Extension) HandleExtension(request *restful.Request, response *restful.Response) {
	target, _ := url.Parse("http://" +  ext.Name + ":" + ext.Port + "/")
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.ServeHTTP(response, request.Request)
}

// getPort - this get the port of the service
func getPort(svc corev1.Service) string {
	return strconv.Itoa(int(svc.Spec.Ports[0].Port))
}
