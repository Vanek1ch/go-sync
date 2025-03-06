package main

import "go-sync/handlers"

type AppManager struct {
	Version      string
	Routes       []*handlers.JSONSyncManager
	RunOnStartup bool
}
