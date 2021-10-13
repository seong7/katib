/*
Copyright 2021 The Kubeflow Authors.

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

package v1beta1

import (
	"github.com/kubeflow/katib/pkg/controller.v1beta1/consts"
	"github.com/kubeflow/katib/pkg/util/v1beta1/katibclient"
)

var (
	TrialTemplateLabel = map[string]string{
		consts.LabelTrialTemplateConfigMapName: consts.LabelTrialTemplateConfigMapValue}
)

type Decoder struct {
	Layers     int            `json:"num_layers"`
	InputSize  []int          `json:"input_size"`
	OutputSize []int          `json:"output_size"`
	Embedding  map[int]*Block `json:"embedding"`
}

type Block struct {
	ID    int    `json:"opt_id"`
	Type  string `json:"opt_type"`
	Param Option `json:"opt_params"`
}

type Option struct {
	FilterNumber string `json:"num_filter"`
	FilterSize   string `json:"filter_size"`
	Stride       string `json:"stride"`
}

type ExperimentView struct {
	Name      string
	Status    string
	Namespace string
	Type      string
}

type TrialTemplatesDataView struct {
	ConfigMapNamespace string
	ConfigMaps         []ConfigMap
}

type TrialTemplatesResponse struct {
	Data []TrialTemplatesDataView
}

type ConfigMap struct {
	ConfigMapName string
	Templates     []Template
}

type Template struct {
	Path string
	Yaml string
}

type KatibUIHandler struct {
	katibClient   katibclient.Client
	dbManagerAddr string
}

type NNView struct {
	Name         string
	TrialName    string
	Architecture string
	MetricsName  []string
	MetricsValue []string
}

type JobType string

const (
	ExperimentTypeHP  = "hp"
	ExperimentTypeNAS = "nas"
	ActionTypeAdd     = "add"
	ActionTypeEdit    = "edit"
	ActionTypeDelete  = "delete"
)
