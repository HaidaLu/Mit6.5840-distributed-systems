# Distributed MapReduce Design
## 1. Functional Requirements:
- Distributed processing of large datasets
- Map phase: process input files into intermediate key-value pairs
- Reduce phase: aggregate intermediate data by key
- Support multiple concurrent workers
- Handle worker failures gracefully

## 2. Non-Functional Requirements:
- Fault tolerance: 10-second task timeout
- Scalability: support arbitrary number of workers
- Consistency: ensure correct processing of all input data
- File naming convention compliance


## 3. Design Diagram
![diagram](../images/MapReduce.jpg)
[rpc design](rpc.go)
## 4. Fault Tolerance

- Task Timeout: 
  - 10-second timeout for tasks
  - Automatic task reassignment
    ```go
    if now.Sub(task.startTime) > 10*time.Second {
        task.status = "idle"
    }
    ```
- Worker Failure Detection:
  - RPC failure handling
  - Task state reset on failure
  - Independent worker processes

- State Recovery:
  - Tasks can be reassigned to different workers
  - Idempotent task execution

## 5. File Management

File Types:

Input files: user-provided
Intermediate: mr-X-Y
Output: mr-out-Y


File Naming Convention:
goCopyintermediateFile := fmt.Sprintf("mr-%d-%d", mapTask, reduceTask)
outputFile := fmt.Sprintf("mr-out-%d", reduceTask)