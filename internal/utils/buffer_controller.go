package utils

import (
	"bytes"
	"fmt"
	"sync"
)

// this data structure can be used whenever you have a stream of random length byte arrays coming to you, and you want to
// convert them into a stream of fixed length byte arrays.
type BufferSizeController struct {
	buffer                bytes.Buffer
	mutex                 sync.Mutex
	outputByteArrayLength int

	outChan chan []byte
}

func NewBufferSizeController(capacity int) BufferSizeController {
	return BufferSizeController{
		buffer:                bytes.Buffer{},
		mutex:                 sync.Mutex{},
		outputByteArrayLength: capacity,

		outChan: make(chan []byte),
	}
}

func (ab *BufferSizeController) GetOutputChannel() <-chan []byte {
	return ab.outChan
}

// this basically sends the leftover data from the internal buffer to the outChan
func (ab *BufferSizeController) Flush() error {
	ab.mutex.Lock()
	defer ab.mutex.Unlock()

	ab.outChan <- ab.buffer.Bytes()
	ab.buffer.Reset()
	return nil
}

// this evaluates the state of the buffer makes sure that the buffer size is less than outputByteArrayLength
// by making max possible number of chunks from the internal buffer and sends it to the outChan
func (ab *BufferSizeController) makeChunksFromBuffer() error {
	for ab.buffer.Len() > ab.outputByteArrayLength {
		outBuf := make([]byte, ab.outputByteArrayLength)
		_, err := ab.buffer.Read(outBuf)
		if err != nil {
			return fmt.Errorf("Could not read bytes: %s", err)
		}
		// send the data to the outChan
		ab.outChan <- outBuf
	}
	return nil
}

func (ab *BufferSizeController) processData(data []byte) error {
	ab.mutex.Lock()
	defer ab.mutex.Unlock()

	ab.buffer.Write(data)
	return ab.makeChunksFromBuffer()
}

func (ab *BufferSizeController) Write(data []byte) error {
	err := ab.processData(data)
	if err != nil {
		return err
	}
	return nil
}
