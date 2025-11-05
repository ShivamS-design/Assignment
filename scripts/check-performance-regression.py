#!/usr/bin/env python3

import sys
import re
import json
import requests
from typing import Dict, List, Tuple

class PerformanceChecker:
    def __init__(self, baseline_url: str = None):
        self.baseline_url = baseline_url
        self.thresholds = {
            'execution_time': 1.2,  # 20% regression threshold
            'memory_usage': 1.15,   # 15% regression threshold
            'throughput': 0.85      # 15% degradation threshold
        }
    
    def parse_benchmark_results(self, file_path: str) -> Dict[str, Dict[str, float]]:
        """Parse Go benchmark results"""
        results = {}
        
        with open(file_path, 'r') as f:
            content = f.read()
        
        # Parse benchmark lines: BenchmarkName-8 1000 1234 ns/op 567 B/op 8 allocs/op
        pattern = r'Benchmark(\w+)-\d+\s+(\d+)\s+(\d+)\s+ns/op\s+(\d+)\s+B/op\s+(\d+)\s+allocs/op'
        
        for match in re.finditer(pattern, content):
            name = match.group(1)
            iterations = int(match.group(2))
            ns_per_op = int(match.group(3))
            bytes_per_op = int(match.group(4))
            allocs_per_op = int(match.group(5))
            
            results[name] = {
                'iterations': iterations,
                'ns_per_op': ns_per_op,
                'bytes_per_op': bytes_per_op,
                'allocs_per_op': allocs_per_op,
                'ops_per_sec': 1_000_000_000 / ns_per_op
            }
        
        return results
    
    def get_baseline_results(self) -> Dict[str, Dict[str, float]]:
        """Get baseline performance results from previous runs"""
        if self.baseline_url:
            try:
                response = requests.get(self.baseline_url)
                return response.json()
            except:
                pass
        
        # Fallback to local baseline file
        try:
            with open('performance-baseline.json', 'r') as f:
                return json.load(f)
        except FileNotFoundError:
            return {}
    
    def check_regressions(self, current: Dict, baseline: Dict) -> List[Tuple[str, str, float, float]]:
        """Check for performance regressions"""
        regressions = []
        
        for benchmark_name in current:
            if benchmark_name not in baseline:
                continue
            
            curr = current[benchmark_name]
            base = baseline[benchmark_name]
            
            # Check execution time regression
            time_ratio = curr['ns_per_op'] / base['ns_per_op']
            if time_ratio > self.thresholds['execution_time']:
                regressions.append((
                    benchmark_name, 'execution_time', 
                    time_ratio, self.thresholds['execution_time']
                ))
            
            # Check memory usage regression
            if base['bytes_per_op'] > 0:
                memory_ratio = curr['bytes_per_op'] / base['bytes_per_op']
                if memory_ratio > self.thresholds['memory_usage']:
                    regressions.append((
                        benchmark_name, 'memory_usage',
                        memory_ratio, self.thresholds['memory_usage']
                    ))
            
            # Check throughput degradation
            throughput_ratio = curr['ops_per_sec'] / base['ops_per_sec']
            if throughput_ratio < self.thresholds['throughput']:
                regressions.append((
                    benchmark_name, 'throughput',
                    throughput_ratio, self.thresholds['throughput']
                ))
        
        return regressions
    
    def generate_report(self, current: Dict, baseline: Dict, regressions: List) -> str:
        """Generate performance report"""
        report = ["# Performance Report\n"]
        
        if not regressions:
            report.append("✅ No performance regressions detected!\n")
        else:
            report.append(f"❌ {len(regressions)} performance regressions detected:\n")
            
            for benchmark, metric, ratio, threshold in regressions:
                if metric == 'throughput':
                    change = f"{(1-ratio)*100:.1f}% slower"
                else:
                    change = f"{(ratio-1)*100:.1f}% increase"
                
                report.append(f"- **{benchmark}** ({metric}): {change} (threshold: {threshold})")
        
        report.append("\n## Benchmark Comparison\n")
        report.append("| Benchmark | Current (ns/op) | Baseline (ns/op) | Change |")
        report.append("|-----------|-----------------|------------------|--------|")
        
        for name in sorted(current.keys()):
            curr_time = current[name]['ns_per_op']
            base_time = baseline.get(name, {}).get('ns_per_op', 0)
            
            if base_time > 0:
                change = f"{((curr_time/base_time-1)*100):+.1f}%"
            else:
                change = "NEW"
            
            report.append(f"| {name} | {curr_time:,} | {base_time:,} | {change} |")
        
        return "\n".join(report)
    
    def save_baseline(self, results: Dict):
        """Save current results as new baseline"""
        with open('performance-baseline.json', 'w') as f:
            json.dump(results, f, indent=2)

def main():
    if len(sys.argv) < 2:
        print("Usage: python3 check-performance-regression.py <benchmark-results.txt>")
        sys.exit(1)
    
    results_file = sys.argv[1]
    checker = PerformanceChecker()
    
    # Parse current benchmark results
    current_results = checker.parse_benchmark_results(results_file)
    
    if not current_results:
        print("No benchmark results found in file")
        sys.exit(1)
    
    # Get baseline results
    baseline_results = checker.get_baseline_results()
    
    # Check for regressions
    regressions = checker.check_regressions(current_results, baseline_results)
    
    # Generate report
    report = checker.generate_report(current_results, baseline_results, regressions)
    
    # Save report
    with open('performance-report.md', 'w') as f:
        f.write(report)
    
    print(report)
    
    # Update baseline if no regressions
    if not regressions:
        checker.save_baseline(current_results)
        print("\n✅ Baseline updated with current results")
    
    # Exit with error code if regressions found
    if regressions:
        sys.exit(1)

if __name__ == "__main__":
    main()