package main

import (
	"context"
	"flag"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

func main() {
	action := flag.String("action", "deploy", "Action to perform: deploy, delete, sequence")
	name := flag.String("name", "empty-go", "Name of the function")
	image := flag.String("image", "empty-go", "Image to deploy")
	amount := flag.Int("amount", 1, "Amount of functions to deploy")
	flag.Parse()

	// Register Knative types with the scheme
	servingv1.AddToScheme(scheme.Scheme)

	// Create k8s client
	cfg, err := config.GetConfig()
	if err != nil {
		panic(err)
	}

	k8sClient, err := client.New(cfg, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		panic(err)
	}

	switch *action {
	case "deploy":
		// Deploy multiple functions
		for i := 0; i < *amount; i++ {
			newName := fmt.Sprintf("%s-%d", *name, i)
			ksvc := createKnativeService(newName, *image)
			err := k8sClient.Create(context.Background(), ksvc)
			if err != nil {
				fmt.Printf("Failed to create service %d: %v\n", i, err)
			} else {
				fmt.Printf("Created service %s\n", ksvc.Name)
			}
		}
	case "delete":
		// Delete multiple functions
		for i := 0; i < *amount; i++ {
			newName := fmt.Sprintf("%s-%d", *name, i)
			ksvc := createKnativeService(newName, *image)
			err := k8sClient.Delete(context.Background(), ksvc)
			if err != nil {
				fmt.Printf("Failed to delete service %d: %v\n", i, err)
			} else {
				fmt.Printf("Deleted service %s\n", ksvc.Name)
			}
		}
	case "sequence":
		// Deploy a sequence of functions
		for i := 1; i < 11; i++ {
			newName := fmt.Sprintf("%s-%d", *name, i)
			ksvc := createKnativeService(newName, *image)
			err := k8sClient.Create(context.Background(), ksvc)
			if err != nil {
				fmt.Printf("Failed to create sequence: %v\n", err)
			}
		}
	default:
		fmt.Println("Invalid action.")
	}
}

func createKnativeService(name string, image string) *servingv1.Service {
	image = fmt.Sprintf("docker.io/luccadibenedetto/%s:latest", image)
	revisionName := fmt.Sprintf("%s-rev-%d", name, time.Now().Unix())
	return &servingv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "functions",
			Labels: map[string]string{
				"boson.dev/function":           "true",
				"boson.dev/runtime":            "go",
				"function.knative.dev":         "true",
				"function.knative.dev/name":    name,
				"function.knative.dev/runtime": "go",
			},
		},
		Spec: servingv1.ServiceSpec{
			ConfigurationSpec: servingv1.ConfigurationSpec{
				Template: servingv1.RevisionTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"boson.dev/function":                "true",
							"boson.dev/runtime":                 "go",
							"function.knative.dev":              "true",
							"function.knative.dev/name":         name,
							"function.knative.dev/runtime":      "go",
							"serving.knative.dev/revision":      revisionName,
							"serving.knative.dev/configuration": name,
							"serving.knative.dev/service":       name,
						},
					},
					Spec: servingv1.RevisionSpec{
						ContainerConcurrency: ptr(int64(0)),
						TimeoutSeconds:       ptr(int64(300)),
						PodSpec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Image: image,
									Env: []corev1.EnvVar{
										{
											Name:  "ADDRESS",
											Value: "0.0.0.0",
										},
									},
									SecurityContext: &corev1.SecurityContext{
										AllowPrivilegeEscalation: ptr(false),
										Capabilities: &corev1.Capabilities{
											Drop: []corev1.Capability{"ALL"},
										},
										RunAsNonRoot: ptr(true),
										SeccompProfile: &corev1.SeccompProfile{
											Type: "RuntimeDefault",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

// Helper function to create pointers to basic types
func ptr[T any](v T) *T {
	return &v
}

func patchFunctionService(client client.Client, name string, params map[string]string) {
	/*

			resources
		requests
		cpu: A CPU resource request for the container with deployed function. See related Kubernetes docs.
		memory: A memory resource request for the container with deployed function. See related Kubernetes docs.
		limits
		cpu: A CPU resource limit for the container with deployed function. See related Kubernetes docs.
		memory: A memory resource limit for the container with deployed function. See related Kubernetes docs.

		from func.yaml:

		options:
		  scale:
		    min: 0
		    max: 10
		    metric: concurrency
		    target: 75
		    utilization: 75
		  resources:
		    requests:
		      cpu: 100m
		      memory: 128Mi
		    limits:
		      cpu: 1000m
		      memory: 256Mi
		      concurrency: 100



						containerConcurrency
				int64	(Optional)
				ContainerConcurrency specifies the maximum allowed in-flight (concurrent) requests per container of the Revision. Defaults to 0 which means concurrency to the application is not limited, and the system decides the target concurrency for the autoscaler.

				timeoutSeconds
				int64	(Optional)
				TimeoutSeconds is the maximum duration in seconds that the request instance is allowed to respond to a request. If unspecified, a system default will be provided.

				responseStartTimeoutSeconds
				int64	(Optional)
				ResponseStartTimeoutSeconds is the maximum duration in seconds that the request routing layer will wait for a request delivered to a container to begin sending any network traffic.

				idleTimeoutSeconds
				int64	(Optional)
				IdleTimeoutSeconds is the maximum duration in seconds a request will be allowed to stay open while not receiving any bytes from the user’s application. If unspecified, a system default will be provided.


					apiVersion: serving.knative.dev/v1
				kind: Service
				metadata:
				  name: helloworld-go
				  namespace: default
				spec:
				  template:
				    metadata:
				      annotations:
					  	-> this needs to be patched to change the autoscaling strategy.
						Possible values: "concurrency", "rps", "cpu", "memory"
						for concurrency:
				        autoscaling.knative.dev/metric: "concurrency"
				        autoscaling.knative.dev/target-utilization-percentage: "70"

						for rps:
						autoscaling.knative.dev/metric: "rps"
				        autoscaling.knative.dev/target: "150"

						for cpu:
						autoscaling.knative.dev/class: "hpa.autoscaling.knative.dev"
				        autoscaling.knative.dev/metric: "cpu"
				        autoscaling.knative.dev/target: "100"

						for memory:
						autoscaling.knative.dev/class: "hpa.autoscaling.knative.dev"
				        autoscaling.knative.dev/metric: "memory"
				        autoscaling.knative.dev/target: "75"
	*/

}
