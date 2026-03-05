package events

type UpdateDetectedMsg struct {
	NewHash string
}

type PullCompletedMsg struct{}

type ErrorMsg struct {
	Err error
}

type LogEmittedMsg struct {
	Log string
}

type CommandExecutedMsg struct{}
