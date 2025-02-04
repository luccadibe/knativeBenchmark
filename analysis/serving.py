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
    r.timestamp,
    r.target,
    r.is_cold
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

def scenario_1_analysis():
    """
    Scenario 1: 
    Languages: Go, Python, Typescript, Rust, Java (Springboot) , Java (Quarkus)
    RPS: 800, 1000, 1200, 1400
    """
    # Get all scenario 1 requests
    scenario1_df = requests_df[requests_df['scenario'] == "1"]
    
    # Analyze status codes
    status_counts = scenario1_df.groupby(['language', 'rps', 'status']).size().unstack(fill_value=0)
    status_ratios = status_counts.div(status_counts.sum(axis=1), axis=0).round(4) * 100
    
    print("\nStatus Code Distribution by Language and RPS:")
    print("\nCounts:")
    print(status_counts)
    print("\nPercentages:")
    print(status_ratios)
    
    # Filter for successful requests and non-cold starts
    successful_requests = scenario1_df[
        (scenario1_df['status'] == 200) & 
        (~scenario1_df['is_cold'])
    ]
    
    # Calculate 5-number summary grouped by language and RPS
    summary = successful_requests.groupby(['language', 'rps'])['ttfb'].agg([
        'min',
        lambda x: x.quantile(0.25),
        'median',
        lambda x: x.quantile(0.75),
        'max'
    ]).round(2)
    
    summary.columns = ['Min', '25%', 'Median', '75%', 'Max']
    print("\nLatency Summary (ms) by Language and RPS (excluding cold starts):")
    print(summary)

    plot_df = scenario1_df.copy()
    plot_df.set_index('timestamp', inplace=True)
    # Resample numeric data (ttfb) and use first() for categorical data (language)
    plot_df = plot_df.resample('1s').agg({
        'ttfb': 'mean',
        'language': 'first'
    })
    plot_df = plot_df.reset_index()
    
    scenario_1_plot(plot_df)
    
    scenario_1_plot(scenario1_df)
    
def scenario_1_plot(df):
    """ECDF of TTFB for each language and RPS - Grouped in a FacetGrid"""
    print(f"Plotting scenario 1: {df.shape[0]} observations")
    g = sns.FacetGrid(df, col='language', row='rps', hue='language', height=4, aspect=1.5)
    g.map(sns.ecdfplot, 'ttfb')
    plt.savefig('plots/scenario_1_plot.png')
    plt.close()

def scenario_2_analysis():
    """
    Scenario 2 (Cold start analysis): 
    Languages: Go, Python, Typescript, Rust, Java (Springboot) , Java (Quarkus)
    RPS: 0.05
    """
    # Get all scenario 2 requests and filter for cold starts
    scenario2_df = requests_df[requests_df['scenario'] == "2"]
    scenario2_df = scenario2_df[scenario2_df['is_cold'] == True]

    # Correct the language column using the target column
    # example: http://empty-ts-http-0.functions.svc.cluster.local -> ts
    scenario2_df['language'] = scenario2_df['target'].str.split('-').str[-1]
    
    # Analyze status codes
    status_counts = scenario2_df.groupby(['language', 'status']).size().unstack(fill_value=0)
    status_ratios = status_counts.div(status_counts.sum(axis=1), axis=0).round(4) * 100
    
    print("\nStatus Code Distribution by Language:")
    print("\nCounts:")
    print(status_counts)
    print("\nPercentages:")
    print(status_ratios)

    successful_requests = scenario2_df[scenario2_df['status'] == 200]
    # Calculate 5-number summary grouped by language
    summary = successful_requests.groupby(['language'])['ttfb'].agg([
        'min',
        lambda x: x.quantile(0.25),
        'median',
        lambda x: x.quantile(0.75),
        'max'
    ]).round(2)

    print("\nLatency Summary (ms) by Language (only cold starts):")
    print(summary)

    scenario_2_plot(scenario2_df)

def scenario_2_plot(df):
    """ECDF of Cold StartTTFB for each language - Grouped in a FacetGrid"""
    print(f"Plotting scenario 2: {df.shape[0]} observations")
    g = sns.FacetGrid(df, col='language', hue='language', height=4, aspect=1.5)
    g.map(sns.ecdfplot, 'ttfb')
    plt.savefig('plots/scenario_2_plot.png')
    plt.close()

def scenario_3_analysis():
    """
    Scenario 3 Container Concurrency set to 1: 
    Languages: Go, Python, Typescript, Rust, Java (Springboot) , Java (Quarkus)
    RPS: 80, 90, 100
    """
    # Get all scenario 3 requests
    scenario3_df = requests_df[requests_df['scenario'] == "3"]
    scenario3_df = scenario3_df[scenario3_df['is_cold'] == False]

    # Analyze status codes
    status_counts = scenario3_df.groupby(['language', 'rps', 'status']).size().unstack(fill_value=0)
    status_ratios = status_counts.div(status_counts.sum(axis=1), axis=0).round(4) * 100
    
    print("\nStatus Code Distribution by Language and RPS:")
    print("\nCounts:")
    print(status_counts)
    print("\nPercentages:")
    print(status_ratios)

    # Calculate 5-number summary grouped by language and RPS
    summary = scenario3_df.groupby(['language', 'rps'])['ttfb'].agg([
        'min',
        lambda x: x.quantile(0.25),
        'median',
        lambda x: x.quantile(0.75),
        'max'
    ]).round(2)

    print("\nLatency Summary (ms) by Language and RPS (excluding cold starts):")
    print(summary)

    # Resample while maintaining the language information
    plot_df = scenario3_df.copy()
    plot_df.set_index('timestamp', inplace=True)
    # Resample numeric data (ttfb) and use first() for categorical data (language)
    plot_df = plot_df.resample('1s').agg({
        'ttfb': 'mean',
        'language': 'first'
    })
    plot_df = plot_df.reset_index()

    scenario_3_plot(scenario3_df)

def scenario_3_plot(df):
    """ECDF of TTFB for each language and RPS - Grouped in a FacetGrid"""
    print(f"Plotting scenario 3: {df.shape[0]} observations")
    g = sns.FacetGrid(df, col='language', row='rps', hue='language', height=4, aspect=1.5)
    g.map(sns.ecdfplot, 'ttfb')
    plt.savefig('plots/scenario_3_plot.png')
    plt.close()

scenario_1_analysis()
scenario_2_analysis()
scenario_3_analysis()
conn.close()
