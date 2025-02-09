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
	//attemptCount int
}

type AssignRequest struct {
	WorkerID int
}

type AssignResponse struct {
	TaskType   Phase  // MAP 或 REDUCE
	TaskNumber int    // 任务编号
	InputFile  string // 对于Map任务是输入文件名，对于Reduce任务可以为空
	NReduce    int    // reduce任务数量（map任务需要知道要生成多少个中间文件）
	NMap       int    // map任务数量（reduce任务需要知道要读取多少个中间文件）
	NoTask     bool   // 当前是否没有可用任务
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
