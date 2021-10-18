//go:build !ignore_autogenerated
// +build !ignore_autogenerated

package v1

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