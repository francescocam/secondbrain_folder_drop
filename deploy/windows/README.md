# Windows deployment

1. Build binary:
```powershell
go build -o secondbrain-folder-drop.exe ./cmd/secondbrain-folder-drop
```

2. Place files:
- Binary: `C:\Program Files\SecondbrainFolderDrop\secondbrain-folder-drop.exe`
- Config: `C:\ProgramData\SecondbrainFolderDrop\config.yaml`

3. Create service:
```powershell
sc.exe create secondbrain-folder-drop binPath= '"C:\Program Files\SecondbrainFolderDrop\secondbrain-folder-drop.exe" run --config "C:\ProgramData\SecondbrainFolderDrop\config.yaml"' start= auto
```

4. Configure recovery:
```powershell
sc.exe failure secondbrain-folder-drop reset= 86400 actions= restart/5000/restart/5000/restart/5000
```

5. Start service:
```powershell
sc.exe start secondbrain-folder-drop
```

6. Optional service account:
```powershell
sc.exe config secondbrain-folder-drop obj= ".\\secondbrain-drop" password= "<password>"
```
