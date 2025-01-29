import sqlite3
import pandas as pd
import matplotlib.pyplot as plt
import seaborn as sns
from datetime import datetime

# Connect to the database
conn = sqlite3.connect('benchmark.db')

# Query for requests data
requests_query = """
SELECT 
    e.id as experiment_id,
    e.scenario,
    e.triggers,
    r.status,
    r.ttfb,
    r.timestamp,
    r.event_id
FROM experiments e
JOIN requests r ON e.id = r.experiment_id
"""

# Query for node metrics
node_metrics_query = """
SELECT 
    timestamp,
    node_name,
    cpu_percentage
FROM node_metrics
ORDER BY timestamp
"""

# Query for pod metrics
pod_metrics_query = """
SELECT 
    timestamp,
    container_name,
    pod_name,
    node_name,
    cpu_percentage,
    memory_percentage,
    cpu_usage,
    memory_usage
FROM pod_metrics
ORDER BY timestamp
"""

# Events data is stored in ./logs/events.csv
# columns are event_id,timestamp
# example: 7304328921599469878,2025-01-29T13:43:37.792661715Z
events_df = pd.read_csv('./logs/events.csv', names=['event_id', 'timestamp'], header=0)
events_df['event_id'] = events_df['event_id'].astype(str)
events_df['timestamp'] = pd.to_datetime(events_df['timestamp'], format='%Y-%m-%dT%H:%M:%S.%fZ').dt.tz_localize('UTC')


requests_df = pd.read_sql_query(requests_query, conn)
requests_df['event_id'] = requests_df['event_id'].astype(str)
requests_df['timestamp'] = pd.to_datetime(requests_df['timestamp'], format='ISO8601')


node_metrics_df = pd.read_sql_query(node_metrics_query, conn)
node_metrics_df['timestamp'] = pd.to_datetime(node_metrics_df['timestamp'], format='ISO8601')
node_metrics_df['timestamp'] = node_metrics_df['timestamp'].dt.tz_convert('UTC')

pod_metrics_df = pd.read_sql_query(pod_metrics_query, conn)
pod_metrics_df['timestamp'] = pd.to_datetime(pod_metrics_df['timestamp'], format='ISO8601')
pod_metrics_df['timestamp'] = pod_metrics_df['timestamp'].dt.tz_convert('UTC')

print(f"\n Node metrics going from {node_metrics_df['timestamp'].min()} to {node_metrics_df['timestamp'].max()}")
print(f"\n Requests going from {requests_df['timestamp'].min()} to {requests_df['timestamp'].max()}")
print(f"\n Amount of requests: {requests_df.shape[0]}")

# Calculate summary statistics for TTFB grouped by triggers
summary_ttfb = requests_df.groupby(
    ['experiment_id', 'triggers', 'status']
)['ttfb'].describe().reset_index()
print("\nTTFB Summary Statistics by experiment configuration and status:")
print(summary_ttfb.to_string(index=False))



def big_plot():
    # Create the overlay plot with seaborn
    fig, ax1 = plt.subplots(figsize=(12, 6))

    # Plot CPU utilization
    sns.lineplot(data=node_metrics_df, x='timestamp', y='cpu_percentage', 
                color='blue', ax=ax1, label='CPU Utilization')
    ax1.set_ylabel('CPU Utilization (%)', color='blue')
    ax1.tick_params(axis='y', labelcolor='blue')

    # Create second y-axis for TTFB
    ax2 = ax1.twinx()
    # Calculate rolling mean for TTFB to smooth the line
    requests_df['ttfb_rolling'] = requests_df['ttfb'].rolling(window=100).mean()
    sns.lineplot(data=requests_df, x='timestamp', y='ttfb_rolling', 
                color='red', ax=ax2, label='TTFB (rolling avg)')
    ax2.set_ylabel('TTFB (ms)', color='red')
    ax2.tick_params(axis='y', labelcolor='red')

    # Improve x-axis readability
    plt.xticks(rotation=45)

    # Add title
    plt.title('CPU Utilization vs TTFB Over Time')

    # Add combined legend
    lines1, labels1 = ax1.get_legend_handles_labels()
    lines2, labels2 = ax2.get_legend_handles_labels()
    ax1.legend(lines1 + lines2, labels1 + labels2, loc='upper left')

    # Adjust layout to prevent label cutoff
    fig.tight_layout()

    # Save the plot
    plt.savefig('cpu_vs_ttfb.png')
    plt.close()


def node_metrics_plot():
    sns.lineplot(data=node_metrics_df, x='timestamp', y='cpu_percentage', hue='node_name')
    time_now = datetime.now().strftime("%Y-%m-%d_%H-%M-%S")
    plt.savefig(f'plots/node_metrics_{time_now}.png')
    plt.close()

def nodes_metrics_plot():
    sns.lineplot(data=node_metrics_df, x='timestamp', y='cpu_percentage', hue='node_name')
    plt.savefig('plots/nodes_metrics.png')
    plt.close()

def ttfb_rolling_mean_plot():
    print("Sorting data...")
    requests_df_sorted = requests_df.sort_values('timestamp')
    
    # Calculate time difference between samples
    avg_time_diff = requests_df_sorted['timestamp'].diff().mean()
    print(f"\nAverage time between samples: {avg_time_diff}")
    
    # Use a window that represents about 1 second of data
    window_size = int(1 / avg_time_diff.total_seconds())
    print(f"Using window size of {window_size} samples (≈1 second of data)")
    
    print("Calculating rolling mean...")
    requests_df_sorted['ttfb_rolling'] = requests_df_sorted['ttfb'].rolling(
        window=window_size,
        min_periods=1
    ).mean()
    
    # Downsample to approximately 1000 points for plotting
    downsample_size = len(requests_df_sorted) // 1000
    plot_data = requests_df_sorted.iloc[::downsample_size]
    
    print(f"Plotting {len(plot_data)} points...")
    plt.figure(figsize=(12, 6))
    sns.lineplot(
        data=plot_data, 
        x='timestamp', 
        y='ttfb_rolling', 
        color='red'
    )
    plt.savefig('plots/ttfb_rolling_mean.png')
    plt.close()

def requests_node_metrics_plot():
    print("Preparing data...")
    # Sort and calculate rolling mean for requests data
    requests_df_sorted = requests_df.sort_values('timestamp')
    avg_time_diff = requests_df_sorted['timestamp'].diff().mean()
    window_size = int(1 / avg_time_diff.total_seconds())
    print(f"Using window size of {window_size} samples (≈1 second of data)")
    
    print("Calculating rolling mean...")
    requests_df_sorted['ttfb_rolling'] = requests_df_sorted['ttfb'].rolling(
        window=window_size,
        min_periods=1
    ).mean()
    
    # Downsample only the requests data
    requests_downsample = len(requests_df_sorted) // 1000
    plot_requests = requests_df_sorted.iloc[::requests_downsample]
    
    print(f"Plotting {len(plot_requests)} request points and {len(node_metrics_df)} metric points...")
    
    # Create the plot
    fig, ax1 = plt.subplots(figsize=(12, 6))
    ax2 = ax1.twinx()
    
    # Plot CPU percentage on left axis (all points)
    sns.lineplot(
        data=node_metrics_df, 
        x='timestamp', 
        y='cpu_percentage', 
        ax=ax1, 
        color='blue',
        label='CPU Usage'
    )
    
    # Plot TTFB on right axis (downsampled)
    sns.lineplot(
        data=plot_requests, 
        x='timestamp', 
        y='ttfb_rolling', 
        ax=ax2, 
        color='red',
        label='TTFB (rolling avg)'
    )
    
    # Improve labels and formatting
    ax1.set_xlabel('Time')
    ax1.set_ylabel('CPU Usage (%)', color='blue')
    ax2.set_ylabel('TTFB (ms)', color='red')
    
    # Rotate x-axis labels for better readability
    plt.xticks(rotation=45)
    
    # Add title
    plt.title('CPU Usage vs TTFB Over Time')
    
    # Add combined legend
    lines1, labels1 = ax1.get_legend_handles_labels()
    lines2, labels2 = ax2.get_legend_handles_labels()
    ax1.legend(lines1 + lines2, labels1 + labels2, loc='upper left')
    
    # Adjust layout to prevent label cutoff
    plt.tight_layout()
    
    plt.savefig('plots/requests_node_metrics.png')
    plt.close()
    
    print("Plot saved successfully!")

def process_events():
    # The basic Idea here is that the timestamp of the event in events.csv is when the event was finished processing.
    # The timestamp of the request is the time when rabbitmq sent the response that the event started processing.
    # So the difference is the time it took to process the event.
    # Calculate summary statistics for total event processing time.
    # Calculating true event processing per second is also interesting.
    # The amount of triggers is in the column triggers in the experiments table.

    # Convert timestamps to datetime objects (format: 2025-01-29T13:43:37.792661715Z)
    events_df['timestamp'] = pd.to_datetime(events_df['timestamp'], format='%Y-%m-%dT%H:%M:%S.%fZ')
    
    # Merge requests and events data on event_id
    merged_df = pd.merge(
        requests_df,
        events_df,
        left_on='event_id', 
        right_on='event_id',
        suffixes=('_request', '_completion')
    )
    
    # Calculate processing time for each event
    merged_df['processing_time'] = (
        merged_df['timestamp_completion'] - merged_df['timestamp_request']
    ).dt.total_seconds()
    
    # Group by experiment to get summary statistics
    summary_stats = merged_df.groupby('experiment_id').agg({
        'processing_time': ['mean', 'std', 'min', 'max', 'count'],
        'triggers': 'first'  # Get number of triggers for each experiment
    }).round(3)
    
    # Calculate events processed per second
    for idx in summary_stats.index:
        experiment_data = merged_df[merged_df['experiment_id'] == idx]
        duration = (experiment_data['timestamp_completion'].max() - 
                   experiment_data['timestamp_request'].min()).total_seconds()
        events_per_second = len(experiment_data) / duration
        summary_stats.loc[idx, ('events_per_second', '')] = round(events_per_second, 3)
    
    print("\nEvent Processing Summary Statistics:")
    print(summary_stats.to_string())
    
    # Create visualization of processing times
    plt.figure(figsize=(10, 6))
    sns.boxplot(data=merged_df, x='experiment_id', y='processing_time')
    plt.title('Event Processing Time Distribution by Experiment')
    plt.xlabel('Experiment ID')
    plt.ylabel('Processing Time (seconds)')
    plt.xticks(rotation=45)
    plt.tight_layout()
    plt.savefig('plots/event_processing_times.png')
    plt.close()

    # Visualisation of knative's cpu and memory usage (per pod in knative-eventing namespace)
    print("Preparing data for visualization...")
    # Get the node where the eventing pods are running
    eventing_node = pod_metrics_df[pod_metrics_df['container_name'] == 'eventing-controller'].iloc[0]['node_name']


    # Get all of the pod metrics for the eventing node
    eventing_node_metrics = pod_metrics_df[pod_metrics_df['node_name'] == eventing_node]

    # Set the time index for resampling
    eventing_node_metrics = eventing_node_metrics.set_index('timestamp')
    merged_df_time = merged_df.set_index('timestamp_request')

    # Calculate rolling mean for event processing times
    print("Resampling event processing times...")
    plot_events = merged_df_time['processing_time'].resample('0.1S').mean()


    print(f"Plotting {len(eventing_node_metrics)} metric points and {len(plot_events)} event points...")
    
    fig, ax1 = plt.subplots(figsize=(12, 6))
    ax2 = ax1.twinx()

    # Plot the cpu usage
    sns.lineplot(data=eventing_node_metrics, x='timestamp', y='cpu_percentage',hue="pod_name", ax=ax1, color='blue')
    ax1.set_ylabel('CPU Usage (%)', color='blue')
    ax1.tick_params(axis='y', labelcolor='blue')

    # Plot the event processing times
    # Plot the event processing times (resampled)
    plot_events.plot(ax=ax2, color='red', label='Event Processing Time')
    ax2.set_ylabel('Event Processing Time (seconds)', color='red')
    ax2.tick_params(axis='y', labelcolor='red')

    plt.tight_layout()
    plt.savefig('plots/eventing_node_metrics.png')
    plt.close()

def plot_cpu_usage(node_name, pod_filter_out = []):
    
    plt.figure(figsize=(20, 10))
    
    # Filter the dataframe to only include pods from the specified node
    pod_metrics_df_filtered = pod_metrics_df[pod_metrics_df['node_name'] == node_name]
    
    # Filter out pods whose names contain any of the strings in pod_filter_out
    for pod in pod_filter_out:
        pod_metrics_df_filtered = pod_metrics_df_filtered[~pod_metrics_df_filtered['pod_name'].str.contains(pod)]
    
    # Plot the filtered data
    sns.lineplot(data=pod_metrics_df_filtered, x='timestamp', y='cpu_percentage', hue="pod_name")
    
    # Move legend outside of plot
    plt.legend(bbox_to_anchor=(1.05, 1), loc='upper left', borderaxespad=0.)
    
    # Adjust layout to prevent legend cutoff
    plt.tight_layout()
    
    plt.savefig(f'plots/cpu_usage_{node_name}.png', bbox_inches='tight', dpi=300)
    plt.close()

#nodes_metrics_plot()
#process_events()
plot_cpu_usage('nodes-europe-west1-b-xw95', ['trigger', 'cilium', "csi", "coredns"])
plot_cpu_usage('nodes-europe-west1-b-vmpb', ['trigger', 'cilium', "csi", "coredns"])
#requests_node_metrics_plot()
#ttfb_rolling_mean_plot()
# Close the database connection
conn.close()