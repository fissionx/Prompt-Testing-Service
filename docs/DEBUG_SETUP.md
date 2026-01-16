# Debug Setup Guide for Cursor/VS Code

This guide explains how to debug the Gego application in Cursor (or VS Code) with remote PostgreSQL.

## Quick Start

1. **Open the Debug Panel**: Press `F5` or click the Debug icon in the sidebar
2. **Select Configuration**: Choose `🐘 Debug API Server (PostgreSQL - Remote Neon)` from the dropdown
3. **Start Debugging**: Press `F5` or click the green play button

## Available Debug Configurations

### PostgreSQL Configurations

#### 🐘 Debug API Server (PostgreSQL - Remote Neon)
- **Purpose**: Debug API server connected to remote Neon PostgreSQL
- **Database**: Remote Neon PostgreSQL (pre-configured)
- **Port**: 8989
- **Host**: 0.0.0.0 (all interfaces)

#### 🐘 Debug API Server (PostgreSQL - Local)
- **Purpose**: Debug API server with local PostgreSQL
- **Database**: Local PostgreSQL (localhost:5432)
- **Requires**: Local PostgreSQL running

#### 🐘 Debug API Server (PostgreSQL - Custom URI)
- **Purpose**: Debug with custom PostgreSQL connection string
- **Database**: Prompts for PostgreSQL URI at startup
- **Use Case**: Testing different database connections

#### 🐘 Debug API Server (PostgreSQL - Config File)
- **Purpose**: Debug using configuration file
- **Database**: Uses `~/.gego/config.yaml` or custom path
- **Use Case**: Using existing configuration

### MongoDB Configurations

#### 🌍 Debug API Server (Cloud MongoDB)
- **Purpose**: Debug with cloud MongoDB Atlas
- **Database**: Remote MongoDB

#### 💻 Debug API Server (Local MongoDB)
- **Purpose**: Debug with local MongoDB
- **Database**: Local MongoDB (localhost:27017)

## Setting Breakpoints

1. **Open any Go file** in your workspace
2. **Click in the gutter** (left of line numbers) to set a breakpoint
3. **Red dot** appears indicating breakpoint is set
4. **Start debugging** - execution will pause at breakpoints

## Debug Features

### Breakpoints
- **Regular Breakpoint**: Pauses execution
- **Conditional Breakpoint**: Right-click breakpoint → Edit Breakpoint → Add condition
- **Logpoint**: Logs message without stopping (right-click → Add Logpoint)

### Debug Console
- **Evaluate expressions**: Type Go expressions in Debug Console
- **View variables**: Hover over variables or check Variables panel
- **Call stack**: See function call hierarchy in Call Stack panel

### Debug Actions
- **Continue (F5)**: Resume execution
- **Step Over (F10)**: Execute current line, don't enter functions
- **Step Into (F11)**: Enter function calls
- **Step Out (Shift+F11)**: Exit current function
- **Restart (Ctrl+Shift+F5)**: Restart debugging session
- **Stop (Shift+F5)**: Stop debugging

## Environment Variables in Debug Config

The remote Neon configuration includes:

```json
{
    "NOSQL_DATABASE_PROVIDER": "postgresql",
    "POSTGRESQL_URI": "postgresql://neondb_owner:***@ep-bold-sky-a11lqfs2-pooler.ap-southeast-1.aws.neon.tech/gego?sslmode=require&channel_binding=require",
    "POSTGRESQL_DATABASE": "gego"
}
```

## Customizing Debug Configuration

To modify the remote PostgreSQL connection:

1. Open `.vscode/launch.json`
2. Find `🐘 Debug API Server (PostgreSQL - Remote Neon)`
3. Update the `POSTGRESQL_URI` environment variable
4. Save the file

## Troubleshooting

### Debugger Not Starting
- **Check Go installation**: Ensure Go is installed and in PATH
- **Check Delve**: Install Delve debugger: `go install github.com/go-delve/delve/cmd/dlv@latest`
- **Check workspace**: Ensure you're in the correct workspace folder

### Connection Issues
- **Verify PostgreSQL URI**: Check connection string is correct
- **Check network**: Ensure you can reach the remote database
- **Check SSL**: Verify SSL mode matches database requirements

### Breakpoints Not Hit
- **Verify code is running**: Check if the code path is executed
- **Check optimization**: Ensure code isn't optimized away
- **Rebuild**: Try rebuilding the project

### Variables Not Showing
- **Check scope**: Variables must be in current scope
- **Check optimization**: Some variables may be optimized away
- **Use Debug Console**: Try evaluating expressions directly

## Debugging Tips

1. **Set breakpoints early**: Set breakpoints at entry points (main, API handlers)
2. **Use conditional breakpoints**: Break only when specific conditions are met
3. **Watch expressions**: Add expressions to Watch panel for monitoring
4. **Use Debug Console**: Evaluate expressions and test code
5. **Check logs**: Monitor console output for additional information

## Example: Debugging Brand Service

To debug the brand service with remote PostgreSQL:

1. **Set breakpoint** in `internal/services/brand_service.go` at line 49 (GetBrandInfo)
2. **Select** `🐘 Debug API Server (PostgreSQL - Remote Neon)` configuration
3. **Start debugging** (F5)
4. **Make API request** to `/api/v1/brands/8ea22500-53ff-42fb-81eb-f7cd512463ac`
5. **Execution pauses** at breakpoint
6. **Inspect variables**: Check `brandID`, `s.cache`, etc.
7. **Step through code**: Use F10/F11 to step through execution

## Remote Debugging

If you need to debug a remote instance:

1. **Use Delve headless mode** on remote server:
   ```bash
   dlv debug --headless --listen=:2345 --api-version=2
   ```

2. **Add attach configuration** in `launch.json`:
   ```json
   {
       "name": "Attach to Remote",
       "type": "go",
       "request": "attach",
       "mode": "remote",
       "remotePath": "/path/on/remote",
       "port": 2345,
       "host": "remote-host-ip"
   }
   ```

## Additional Resources

- [Go Debugging in VS Code](https://github.com/golang/vscode-go/wiki/debugging)
- [Delve Documentation](https://github.com/go-delve/delve)
- [VS Code Debugging](https://code.visualstudio.com/docs/editor/debugging)

