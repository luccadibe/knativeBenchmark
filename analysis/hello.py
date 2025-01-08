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
    
    # Extract date
    df['date'] = df['timestamp'].dt.date

    # Separate container and node metrics
    container_metrics = df[df['msg'] == 'container metrics'].copy()
    node_metrics = df[df['msg'] == 'node metrics'].copy()

    return container_metrics, node_metrics

def analyze_and_visualize_container_metrics(container_metrics_df):
        """Analyzes container metrics and creates visualizations."""
        
        # Basic statistics for container metrics
        print("\nContainer Metrics Analysis:")
        print(container_metrics_df[['cpu', 'memory_bytes']].describe())

        # Visualize CPU and Memory Usage by Namespace and Container
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

        # Visualize CPU and Memory Usage by Date, Namespace
        plt.figure(figsize=(14, 8))
        sns.boxplot(data=container_metrics_df, x='date', y='cpu', hue='namespace')
        plt.title('CPU Usage by Date, Namespace')
        plt.xticks(rotation=45, ha='right')
        plt.tight_layout()
        save_plot('container_cpu_by_date')

        plt.figure(figsize=(14, 8))
        sns.boxplot(data=container_metrics_df, x='date', y='memory_bytes', hue='namespace')
        plt.title('Memory Usage by Date, Namespace')
        plt.xticks(rotation=45, ha='right')
        plt.tight_layout()
        save_plot('container_memory_by_date')


def analyze_and_visualize_node_metrics(node_metrics_df):
        """Analyzes node metrics and creates visualizations."""
        
        # Basic statistics for node metrics
        print("\nNode Metrics Analysis:")
        print(node_metrics_df[['cpu', 'memory_bytes']].describe())

        # Visualize CPU and Memory Usage by Node
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

        # Visualize CPU and Memory Usage by Date and Node
        plt.figure(figsize=(14, 8))
        sns.boxplot(data=node_metrics_df, x='date', y='cpu', hue='node')
        plt.title('CPU Usage by Date and Node')
        plt.xticks(rotation=45, ha='right')
        plt.tight_layout()
        save_plot('node_cpu_by_date')

        plt.figure(figsize=(14, 8))
        sns.boxplot(data=node_metrics_df, x='date', y='memory_bytes', hue='node')
        plt.title('Memory Usage by Date and Node')
        plt.xticks(rotation=45, ha='right')
        plt.tight_layout()
        save_plot('node_memory_by_date')


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