package mr

import (
	"log"
	"sync"
	"time"
)
import "net"
import "os"
import "net/rpc"
import "net/http"

type Coordinator struct {
	// Your definitions here.
	mu          sync.Mutex
	mapTasks    []Task
	reduceTasks []Task
	phase       Phase
	nReduce     int
	nMap        int
	done        bool
}

// RequestHandler Your code here -- RPC handlers for the worker to call.
func (c *Coordinator) AssignTasks(args *AssignRequest, response *AssignResponse) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Otherwise, try to assign a pending task
	if c.phase == MapPhase {
		for i := range c.mapTasks {
			if c.mapTasks[i].status == "idle" {
				c.mapTasks[i].status = "in_progress"
				c.mapTasks[i].startTime = time.Now()
				//c.mapTasks[i].attemptCount++
				//
				//// 可以添加最大重试次数的检查
				//if c.mapTasks[i].attemptCount > 3 {
				//	log.Printf("Warning: Map task %v has been attempted %v times",
				//		i, c.mapTasks[i].attemptCount)
				//}

				response.TaskType = MapPhase
				response.TaskNumber = i
				response.InputFile = c.mapTasks[i].inputFile
				response.NReduce = c.nReduce
				response.NMap = c.nMap
				return nil
			}
		}
		if c.allMapTasksDone() {
			c.phase = ReducePhase
		}
	}

	if c.phase == ReducePhase {
		for i := range c.reduceTasks {
			if c.reduceTasks[i].status == "idle" {
				c.reduceTasks[i].status = "in_progress"
				c.reduceTasks[i].startTime = time.Now()

				response.TaskType = ReducePhase
				response.TaskNumber = i
				response.NReduce = c.nReduce
				response.NMap = c.nMap
				return nil
			}
		}
	}

	response.NoTask = true
	return nil
}

func (c *Coordinator) TaskComplete(args *CompleteRequest, response *CompleteResponse) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if args.TaskType == MapPhase {
		if c.mapTasks[args.TaskID].status == "in_progress" {
			if args.Success {
				c.mapTasks[args.TaskID].status = "completed"
			} else {
				c.mapTasks[args.TaskID].status = "idle"
			}
		}
	} else {
		if c.reduceTasks[args.TaskID].status == "in_progress" {
			if args.Success {
				c.reduceTasks[args.TaskID].status = "completed"
			} else {
				c.reduceTasks[args.TaskID].status = "idle"
			}
		}
	}
	response.Acknowledged = true
	return nil
}

// start a thread that listens for RPCs from worker.go
func (c *Coordinator) server() {
	rpc.Register(c)
	rpc.HandleHTTP()
	//l, e := net.Listen("tcp", ":1234")
	sockname := coordinatorSock()
	os.Remove(sockname)
	l, e := net.Listen("unix", sockname)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)
}

func (c *Coordinator) allMapTasksDone() bool {
	for _, task := range c.mapTasks {
		if task.status != "completed" {
			return false
		}
	}
	return true
}

func (c *Coordinator) Done() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.done {
		return true
	}

	if c.phase == MapPhase {
		return false
	}

	for _, task := range c.reduceTasks {
		if task.status != "completed" {
			return false
		}
	}

	c.done = true
	return true
}

func (c *Coordinator) monitorTasks() {
	for !c.Done() {
		c.mu.Lock()
		now := time.Now()

		for i := range c.mapTasks {
			if c.mapTasks[i].status == "in_progress" {
				if now.Sub(c.mapTasks[i].startTime) > 10*time.Second {
					log.Printf("Map task %v timed out, resetting", i)
					c.mapTasks[i].status = "idle" // 重置任务状态
				}
			}
		}

		for i := range c.reduceTasks {
			if c.reduceTasks[i].status == "in_progress" {
				if now.Sub(c.reduceTasks[i].startTime) > 10*time.Second {
					log.Printf("Reduce task %v timed out, resetting", i)
					c.reduceTasks[i].status = "idle"
				}
			}
		}
		c.mu.Unlock()
		time.Sleep(1 * time.Second)
	}

}

// create a Coordinator.
// main/mrcoordinator.go calls this function.
// NReduce is the number of reduce tasks to use.
func MakeCoordinator(files []string, nReduce int) *Coordinator {
	c := Coordinator{
		nReduce: nReduce,
		nMap:    len(files),
		phase:   MapPhase,
		done:    false,
	}

	// we assume one file per Map task
	// creating task
	numTasks := len(files)
	c.mapTasks = make([]Task, numTasks)
	for i := 0; i < numTasks; i++ {
		//startIdx := i * filesPerTask
		//endIdx := min(startIdx+filesPerTask, len(files))

		c.mapTasks[i] = Task{
			phase:      MapPhase,
			status:     "idle",
			taskNumber: i,
			inputFile:  files[i],
		}
	}

	// reduce tasks
	c.reduceTasks = make([]Task, nReduce)
	for i := 0; i < nReduce; i++ {
		c.reduceTasks[i] = Task{
			phase:      ReducePhase,
			status:     "idle",
			taskNumber: i,
		}
	}
	go c.monitorTasks()

	c.server()
	return &c
}
