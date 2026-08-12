# AGENTS.md

- Preserve Windows-native and Ubuntu-native designs; do not introduce WSL as a requirement.
- Keep the runtime localhost-only and the control connection outbound-only.
- Never add arbitrary remote shell or unrelated host-data collection.
- Verify signatures or hashes for runtime, model, and update artifacts.
- Keep platform-specific behavior behind adapters and test on the target OS.
- Do not claim compatibility without matrix evidence.
