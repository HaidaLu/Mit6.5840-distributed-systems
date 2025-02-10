package mr

//
// RPC definitions.
//
// remember to capitalize all names.
//

import (
	"os"
	"time"
)
import "strconv"

// Add your RPC definitions here.
type Phase int

const (
	MapPhase Phase = iota
	ReducePhase
)

type Task struct {
	phase      Phase
	status     string
	taskNumber int
	inputFile  string
	startTime  time.Time
}

type AssignRequest struct {
	WorkerID int
}

type AssignResponse struct {
	TaskType   Phase
	TaskNumber int
	InputFile  string
	NReduce    int
	NMap       int
	NoTask     bool
}

// For task completion communication
type CompleteRequest struct {
	TaskID   int
	WorkerID string // To track which worker completed the task
	TaskType Phase
	Success  bool
}

type CompleteResponse struct {
	Acknowledged bool // Coordinator confirms receipt
}

// Cook up a unique-ish UNIX-domain socket name
// in /var/tmp, for the coordinator.
// Can't use the current directory since
// Athena AFS doesn't support UNIX-domain sockets.
func coordinatorSock() string {
	s := "/var/tmp/5840-mr-"
	s += strconv.Itoa(os.Getuid())
	return s
}
