initial readme



Knative requirements:
For production purposes, it is recommended that:

If you have only one node in your cluster, you need at least 6 CPUs, 6 GB of memory, and 30 GB of disk storage.
If you have multiple nodes in your cluster, for each node you need at least 2 CPUs, 4 GB of memory, and 20 GB of disk storage.
You have a cluster that uses Kubernetes v1.28 or newer.
You have installed the kubectl CLI.
Your Kubernetes cluster must have access to the internet, because Kubernetes needs to be able to fetch images. To pull from a private registry, see Deploying images from a private container registry.

The system requirements provided are recommendations only. The requirements for your installation might vary, depending on whether you use optional components, such as a networking layer.

Knative supports encryption features through cert-manager. Follow the documentation in Serving encryption for more information.
maybe benchmark with encryption vs no encryption

func languages
go
node
python
quarkus
rust
springboot
typescript

func templates
LANGUAGE     TEMPLATE
go           cloudevents
go           http
node         cloudevents
node         http
python       cloudevents
python       flask
python       http
python       wsgi
quarkus      cloudevents
quarkus      http
rust         cloudevents
rust         http
springboot   cloudevents
springboot   http
typescript   cloudevents
typescript   http


func build --help

NAME
        func build - Build a function container locally without deploying

SYNOPSIS
        func build [-r|--registry] [--builder] [--builder-image]
                         [--push] [--username] [--password] [--token]
                     [--platform] [-p|--path] [-c|--confirm] [-v|--verbose]
                         [--build-timestamp] [--registry-insecure]

DESCRIPTION

        Builds a function's container image and optionally pushes it to the
        configured container registry.

        By default building is handled automatically when deploying (see the deploy
        subcommand). However, sometimes it is useful to build a function container
        outside of this normal deployment process, for example for testing or during
        composition when integrating with other systems. Additionally, the container
        can be pushed to the configured registry using the --push option.

        When building a function for the first time, either a registry or explicit
        image name is required.  Subsequent builds will reuse these option values.

EXAMPLES

        o Build a function container using the given registry.
          The full image name will be calculated using the registry and function name.
          $ func build --registry registry.example.com/alice

        o Build a function container using an explicit image name, ignoring registry
          and function name.
          $ func build --image registry.example.com/alice/f:latest

        o Rebuild a function using prior values to determine container name.
          $ func build

        o Build a function specifying the Source-to-Image (S2I) builder
          $ func build --builder=s2i

        o Build a function specifying the Pack builder with a custom Buildpack
          builder image.
          $ func build --builder=pack --builder-image=cnbs/sample-builder:bionic



Usage:
  func build [flags]

Flags:
      --build-timestamp        Use the actual time as the created time for the docker image. This is only useful
                               for buildpacks builder.
  -b, --builder string         Builder to use when creating the function's container. Currently supported builders
                               are "pack" and "s2i". ($FUNC_BUILDER) (default "pack")
      --builder-image string   Specify a custom builder image for use by the builder other than its default.
                               ($FUNC_BUILDER_IMAGE)
  -c, --confirm                Prompt to confirm options interactively ($FUNC_CONFIRM)
  -i, --image string           Full image name in the form [registry]/[namespace]/[name]:[tag] (optional). This
                               option takes precedence over --registry ($FUNC_IMAGE)
  -p, --path string            Path to the function.  Default is current directory ($FUNC_PATH)
      --platform string        Optionally specify a target platform, for example "linux/amd64" when using the s2i
                               build strategy
  -u, --push                   Attempt to push the function image to the configured registry after being
                               successfully built
  -r, --registry string        Container registry + registry namespace. (ex 'ghcr.io/myuser').  The full image
                               name is automatically determined using this along with function name. ($FUNC_REGISTRY)
      --registry-insecure      Skip TLS certificate verification when communicating in HTTPS with the registry
                               ($FUNC_REGISTRY_INSECURE)
  -v, --verbose                Print verbose logs ($FUNC_VERBOSE)


  https://knative.dev/docs/serving/revisions/revision-admin-config-options/

  en el activator ahora mismo hay un error:
  error while getting revision for commit noseque , claro cuando deployeamos el service con la API la revision no se crea automaticamente sino q se creaa cuendo pusheas la funcion con func deploy


  luccadibe@DESKTOP-HT1I41R:~/projects/knative_benchmark/functions/echo-go-http$ func build --image=docker.io/luccadibenedetto/echo-go-http:latest
Building function image
Still building
Still building
Yes, still building
🙌 Function built: docker.io/luccadibenedetto/echo-go-http:latest


luccadibe@DESKTOP-HT1I41R:~/projects/knative_benchmark/functions/echo-go-http$ func deploy --image=docker.io/luccadibenedetto/echo-go-http:latest --build=false --namespace=functions
Warning: namespace chosen is 'functions', but currently active namespace is 'default'. Continuing with deployment to 'functions'.
Pushing function image to the registry "index.docker.io" using the "luccadibenedetto" user credentials
🎯 Creating Triggers on the cluster
✅ Function deployed in namespace "functions" and exposed at URL: 
   http://echo-go-http.functions.example.com


   luccadibe@DESKTOP-HT1I41R:~/projects/knative_benchmark/functions/echo-go-http$ func deploy --image=docker.io/luccadibenedetto/echo-go-http:latest --build=true --namespace=functions
Warning: namespace chosen is 'functions', but currently active namespace is 'default'. Continuing with deployment to 'functions'.
Building function image
Still building
Still building
🙌 Function built: docker.io/luccadibenedetto/echo-go-http:latest
Pushing function image to the registry "index.docker.io" using the "luccadibenedetto" user credentials
🎯 Creating Triggers on the cluster
✅ Function deployed in namespace "functions" and exposed at URL: 
   http://echo-go-http.functions.example.com

current issue is that there is no revision created automatically when we deploy with the API.
In the end the services work fine, but I suspect that this could be a problem in the future.
Maybe it causes knative to do extra work checking and reconciling.
So i need a way to create a revision automatically when we deploy with the API.

Another idea is to create all the revisions "manually" (small script that uses func deploy to create the revisions)
and then point to the correct revision when we deploy with the API.

-> also the metrics exported by knative are working fine . The collector is recieving them.
Currently I only verified this by running:
kubectl port-forward --namespace metrics deployment/otel-collector 8889
then visiting http://localhost:8889/metrics

Next step is adding a simple victoriametrics database to store the metrics.
Hopefully we can do this easily directly with a deployment and a pvc, i dont want more than that.

para deploy simepre:
docker login -u luccadibenedetto
-> token

func deploy --image=docker.io/luccadibenedetto/<name of the function>:latest --build=true --namespace=functions



current issue

Error Summary
Error Type: 503 Service Unavailable
Source: Envoy/Kourier Gateway
Key Indicators:
X-Envoy-Overloaded: true
Error message: "upstream connect error or disconnect/reset before headers. reset reason: overflow"
Very high latencies (~19 seconds)
Root Cause:
The gateway (Envoy) is becoming overloaded with too many concurrent requests. When this happens, Envoy's circuit breaker triggers and starts rejecting requests to protect the system from cascading failures. This is likely due to:
Too many requests being sent without adequate backoff/throttling
Default circuit breaker limits being too low for your workload
3. Possible bottleneck in the Knative service scaling
Think of it like a bouncer at a crowded club - when too many people try to enter at once, the bouncer starts turning people away (503 errors) to prevent overcrowding inside.

I need to find the actual correct casue. maybe changiing config is not enough bc the envoy might be missing more resources.
-> find bottleneck
-> see knative metrics . do the components loook like they are working fine?

Other than that, the next step is to automatise the whole benchmark.
the workflow will more or less be:
1. "just up"  and "just install-knative"
2. "just benchmark"
  a. The deployer will deploy a function / a set of functions / an event based chain of functions with different configurations
  b. We trigger the workload generator to send requests to the functions
  c. We run health checks on knative
3. "just collect-results"
4. "just down"


For that I first need to correctly specify the benchmark design.
The event based chain of functions is not done yet.
Eventing flows: https://knative.dev/docs/eventing/flows/

Sequence Spec¶ https://knative.dev/docs/eventing/flows/sequence/sequence-terminal/
Sequence has three parts for the Spec:

Steps which defines the in-order list of Subscribers, aka, which functions are executed in the listed order. These are specified using the messaging.v1.SubscriberSpec just like you would when creating Subscription. Each step should be Addressable.
ChannelTemplate defines the Template which will be used to create Channels between the steps.
Reply (Optional) Reference to where the results of the final step in the sequence are sent to.
Sequence Status¶
Sequence has four parts for the Status:

Conditions which detail the overall Status of the Sequence object
ChannelStatuses which convey the Status of underlying Channel resources that are created as part of this Sequence. It is an array and each Status corresponds to the Step number, so the first entry in the array is the Status of the Channel before the first Step.
SubscriptionStatuses which convey the Status of underlying Subscription resources that are created as part of this Sequence. It is an array and each Status corresponds to the Step number, so the first entry in the array is the Subscription which is created to wire the first channel to the first step in the Steps array.
AddressStatus which is exposed so that Sequence can be used where Addressable can be used. Sending to this address will target the Channel which is fronting the first Step in the Sequence.
//FROM THE EVENTING DOCS
apiVersion: flows.knative.dev/v1
kind: Sequence
metadata:
  name: sequence
spec:
  channelTemplate:
    apiVersion: messaging.knative.dev/v1
    kind: InMemoryChannel
  steps:
    - ref:
        apiVersion: serving.knative.dev/v1
        kind: Service
        name: first
    - ref:
        apiVersion: serving.knative.dev/v1
        kind: Service
        name: second
    - ref:
        apiVersion: serving.knative.dev/v1
        kind: Service
        name: third

the actual event is base64 encoded in the data field of the event.
because we want to benchmark the performance of the knative components, we really dont care about the actual event or the logic that the functions do. 
I suspect that data serialization and deserialization is gonna be the bottleneck, but not sure.
So the functions will just recieve the event and append their name to the data field.

Important for the report:
Developers can create their own sources via CRDs and use the knative duck type to declare them as sources.
https://github.com/knative/eventing/blob/52792ea9874fae8e2cbe1a6387ebe8bb3d6184b3/docs/spec/sources.md#L4
Knative Eventing defines an EventType object to make it easier for consumers to discover the types of events they can consume from Brokers or Channels.

https://knative.dev/docs/eventing/event-registry/

Table with the implemented event sources: https://knative.dev/docs/eventing/sources/#knative-sources
ContainerSource looks like the one I need to generate the events to trigger the event based chain of functions.
The ContainerSource instantiates container image(s) that can generate events until the ContainerSource is deleted. This may be used, for example, to poll an FTP server for new files or generate events at a set time interval. Given a spec.template with at least a container image specified, the ContainerSource keeps a Pod running with the specified image(s). K_SINK (destination address) and KE_CE_OVERRIDES (JSON CloudEvents attributes) environment variables are injected into the running image(s). It is used by multiple other Sources as underlying infrastructure. Refer to the Container Source example for more details.
https://knative.dev/docs/eventing/custom-event-source/containersource/


- test the metrics collector -> done , works fine

- test the knative sequence of functions that process the event. -> done , works fine

- make the deployer able to deploy a sequence of functions that process the event. -> done , works fin
     - for this we need the image !

go run deployer/main.go --action=sequence --image=go-handler-event --name=event-handler

2m29s (x16 over 5m13s)   Warning   TrackerFailed                       Sequence/sequence                               unable to track changes to channel {Kind:InMemoryChannel Namespace:default Name:sequence-kn-sequence-0 UID: APIVersion:messaging.knative.dev/v1 ResourceVersion: FieldPath:} : inmemorychannels.messaging.knative.dev is forbidden: User "system:serviceaccount:knative-eventing:eventing-controller" cannot list resource "inmemorychannels" in API group "messaging.knative.dev" at the cluster scope
2m17s (x16 over 4m51s)   Warning   InternalError                       SinkBinding/event-source-sinkbinding            URL missing in address of Kind = Sequence, Namespace = default, Name = sequence, APIVersion = flows.knative.dev/v1, Group = , Address =


func deploy --image=docker.io/luccadibenedetto/go-handler-event:latest --build=true --namespace=functions

current setup worked for 10 req / s for 1 min. to the sequece:
time=2025-01-08T17:07:38.035Z level=INFO msg=Success target=http://sequence-kn-sequence-0-kn-channel.functions.svc.cluster.local latency=61.911697ms status=202

I saved the logs in home/random
initial implementation of node and container metrics is working in analysis/2.py
todo:
- add cpu and memory in % over total memory and cpu . keep in mind that this depends on the VM size.
- add prometheus metrics - what is happening with each knative component?
- finish up eventing benchark. simple scenario broker / trigger .
- make more preliminary tests with the function sequence, see what is the limit, I suspect  its the inmemorychannel.

- start a rough draft of the report.
just eventing -> deployes the sequence and the workload generator container source.
just destroy-eventing -> destroys the sequence and the workload generator container source.

for the report:
  im using in memory channels here, not kafka or rabbitmq, knative docs say that in memory channels are  not pod ready,
  but the other options are out of scope for this benchmark.


The cloudevent we generate will have counter.
We can compare sent vs received timestaps for each event id.

Eventing scenario 1:
  broker / trigger
  a)
  ContainerSource -> broker -> trigger -> Service
  - container-source-1.yaml points to the broker
  - broker.yaml points to the trigger
  - trigger.yaml points to the service
  - the service is a very simple http server that logs timestamps to a file.
  this way we are interested in the latency of the broker and the trigger.
  b)
  broker / trigger with many triggers
  - we can deploy many triggers and see how they scale.

Eventing scenario 2:
  sequence
  ContainerSource -> sequence -> Service
  - container-source-2.yaml points to the sequence
  - sequence.yaml points to the service
  - the service is a very simple http server that logs timestamps to a file.
  this way we are interested in the latency of the sequence.
  - we can analyse if the number of steps in the sequence is a bottleneck.

Knative Functions - Serving
Scenario 1:
  hit a single function with max throughput.
  - do this with different programming languages and templates.
  - go , python , node , quarkus , springboot , typescript, rust
  - do this with different function configurations.
  specifically interested in the autoscaling strategy. Possible values: "concurrency", "rps", "cpu", "memory"
Scenario 2:
  scale up the number of functions and distribute the load among them.
  - do this with just the go http template.


-	resources

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

TODO rn:

find out default autoscaling strategy. -> idk .(
test new actions of deployer. -> done they work!
Prometheus metrics confirm everything is working fine, and extract metrics via justfile -> still todo
Create justfile pipeline for the benchmark. -> part 1 done  
Part 2 is knative eventing pipeline
1) test all functions:
curl http://localhost:8080 --header "Host:  empty-rust-http.functions.example.com" -> true
curl http://localhost:8080 --header "Host:  empty-quarkus-http.functions.example.com" -> "true"
curl http://localhost:8080 --header "Host:  empty-springboot-http.functions.example.com -> true
curl http://localhost:8080 --header "Host:  empty-go.functions.example.com -> true
curl http://localhost:8080 --header "Host:  empty-ts-http.functions.example.com -> true
curl http://localhost:8080 --header "Host:  empty-py-http.functions.example.com -> True



todo rn:

-> check that the rabbitmq cluster has enough resources.

run scenarios.
find a way to persist the logs of the reciever.

