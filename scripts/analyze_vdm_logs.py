#!/usr/bin/env python3
import json
import os
import sys
from collections import Counter, defaultdict

def analyze_logs(log_file):
    if not os.path.exists(log_file):
        print(f"Log file {log_file} not found.")
        return

    tool_calls = Counter()
    generic_vd_cmds = Counter()
    tool_errors = Counter()
    python_attempts = 0
    total_iterations = 0
    
    with open(log_file, "r") as f:
        for line in f:
            # logs are in slog Text format or JSON depending on handler.
            # slog.NewTextHandler produces: time=... level=... msg=... key=val
            # Let's try to find our keys.
            
            if "msg=TOOL_CALL" in line:
                # Naive parsing
                parts = line.split()
                tool = next((p.split("=")[1] for p in parts if p.startswith("tool=")), "unknown")
                tool_calls[tool] += 1
                
                if tool == "vd":
                    cmd = next((p.split("=")[1] for p in parts if p.startswith("arguments=")), "")
                    generic_vd_cmds[cmd] += 1
                
                exit_code = next((p.split("=")[1] for p in parts if p.startswith("exit_code=")), "0")
                if exit_code != "0":
                    tool_errors[tool] += 1
            
            if "msg=PYTHON_STATE_ATTEMPT" in line:
                python_attempts += 1
            
            if "msg=GENERATION_COMPLETE" in line:
                total_iterations += 1

    print("--- VDM PHASE 4 TELEMETRY REPORT ---")
    print(f"Total DM sessions analyzed: {total_iterations}")
    print(f"Python state mutation attempts detected: {python_attempts}")
    print("
Tool Usage:")
    for tool, count in tool_calls.most_common():
        errs = tool_errors[tool]
        err_rate = (errs / count * 100) if count > 0 else 0
        print(f"  - {tool}: {count} calls ({errs} errors, {err_rate:.1f}% error rate)")
    
    print("
Most Common Generic 'vd' Commands:")
    for cmd, count in generic_vd_cmds.most_common(10):
        print(f"  - {cmd}: {count} times")

if __name__ == "__main__":
    log_path = "vdm.log"
    if len(sys.argv) > 1:
        log_path = sys.argv[1]
    analyze_logs(log_path)
