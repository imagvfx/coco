package main

// Command is a command to be run in a worker.
// First string is the executable and others are arguments.
// When a Command is nil or empty, the command will be skipped.
type Command []string
