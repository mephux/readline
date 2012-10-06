// Access to basic readline functions. Support for history manipulation and tab
// completion are provided. Not much of the library is exposed though.
package readline

/*

#cgo LDFLAGS: -lreadline -lhistory

#include <stdio.h>
#include <stdlib.h>
#include <readline/readline.h>
#include <readline/history.h>

extern char *_completion_function(char *s, int i);

static char *_completion_function_trans(const char *s, int i) {
	return _completion_function((char *) s, i);
}

static void register_readline() {
	rl_completion_entry_function = _completion_function_trans;
	using_history();
}

*/
import "C"

import (
	"io"
	"unsafe"
	"syscall"
)

// The default prompt used by Reader().
var Prompt = "> "

// The continue prompt used by Reader().
var Continue = ".."

type state byte

const (
	readerStart state = iota
	readerContinue
	readerEnd
)

type reader struct {
	buf []byte
	state state
}

// Begin reading lines. If more than one line is required, the continue prompt
// is used for subsequent lines.
func Reader() io.Reader {
	return new(reader)
}

func (r *reader) getLine() error {
	prompt := Prompt
	if r.state == readerContinue {
		prompt = Continue
	}
	s, err := String(prompt)
	if err != nil {
		return err
	}
	r.buf = []byte(s)
	return nil
}

func (r *reader) Read(buf []byte) (int, error) {
	if r.state == readerEnd {
		return 0, io.EOF
	}
	if len(r.buf) == 0 {
		err := r.getLine()
		if err == io.EOF {
			r.state = readerEnd
		}
		if err != nil {
			return 0, err
		}
		r.state = readerContinue
	}
	copy(buf, r.buf)
	l := len(buf)
	if len(buf) > len(r.buf) {
		l = len(r.buf)
	}
	r.buf = r.buf[l:]
	return l, nil
}

// Read a line with the given prompt.
func String(prompt string) (string, error) {
	p := C.CString(prompt)
	rp := C.readline(p)
	s := C.GoString(rp)
	C.free(unsafe.Pointer(p))
	if rp != nil {
		C.free(unsafe.Pointer(rp))
		return s, nil
	}
	return s, io.EOF
}

// This function provides entries for the tab completer.
var Completer = func(query, ctx string) []string {
	return nil
}

var entries []*C.char

//export _completion_function
func _completion_function(p *C.char, _i C.int) *C.char {
	C.rl_completion_suppress_append = 1
	i := int(_i)
	if i == 0 {
		es := Completer(C.GoString(p), C.GoString(C.rl_line_buffer))
		entries = make([]*C.char, len(es))
		for i, x := range es {
			entries[i] = C.CString(x)
		}
	}
	if i >= len(entries) {
		return nil
	}
	return entries[i]
}

func SetWordBreaks(cs string) {
	C.rl_completer_word_break_characters = C.CString(cs)
}

// Add an item to the history.
func AddHistory(s string) {
	n := HistorySize()
	if n == 0 || s != GetHistory(n - 1) {
		C.add_history(C.CString(s))
	}
}

// Retrieve a line from the history.
func GetHistory(i int) string {
	e := C.history_get(C.int(i+1))
	if e == nil {
		return ""
	}
	return C.GoString(e.line)
}

// Deletes all the items in the history.
func ClearHistory() {
	C.clear_history()
}

// Returns the number of items in the history.
func HistorySize() int {
	return int(C.history_length)
}

// Load the history from a file.
func LoadHistory(path string) error {
	p := C.CString(path)
	e := C.read_history(p)
	C.free(unsafe.Pointer(p))

	if e == 0 {
		return nil
	}
	return syscall.Errno(e)
}

// Save the history to a file.
func SaveHistory(path string) error {
	p := C.CString(path)
	e := C.write_history(p)
	C.free(unsafe.Pointer(p))

	if e == 0 {
		return nil
	}
	return syscall.Errno(e)
}

// Frees internal memory and restores terminal attributes. This
// function should be called when readline doesn't return and would
// leave the terminal in a corrupted state.
func Cleanup() {
	C.rl_free_line_state()
	C.rl_cleanup_after_signal()
}

func init() {
	C.register_readline()
}
