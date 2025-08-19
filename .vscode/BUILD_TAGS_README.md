# VS Code Go Configuration for Message Bus

This workspace uses build tags to switch between production Kafka implementation and local mock implementation.

## Why Some Files Show "Errors" in VS Code

**This is normal behavior!** VS Code's Go language server (gopls) can only analyze one build configuration at a time:

- In **production mode**: `kafkabus.go` shows normally, `localbus.go` appears "excluded" with build tag errors
- In **local mode**: `localbus.go` shows normally, `kafkabus.go` appears "excluded" with build tag errors

**The builds still work perfectly** - this is just a VS Code display limitation.

## Current Configuration

By default, VS Code is configured for **production mode** (Kafka implementation).

## Quick Switching with Tasks

Use VS Code's Command Palette (`Cmd+Shift+P`) and run:
- **"Tasks: Run Task"** → **"Switch to Local Development Mode"**
- **"Tasks: Run Task"** → **"Switch to Production Mode"**

After switching, reload the VS Code window: `Cmd+Shift+P` → **"Developer: Reload Window"**

## Manual Configuration

### For Local Development (Mock Implementation)
Edit `.vscode/settings.json`:
```json
{
    "go.buildTags": "local",
    "gopls": {
        "buildFlags": ["-tags=local"]
    },
    "go.toolsEnvVars": {
        "GOFLAGS": "-tags=local"
    }
}
```

### For Production (Kafka Implementation)  
Edit `.vscode/settings.json`:
```json
{
    "go.buildTags": "",
    "gopls": {
        "buildFlags": []
    },
    "go.toolsEnvVars": {
        "GOFLAGS": ""
    }
}
```

## Build Commands (Always Work)

- Production build: `make package`
- Local build: `make package-local`

Both builds work regardless of VS Code configuration!
