//go:build !ignore_autogenerated
// +build !ignore_autogenerated

package v1

import "fmt"

func (j *TestJob) SetStaticStrategyKeys(keys []string) error {
	if j.Spec.Strategy == nil {
		return fmt.Errorf("kubetest: spec.strategy is undefined")
	}
	j.Spec.Strategy.Key.Source.Static = keys
	return nil
}
