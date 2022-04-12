// Copyright (c) 2021 Brian J. Downs
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package spinner is a simple package to add a spinner / progress indicator to any terminal application.
package spinner

import (
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/fatih/color"
	"github.com/mattn/go-isatty"
)

// errInvalidColor is returned when attempting to set an invalid color
var errInvalidColor = errors.New("invalid color")

// validColors holds an array of the only colors allowed
var validColors = map[string]bool{
	// default colors for backwards compatibility
	"black":   true,
	"red":     true,
	"green":   true,
	"yellow":  true,
	"blue":    true,
	"magenta": true,
	"cyan":    true,
	"white":   true,

	// attributes
	"reset":        true,
	"bold":         true,
	"faint":        true,
	"italic":       true,
	"underline":    true,
	"blinkslow":    true,
	"blinkrapid":   true,
	"reversevideo": true,
	"concealed":    true,
	"crossedout":   true,

	// foreground text
	"fgBlack":   true,
	"fgRed":     true,
	"fgGreen":   true,
	"fgYellow":  true,
	"fgBlue":    true,
	"fgMagenta": true,
	"fgCyan":    true,
	"fgWhite":   true,

	// foreground Hi-Intensity text
	"fgHiBlack":   true,
	"fgHiRed":     true,
	"fgHiGreen":   true,
	"fgHiYellow":  true,
	"fgHiBlue":    true,
	"fgHiMagenta": true,
	"fgHiCyan":    true,
	"fgHiWhite":   true,

	// background text
	"bgBlack":   true,
	"bgRed":     true,
	"bgGreen":   true,
	"bgYellow":  true,
	"bgBlue":    true,
	"bgMagenta": true,
	"bgCyan":    true,
	"bgWhite":   true,

	// background Hi-Intensity text
	"bgHiBlack":   true,
	"bgHiRed":     true,
	"bgHiGreen":   true,
	"bgHiYellow":  true,
	"bgHiBlue":    true,
	"bgHiMagenta": true,
	"bgHiCyan":    true,
	"bgHiWhite":   true,
}

// returns true if the OS is windows and the WT_SESSION env variable is set.
var isWindowsTerminalOnWindows = len(os.Getenv("WT_SESSION")) > 0 && runtime.GOOS == "windows"

// returns a valid color's foreground text color attribute
var colorAttributeMap = map[string]color.Attribute{
	// default colors for backwards compatibility
	"black":   color.FgBlack,
	"red":     color.FgRed,
	"green":   color.FgGreen,
	"yellow":  color.FgYellow,
	"blue":    color.FgBlue,
	"magenta": color.FgMagenta,
	"cyan":    color.FgCyan,
	"white":   color.FgWhite,

	// attributes
	"reset":        color.Reset,
	"bold":         color.Bold,
	"faint":        color.Faint,
	"italic":       color.Italic,
	"underline":    color.Underline,
	"blinkslow":    color.BlinkSlow,
	"blinkrapid":   color.BlinkRapid,
	"reversevideo": color.ReverseVideo,
	"concealed":    color.Concealed,
	"crossedout":   color.CrossedOut,

	// foreground text colors
	"fgBlack":   color.FgBlack,
	"fgRed":     color.FgRed,
	"fgGreen":   color.FgGreen,
	"fgYellow":  color.FgYellow,
	"fgBlue":    color.FgBlue,
	"fgMagenta": color.FgMagenta,
	"fgCyan":    color.FgCyan,
	"fgWhite":   color.FgWhite,

	// foreground Hi-Intensity text colors
	"fgHiBlack":   color.FgHiBlack,
	"fgHiRed":     color.FgHiRed,
	"fgHiGreen":   color.FgHiGreen,
	"fgHiYellow":  color.FgHiYellow,
	"fgHiBlue":    color.FgHiBlue,
	"fgHiMagenta": color.FgHiMagenta,
	"fgHiCyan":    color.FgHiCyan,
	"fgHiWhite":   color.FgHiWhite,

	// background text colors
	"bgBlack":   color.BgBlack,
	"bgRed":     color.BgRed,
	"bgGreen":   color.BgGreen,
	"bgYellow":  color.BgYellow,
	"bgBlue":    color.BgBlue,
	"bgMagenta": color.BgMagenta,
	"bgCyan":    color.BgCyan,
	"bgWhite":   color.BgWhite,

	// background Hi-Intensity text colors
	"bgHiBlack":   color.BgHiBlack,
	"bgHiRed":     color.BgHiRed,
	"bgHiGreen":   color.BgHiGreen,
	"bgHiYellow":  color.BgHiYellow,
	"bgHiBlue":    color.BgHiBlue,
	"bgHiMagenta": color.BgHiMagenta,
	"bgHiCyan":    color.BgHiCyan,
	"bgHiWhite":   color.BgHiWhite,
}

// validColor will make sure the given color is actually allowed.
func validColor(c string) bool {
	return validColors[c]
}

// Spinner struct to hold the provided options.
type Spinner struct {
	mu         *sync.RWMutex
	Delay      time.Duration                 // Delay is the speed of the indicator
	chars      []string                      // chars holds the chosen character set
	Text       string                        // Text shown after the Spinner
	lastOutput string                        // last character(set) written
	color      func(a ...interface{}) string // default color is white
	Writer     io.Writer                     // to make testing better, exported so users have access. Use `WithWriter` to update after initialization.
	active     bool                          // active holds the state of the spinner
	stopChan   chan struct{}                 // stopChan is a channel used to stop the indicator
	HideCursor bool                          // hideCursor determines if the cursor is visible
	PreUpdate  func(s *Spinner)              // will be triggered before every spinner update
	PostUpdate func(s *Spinner)              // will be triggered after every spinner update
	Symbol     string                        // Symbol for the spinner, show before PrefixText
	PrefixText string                        // PrefixText for the spinner, shown before the spinner and after the Symbol
}

// New provides a pointer to an instance of Spinner with the supplied options.
func New(options Options) *Spinner {
	s := &Spinner{
		Delay:      100 * time.Millisecond,
		chars:      CharSets[11],
		color:      color.New(color.FgWhite).SprintFunc(),
		mu:         &sync.RWMutex{},
		Writer:     color.Output,
		stopChan:   make(chan struct{}, 1),
		active:     false,
		HideCursor: true,
	}

	if options.Writer != nil {
		s.mu.Lock()
		s.Writer = options.Writer
		s.mu.Unlock()
	}

	if options.PrefixText != "" {
		s.PrefixText = options.PrefixText
	}

	if options.Symbol != "" {
		s.Symbol = options.Symbol
	}

	if options.HideCursor {
		s.HideCursor = options.HideCursor
	}

	if options.Symbol != "" {
		s.Symbol = options.Symbol
	}

	if options.Color != "" {
		s.Color(options.Color)
	}

	if options.Text != "" {
		s.Text = options.Text
	}

	if options.Delay != 0 {
		s.Delay = options.Delay
	}

	return s
}

// Options contains fields to configure the spinner.
type Options struct {
	Color        string
	Text         string
	HideCursor   bool
	Symbol       string
	PrefixText   string
	CharacterSet []string
	Writer       io.Writer
	Delay        time.Duration
}

// Start will start the spinner.
func (s *Spinner) Start() {
	s.mu.Lock()
	if s.active || !isRunningInTerminal() {
		s.mu.Unlock()
		return
	}
	if s.HideCursor && !isWindowsTerminalOnWindows {
		// hides the cursor
		fmt.Fprint(s.Writer, "\033[?25l")
	}
	s.active = true
	s.mu.Unlock()

	go func() {
		for {
			for i := 0; i < len(s.chars); i++ {
				select {
				case <-s.stopChan:
					return
				default:
					s.mu.Lock()
					if !s.active {
						s.mu.Unlock()
						return
					}
					if !isWindowsTerminalOnWindows {
						s.erase()
					}

					if s.PreUpdate != nil {
						s.PreUpdate(s)
					}

					var fullSymbol string
					if s.Symbol != "" {
						fullSymbol = s.Symbol + " "
					} else {
						fullSymbol = ""
					}

					var fullPrefixText string
					if s.PrefixText != "" {
						fullPrefixText = s.PrefixText + " "
					} else {
						fullPrefixText = ""
					}

					var fullText string
					if s.Text != "" {
						fullText = " " + s.Text
					} else {
						fullText = ""
					}

					var charStyled string
					if runtime.GOOS == "windows" {
						if s.Writer == os.Stderr {
							charStyled = s.chars[i]
						} else {
							charStyled = s.color(s.chars[i])
						}
					} else {
						charStyled = s.color(s.chars[i])
					}

					outColor := fmt.Sprintf("\r%s%s%s%s", fullSymbol, fullPrefixText, charStyled, fullText)
					outPlain := fmt.Sprintf("\r%s%s%s%s", fullSymbol, fullPrefixText, s.chars[i], fullText)

					fmt.Fprint(s.Writer, outColor)
					s.lastOutput = outPlain
					delay := s.Delay

					if s.PostUpdate != nil {
						s.PostUpdate(s)
					}

					s.mu.Unlock()
					time.Sleep(delay)
				}
			}
		}
	}()
}

// Stops the spinner.
func (s *Spinner) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.active {
		s.active = false
		if s.HideCursor && !isWindowsTerminalOnWindows {
			// makes the cursor visible
			fmt.Fprint(s.Writer, "\033[?25h")
		}
		s.erase()

		s.stopChan <- struct{}{}
	}
}

// Stops the spinner and prits out a message, used later for success, fail, etc.
func (s *Spinner) StopAndPersist(symbol string, text string) {
	s.Stop()

	var fullSymbol string
	if s.Symbol != "" {
		fullSymbol = s.Symbol + " "
	} else {
		fullSymbol = ""
	}

	var fullText string
	if text != "" {
		fullText = " " + text
	} else {
		fullText = ""
	}

	fmt.Fprintf(s.Writer, "\r%s%s%s\n", fullSymbol, symbol, fullText)
}

// Stops the spinner and prints out a success message.
func (s *Spinner) Succeed(text string) {
	s.StopAndPersist(Symbols["success"], text)
}

// Stops the spinner and prints out a failure message.
func (s *Spinner) Fail(text string) {
	s.StopAndPersist(Symbols["failure"], text)
}

// Restart will stop and start the indicator.
func (s *Spinner) Restart() {
	s.Stop()
	s.Start()
}

// Reverse will reverse the order of the slice assigned to the indicator.
func (s *Spinner) Reverse() {
	s.mu.Lock()
	for i, j := 0, len(s.chars)-1; i < j; i, j = i+1, j-1 {
		s.chars[i], s.chars[j] = s.chars[j], s.chars[i]
	}
	s.mu.Unlock()
}

// Color will set the struct field for the given color to be used. The spinner
// will need to be explicitly restarted.
func (s *Spinner) Color(colors ...string) error {
	colorAttributes := make([]color.Attribute, len(colors))

	// Verify colours are valid and place the appropriate attribute in the array
	for index, c := range colors {
		if !validColor(c) {
			return errInvalidColor
		}
		colorAttributes[index] = colorAttributeMap[c]
	}

	s.mu.Lock()
	s.color = color.New(colorAttributes...).SprintFunc()
	s.mu.Unlock()
	return nil
}

// UpdateSpeed will set the indicator delay to the given value.
func (s *Spinner) UpdateSpeed(d time.Duration) {
	s.mu.Lock()
	s.Delay = d
	s.mu.Unlock()
}

// UpdateCharSet will change the current character set to the given one.
func (s *Spinner) UpdateCharSet(cs []string) {
	s.mu.Lock()
	s.chars = cs
	s.mu.Unlock()
}

// erase deletes written characters on the current line.
// Caller must already hold s.lock.
func (s *Spinner) erase() {
	n := utf8.RuneCountInString(s.lastOutput)
	if runtime.GOOS == "windows" && !isWindowsTerminalOnWindows {
		clearString := "\r" + strings.Repeat(" ", n) + "\r"
		fmt.Fprint(s.Writer, clearString)
		s.lastOutput = ""
		return
	}

	// Taken from https://en.wikipedia.org/wiki/ANSI_escape_code:
	// \r     - Carriage return - Moves the cursor to column zero
	// \033[K - Erases part of the line. If n is 0 (or missing), clear from
	// cursor to the end of the line. If n is 1, clear from cursor to beginning
	// of the line. If n is 2, clear entire line. Cursor position does not
	// change.
	fmt.Fprintf(s.Writer, "\r\033[K")
	s.lastOutput = ""
}

// Lock allows for manual control to lock the spinner.
func (s *Spinner) Lock() {
	s.mu.Lock()
}

// Unlock allows for manual control to unlock the spinner.
func (s *Spinner) Unlock() {
	s.mu.Unlock()
}

// GenerateNumberSequence will generate a slice of integers at the
// provided length and convert them each to a string.
func GenerateNumberSequence(length int) []string {
	numSeq := make([]string, length)
	for i := 0; i < length; i++ {
		numSeq[i] = strconv.Itoa(i)
	}
	return numSeq
}

// isRunningInTerminal check if stdout file descriptor is terminal
func isRunningInTerminal() bool {
	return isatty.IsTerminal(os.Stdout.Fd())
}
