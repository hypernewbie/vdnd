import subprocess
import json
import os

def run_repl_test(code_blocks):
    process = subprocess.Popen(
        ["python3", "py/restricted_python.py"],
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True
    )
    
    results = []
    for code in code_blocks:
        process.stdin.write(json.dumps({"code": code}) + "\n")
        process.stdin.flush()
        line = process.stdout.readline()
        if line:
            results.append(json.loads(line))
        else:
            print("Error: No output from REPL")
            break
            
    process.terminate()
    return results

def test_persistence():
    print("Testing persistence...")
    blocks = [
        "x = 10",
        "print(x + 5)"
    ]
    results = run_repl_test(blocks)
    if "15" not in results[1]["stdout"]:
        print(f"DEBUG: results[1] = {results[1]}")
    assert "15" in results[1]["stdout"]
    print("Persistence test passed!")

def test_stdout_capture():
    print("Testing stdout capture...")
    blocks = [
        "print('hello world')",
        "for i in range(3): print(i)"
    ]
    results = run_repl_test(blocks)
    assert "hello world" in results[0]["stdout"]
    assert "0\n1\n2" in results[1]["stdout"]
    print("Stdout capture test passed!")

def test_restricted_access():
    print("Testing restricted access...")
    blocks = [
        "with open('/etc/passwd') as f: print(f.read())"
    ]
    results = run_repl_test(blocks)
    assert "PermissionError" in results[0]["error"]
    print("Restricted access test passed!")

if __name__ == "__main__":
    try:
        test_persistence()
        test_stdout_capture()
        test_restricted_access()
        print("\nAll tests passed successfully!")
    except Exception as e:
        print(f"\nTest failed: {e}")
        import traceback
        traceback.print_exc()
