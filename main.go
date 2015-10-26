/*
Copyright 2015 Kelsey Hightower All rights reserved.
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

package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	c2k "github.com/cilium-team/compose2kube/pkg/compose2kube"
)

type K8sConfig struct {
	Name     string
	ObjType  string
	JsonData []byte
}

var (
	composeFile string
	outputDir   string
)

func init() {
	flag.StringVar(&composeFile, "compose-file", "docker-compose.yml", "Specify an alternate compose `file`")
	flag.StringVar(&outputDir, "output-dir", "output", "Kubernetes configs output `directory`")
}

func main() {
	flag.Parse()

	file, err := os.Open(composeFile)
	if err != nil {
		log.Fatal(err)
	}
	k8sConfigs, err := c2k.Compose2kube(file)
	if err != nil {
		log.Fatal(err)
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatal("Failed to create the output directory %s: %v", outputDir, err)
	}

	for _, k8c := range k8sConfigs {
		// Save the replication controller for the Docker compose service to the
		// configs directory.
		outputFileName := fmt.Sprintf("%s-%s.json", k8c.Name, k8c.ObjType)
		outputFilePath := filepath.Join(outputDir, outputFileName)
		if err := ioutil.WriteFile(outputFilePath, k8c.JsonData, 0644); err != nil {
			log.Fatalf("Failed to write replication controller %s: %v", outputFileName, err)
		}
		fmt.Println(outputFilePath)
	}
}

