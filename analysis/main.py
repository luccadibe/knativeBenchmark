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
    e.language,
    e.scenario,
    e.concurrency,
    e.rps,
    r.status,
    r.ttfb,
    r.timestamp
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
# WHERE node_name = 'nodes-europe-west1-b-s5tc' 
# thats the node where knative is running
# Load data into pandas
requests_df = pd.read_sql_query(requests_query, conn)
node_metrics_df = pd.read_sql_query(node_metrics_query, conn)

# Convert timestamps to datetime with appropriate formats
requests_df['timestamp'] = pd.to_datetime(requests_df['timestamp'], format='ISO8601')
node_metrics_df['timestamp'] = pd.to_datetime(node_metrics_df['timestamp'], format='ISO8601')

# Ensure all timestamps are in UTC
node_metrics_df['timestamp'] = node_metrics_df['timestamp'].dt.tz_convert('UTC')

# Calculate median TTFB grouped by experiment configuration and status
median_ttfb = requests_df.groupby(
    ['experiment_id', 'language', 'scenario', 'concurrency', 'rps', 'status']
)['ttfb'].median().reset_index()

print(f"\n Node metrics going from {node_metrics_df['timestamp'].min()} to {node_metrics_df['timestamp'].max()}")
print(f"\n Requests going from {requests_df['timestamp'].min()} to {requests_df['timestamp'].max()}")
print(f"\n Amount of requests: {requests_df.shape[0]}")

# Print the results
print("\nMedian TTFB by experiment configuration and status:")
print(median_ttfb.to_string(index=False))



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
    plt.savefig('plots/node_metrics.png')
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


nodes_metrics_plot()
#requests_node_metrics_plot()
#ttfb_rolling_mean_plot()
# Close the database connection
conn.close()
