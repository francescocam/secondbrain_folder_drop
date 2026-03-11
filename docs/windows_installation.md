# Windows Installation Guide

This guide explains how to install and run the `secondbrain-folder-drop` service on Windows using PowerShell and `sc.exe`.

## Prerequisites

- Go is only required if you want to build the binary on Windows.
- If you use the AFFiNE target, Node.js must be installed and `node.exe` must be available on `PATH` on the machine running the service.

## Installation Steps

Open PowerShell **as Administrator** and run the following commands to build the binary, prepare the filesystem layout, and register the application as a Windows service.

### 1. Build the Binary

If Go is installed on Windows, build the executable:

```powershell
go build -o secondbrain-folder-drop.exe ./cmd/secondbrain-folder-drop
```

### 2. Go to the Installation Directory

Move to the directory where you want to build or stage the executable:

```powershell
cd "C:\Users\fcamisa\OneDrive\Documents\SecondBrain"
```

### 3. Prepare Directories and Place Files

Recommended locations:

- Executable: `C:\Program Files\SecondbrainFolderDrop`
- Configuration: `C:\ProgramData\SecondbrainFolderDrop`

```powershell
# Create directories
New-Item -ItemType Directory -Force -Path "C:\Program Files\SecondbrainFolderDrop"
New-Item -ItemType Directory -Force -Path "C:\ProgramData\SecondbrainFolderDrop"

# Copy the executable from the current directory
Copy-Item -Path ".\secondbrain-folder-drop.exe" -Destination "C:\Program Files\SecondbrainFolderDrop\secondbrain-folder-drop.exe" -Force
```

### 4. Create or Copy the Configuration File

Place `config.yaml` in `C:\ProgramData\SecondbrainFolderDrop` and adjust it for your environment.

If you want to create it directly from PowerShell:

```powershell
$YamlConfig = @"
watcher:
  input_dir: "C:\\ProgramData\\SecondbrainFolderDrop\\data"
  failed_dir: "C:\\ProgramData\\SecondbrainFolderDrop\\failed"

blinko:
  base_url: "http://your-blinko-instance.local"
  jwt_token: "your-jwt-token"

affine:
  base_url: "http://your-affine-instance.local"
  auth_token: "your-auth-token"
  workspace_id: "your-workspace-id"
"@

# Write config.yaml to disk
Set-Content -Path "C:\ProgramData\SecondbrainFolderDrop\config.yaml" -Value $YamlConfig -Encoding utf8
```

If you already have a `config.yaml`, copy it instead:

```powershell
# Copy an existing config file from the current directory
Copy-Item -Path ".\config.yaml" -Destination "C:\ProgramData\SecondbrainFolderDrop\config.yaml" -Force
```

### 5. Create the Windows Service

Register the background service with `sc.exe`:

```powershell
sc.exe create secondbrain-folder-drop binPath= '"C:\Program Files\SecondbrainFolderDrop\secondbrain-folder-drop.exe" run --config "C:\ProgramData\SecondbrainFolderDrop\config.yaml"' start= auto
```

### 6. Configure Service Recovery

Restart the service automatically if it fails:

```powershell
sc.exe failure secondbrain-folder-drop reset= 86400 actions= restart/5000/restart/5000/restart/5000
```

### 7. Start the Service

Start the application as a background service:

```powershell
sc.exe start secondbrain-folder-drop
```

### 8. Optional: Configure a Service Account

If the application needs to run under a specific service account:

```powershell
sc.exe config secondbrain-folder-drop obj= ".\secondbrain-drop" password= "<password>"
```

## Verification

Check the service status at any time:

```powershell
sc.exe query secondbrain-folder-drop
```
