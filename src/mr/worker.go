package mr

import (
	"encoding/json"
	"fmt"
	"io"
	_ "io/ioutil"
	"os"
	"sort"
	"time"
)
import "log"
import "net/rpc"
import "hash/fnv"

// Map functions return a slice of KeyValue.
type KeyValue struct {
	Key   string
	Value string
}
type ByKey []KeyValue

func (a ByKey) Len() int           { return len(a) }
func (a ByKey) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByKey) Less(i, j int) bool { return a[i].Key < a[j].Key }

// use ihash(key) % NReduce to choose the reduce
// task number for each KeyValue emitted by Map.
func ihash(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32() & 0x7fffffff)
}

// main/mrworker.go calls this function.
func Worker(mapf func(string, string) []KeyValue,
	reducef func(string, []string) string) {
	for {
		request := AssignRequest{
			WorkerID: os.Getpid(),
		}
		response := AssignResponse{}

		ok := call("Coordinator.AssignTasks", &request, &response)

		if !ok {
			log.Printf("Cannot contact coordinator, worker exiting")
			return
		}

		if response.NoTask {
			log.Printf("No task available, waiting...")
			time.Sleep(time.Second)
			continue
		}

		success := false
		if response.TaskType == MapPhase {
			success = doMap(mapf, response.TaskNumber, response.InputFile, response.NReduce)
		} else {
			success = doReduce(reducef, response.TaskNumber, response.NMap)
		}

		completeReq := CompleteRequest{
			TaskType: response.TaskType,
			TaskID:   response.TaskNumber,
			Success:  success,
		}

		completeRes := CompleteResponse{}
		call("Coordinator.TaskComplete", &completeReq, &completeRes)

	}
}

func doMap(mapf func(string, string) []KeyValue,
	taskNum int, filename string, nReduce int) bool {
	// read the file
	file, err := os.Open(filename)
	if err != nil {
		log.Printf("cannot open %v", filename)
		return false
	}
	content, err := io.ReadAll(file)
	if err != nil {
		log.Printf("cannot read %v", filename)
		return false
	}
	kva := mapf(filename, string(content))

	// intermediate
	intermediate := make([][]KeyValue, nReduce)
	for _, kv := range kva {
		bucket := ihash(kv.Key) % nReduce
		intermediate[bucket] = append(intermediate[bucket], kv)
	}

	for reduceTask := 0; reduceTask < nReduce; reduceTask++ {
		oname := fmt.Sprintf("mr-%d-%d", taskNum, reduceTask)
		ofile, _ := os.Create(oname)
		enc := json.NewEncoder(ofile)
		for _, kv := range intermediate[reduceTask] {
			enc.Encode(&kv)
		}
		ofile.Close()
	}
	return true
}

func doReduce(reducef func(string, []string) string, reduceNum int, nMap int) bool {
	intermediate := []KeyValue{}
	for mapTask := 0; mapTask < nMap; mapTask++ {
		iname := fmt.Sprintf("mr-%d-%d", mapTask, reduceNum)
		file, err := os.Open(iname)
		if err != nil {
			continue
		}
		dec := json.NewDecoder(file)
		for {
			var kv KeyValue
			if err := dec.Decode(&kv); err != nil {
				break
			}
			intermediate = append(intermediate, kv)
		}
		file.Close()
	}

	// sort by key
	sort.Sort(ByKey(intermediate))

	oname := fmt.Sprintf("mr-out-%d", reduceNum)
	ofile, _ := os.Create(oname)

	i := 0
	for i < len(intermediate) {
		j := i + 1
		for j < len(intermediate) &&
			intermediate[j].Key == intermediate[i].Key {
			j++
		}
		values := []string{}
		for k := i; k < j; k++ {
			values = append(values, intermediate[k].Value)
		}
		output := reducef(intermediate[i].Key, values)
		fmt.Fprintf(ofile, "%v %v\n", intermediate[i].Key, output)
		i = j
	}
	ofile.Close()
	return true

}

// send an RPC request to the coordinator, wait for the response.
// usually returns true.
// returns false if something goes wrong.
func call(rpcname string, args interface{}, reply interface{}) bool {
	// c, err := rpc.DialHTTP("tcp", "127.0.0.1"+":1234")
	sockname := coordinatorSock()
	c, err := rpc.DialHTTP("unix", sockname)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	defer c.Close()

	err = c.Call(rpcname, args, reply)
	if err == nil {
		return true
	}

	fmt.Println(err)
	return false
}
