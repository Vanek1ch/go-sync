package main

import "proj/handlers"

type AppManager struct {
	Version      string
	Routes       []*handlers.JSONSyncManager
	RunOnStartup bool
}
