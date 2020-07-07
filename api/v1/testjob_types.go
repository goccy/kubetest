/*


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

// TestJobSpec defines the desired state of TestJob
type TestJobSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Image name.
	Image string `json:"image"`
	// Repository name.
	Repo string `json:"repo"`
	// Command for testing.
	Command []string `json:"command"`
	// Branch name.
	Branch string `json:"branch,omitempty"`
	// Revision.
	Rev string `json:"rev,omitempty"`
	// OAuth token to fetch private repository
	Token *TestJobToken `json:"token,omitempty"`
	// Distributed testing parameter
	DistributedTest *DistributedTestSpec `json:"distributedTest,omitempty"`
}

type TestJobToken struct {
	SecretKeyRef TestJobSecretKeyRef `json:"secretKeyRef"`
}

type TestJobSecretKeyRef struct {
	Name string `json:"name"`
	Key  string `json:"key"`
}

type DistributedTestSpec struct {
	// Output testing list to stdout
	ListCommand []string `json:"listCommand"`
	// Delimiter for testing list ( default: new line character ( \n ) )
	ListDelimiter string `json:"listDelimiter,omitempty"`
	// Test name pattern ( enable use regular expression )
	Pattern string `json:"pattern,omitempty"`
	// Concurrent number of process of testing
	Concurrent int `json:"concurrent"`
	// Restart testing for failed tests
	Retest bool `json:"retest"`
	// Delimiter for testing list of retest ( default: white space )
	RetestDelimiter string `json:"retestDelimiter,omitempty"`
}

// TestJobStatus defines the observed state of TestJob
type TestJobStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true

// TestJob is the Schema for the testjobs API
type TestJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TestJobSpec   `json:"spec,omitempty"`
	Status TestJobStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TestJobList contains a list of TestJob
type TestJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TestJob `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TestJob{}, &TestJobList{})
}
