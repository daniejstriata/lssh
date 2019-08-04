package ssh

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

//
func checkBuildInCommand(cmd string) (isLocalCmd bool) {
	// check local command regex
	buildinRegex := regexp.MustCompile(`^%%.*`)

	// check build-in command
	switch c {
	case "exit", "quit", "clear": // build-in command
		isLocalCmd = true

	case "%history", "%out": // parsent build-in command.
		isLocalCmd = true
	}

	// local command
	switch {
	case buildinRegex.MatchString(c):
		isLocalCmd = true
	}

	return
}

// checkLocalCommand return bool  if each pipeline contains built-in commands or local machine commands.
func checkBuildInCommandInSlice(pslice [][]pipeLine) (isInLocalCmd bool) {
	for _, pipelines := range pslice {
		for _, p := range pipelines {
			// get pipeline command
			c := p.Args[0]

			if checkBuildInCommand(c) {
				isInLocalCmd = true
				return
			}
		}
	}

	return
}

// runBuildInCommand is run buildin or local machine command.
func (ps *pShell) run(pLine pipeLine, in *io.PipeReader, out *io.PipeWriter, ch chan<- bool) (err error) {
	// get 1st element
	command := pLine.Args[0]

	// check and exec build-in command
	switch command {
	// exit or quit
	case "exit", "quit":
		os.Exit(0)

	// clear
	case "clear":
		fmt.Printf("\033[H\033[2J")
		return

	// %history
	case "%history":
		ps.buildin_history(stdout)
		return

	// %out [num]
	case "%out":
		num := 0
		if len(pLine.Args) > 1 {
			num, err = strconv.Atoi(pLine.Args[1])
			if err != nil {
				return
			}
		}

		ps.buildin_out(num, stdout)
		return
	}

	// Create History
	ps.History[ps.Count] = map[string]*pShellHistory{}

	// check and exec local command
	buildinRegex := regexp.MustCompile(`^%%.*`)
	switch {
	case buildinRegex.MatchString(command):
		// exec local machine
		ps.executePipeLineLocal(pline, in, out, ch)
		return
	}

	// exec remote machine
	ps.executePipeLineRemote(pline, in, out, ch)

	return
}

// localCmd_history is printout history (shell history)
// TODO(blacknon): 通番をつけて、bash等のように `!<N>` で実行できるようにする
func (ps *pShell) buildin_history(out *io.Writer) {
	data, err := ps.GetHistoryFromFile()
	if err != nil {
		return
	}

	for _, h := range data {
		fmt.Fprintf(out, "%s: %s\n", h.Timestamp, h.Command)
	}
}

// localCmd_out is print exec history at number
// example:
//     - %out
//     - %out <num>
func (ps *pShell) buildin_out(num int, out *io.Writer) {
	histories := ps.History[num]

	i := 0
	for _, h := range histories {
		// if first, print out command
		if i == 0 {
			fmt.Fprintf(out, "Command: %s\n", h.Command)
		}
		i += 1

		// print out result
		fmt.Fprintf(out, h.Result)
	}
}

// executePipeLineRemote is exec command in remote machine.
func (ps *pShell) executePipeLineRemote(pline pipeLine, in io.Reader, out io.Writer, ch chan<- bool) {
	exitInput := make(chan bool)  // Input finish channel
	exitSignal := make(chan bool) // Send kill signal finish channel

	// create []io.Writer, this after in MultiWriter.
	var writers []io.Writer

	// set stdin
	// TODO(blacknon): io.MultiWriterで書き込ませる必要がある(io.Copyで処理？)
	var stdin io.Reader
	if in != nil {
		stdin = in
	} else {
		stdin = os.Stdin
	}

	// set stdout
	var stdout io.Writer
	if out != nil {
		// こっちはそのままpipeに渡せば対処できそう？
		stdout = out
	} else {
		// TODO(blacknon):
		//     出力をOutput経由にさせて、headerを出させるようにしないといけない。
		//     どうしたら良いかは不明。bufio.Writerでうまいこと対処できると嬉しいが…？
		stdout = os.Stdout
	}
}

// executePipeLineLocal is exec command in local machine.
func (ps *pShell) executePipeLineLocal(pline pipeLine, in io.Reader, out io.Writer, ch chan<- bool) (err error) {
	// set stdin
	var stdin io.Reader
	if in != nil {
		stdin = in
	} else {
		stdin = os.Stdin
	}

	// set stdout
	var stdout io.Writer
	if out != nil {
		stdout = out
	} else {
		stdout = os.Stdout
	}

	// delete command prefix(`%%`)
	rep := regexp.MustCompile(`^%%`)
	pline[0] = rep.ReplaceAllString(pline[0], "")

	// join command
	command := strings.Join(pline, " ")

	// execute command
	cmd := exec.Command("sh", "-c", command)

	// set stdin,stdout
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = os.Stderr

	// run command
	err = cmd.Run()

	// send exit
	ch <- true

	return
}
