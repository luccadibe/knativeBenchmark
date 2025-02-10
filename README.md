# Knative Serving and Eventing Benchmark
This benchmarking orchestrator was tested using Ubuntu 22.04 via WSL2.
## Running the benchmark
To get started, you need the following dependencies:

- [just](https://github.com/casey/just)
- [Go](https://go.dev/doc/install)
- [Python](https://www.python.org/downloads/)
- [uv](https://docs.astral.sh/uv/getting-started/installation/)
- [kubectl](https://kubernetes.io/docs/tasks/tools/#kubectl)
- [Terraform](https://www.terraform.io/downloads)
- [SQLite](https://www.sqlite.org/download.html)

You need to be able to run bash commands as well.



This benchmark is controlled mainly by the justfile in the infrastructure directory.

The justfile allows for running many commands at once, acting as a central wrapper for bash scripting.

You can quickly see all the commands available by running `just` in your terminal.

To start the benchmark, you need to open 2 terminals, go to the infrastructure directory and in one of them run:

```
# Initializes the terraform client
just init
```

```
# Creates the cluster
just up
```
Then, in order to reduce node crashes, you can edit the cluster configuration to reserve resources for the kubelet and the system.
I tried very hard to automate this, but it does not seem to work.

```
export KOPS_STATE_STORE="gs://$(terraform output -raw kops_state_store_bucket_name)"
kops edit cluster
```
Add the following to the spec:
```
spec:
  kubelet:
    kubeReserved:
        cpu: "1"
        memory: "2Gi"
        ephemeral-storage: "1Gi"
    kubeReservedCgroup: "/kube-reserved"
    kubeletCgroups: "/kube-reserved"
    runtimeCgroups: "/kube-reserved"
    systemReserved:
        cpu: "500m"
        memory: "1Gi"
        ephemeral-storage: "1Gi"
    systemReservedCgroup: "/system-reserved"
    enforceNodeAllocatable: "pods,system-reserved,kube-reserved"
```

Then, install the crds and the knative components:
```
# Sets up node taints
just setup-cluster
# Installs the crds and the knative components
just install
```

In the other terminal, run:
```
just top
```
This will start a process that will save the cpu and memory metrics of the cluster to a sqlite database.

Then, to run the benchmarks, run:
```
# Runs the benchmarks
just runserving
```
Three scenarios are run:
- serving-scenario-1: warm latency benchmark. Per language. RPS and languages can be configured in the justfile
- serving-scenario-2: cold start benchmark. Per language.
- serving-scenario-3: warm latency benchmark with container concurrency set to 1. Per language. RPS and languages can be configured in the justfile
Then , the logs are extracted and processed to produce the final results.


After the benchmarks are done, you should stop the metrics process by pressing `ctrl+c` in the terminal where you ran `just top`.

Then, you can combine the metrics db with the benchmark db by running:
```
# From the infrastructure directory
sqlite3 ../data/benchmark.db "ATTACH DATABASE 'metrics.db' AS metrics;"
sqlite3 ../data/benchmark.db "CREATE TABLE node_metrics as select * from metrics.node_metrics;"
sqlite3 ../data/benchmark.db "CREATE TABLE pod_metrics as select * from metrics.pod_metrics;"
sqlite3 ../data/benchmark.db "DETACH DATABASE metrics;"
```

All "ttfb" values are in milliseconds.



To store prometheus metrics (optional):
port forward the prometheus service
```
kubectl port-forward svc/prometheus-prometheus-kube-prometheus-prometheus 9090:9090 -n metrics
```

then create a snapshot:
```
curl -XPOST http://localhost:9090/api/v1/admin/tsdb/snapshot
```

then copy the snapshot to the local directory:

kubectl cp prometheus-prometheus-kube-prometheus-prometheus-0:/prometheus/snapshots/<name_of_snapshot> <local_directory> -n metrics

then spin up a local prometheus server and import the metrics.
first create a simple prometheus.yml file inside the directory where you want to store the metrics:
```
scrape_configs:
  - job_name: 'prometheus'
    static_configs:
      - targets: ['localhost:9090']
```
then run the following command:
```
docker run -d \
  --name prometheus \
  -p 9090:9090 \
  -v $(pwd)/<name_of_snapshot>:/prometheus/<name_of_snapshot> \
  -v $(pwd)/prometheus.yml:/etc/prometheus/prometheus.yml \
  prom/prometheus \
  --config.file=/etc/prometheus/prometheus.yml \
  --storage.tsdb.path=/prometheus \
  --storage.tsdb.retention.time=1000d \
  --web.enable-lifecycle
```


To remove the infrastructure:
```
just down
```

## Implementation
The workload generator entrypoint is in `cmd/workload-generator/main.go`.
It is containerized, the dockerfile is in `Dockerfile.wg`.
It is deployed to the cluster via a kubernetes deployment.
This container has the binary which takes a config file and flags as input.
Additionally, it has all the config files which are in the `experiments` directory.
To run different benchmarks, we trigger the program via kubectl exec.
Example:
```
kubectl exec -it $(kubectl get pods -o jsonpath="{.items[0].metadata.name}" -n workload-generator) -n workload-generator -- ./workload-generator --config=serving-scenario-2-go.yaml --prefix=serving-scenario-1_all --cold-start=true
```
This runs the scenario 2 and prefixes the logs with `serving-scenario-1_all` (Because the config file triggers the endpoints of all the languages).

The default timeout for http requests is 30 seconds.

The workload generator has a cloud-event mode to generate cloud-events for the eventing benchmarks.
Due to time constraints, the eventing benchmark is not ran by default.

## Important notes 
### Eventing
In the eventing benchmark, there is a possibility that the containersource (workload-generator) may be stuck pending (with one pod running without a key environment variable set called K_SINK, that must be set by knative) .
I suspect that this is due to a bug in the knative eventing controller.
Please check the replicasets and delete the one that is NOT pending. The other one should run with the environment variable set after a few seconds.

