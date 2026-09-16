/*
Copyright 2021 The Skaffold Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package output

import (
	"context"
	"io"
	"time"

	"github.com/lucky-tools/devloop/pkg/devloop/constants"
	eventV2 "github.com/lucky-tools/devloop/pkg/devloop/event/v2"
	"github.com/lucky-tools/devloop/pkg/devloop/output/log"
)

const timestampFormat = "2006-01-02 15:04:05"

type devloopWriter struct {
	MainWriter  io.Writer
	EventWriter io.Writer
	task        constants.Phase
	subtask     string

	timestamps bool
}

func (s devloopWriter) Write(p []byte) (int, error) {
	written := 0
	if s.timestamps {
		t, err := s.MainWriter.Write([]byte(time.Now().Format(timestampFormat) + " "))
		if err != nil {
			return t, err
		}

		written += t
	}

	n, err := s.MainWriter.Write(p)
	if err != nil {
		return n, err
	}
	if n != len(p) {
		return n, io.ErrShortWrite
	}

	written += n

	s.EventWriter.Write(p)

	return written, nil
}

func GetWriter(ctx context.Context, out io.Writer, defaultColor int, forceColors bool, timestamps bool) io.Writer {
	if _, isSW := out.(devloopWriter); isSW {
		return out
	}

	return devloopWriter{
		MainWriter:  SetupColors(ctx, out, defaultColor, forceColors),
		EventWriter: eventV2.NewLogger(constants.DevLoop, "-1"),
		timestamps:  timestamps,
	}
}

// GetUnderlyingWriter returns the underlying writer if out is a colorableWriter
func GetUnderlyingWriter(out io.Writer) io.Writer {
	sw, isSW := out.(devloopWriter)
	if isSW {
		out = sw.MainWriter
	}
	cw, isCW := out.(colorableWriter)
	if isCW {
		out = cw.Writer
	}
	return out
}

// WithEventContext will return a new devloopWriter with the given parameters to be used for the event writer.
// If the passed io.Writer is not a devloopWriter, then it is simply returned.
func WithEventContext(ctx context.Context, out io.Writer, phase constants.Phase, subtaskID string) (io.Writer, context.Context) {
	ctx = context.WithValue(ctx, log.ContextKey, log.EventContext{
		Task:    phase,
		Subtask: subtaskID,
	})

	if sw, isSW := out.(devloopWriter); isSW {
		return devloopWriter{
			MainWriter:  sw.MainWriter,
			EventWriter: eventV2.NewLogger(phase, subtaskID),
			task:        phase,
			subtask:     subtaskID,
			timestamps:  sw.timestamps,
		}, ctx
	}

	return out, ctx
}
