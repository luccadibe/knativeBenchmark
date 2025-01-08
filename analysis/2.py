import pandas as pd
import seaborn as sns
import matplotlib.pyplot as plt
import json
from datetime import datetime
import os

# Create plots directory if it doesn't exist
PLOTS_DIR = 'plots'
os.makedirs(PLOTS_DIR, exist_ok=True)

def save_plot(name):
    """Helper function to save plots with consistent naming"""
    plt.savefig(os.path.join(PLOTS_DIR, f"{name}.png"), bbox_inches='tight', dpi=300)
    plt.close()

def parse_log_line(line):
    """Parses a single log line and returns a dict."""
    try:
      return json.loads(line)
    except json.JSONDecodeError:
        return None  # skip malformed lines


def load_and_preprocess_data(log_file):
    """Loads the log data, preprocesses and returns as a DataFrame."""
    
    with open(log_file, 'r') as f:
        log_lines = f.readlines()
    
    records = [parse_log_line(line) for line in log_lines if parse_log_line(line)]
    
    df = pd.DataFrame(records)

    # Convert timestamp to datetime objects for time-based analysis
    df['timestamp'] = pd.to_datetime(df['timestamp'])

    # Separate container and node metrics
    container_metrics = df[df['msg'] == 'container metrics'].copy()
    node_metrics = df[df['msg'] == 'node metrics'].copy()

    return container_metrics, node_metrics

def analyze_and_visualize_container_metrics(container_metrics_df):
    """Analyzes container metrics and creates time series visualizations."""
    
    print("\nContainer Metrics Analysis:")
    print(container_metrics_df[['cpu', 'memory_bytes']].describe())
    
    # Aggregate CPU and memory usage over time for all containers
    container_metrics_df_agg = container_metrics_df.groupby('timestamp').agg({'cpu': 'sum', 'memory_bytes': 'sum'}).reset_index()
    
    # CPU Time Series
    plt.figure(figsize=(14, 6))
    plt.plot(container_metrics_df_agg['timestamp'], container_metrics_df_agg['cpu'])
    plt.title('Aggregated CPU Usage Over Time (All Containers)')
    plt.xlabel('Time')
    plt.ylabel('Total CPU (milli cores)')
    plt.xticks(rotation=45, ha='right')
    plt.tight_layout()
    save_plot('container_agg_cpu_timeseries')
    
    # Memory Time Series
    plt.figure(figsize=(14, 6))
    plt.plot(container_metrics_df_agg['timestamp'], container_metrics_df_agg['memory_bytes'])
    plt.title('Aggregated Memory Usage Over Time (All Containers)')
    plt.xlabel('Time')
    plt.ylabel('Total Memory (bytes)')
    plt.xticks(rotation=45, ha='right')
    plt.tight_layout()
    save_plot('container_agg_memory_timeseries')
    
    # Boxplots for overview
    plt.figure(figsize=(14, 8))
    sns.boxplot(data=container_metrics_df, x='namespace', y='cpu', hue='container')
    plt.title('CPU Usage by Namespace and Container')
    plt.xticks(rotation=45, ha='right')
    plt.tight_layout()
    save_plot('container_cpu_by_namespace')

    plt.figure(figsize=(14, 8))
    sns.boxplot(data=container_metrics_df, x='namespace', y='memory_bytes', hue='container')
    plt.title('Memory Usage by Namespace and Container')
    plt.xticks(rotation=45, ha='right')
    plt.tight_layout()
    save_plot('container_memory_by_namespace')
    

def analyze_and_visualize_node_metrics(node_metrics_df):
        """Analyzes node metrics and creates time series visualizations."""

        print("\nNode Metrics Analysis:")
        print(node_metrics_df[['cpu', 'memory_bytes']].describe())
        
        # Aggregate CPU and memory usage over time for all nodes
        node_metrics_df_agg = node_metrics_df.groupby('timestamp').agg({'cpu': 'sum', 'memory_bytes': 'sum'}).reset_index()
        
        # CPU Time Series
        plt.figure(figsize=(14, 6))
        for node, group in node_metrics_df.groupby('node'):
          plt.plot(group['timestamp'], group['cpu'], label=node)
        plt.title('CPU Usage Over Time (All Nodes)')
        plt.xlabel('Time')
        plt.ylabel('Total CPU (milli cores)')
        plt.xticks(rotation=45, ha='right')
        plt.legend(loc='upper left')
        plt.tight_layout()
        save_plot('node_cpu_timeseries')
        
        # Memory Time Series
        plt.figure(figsize=(14, 6))
        for node, group in node_metrics_df.groupby('node'):
          plt.plot(group['timestamp'], group['memory_bytes'], label=node)
        plt.title('Memory Usage Over Time (All Nodes)')
        plt.xlabel('Time')
        plt.ylabel('Total Memory (bytes)')
        plt.xticks(rotation=45, ha='right')
        plt.legend(loc='upper left')
        plt.tight_layout()
        save_plot('node_memory_timeseries')

        # Boxplots for overview
        plt.figure(figsize=(12, 6))
        sns.barplot(data=node_metrics_df, x='node', y='cpu')
        plt.title('CPU Usage by Node')
        plt.xticks(rotation=45, ha='right')
        plt.tight_layout()
        save_plot('node_cpu_usage')

        plt.figure(figsize=(12, 6))
        sns.barplot(data=node_metrics_df, x='node', y='memory_bytes')
        plt.title('Memory Usage by Node')
        plt.xticks(rotation=45, ha='right')
        plt.tight_layout()
        save_plot('node_memory_usage')



def main():
    """Main function to execute the script."""
    log_file = 'metrics.log'
    container_metrics, node_metrics = load_and_preprocess_data(log_file)

    if not container_metrics.empty:
        analyze_and_visualize_container_metrics(container_metrics)

    if not node_metrics.empty:
      analyze_and_visualize_node_metrics(node_metrics)

if __name__ == "__main__":
    main()