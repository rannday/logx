# ============================================================
# GLOBAL DESTRUCTIVE COMMANDS
# ============================================================
# Prevent catastrophic filesystem operations or privilege use.

[[rules]]
pattern = ["rm", "-rf", "/"]
decision = "forbidden"
justification = "Recursive delete of system root is forbidden."

[[rules]]
pattern = ["rm", "-rf", "*"]
decision = "forbidden"
justification = "Unscoped recursive delete is forbidden."

[[rules]]
pattern = ["mv", ".."]
decision = "forbidden"
justification = "Prevent moving files outside workspace."

[[rules]]
pattern = ["cd", ".."]
decision = "forbidden"
justification = "Prevent navigating outside workspace."

[[rules]]
pattern = ["sudo"]
decision = "prompt"
justification = "Privilege escalation requires explicit approval."


# ============================================================
# WINDOWS SYSTEM PROTECTION
# ============================================================
# Prevent system-level modification, persistence, or damage.

[[rules]]
pattern = ["del", "/s", "/q", "C:\\"]
decision = "forbidden"
justification = "Recursive deletion of system drive is forbidden."

[[rules]]
pattern = ["rmdir", "/s", "/q", "C:\\"]
decision = "forbidden"
justification = "Recursive removal of system drive is forbidden."

[[rules]]
pattern = ["Remove-Item", "-Recurse", "-Force", "C:\\"]
decision = "forbidden"
justification = "PowerShell recursive delete of system root is forbidden."

[[rules]]
pattern = ["reg"]
decision = "forbidden"
justification = "Registry modification is outside project scope."

[[rules]]
pattern = ["sc"]
decision = "forbidden"
justification = "Windows service manipulation is forbidden."

[[rules]]
pattern = ["Set-Service"]
decision = "forbidden"
justification = "Windows service modification is forbidden."

[[rules]]
pattern = ["schtasks"]
decision = "forbidden"
justification = "Scheduled task creation/modification is forbidden."

[[rules]]
pattern = ["netsh"]
decision = "forbidden"
justification = "Firewall or network stack modification is forbidden."

[[rules]]
pattern = ["net", "user"]
decision = "forbidden"
justification = "Local user manipulation is forbidden."


# ============================================================
# NETWORK TOOLING (REQUIRES APPROVAL)
# ============================================================
# Allow research but prevent silent script execution.

[[rules]]
pattern = ["curl"]
decision = "prompt"
justification = "Network execution requires approval."

[[rules]]
pattern = ["Invoke-WebRequest"]
decision = "prompt"
justification = "Network execution requires approval."

[[rules]]
pattern = ["wget"]
decision = "prompt"
justification = "Network execution requires approval."


# ============================================================
# BLOCK ALL GIT OPERATIONS
# ============================================================
# Codex must not modify repo history or interact with git.

[[rules]]
pattern = ["git"]
decision = "forbidden"
justification = "All git operations are forbidden."

[[rules]]
pattern = [".git\\"]
decision = "forbidden"
justification = "Direct modification of Git internals is forbidden."

[[rules]]
pattern = [".git/"]
decision = "forbidden"
justification = "Direct modification of Git internals is forbidden."


# ============================================================
# PROTECT CODEX CONFIGURATION
# ============================================================
# Prevent Codex from modifying its own configuration or rules.

[[rules]]
pattern = [".codex\\"]
decision = "forbidden"
justification = "Modification of Codex configuration is forbidden."

[[rules]]
pattern = [".codex/"]
decision = "forbidden"
justification = "Modification of Codex configuration is forbidden."


# ============================================================
# BLOCK PACKAGE / SYSTEM MANAGERS
# ============================================================
# Prevent installing software or modifying system packages.

[[rules]]
pattern = ["choco"]
decision = "forbidden"
justification = "System package manager usage is forbidden."

[[rules]]
pattern = ["winget"]
decision = "forbidden"
justification = "System package manager usage is forbidden."

[[rules]]
pattern = ["npm"]
decision = "forbidden"
justification = "External package manager usage is forbidden."

[[rules]]
pattern = ["pip"]
decision = "forbidden"
justification = "External package manager usage is forbidden."


# ============================================================
# ALLOW SAFE DEVELOPMENT COMMANDS
# ============================================================
# Explicitly allow common Go development tasks.

[[rules]]
pattern = ["go", "build"]
decision = "allow"
justification = "Allow Go builds inside project."

[[rules]]
pattern = ["go", "test"]
decision = "allow"
justification = "Allow Go tests inside project."

[[rules]]
pattern = ["go", "run"]
decision = "allow"
justification = "Allow running local Go project code."
