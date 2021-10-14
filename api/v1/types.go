package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestJobSpec defines the desired state of TestJob
type TestJobSpec struct {
	// Template defines the behavior when running the test. Reuse this definition during distributed execution.
	Template TestJobTemplateSpec `json:"template"`
	// Tokens list of token for access to the repository and other resources in test.
	// +optional
	Tokens []TokenSpec `json:"tokens,omitempty"`
	// Repos defines list of repositories to use for testing.
	// +optional
	Repos []RepositorySpec `json:"repos,omitempty"`
	// PreSteps defines pre-processing to prepare files for testing that are not included in the repository (e.g. downloading dependent modules or building binaries).
	// This reduces the work that must be done inside the container when running the test, allowing the test to run with the minimum required privileges.
	// In addition, when performing distributed execution, the work that must be performed at the distributed execution destination is reduced,
	// so the resources of kubernetes cluster can be used efficiently.
	// +optional
	PreSteps []PreStep `json:"preSteps,omitempty"`
	// PostSteps defines post-processing to export artifacts.
	// +optional
	PostSteps []PostStep `json:"postSteps,omitempty"`
	// ExportArtifacts export what was saved as an artifact to any path.
	// +optional
	ExportArtifacts []ExportArtifact `json:"exportArtifacts,omitempty"`
	// Log extend parameter to output log.
	// +optional
	Log LogSpec `json:"log,omitempty"`
	// Strategy strategy for distributed testing
	// +optional
	Strategy *Strategy `json:"strategy,omitempty"`
}

// RepositorySpec describes the specification of repository.
type RepositorySpec struct {
	// Name specify the name to be used when referencing the repository in the TestJob resource.
	// The name must be unique within the TestJob resource.
	Name string `json:"name"`
	// Repo defines the repository.
	Value Repository `json:"value"`
}

// Repository describes the repository.
type Repository struct {
	// URL to the repository.
	URL string `json:"url"`
	// Branch name.
	Branch string `json:"branch,omitempty"`
	// Revision.
	Rev string `json:"rev,omitempty"`
	// This must match the Name of a Token.
	Token string `json:"token,omitempty"`
	// Merge base branch
	Merge *MergeSpec `json:"merge,omitempty"`
}

// MergeSpec describes the specification of merge behavior.
type MergeSpec struct {
	// Base branch name
	Base string `json:"base"`
}

// TokenSpec describes the specification of token for the repository or other resources.
type TokenSpec struct {
	// Name specify the name to be used when referencing the token in the TestJob resource.
	// The name must be unique within the TestJob resource.
	Name string `json:"name"`
	// Value specify what information the token is based on.
	Value TokenSource `json:"value"`
}

// TokenSource describes what information the token is based on.
type TokenSource struct {
	GitHubApp   *GitHubAppTokenSource `json:"githubApp,omitempty"`
	GitHubToken *GitHubTokenSource    `json:"githubToken,omitempty"`
}

// GitHubAppTokenSource describes the specification of github app based token.
type GitHubAppTokenSource struct {
	Organization   string                    `json:"organization,omitempty"`
	AppID          int64                     `json:"appId"`
	InstallationID int64                     `json:"installationId,omitempty"`
	KeyFile        *corev1.SecretKeySelector `json:"keyFile"`
}

// GitHubTokenSource describes the specification of github token.
type GitHubTokenSource corev1.SecretKeySelector

type StepType string

const (
	PreStepType  StepType = "preStep"
	MainStepType          = "mainStep"
	PostStepType          = "postStep"
)

type Step interface {
	GetName() string
	GetType() StepType
	GetTemplate() TestJobTemplateSpec
}

// PreStep defines pre-processing to prepare files for testing that are not included in the repository.
type PreStep struct {
	Name     string              `json:"name"`
	Template TestJobTemplateSpec `json:"template"`
}

func (s *PreStep) GetName() string {
	return s.Name
}

func (s *PreStep) GetType() StepType {
	return PreStepType
}

func (s *PreStep) GetTemplate() TestJobTemplateSpec {
	return s.Template
}

// MainStep defines main process
type MainStep struct {
	Template TestJobTemplateSpec `json:"template"`
}

func (s *MainStep) GetName() string {
	return ""
}

func (s *MainStep) GetType() StepType {
	return MainStepType
}

func (s *MainStep) GetTemplate() TestJobTemplateSpec {
	return s.Template
}

// PostStep defines post-processing to export artifacts.
type PostStep struct {
	Name     string              `json:"name"`
	Template TestJobTemplateSpec `json:"template"`
}

func (s *PostStep) GetName() string {
	return s.Name
}

func (s *PostStep) GetType() StepType {
	return PostStepType
}

func (s *PostStep) GetTemplate() TestJobTemplateSpec {
	return s.Template
}

// TestJobTemplateSpec
type TestJobTemplateSpec struct {
	// ObjectMeta standard object's metadata.
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Main is the main container name ( not sidecar container ).
	// If used multiple containers, this parameter must be specified.
	Main string `json:"main,omitempty"`
	// Spec specification of the desired behavior of the pod for TestJob.
	Spec TestJobPodSpec `json:"spec"`
}

// TestJobPodSpec
type TestJobPodSpec struct {
	corev1.PodSpec `json:",inline"`
	Volumes        []TestJobVolume `json:"volumes,omitempty"`
	Artifacts      []ArtifactSpec  `json:"artifacts,omitempty"`
}

// ArtifactSpec describes the specification of artifact for each process.
type ArtifactSpec struct {
	// Name specify the name to be used when referencing the token in the TestJob resource.
	// The name must be unique within the TestJob resource.
	Name string `json:"name"`
	// Container
	Container ArtifactContainer `json:"container"`
}

// ArtifactContainer
type ArtifactContainer struct {
	// Name for the container
	Name string `json:"name"`
	// Path to the artifact.
	Path string `json:"path"`
}

// TestJobVolume describes volume for TestJob.
type TestJobVolume struct {
	Name                string `json:"name"`
	TestJobVolumeSource `json:",inline"`
}

// TestJobVolumeSource describes volume sources for TestJob.
type TestJobVolumeSource struct {
	corev1.VolumeSource `json:",inline"`
	// Repo volume source for repository.
	Repo *RepositoryVolumeSource `json:"repo,omitempty"`
	// Artifact volume source for artifact.
	Artifact *ArtifactVolumeSource `json:"artifact,omitempty"`
	// Token volume source for token.
	Token *TokenVolumeSource `json:"token,omitempty"`
	// Log volume source for captured all logs
	Log *LogVolumeSource `json:"log,omitempty"`
	// Report volume source for result of kubetest
	Report *ReportVolumeSource `json:"report,omitempty"`
}

// RepositoryVolumeSource
type RepositoryVolumeSource struct {
	// This must match the Name of a RepositorySpec.
	Name string `json:"name"`
}

// ArtifactVolumeSource
type ArtifactVolumeSource struct {
	// This must match the Name of a ArtifactSpec.
	Name string `json:"name"`
}

// TokenVolumeSource
type TokenVolumeSource struct {
	// This must match the Name of a TokenSpec.
	Name string `json:"name"`
}

// LogVolumeSource
type LogVolumeSource struct{}

// ReportFormatType format type of report
type ReportFormatType string

const (
	ReportFormatTypeJSON ReportFormatType = "json"
)

// ReportVolumeSource
type ReportVolumeSource struct {
	Format ReportFormatType `json:"format"`
}

// ExportArtifact
type ExportArtifact struct {
	// This must match the Name of a ArtifactSpec.
	Name string `json:"name"`
	// Path path to the artifact.
	Path string `json:"path"`
}

// LogSpec
type LogSpec struct {
	ExtParam map[string]string `json:"extParam"`
}

// Strategy
type Strategy struct {
	// Key
	Key StrategyKeySpec `json:"key"`
	// Scheduler
	Scheduler Scheduler `json:"scheduler"`
	// Restart testing for failed tests
	Retest bool `json:"retest,omitempty"`
}

// StrategyKeySpec
type StrategyKeySpec struct {
	// Env name of env value for strategy key
	Env string `json:"env"`
	// Source
	Source StrategyKeySource `json:"source"`
}

// StrategyKeySource
type StrategyKeySource struct {
	// Static
	Static []string `json:"static,omitempty"`
	// Dynamic
	Dynamic *StrategyDynamicKeySource `json:"dynamic,omitempty"`
}

type StrategyDynamicKeySource struct {
	// Spec
	Template TestJobTemplateSpec `json:"template"`
	// Delimiter for strategy keys ( default: new line character ( \n ) )
	Delim string `json:"delimiter,omitempty"`
	// Filter filter got strategy keys ( use regular expression )
	Filter string `json:"filter,omitempty"`
}

// Scheduler
type Scheduler struct {
	// MaxContainersPerPod maximum number of container per pod.
	MaxContainersPerPod int `json:"maxContainersPerPod"`
	// MaxConcurrentNumPerPod maximum number of concurrent per pod.
	MaxConcurrentNumPerPod int `json:"maxConcurrentNumPerPod"`
}

// TestJobStatus defines the observed state of TestJob
type TestJobStatus struct {
	// Whether the testjob is running
	Running bool `json:"running,omitempty"`
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
