import os
import sys
import json
import io
import traceback
import re
import random
import math
from contextlib import redirect_stdout
from RestrictedPython import compile_restricted
from RestrictedPython.Guards import safe_builtins, full_write_guard, safer_getattr, guarded_iter_unpack_sequence
from RestrictedPython.PrintCollector import PrintCollector

# Determine project root (one level up from this script in py/)
SCRIPT_DIR = os.path.dirname(os.path.realpath(__file__))
PROJECT_ROOT = os.path.dirname(SCRIPT_DIR)

# Security Configuration
ALLOWED_DIRS = {
    os.path.join(PROJECT_ROOT, "rules"),
    os.path.join(PROJECT_ROOT, "rules_derived"),
    os.path.join(PROJECT_ROOT, "sandbox"),
}

def safe_open(filename, mode="r", *args, **kwargs):
    # Normalize path
    filename = os.path.realpath(filename)
    sandbox_dir = os.path.join(PROJECT_ROOT, "sandbox")

    # Check directory whitelist
    allowed = False
    is_sandbox = filename.startswith(sandbox_dir + os.sep)
    
    # Check whitelist
    for allowed_dir in ALLOWED_DIRS:
        if filename.startswith(allowed_dir + os.sep) and filename.endswith(".md"):
            allowed = True
            break

    if not allowed:
        raise PermissionError(f"Access to '{filename}' is not allowed")

    # Enforce read-only for non-sandbox areas
    if not is_sandbox and mode not in ("r", "rb"):
        raise PermissionError("Write access is only allowed in the sandbox directory")

    return open(filename, mode, *args, **kwargs)

class UniversalPrint:
    def __init__(self, _getattr_=None):
        pass
    def __call__(self, *args, **kwargs):
        # RestrictedPython calls _print_(_getattr_) to get the printer
        return self
    def _call_print(self, *args, **kwargs):
        print(*args, **kwargs)
    def write(self, s):
        print(s, end="")

def _inplacevar_(op, var, expr):
    if op == '+=':
        return var + expr
    if op == '-=':
        return var - expr
    if op == '*=':
        return var * expr
    if op == '/=':
        return var / expr
    if op == '//=':
        return var // expr
    if op == '%=':
        return var % expr
    if op == '**=':
        return var ** expr
    if op == '<<=':
        return var << expr
    if op == '>>=':
        return var >> expr
    if op == '&=':
        return var & expr
    if op == '^=':
        return var ^ expr
    if op == '|=':
        return var | expr
    raise ValueError(f"Unknown in-place operator: {op}")

class SandboxREPL:
    def __init__(self):
        # Save original stdout for communication with Go
        self.original_stdout = sys.stdout
        
        # Initialize persistent state
        self.globals_dict = safe_builtins.copy()
        self.globals_dict["open"] = safe_open
        self.globals_dict["message_history"] = []
        self.globals_dict["json"] = json
        self.globals_dict["re"] = re
        self.globals_dict["random"] = random
        self.globals_dict["math"] = math
        self.globals_dict["min"] = min
        self.globals_dict["max"] = max
        self.globals_dict["sum"] = sum
        self.globals_dict["dir"] = dir
        
        def guarded_import(name, globals=None, locals=None, fromlist=(), level=0):
            if name in ("re", "json", "random", "math"):
                return __import__(name, globals, locals, fromlist, level)
            raise ImportError(f"import of '{name}' is not allowed")
        self.globals_dict["__import__"] = guarded_import
        
        # Necessary guards for RestrictedPython
        self.globals_dict["_getattr_"] = safer_getattr
        self.globals_dict["_write_"] = full_write_guard
        self.globals_dict["_getiter_"] = iter
        self.globals_dict["_getitem_"] = lambda obj, key: obj[key]
        self.globals_dict["_setitem_"] = lambda obj, key, val: obj.__setitem__(key, val)
        self.globals_dict["_delitem_"] = lambda obj, key: obj.__delitem__(key)
        self.globals_dict["_unpack_sequence_"] = guarded_iter_unpack_sequence
        self.globals_dict["_iter_unpack_sequence_"] = guarded_iter_unpack_sequence
        self.globals_dict["_inplacevar_"] = _inplacevar_
        
        # In RestrictedPython, 'print' is transformed to a call to '_print_'.
        # UniversalPrint handles both factory style and direct call style.
        self.globals_dict["_print_"] = UniversalPrint()
        
        # Recursive LLM callback
        self.globals_dict["recursive_llm"] = self.recursive_llm
        
        
        # Files inspection helper
        self.globals_dict["list_dir"] = self.list_dir
        self.globals_dict["search_files"] = self.search_files
        
        # Ensure we have a reference to builtins that RestrictedPython expects
        self.globals_dict["__builtins__"] = self.globals_dict.copy()
        
        self.locals_dict = {}

    def list_dir(self, path="."):
        """
        Lists files and directories in the given path (if allowed).
        """
        abs_path = os.path.realpath(os.path.join(PROJECT_ROOT, path))
        allowed = False
        for allowed_dir in ALLOWED_DIRS:
            if abs_path.startswith(allowed_dir):
                allowed = True
                break
        
        if not allowed and abs_path != PROJECT_ROOT:
             # Allow listing PROJECT_ROOT to see 'rules', etc.
             if abs_path == PROJECT_ROOT:
                 allowed = True
             else:
                 raise PermissionError(f"Access to '{path}' is not allowed")
        
        try:
            items = os.listdir(abs_path)
            # Filter for .md files or directories
            res = []
            for item in items:
                full = os.path.join(abs_path, item)
                if os.path.isdir(full) or item.endswith(".md"):
                    res.append(item + ("/" if os.path.isdir(full) else ""))
            return sorted(res)
        except Exception as e:
            return str(e)

    def search_files(self, query):
        """
        Finds files matching the query in allowed directories.
        """
        res = []
        for allowed_dir in ALLOWED_DIRS:
            if not os.path.exists(allowed_dir):
                continue
            for root, _, filenames in os.walk(allowed_dir):
                for filename in filenames:
                    if query.lower() in filename.lower() and filename.endswith(".md"):
                        rel = os.path.relpath(os.path.join(root, filename), PROJECT_ROOT)
                        res.append(rel)
        return sorted(res)[:50] # Limit results

    def recursive_llm(self, query, context):
        """
        Signals the Go process to perform a recursive LLM call.
        """
        # Output special JSON to signal recursion to the ORIGINAL stdout
        self.original_stdout.write(json.dumps({
            "type": "recursive_call",
            "query": query,
            "context": context
        }) + "\n")
        self.original_stdout.flush()
        
        # Wait for result on stdin
        for line in sys.stdin:
            line = line.strip()
            if not line:
                continue
            try:
                response = json.loads(line)
                if response.get("type") == "recursive_response":
                    return response.get("result", "")
                else:
                    # Ignore other messages while waiting for response
                    continue
            except json.JSONDecodeError:
                continue
        return "Error: No response from recursive call"

    def execute(self, code):
        f = io.StringIO()
        error = None
        result = None
        
        try:
            # Compile in restricted mode
            # We use 'exec' to allow multiple statements.
            # Python 3.13+ may emit SyntaxWarning for the 'printed' variable 
            # injected by RestrictedPython if not read within the same scope.
            import warnings
            with warnings.catch_warnings():
                warnings.filterwarnings("ignore", category=SyntaxWarning)
                compiled = compile_restricted(code, "<sandbox>", "exec")
            
            # Execute with stdout redirection
            with redirect_stdout(f):
                exec(compiled, self.globals_dict, self.locals_dict)
            
            # If the last line is an expression, we don't automatically capture it in 'exec'
            # Unlike Jupyter which captures the result of the last expression.
            # To simulate that, we'd need more complex parsing or use 'eval' for single lines.
            # For now, we'll rely on explicit prints or inspecting locals.
            
        except Exception:
            error = traceback.format_exc()
        
        return {
            "stdout": f.getvalue(),
            "error": error
        }

def main():
    repl = SandboxREPL()
    
    # Process requests from stdin
    # Format: {"code": "..."}
    for line in sys.stdin:
        line = line.strip()
        if not line:
            continue
            
        try:
            request = json.loads(line)
            code = request.get("code", "")
            
            response = repl.execute(code)
            response["type"] = "result"
            
            # Output structured response
            print(json.dumps(response), flush=True)
            
        except json.JSONDecodeError:
            print(json.dumps({"error": "Invalid JSON input"}), flush=True)
        except Exception as e:
            print(json.dumps({"error": str(e)}), flush=True)

if __name__ == "__main__":
    main()
