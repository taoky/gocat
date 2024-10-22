// Copyright 2018 SumUp Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package testing

import (
	"errors"
	"io"
	"net"
	"os"

	"github.com/stretchr/testify/require"
)

type UnixServer struct {
	bufferSize int
	t          T
}

func NewUnixServer(t T, bufferSize int) *UnixServer {
	return &UnixServer{
		bufferSize: bufferSize,
		t:          t,
	}
}

func (us *UnixServer) Serve(started chan<- *ListenResult) {
	fd, err := os.CreateTemp("", "unix-server")
	require.NoError(us.t, err, "Failed to create tempfile")
	err = os.RemoveAll(fd.Name())
	require.NoError(us.t, err, "Failed to remove tempfile")

	ln, err := net.Listen("unix", fd.Name())
	if err != nil {
		started <- &ListenResult{
			Err: err,
		}

		return
	}

	defer func() {
		_ = ln.Close()
		_ = os.RemoveAll(fd.Name())
	}()

	addr := ln.Addr().String()
	if err != nil {
		started <- &ListenResult{
			Address: addr,
			Err:     err,
		}
	}
	started <- &ListenResult{
		Address: addr,
	}

	for {
		c, err := ln.Accept()
		require.NoError(us.t, err, "Failed to accept connection")

		go us.handleConnection(c)
	}
}

func (us *UnixServer) handleConnection(c net.Conn) {
	defer c.Close()

	buffer := make([]byte, us.bufferSize)
	for {
		readBytes, err := c.Read(buffer)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return
			}

			us.t.Logf("Copy Err: %s", err.Error())
			return
		}

		if readBytes < 1 {
			continue
		}

		msg := buffer[:readBytes]
		writtenBytes, err := c.Write(msg)
		require.NoError(us.t, err, "Failed to write data")

		expectedWrittenBytes := len(msg)
		if expectedWrittenBytes != writtenBytes {
			us.t.Fatalf(
				"Incomplete write back, written: %d, expected: %d",
				writtenBytes,
				expectedWrittenBytes,
			)
		}
	}
}
