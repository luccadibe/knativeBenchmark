This text file contains all of the exported metrics by my knative system on my k8s cluster.
I am in the process of designing a benchmark with the goal of measuring the performance of knative under different scenarios.
For example, I will do maximum throughput tests. Because knative has many components in its architecture (like the Activator service , the Autoscaler and all of the Knative Eventing components) I want to do a drill-down measurement so i can see which component is the slowest / how much actual time each component needs to process requests.
The list of metrics is comically large. Which metrics specifically do you think will be helpful? how should i go about processing them and correctly storing them for analysis later?


Okay, this is a great starting point for a detailed Knative benchmark! You're right, the sheer number of metrics can be overwhelming, but we can definitely narrow it down to the most relevant ones for performance analysis, especially when it comes to individual component breakdown.

Here's a breakdown of helpful metrics, how to process them, and ways to store them for later analysis:

**Key Metric Categories & Specific Metrics**

We'll focus on these categories, with specific metric examples that are highly informative:

1.  **Request Latency (Component Timing):**  These will show you where time is spent within your Knative system.
    *   `knative_dev_eventing_controller_client_latency_seconds`: Latency of Kubernetes API requests from eventing controller.
        *   `knative_dev_eventing_eventing_webhook_client_latency_seconds`: Latency of Kubernetes API requests from webhook for eventing.
        *   `knative_dev_internal_serving_controller_client_latency_seconds`: Latency of Kubernetes API requests from serving controller.
        *   `knative_dev_internal_serving_autoscaler_client_latency_seconds`: Latency of Kubernetes API requests from autoscaler.
        *   `knative_dev_internal_eventing_job_sink_client_latency_seconds`: Latency of K8s API requests from jobsink.
            *   *These are all good "entry-level" metrics that show how long different knative components are taking to interact with K8s.*
    *   `knative_dev_internal_serving_activator_request_latencies_milliseconds`: Response times of the Activator itself.
        *   *This will help understand the request latency inside the serving component.*
2.  **Reconcile Latency & Counts (Control Loop Timing):** How long reconciliation cycles take, broken down by component.
    *   `knative_dev_eventing_controller_reconcile_latency_milliseconds`: How long reconcile operations take within the eventing controller.
    *   `knative_dev_internal_serving_controller_reconcile_latency_milliseconds`: How long reconcile operations take within the serving controller
     *   `knative_dev_serving_autoscaler_reconcile_latency_milliseconds`: How long reconcile operations take within the autoscaler.
        *   *These are crucial for control plane performance. Look for variations and long times.*
    *  `knative_dev_eventing_controller_reconcile_count_total`: number of reconcile operations, broken down by component and success/failure.
    *   `knative_dev_internal_serving_controller_reconcile_count_total`: number of reconcile operations, broken down by component and success/failure.
    *   `knative_dev_serving_autoscaler_reconcile_count_total`: Number of reconcile operations, broken down by success/failure.
        *    *Use these counts to compare against latencies to find if certain components are handling a larger set of operations.*
3.  **Workqueue Metrics (Internal Queue Analysis):** How well the workqueues are working.
    *   `knative_dev_serving_autoscaler_workqueue_queue_latency_seconds`: How long items wait in the Autoscaler's workqueue
    *   `knative_dev_internal_serving_controller_workqueue_queue_latency_seconds`: How long items wait in the Serving controller's workqueue
        *   `knative_dev_eventing_controller_workqueue_queue_latency_seconds`: How long items wait in the Eventing controller's workqueue.
        *    *High queue latency means components are struggling to keep up with incoming events/requests.*
    *    `knative_dev_serving_autoscaler_workqueue_work_duration_seconds`: How long processing an item takes from Autoscaler's workqueue.
        *    `knative_dev_internal_serving_controller_workqueue_work_duration_seconds`: How long processing an item takes from Serving controller's workqueue
        *   `knative_dev_eventing_controller_workqueue_work_duration_seconds`: How long processing an item takes from Eventing controller's workqueue.
        *    *High work duration time combined with high queue latency shows that the component is getting overloaded.*
    *    `knative_dev_serving_autoscaler_workqueue_unfinished_work_seconds`: How long workqueue items have been in flight (Autoscaler)
     *   `knative_dev_internal_serving_controller_workqueue_unfinished_work_seconds`: How long workqueue items have been in flight (Serving controller)
        *  `knative_dev_eventing_controller_workqueue_unfinished_work_seconds`: How long workqueue items have been in flight (Eventing controller)
        *     *Shows the load/congestion in a component's workqueue.*
    * `knative_dev_serving_autoscaler_workqueue_adds_total`: Total number of adds handled by workqueue, broken down by the type of queue.
        *`knative_dev_internal_serving_controller_workqueue_adds_total`: Total number of adds handled by workqueue, broken down by the type of queue.
        *   `knative_dev_eventing_controller_workqueue_adds_total`: Total number of adds handled by workqueue, broken down by the type of queue.
        *  *Compare these numbers with the latency numbers.*
    *   `knative_dev_serving_autoscaler_workqueue_retries_total`: Total number of retries handled by the workqueue, broken down by queue.
    *   `knative_dev_internal_serving_webhook_workqueue_retries_total`: Total number of retries handled by workqueue, broken down by queue.
        *   *Retries indicate errors or failures, which can impact overall throughput.*
4.  **Autoscaling Metrics (Capacity):** How well the autoscaler is scaling the system under load.
     *   `knative_dev_serving_autoscaler_actual_pods_ratio`:  Number of pods currently running for a service or component.
     *   `knative_dev_serving_autoscaler_not_ready_pods_ratio`: Number of pods that are not ready.
     *   `knative_dev_serving_autoscaler_pending_pods_ratio`: Number of pods that are pending.
          *  *Track how well autoscaling is responding under different throughput loads. The difference between `actual`, `not_ready` and `pending` will tell you if the system is keeping up.*
    *   `knative_dev_serving_autoscaler_panic_mode_ratio`: Indicates if the autoscaler is in panic mode (adjusting rapidly).
         *     *Panic mode may point towards an issue. In normal high throughput scenarios, the autoscaler should be reacting to the load, not "panicking".*
5.  **Request Counts (Throughput):** How many requests/operations the component is handling.
     *   `knative_dev_internal_serving_activator_request_count_total`: total requests to the activator.
    *  `knative_dev_internal_serving_webhook_request_count_total`: total requests to the webhook.
    *  `knative_dev_internal_serving_controller_client_results_total`: total K8s API request from serving controller, broken down by status code.
    *   `knative_dev_eventing_controller_client_results_total`: Total K8s API request from eventing controller.
        * `knative_dev_eventing_eventing_webhook_client_results_total`: total K8s API request from webhook for eventing.
        *  *Use these to determine overall throughput for each component.*

**Processing & Storing Metrics**

1.  **Scraping:**  You already have the metrics in a text format. This is useful for quick viewing, but not for long-term storage and analysis. You'll want a time-series database like Prometheus to gather, store, and query these metrics over time.

2.  **Prometheus (Recommended):**
    *   **Setup:** If you aren't already using it, deploy Prometheus to your Kubernetes cluster. Prometheus excels at scraping metrics from targets, storing them as time-series data, and enabling querying via PromQL.
    *   **Configuration:** Configure Prometheus to scrape from the pods exposing metrics.
        *You will likely need to configure service monitors to target the pods that emit these metrics. Service monitors are CRDs that Prometheus uses to get info about the target metrics. A service monitor will typically include:*
            * A selector for the Pods
            * An optional namespace selector
            * A target port and path for scraping metrics from
    *   **PromQL:** Learn PromQL, Prometheus's query language. It's perfect for calculating rates, averages, percentiles, and creating graphs.

3.  **Data Storage:** Prometheus itself is the primary storage engine. However, if you anticipate massive data volumes, have more stringent storage needs, or want the ability to query from multiple locations, consider using:
    *   **Thanos/Cortex:** These scale out the Prometheus querying and storage layers. Good options for very large deployments
    *   **Long-Term Storage Backends:** Prometheus supports integration with various long-term storage solutions (like Google Cloud Storage, AWS S3, etc). Good if you don't want to self-host your storage.

4.  **Processing:**
    *   **Initial Filtering:** Start by focusing on the metrics I mentioned earlier. Don't try to analyze everything at once.
    *   **PromQL Queries:**
        *   Use `rate()` to calculate per-second rates (e.g., requests per second).
        *   Use `histogram_quantile()` to get latencies at specific percentiles (p50, p90, p99).
        *   Use `sum()` and `by()` to group data by labels (e.g., average latency per component).
        *   Use `increase()` on a counter to get the total count between two time points (e.g. total reconcile operations in the last 5 minutes).
        *   Use `irate()` on a counter to calculate a per-second instant rate
        *   `sum(irate(knative_dev_internal_serving_activator_request_count_total[5m]))` : This will measure the request volume at each second, and sum it up for 5 minutes.
        *   `histogram_quantile(0.99, sum(rate(knative_dev_internal_serving_activator_request_latencies_milliseconds_bucket[5m])) by (le))` : calculates the p99 latencies for activator over 5m intervals.
        *   `sum(increase(knative_dev_serving_autoscaler_reconcile_count_total[5m])) by (reconciler, success)` : number of reconciliation cycles performed for the autoscaler over 5m intervals, broken down by success/failure.

5.  **Analysis:**
    *   **Dashboards:** Use Grafana to visualize Prometheus metrics and create interactive dashboards.
        *   Create dashboards to show request latency and throughput on a per-component basis.
        *   Create dashboards that monitor the metrics you think might be important.
        *   Combine graphs that show rate and latency of the same component side-by-side.
        *    Build a dashboard that shows the autoscaler and the pods its creating (pods_pending, pods_ready).
    *  **Statistical Analysis:** Look at trends, spikes, outliers, and compare results under different load conditions.
        *   Calculate averages, means, medians, and standard deviations.
        *   Compare throughput at different scale settings.

**How to Start**

1.  **Prometheus Setup:** Get Prometheus running in your cluster and configure it to scrape key metrics from knative pods (autoscaler, activator, controllers, etc.).
2. **Define Service Monitors**: define kubernetes service monitors for your pods. You can do this in a declarative way by applying service monitors yamls.
3.  **Initial Queries:** Start with a small set of PromQL queries for latency, throughput, and workqueue depth.
4.  **Grafana:** Set up a Grafana dashboard with the basic queries so you have initial results.
5.  **Testing:** Design a test scenario and run it (e.g., create a service under high traffic) and observe initial metrics. Refine your queries to get the required granularity of information.
6.  **Expand:**
    *    Expand your dashboard and data collection to include more metrics as you get more proficient.
    *   Start stress testing different components, or whole knative services.
7.  **Analysis:** Analyse the data for bottlenecks and weak spots.

**Important Considerations**

*   **Test Environment:**  Use a dedicated environment for benchmarking. Don't do this in your production cluster.
*   **Realistic Load:** Your benchmark should try to simulate realistic traffic patterns, including burst traffic.
*   **Configuration:** Explore Knative's configuration options and see how they affect performance.
*   **Resource Limits:** Pay attention to resource consumption during your tests (CPU, Memory, etc.).

Let me know if you'd like more specific PromQL examples or further clarification. Good luck with your benchmarking!
