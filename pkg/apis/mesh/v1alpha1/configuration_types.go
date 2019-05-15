/*
 * Copyright (c) 2018 WSO2 Inc. (http:www.wso2.org) All Rights Reserved.
 *
 * WSO2 Inc. licenses this file to you under the Apache License,
 * Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http:www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Configuration struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ConfigurationSpec   `json:"spec"`
	Status ConfigurationStatus `json:"status"`
}

type ConfigurationSpec struct {
	Rules []Rule `json:"rules"`
}

type Rule struct {
	Match    map[string]string `json:"match,omitempty"`
	Filters  []Filter          `json:"filters,omitempty"`
	Clusters []Cluster         `json:"clusters,omitempty"`
}

type Filter struct {
	Port uint32 `json:"port,omitempty"`
	Http []Http `json:"http,omitempty"`
	Tcp  Tcp    `json:"tcp,omitempty"`
}

type Http struct {
	Domains []string `json:"domains,omitempty"`
	Cluster string   `json:"cluster,omitempty"`
}

type Tcp struct {
	Cluster string `json:"cluster,omitempty"`
}

type Cluster struct {
	Name string `json:"name,omitempty"`
	Host string `json:"host,omitempty"`
	Port uint32 `json:"port,omitempty"`
}

type ConfigurationStatus struct {
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ConfigurationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []Configuration `json:"items"`
}
