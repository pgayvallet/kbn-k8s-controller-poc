package controllers

import (
	elasticv1 "co.elastic/kibana-crd/api/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CreateExpandJob(kibana *elasticv1.Kibana) *batchv1.Job {
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				// Set the deploymentId label to easily retrieve it later
				deploymentIdLabel: kibana.Name,
				jobRole:           "expand-task",
			},
			Annotations: make(map[string]string),
			Name:        "some-job", // TODO
			Namespace:   kibana.Namespace,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "job",
						Image: "hello-world:latest",
					}},
					RestartPolicy: corev1.RestartPolicyNever,
				},
			},
		},
	}
	return job
}
