I'll start by exploring the repository structure to understand the current state of the custom decoder plugin and related files.

<bash>
find . -type d -name "custom-decoder" 2>/dev/null
</bash>

<bash>
ls -la examples/plugins/ 2>/dev/null
</bash>

<bash>
cat examples/plugins/custom-decoder/README.md 2>/dev/null || echo "FILE NOT FOUND"
</bash>

<bash>
cat go.mod 2>/dev/null | head -20
</bash>

<bash>
cat Makefile 2>/dev/null
</bash>