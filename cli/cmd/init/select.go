// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package init

import (
	"bufio"
	"fmt"
	"io"
	"os"

	"github.com/agntcy/dir/cli/internal/agentcfg"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// Control bytes read from the terminal in raw mode.
const (
	byteCtrlC = 0x03
	byteEsc   = 0x1b
)

// key is a decoded keypress from the interactive selector.
type key int

const (
	keyNone key = iota
	keyUp
	keyDown
	keyToggle  // space
	keyConfirm // enter
	keyAbort   // q / esc / ctrl-c
)

// selectState is the mutable state of the checkbox list: which line the cursor
// is on and which entries are checked.
type selectState struct {
	names   []string
	checked []bool
	cursor  int
}

func newSelectState(names []string) *selectState {
	checked := make([]bool, len(names))
	for i := range checked {
		checked[i] = true // all selected by default
	}

	return &selectState{names: names, checked: checked}
}

// apply mutates the state for one keypress (movement or toggle). Enter/abort are
// handled by the caller and are no-ops here.
func (s *selectState) apply(k key) {
	switch k {
	case keyUp:
		if s.cursor > 0 {
			s.cursor--
		}
	case keyDown:
		if s.cursor < len(s.names)-1 {
			s.cursor++
		}
	case keyToggle:
		s.checked[s.cursor] = !s.checked[s.cursor]
	case keyNone, keyConfirm, keyAbort:
		// no state change
	}
}

// checkedIndexes returns the indexes of the checked entries, in list order.
func (s *selectState) checkedIndexes() []int {
	var out []int

	for i, on := range s.checked {
		if on {
			out = append(out, i)
		}
	}

	return out
}

// decodeKey reads one logical keypress, translating arrow-key escape sequences
// and vim-style j/k into movement. Unknown bytes decode to keyNone.
func decodeKey(r *bufio.Reader) (key, error) {
	b, err := r.ReadByte()
	if err != nil {
		return keyNone, err //nolint:wrapcheck
	}

	switch b {
	case '\r', '\n':
		return keyConfirm, nil
	case ' ':
		return keyToggle, nil
	case 'k':
		return keyUp, nil
	case 'j':
		return keyDown, nil
	case 'q', byteCtrlC: // q or Ctrl-C
		return keyAbort, nil
	case byteEsc: // ESC: bare escape aborts; ESC-[-A/B is an arrow key
		return decodeEscape(r)
	default:
		return keyNone, nil
	}
}

// decodeEscape decodes the bytes following an ESC (0x1b). A bare ESC (nothing
// buffered) aborts; ESC-[-A / ESC-[-B are the up / down arrows. Read errors
// mid-sequence are propagated rather than silently decoded as keyNone.
func decodeEscape(r *bufio.Reader) (key, error) {
	if r.Buffered() == 0 {
		return keyAbort, nil
	}

	b2, err := r.ReadByte()
	if err != nil {
		return keyNone, err //nolint:wrapcheck
	}

	if b2 != '[' {
		return keyNone, nil
	}

	b3, err := r.ReadByte()
	if err != nil {
		return keyNone, err //nolint:wrapcheck
	}

	switch b3 {
	case 'A':
		return keyUp, nil
	case 'B':
		return keyDown, nil
	default:
		return keyNone, nil
	}
}

// promptMultiSelect renders a minimal interactive checkbox list — arrow keys (or
// j/k) move, space toggles, enter confirms, q/esc skips — with every candidate
// checked by default. It uses raw terminal mode via golang.org/x/term (already a
// dependency); no extra package. When stdin is not a terminal it selects all
// candidates (callers gate on interactivity before reaching here).
func promptMultiSelect(cmd *cobra.Command, title string, candidates []agentcfg.Agent) ([]agentcfg.Agent, error) {
	in, ok := cmd.InOrStdin().(*os.File)
	if !ok || !term.IsTerminal(int(in.Fd())) { //nolint:gosec // G115: fd fits an int.
		return candidates, nil
	}

	names := make([]string, len(candidates))
	for i, a := range candidates {
		names[i] = a.Name
	}

	state := newSelectState(names)
	out := cmd.OutOrStdout()

	restore, err := term.MakeRaw(int(in.Fd())) //nolint:gosec // G115: fd fits an int.
	if err != nil {
		return nil, fmt.Errorf("enter raw mode: %w", err)
	}

	defer func() { _ = term.Restore(int(in.Fd()), restore) }() //nolint:gosec // G115: fd fits an int.

	fmt.Fprintf(out, "%s  (↑/↓ move · space toggles · enter confirms · q skips)\r\n", title)
	render(out, state, true)

	reader := bufio.NewReader(in)

	for {
		k, err := decodeKey(reader)
		if err != nil {
			return nil, err
		}

		switch k {
		case keyConfirm:
			fmt.Fprint(out, "\r\n")

			return pick(candidates, state.checkedIndexes()), nil
		case keyAbort:
			fmt.Fprint(out, "\r\n")

			return nil, nil
		case keyNone, keyUp, keyDown, keyToggle:
			state.apply(k)
			render(out, state, false)
		}
	}
}

// render draws the checkbox list. On redraws (first=false) it moves the cursor
// up over the previous rendering first, so the list updates in place.
func render(w io.Writer, s *selectState, first bool) {
	if !first {
		fmt.Fprintf(w, "\x1b[%dA", len(s.names)) // cursor up to the first row
	}

	for i, name := range s.names {
		pointer := " "
		if i == s.cursor {
			pointer = ">"
		}

		box := " "
		if s.checked[i] {
			box = "x"
		}

		fmt.Fprintf(w, "\r\x1b[K%s [%s] %s\r\n", pointer, box, name)
	}
}

// pick returns the candidates at the given indexes, in order.
func pick(candidates []agentcfg.Agent, indexes []int) []agentcfg.Agent {
	var out []agentcfg.Agent
	for _, i := range indexes {
		out = append(out, candidates[i])
	}

	return out
}
