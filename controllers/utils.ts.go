package controllers

import (
	elasticv1 "co.elastic/kibana-crd/api/v1"
)

// TODO: very naive, need to fix
func ShouldStartOrContinueRolling(kibana *elasticv1.Kibana) bool {
	status := kibana.Status

	if status.OverallStatus == elasticv1.KibanaOverallStatusUnknown || status.OverallStatus == elasticv1.KibanaOverallStatusProgressing {
		return true
	}
	// TODO: probably false, need to check something with CurrentlyDeployingVersion instead
	if status.LastDeployedVersion != "" && status.LastDeployedVersion != kibana.Spec.Version {
		return true
	}

	return false
}
