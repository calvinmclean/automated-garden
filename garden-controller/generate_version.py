import subprocess
import os

# Use current working directory (PlatformIO runs this from the project dir)
project_dir = os.getcwd()

try:
    result = subprocess.run(
        ["git", "rev-parse", "--short", "HEAD"],
        capture_output=True,
        text=True,
        check=True,
        cwd=project_dir,
    )
    version = result.stdout.strip()
except (subprocess.CalledProcessError, FileNotFoundError):
    version = "unknown"

version_h_path = os.path.join(project_dir, "include", "version.h")
with open(version_h_path, "w") as f:
    f.write(f'#ifndef version_h\n#define version_h\n\n#define FIRMWARE_VERSION "{version}"\n\n#endif\n')

print(f"Generated version.h with version: {version}")
