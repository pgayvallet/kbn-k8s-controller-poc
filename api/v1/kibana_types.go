/*
Copyright 2022.

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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// KibanaSpec defines the desired state of Kibana
type KibanaSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Version string `json:"version,omitempty"`
}

// the overall status of the Kibana Kind
type KibanaOverallStatus string

const (
	KibanaOverallStatusUnknown     KibanaOverallStatus = ""
	KibanaOverallStatusAvailable   KibanaOverallStatus = "Available"
	KibanaOverallStatusProgressing KibanaOverallStatus = "Progressing"
	KibanaOverallStatusError       KibanaOverallStatus = "Error"
)

// the stage of the rolling upgrade
type KibanaDeploymentStage string

const (
	// all steps complete
	KibanaDeployingStatusUnknown KibanaDeploymentStage = ""
	// running the expand job
	KibanaDeployingStatusRunningExpandJob KibanaDeploymentStage = "RunningExpandJob"
	// waiting for the deployments to have rolled out to the new version
	KibanaDeployingStatusRollingDeployments KibanaDeploymentStage = "RollingDeployments"
	// running the migrate job
	KibanaDeployingStatusRunningMigrateJob KibanaDeploymentStage = "RunningMigrateJob"
	// all steps complete
	KibanaDeployingStatusComplete KibanaDeploymentStage = "Complete"
)

// the status of a stage of the rolling upgrade
type KibanaDeploymentStageStatus string

const (
	KibanaDeploymentStageStatusScheduled            = "Scheduled"
	KibanaDeploymentStageStatusWaitingForCompletion = "WaitingForCompletion"
)

// KibanaStatus defines the observed state of Kibana
type KibanaStatus struct {
	// The overall status of the deployment
	OverallStatus KibanaOverallStatus `json:"overallStatus,omitempty"`
	// The last deployed version (before the currently deploying version, eventually)
	LastDeployedVersion string `json:"lastDeployedVersion,omitempty"`
	// The version that is currently deploying. Empty if the controller isn't
	CurrentlyDeployingVersion string `json:"currentlyDeployingVersion,omitempty"`
	// The current stage of the rolling deployment
	DeploymentStage KibanaDeploymentStage `json:"deploymentStage,omitempty"`
	// The current status of the current stage of rolling deployment
	DeploymentStageStatus KibanaDeploymentStageStatus `json:"deploymentStageStatus,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Kibana is the Schema for the kibanas API
type Kibana struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KibanaSpec   `json:"spec,omitempty"`
	Status KibanaStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// KibanaList contains a list of Kibana
type KibanaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Kibana `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Kibana{}, &KibanaList{})
}
