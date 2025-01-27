package main

import (
	"context"
	"flag"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	eventingv1 "knative.dev/eventing/pkg/apis/eventing/v1"
	flowsv1 "knative.dev/eventing/pkg/apis/flows/v1"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

func main() {
	action := flag.String("action", "deploy", "Action to perform: deploy, delete, sequence, broker, trigger, delete-trigger")
	name := flag.String("name", "empty-go", "Name of the function")
	image := flag.String("image", "empty-go", "Image to deploy")
	amount := flag.Int("amount", 1, "Amount of functions/triggers to deploy")
	brokerName := flag.String("broker", "default", "Name of the broker to use")
	metric := flag.String("metric", "concurrency", "Autoscaling strategy to use")
	target := flag.String("target", "80", "Target value for autoscaling")
	flag.Parse()
	vMap := make(map[string]string)
	vMap["metric"] = *metric
	vMap["target"] = *target
	// Register Knative types with the scheme
	servingv1.AddToScheme(scheme.Scheme)
	eventingv1.AddToScheme(scheme.Scheme)
	flowsv1.AddToScheme(scheme.Scheme)

	cfg, err := config.GetConfig()
	if err != nil {
		panic(err)
	}

	k8sClient, err := client.New(cfg, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		panic(err)
	}

	ctx := context.Background()

	switch *action {
	case "deploy", "delete":
		handleServiceAction(ctx, k8sClient, *action, *name, *image, *amount)
	case "sequence":
		deploySequence(ctx, k8sClient, *name, *image, *amount)
	case "broker":
		deployBrokerScenario(ctx, k8sClient, *brokerName, *name, *image, *amount)
	case "patch":
		// TODO: add patch params
		err := patchFunctionService(k8sClient, *name, vMap)
		if err != nil {
			fmt.Printf("Failed to patch service %s: %v\n", *name, err)
		}
	case "trigger":
		deployTrigger(ctx, k8sClient, *name, *brokerName, *amount)
	case "delete-trigger":
		deleteTrigger(ctx, k8sClient, *name, *brokerName, *amount)
	default:
		fmt.Println("Invalid action.")
	}
}

// Always targets the receiver service
func deployTrigger(ctx context.Context, k8sClient client.Client, name, brokerName string, amount int) {
	for i := 0; i < amount; i++ {
		trigger := &eventingv1.Trigger{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-trigger-%d", name, i),
				Namespace: "functions",
			},
			Spec: eventingv1.TriggerSpec{
				Broker: brokerName,
				Subscriber: duckv1.Destination{
					Ref: &duckv1.KReference{
						APIVersion: "serving.knative.dev/v1",
						Kind:       "Service",
						Name:       "reciever",
						Namespace:  "functions",
					},
				},
			},
		}
		if err := k8sClient.Create(ctx, trigger); err != nil {
			fmt.Printf("Failed to create trigger %d: %v\n", i, err)
		}
	}
}

// Delete triggers for a given name prefix
func deleteTrigger(ctx context.Context, k8sClient client.Client, name, brokerName string, amount int) {
	for i := 0; i < amount; i++ {
		trigger := &eventingv1.Trigger{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-trigger-%d", name, i),
				Namespace: "functions",
			},
		}
		if err := k8sClient.Delete(ctx, trigger); err != nil {
			fmt.Printf("Failed to delete trigger %d: %v\n", i, err)
		} else {
			fmt.Printf("Deleted trigger %s-trigger-%d\n", name, i)
		}
	}
}

// TODO
func deployBrokerScenario(ctx context.Context, c client.Client, brokerName, name, image string, amount int) {
	// Deploy broker
	broker := &eventingv1.Broker{
		ObjectMeta: metav1.ObjectMeta{
			Name:      brokerName,
			Namespace: "functions",
		},
		Spec: eventingv1.BrokerSpec{},
	}
	if err := c.Create(ctx, broker); err != nil {
		fmt.Printf("Failed to create broker: %v\n", err)
		return
	}

	// Deploy triggers
	for i := 0; i < amount; i++ {
		trigger := &eventingv1.Trigger{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-trigger-%d", name, i),
				Namespace: "functions",
			},
			Spec: eventingv1.TriggerSpec{
				Broker: brokerName,
				Subscriber: duckv1.Destination{
					Ref: &duckv1.KReference{
						APIVersion: "serving.knative.dev/v1",
						Kind:       "Service",
						Name:       fmt.Sprintf("%s-%d", name, i),
						Namespace:  "functions",
					},
				},
			},
		}
		if err := c.Create(ctx, trigger); err != nil {
			fmt.Printf("Failed to create trigger %d: %v\n", i, err)
		}
	}
}

func deploySequence(ctx context.Context, c client.Client, name, image string, steps int) {
	// Create sequence steps
	sequenceSteps := make([]flowsv1.SequenceStep, steps)
	for i := 0; i < steps; i++ {
		sequenceSteps[i] = flowsv1.SequenceStep{
			Destination: duckv1.Destination{
				Ref: &duckv1.KReference{
					APIVersion: "serving.knative.dev/v1",
					Kind:       "Service",
					Name:       fmt.Sprintf("%s-%d", name, i),
					Namespace:  "functions",
				},
			},
		}
	}

	sequence := &flowsv1.Sequence{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "functions",
		},
		Spec: flowsv1.SequenceSpec{
			Steps: sequenceSteps,
			Reply: &duckv1.Destination{
				Ref: &duckv1.KReference{
					APIVersion: "serving.knative.dev/v1",
					Kind:       "Service",
					Name:       "receiver",
					Namespace:  "functions",
				},
			},
		},
	}

	if err := c.Create(ctx, sequence); err != nil {
		fmt.Printf("Failed to create sequence: %v\n", err)
	}
}

func patchFunctionService(c client.Client, name string, params map[string]string) error {
	ctx := context.Background()
	svc := &servingv1.Service{}
	if err := c.Get(ctx, client.ObjectKey{Name: name, Namespace: "functions"}, svc); err != nil {
		return err
	}

	fmt.Printf("Patching service %s with params: %v\n", name, params)

	// Update annotations for autoscaling
	if svc.Spec.Template.Annotations == nil {
		svc.Spec.Template.Annotations = make(map[string]string)
	}

	// Handle autoscaling configuration
	if metric, ok := params["metric"]; ok {
		svc.Spec.Template.Annotations["autoscaling.knative.dev/metric"] = metric
		switch metric {
		case "concurrency":
			svc.Spec.Template.Annotations["autoscaling.knative.dev/target-utilization-percentage"] = params["target"]
		case "rps":
			svc.Spec.Template.Annotations["autoscaling.knative.dev/target"] = params["target"]
		case "cpu", "memory":
			svc.Spec.Template.Annotations["autoscaling.knative.dev/class"] = "hpa.autoscaling.knative.dev"
			svc.Spec.Template.Annotations["autoscaling.knative.dev/target"] = params["target"]
		}
	}

	// Handle resource configuration
	if cpu, ok := params["cpu-limit"]; ok {
		svc.Spec.Template.Spec.Containers[0].Resources.Limits[corev1.ResourceCPU] = resource.MustParse(cpu)
	}
	if mem, ok := params["memory-limit"]; ok {
		svc.Spec.Template.Spec.Containers[0].Resources.Limits[corev1.ResourceMemory] = resource.MustParse(mem)
	}
	if cpu, ok := params["cpu-request"]; ok {
		svc.Spec.Template.Spec.Containers[0].Resources.Requests[corev1.ResourceCPU] = resource.MustParse(cpu)
	}
	if mem, ok := params["memory-request"]; ok {
		svc.Spec.Template.Spec.Containers[0].Resources.Requests[corev1.ResourceMemory] = resource.MustParse(mem)
	}

	// Update the service
	return c.Update(ctx, svc)
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
						Annotations: map[string]string{
							"autoscaling.knative.dev/metric":                        "concurrency",
							"autoscaling.knative.dev/target-utilization-percentage": "10",
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
										AllowPrivilegeEscalation: ptr(true),
										Capabilities: &corev1.Capabilities{
											Drop: []corev1.Capability{"ALL"},
										},
										RunAsNonRoot: ptr(false),
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

func handleServiceAction(ctx context.Context, c client.Client, action, name, image string, amount int) {
	for i := 0; i < amount; i++ {
		newName := fmt.Sprintf("%s-%d", name, i)
		ksvc := createKnativeService(newName, image)

		switch action {
		case "deploy":
			if err := c.Create(ctx, ksvc); err != nil {
				fmt.Printf("Failed to create service %d: %v\n", i, err)
			} else {
				fmt.Printf("Created service %s\n", ksvc.Name)
			}
		case "delete":
			if err := c.Delete(ctx, ksvc); err != nil {
				fmt.Printf("Failed to delete service %d: %v\n", i, err)
			} else {
				fmt.Printf("Deleted service %s\n", ksvc.Name)
			}
		}
	}

	// For sequence and broker scenarios, also deploy the receiver service if it doesn't exist
	if action == "deploy" {
		receiverSvc := &servingv1.Service{}
		err := c.Get(ctx, client.ObjectKey{Name: "receiver", Namespace: "functions"}, receiverSvc)
		if err != nil {
			// Receiver doesn't exist, create it
			receiverSvc = createKnativeService("receiver", "cloudevent-receiver")
			if err := c.Create(ctx, receiverSvc); err != nil {
				fmt.Printf("Failed to create receiver service: %v\n", err)
			} else {
				fmt.Printf("Created receiver service\n")
			}
		}
	}
}
