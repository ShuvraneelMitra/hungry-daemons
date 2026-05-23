package profiler

import (
	"bytes"
	"fmt"
	"strconv"
	"time"
)

/*
Credits to DataDog/gostackparse; I read the code once before starting to
write my own. The design of the parser remains similar, though the implementation
is purely mine.
*/

const (
	heading_prefix = "goroutine "
	created_by_prefix = "created by "
)

func firstWord(b []byte) []byte {
	for i, c := range b {
		if (c < 'A' || c > 'Z') &&
		   (c < 'a' || c > 'z') &&
		   (c < '0' || c > '9') {
			return b[ : i]
		}
	}
	return b
}


type GoRoutineState int

const (
	RUNNING GoRoutineState = iota
	RUNNABLE

	// Do not reorder this enum! If you want to add states please append to the bottom, after DEAD.
	BLOCK_SLEEP // 2
	BLOCK_CHAN_SEND
	BLOCK_CHAN_RCV
	BLOCK_SELECT
	BLOCK_MUTEX
	BLOCK_SEM
	BLOCK_IO

	WAIT
	SLEEP // 10
	SYSCALL
	GC
	DEAD
)

var stateMap = map[string]GoRoutineState{
	"running": RUNNING,
	"runnable": RUNNABLE,
	"syscall": SYSCALL,
	"waiting": WAIT,
	"dead": DEAD,

	"chan send": BLOCK_CHAN_SEND,
	"chan receive": BLOCK_CHAN_RCV,
	"select": BLOCK_SELECT,
	"sync.Mutex.Lock": BLOCK_MUTEX,
	"sync.RWMutex.RLock": BLOCK_MUTEX,
	"sync.WaitGroup.Wait": WAIT,
	"time.Sleep": SLEEP,
	"IO wait": BLOCK_IO,
	"GC": GC,
	"finalizer wait": WAIT,
}

type Frame struct {
	Func string
	File string
	Line int
}

type GoRoutine struct {
	Id             int
	State          GoRoutineState
	Waiting        bool
	Stack          []Frame
	CreatedBy      *Frame
}

type Sample struct {
	Timestamp      time.Time
	GoRoutineCount int
	List           []GoRoutine
}

type ParserState int

const (
	StateHeading ParserState = iota
	StateStackFunc
	StateStackFuncAddr
	StateCreatedByFunc
	StateCreatedByAddr
)
const NUM_PARSER_STATES int = 5

func NewSample(t time.Time, nGo int) *Sample {
	return &Sample{
		Timestamp:      t,
		GoRoutineCount: nGo,
		List:           make([]GoRoutine, 0),
	}
}

func Parse(stop <-chan any, dataStream <-chan Metadata) <-chan Sample {

	/*
		The expected behaviour is to block the parsing goroutine until it
		receives a Metadata object from the dataStream created by the sampler.
	*/
	parsedStream := make(chan Sample)

	go func() {
		defer close(parsedStream)
		for {
			select {
			case <-stop:
				return
			case metadata, ok := <-dataStream:
				if ok == false {
					return
				}

				sample := NewSample(metadata.Timestamp, metadata.numGoroutines)
				var cur_state ParserState
				next_state := StateHeading

				visited := make([]bool, NUM_PARSER_STATES)

				for line := range bytes.SplitSeq(metadata.stackDump, []byte("\n")) {
					if len(line) == 0 {
						continue
					}
					
					for i := range visited {
						visited[i] = false
					}

					state_machine:
						cur_state = next_state 
						// Having 2 such states per iteration is something I picked up from my Verilog days

						switch cur_state {
						case StateHeading:
							if visited[StateHeading] == true {
								fmt.Println("[STACK PARSER] Cycle detected in parser state machine! Moving to next line")
								continue
							}

							if !bytes.HasPrefix(line, []byte(heading_prefix)) {
								next_state = StateStackFunc // retry with a different state
								goto state_machine
							}
							if !parseHeading(sample, line) {
								next_state = StateHeading // look for heading in next line
								continue
							}
							visited[StateHeading] = true
							next_state = StateStackFunc
						
						case StateStackFunc:
							if visited[StateStackFunc] == true {
								fmt.Println("[STACK PARSER] Cycle detected in parser state machine! Moving to next line")
								continue
							}

							if bytes.HasPrefix(line, []byte(created_by_prefix)) {
								next_state = StateCreatedByFunc
								goto state_machine
							} 
							if !parseStackFunc(sample, line) {
								next_state = StateCreatedByFunc // retry with a different state
								goto state_machine
							}
							visited[StateStackFunc] = true
							next_state = StateStackFuncAddr

						case StateStackFuncAddr:
							if visited[StateStackFuncAddr] == true {
								fmt.Println("[STACK PARSER] Cycle detected in parser state machine! Moving to next line")
								continue
							}

							if !parseStackFuncAddr(sample, line) {
								next_state = StateCreatedByAddr // retry with a different state
								goto state_machine
							}
							visited[StateStackFuncAddr] = true
							next_state = StateStackFunc

						case StateCreatedByFunc:
							if visited[StateCreatedByFunc] == true {
								fmt.Println("[STACK PARSER] Cycle detected in parser state machine! Moving to next line")
								continue
							}

							if !bytes.HasPrefix(line, []byte(created_by_prefix)) {
								next_state = StateHeading // retry with a different state
								goto state_machine
							}

							if !parseCreatedByFunc(sample, line) {
								next_state = StateHeading // retry with a different state
								goto state_machine
							}
							visited[StateCreatedByFunc] = true
							next_state = StateCreatedByAddr

						case StateCreatedByAddr:
							if visited[StateCreatedByAddr] == true {
								fmt.Println("[STACK PARSER] Cycle detected in parser state machine! Moving to next line")
								continue
							}

							if !parseCreatedByAddr(sample, line) {
								next_state = StateHeading // retry with a different state
								goto state_machine
							}
							visited[StateCreatedByAddr] = true
							next_state = StateHeading
						}
				}
				if sample.GoRoutineCount != len(sample.List) {
					panic("[STACK PARSER] Goroutine count not same as number of goroutines in sample")
				}
				parsedStream<- *sample
			}
		}
	}()

	return parsedStream
}

func parseHeading(sample *Sample, line []byte) bool {
	//-------------------------------- ID PARSING -----------------------------------
	/*
	We have already determined at the state machine level that the line does 
	begin with "goroutine ..." and so we need not check it once again.
	*/
	
	line = bytes.TrimSpace(line[len(heading_prefix) : ]) // faster than regex and works because the string beginning is deterministic
	id := bytes.Split(line, []byte(" "))[0]

	newGoRoutine := GoRoutine{
		Stack: make([]Frame, 0),
		CreatedBy: &Frame{},
	}
	idVal, err := strconv.Atoi(string(id))
	if err != nil {
		fmt.Println("[STACK PARSER] Could not parse header pattern:", err)
		return false
	}
	newGoRoutine.Id = idVal

	// ------------------------- GOROUTINE STATE PARSING ------------------------------
	
	idx_open := bytes.LastIndex(line, []byte("["))
	idx_close := bytes.LastIndex(line, []byte("]"))

	if idx_open == -1 || idx_close == -1 {
		fmt.Println("[STACK PARSER] Cannot find the goroutine state in header")
		return false
	}

	stateString := line[idx_open + 1 : idx_close]

	newGoRoutine.State = stateMap[string(firstWord(stateString))]
	if 2 <= newGoRoutine.State && newGoRoutine.State <= 10 {
		newGoRoutine.Waiting = true
	}
	sample.List = append(sample.List, newGoRoutine)
	return true
}

func parseStackFunc(sample *Sample, line []byte) bool {

	idx := bytes.LastIndex(line, []byte("("))
	if idx == -1 {
		return false
	}

	stack_function := line[ : idx]

	latestGoRoutine := &sample.List[len(sample.List) - 1]
	latestGoRoutine.Stack = append(latestGoRoutine.Stack, Frame{
		Func: string(stack_function),
	})
	return true
}

func parseStackFuncAddr(sample *Sample, line []byte) bool {
	/*
	We want to support absolute paths with spaces in intermediate or final
	file/folder names, which means that we will utilise LastIndex multiple
	times in this function.
	*/
	splitfrom := bytes.LastIndex(line, []byte("+"))
	if splitfrom != -1 {
		line = line[ : splitfrom]
	}

	idx := bytes.LastIndex(line, []byte(":"))
	if idx == -1 {
		return false
	}

	filepath := line[ : idx]
	line_num := line[idx + 1 : ]

	if len(sample.List) == 0 {
		fmt.Println("[STACK PARSER] Could not parse header pattern: No header detected")
		return false
	}
	latestGoRoutine := &sample.List[len(sample.List) - 1]

	if len(latestGoRoutine.Stack) == 0 {
		fmt.Println("[STACK PARSER] Could not parse header pattern: No stack function detected")
		return false
	}
	latestFrame := &latestGoRoutine.Stack[len(latestGoRoutine.Stack) - 1]

	latestFrame.File = string(filepath)
	lnum, err := strconv.Atoi(string(line_num))

	if err != nil {
		latestFrame.File = string(filepath) + string(line_num)
		return true
	}

	latestFrame.Line = lnum
	return true
}

func parseCreatedByFunc(sample *Sample, line []byte) bool {
	createdBy := bytes.TrimSpace(line[len(created_by_prefix) : ])
	
	if len(sample.List) == 0 {
		fmt.Println("[STACK PARSER] Could not parse header pattern: No header detected")
		return false
	}
	latestGoRoutine := &sample.List[len(sample.List) - 1]

	latestGoRoutine.CreatedBy.Func = string(createdBy)
	return true
}

func parseCreatedByAddr(sample *Sample, line []byte) bool {
	/*
	Again, we want to support absolute paths with spaces in intermediate or final
	file/folder names.
	*/
	splitfrom := bytes.LastIndex(line, []byte("+"))
	if splitfrom == -1 {
		return false
	}
	line = line[ : splitfrom]

	idx := bytes.LastIndex(line, []byte(":"))
	if idx == -1 {
		return false
	}

	filepath := line[ : idx]
	line_num := line[idx + 1 : ]

	if len(sample.List) == 0 {
		fmt.Println("[STACK PARSER] Could not parse header pattern: No header detected")
		return false
	}
	latestGoRoutine := &sample.List[len(sample.List) - 1]

	latestFrame := latestGoRoutine.CreatedBy

	latestFrame.File = string(filepath)
	latestFrame.Line, _ = strconv.Atoi(string(line_num))

	return true
}


/*
DESIGN MUSINGS: Is a state machine the right approach? It is certainly more elegant,
given that in a file with a given structure we always know what to expect next, and
hence can skip having to check multiple "if" statements on strings which seem slightly
contrived or "hardcoded", for lack of a better word.

The gostackparse parser uses a lot more states than I feel are necessary, and I instead
assign one state for each line that is consumed, instead of one state for each semantic
symbol that is consumed.

21 DECEMBER 2025, 7:30 PM -- 
I drew the state machine diagram now, and it appears to me that there might be a lot of 
potential cycles in the state graph. This might stick the parser inside an infinite loop.
So a plausible solution to this problem might be the classical technique borrowed from 
graph algorithms: Maintain a `visited` array for all the states, initially null for each 
line. Then if we encounter a state which has already been visited, there is no point in 
executing further because the processing function as well as the file contents, are 
deterministic and will give us the same cycle of states, over and over.  
Thus when we arrive at a previously visited state we just abort processing for that line. 

11:30 PM -- 
Just checked the DataDog/gostackparse repo once more: the parser file is almost the same 
length for both our implementations. That's a pretty cool fact for me :3
*/
