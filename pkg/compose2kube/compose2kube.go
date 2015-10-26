package compose2kube

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"strconv"
	"strings"

	"github.com/docker/libcompose/project"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/resource"
	"k8s.io/kubernetes/pkg/api/unversioned"
)

type K8sConfig struct {
	Name     string
	ObjType  string
	JsonData []byte
}
func Compose2kube(in io.Reader) ([]K8sConfig, error) {

	inBytes, err := ioutil.ReadAll(in)
	if err != nil {
		return nil, err
	}

	p := project.NewProject(&project.Context{
		ProjectName:  "kube",
		ComposeBytes: inBytes,
	})

	if err := p.Parse(); err != nil {
		return nil, fmt.Errorf("Failed to parse the compose project: %v", err)
	}

	k8sConfigs := []K8sConfig{}

	for name, service := range p.Configs {
		pod := &api.Pod{
			TypeMeta: unversioned.TypeMeta{
				Kind:       "Pod",
				APIVersion: "v1",
			},
			ObjectMeta: api.ObjectMeta{
				Name:   name,
				Labels: map[string]string{"service": name},
			},
			Spec: api.PodSpec{
				Containers: []api.Container{
					{
						Name:  name,
						Image: service.Image,
						Args:  service.Command.Slice(),
						Resources: api.ResourceRequirements{
							Limits: api.ResourceList{},
						},
					},
				},
			},
		}

		if service.CPUShares != 0 {
			pod.Spec.Containers[0].Resources.Limits[api.ResourceCPU] = *resource.NewQuantity(service.CPUShares, "decimalSI")
		}

		if service.MemLimit != 0 {
			pod.Spec.Containers[0].Resources.Limits[api.ResourceMemory] = *resource.NewQuantity(service.MemLimit, "decimalSI")
		}

		// Configure the environment variables
		var environment []api.EnvVar
		for _, envs := range service.Environment.Slice() {
			value := strings.Split(envs, "=")
			environment = append(environment, api.EnvVar{Name: value[0], Value: value[1]})
		}

		pod.Spec.Containers[0].Env = environment

		// Configure the container ports.
		var ports []api.ContainerPort
		for _, port := range service.Ports {
			portNumber, err := strconv.Atoi(port)
			if err != nil {
				log.Fatalf("Invalid container port %s for service %s", port, name)
			}
			ports = append(ports, api.ContainerPort{ContainerPort: portNumber})
		}

		pod.Spec.Containers[0].Ports = ports

		// Configure the container restart policy.
		var (
			rc      *api.ReplicationController
			objType string
			data    []byte
			err     error
		)
		switch service.Restart {
		case "", "always":
			objType = "rc"
			rc = replicationController(name, pod)
			pod.Spec.RestartPolicy = api.RestartPolicyAlways
			data, err = json.MarshalIndent(rc, "", "  ")
		case "no", "false":
			objType = "pod"
			pod.Spec.RestartPolicy = api.RestartPolicyNever
			data, err = json.MarshalIndent(pod, "", "  ")
		case "on-failure":
			objType = "rc"
			rc = replicationController(name, pod)
			pod.Spec.RestartPolicy = api.RestartPolicyOnFailure
			data, err = json.MarshalIndent(rc, "", "  ")
		default:
			log.Fatalf("Unknown restart policy %s for service %s", service.Restart, name)
		}

		if err != nil {
			log.Fatalf("Failed to marshal the replication controller: %v", err)
		}
		k8c := K8sConfig{
			Name:     name,
			ObjType:  objType,
			JsonData: data,
		}
		k8sConfigs = append(k8sConfigs, k8c)
	}
	return k8sConfigs, nil
}

func replicationController(name string, pod *api.Pod) *api.ReplicationController {
	return &api.ReplicationController{
		TypeMeta: unversioned.TypeMeta{
			Kind:       "ReplicationController",
			APIVersion: "v1",
		},
		ObjectMeta: api.ObjectMeta{
			Name:   name,
			Labels: map[string]string{"service": name},
		},
		Spec: api.ReplicationControllerSpec{
			Replicas: 1,
			Selector: map[string]string{"service": name},
			Template: &api.PodTemplateSpec{
				ObjectMeta: api.ObjectMeta{
					Labels: map[string]string{"service": name},
				},
				Spec: pod.Spec,
			},
		},
	}
}