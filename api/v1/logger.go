package v1

import (
	"fmt"
	"sync"
)

type Logger struct {
	mu  sync.Mutex
	msg *MaskedMessage
}

func newLogger() *Logger {
	return &Logger{msg: newMaskedMessage("", nil)}
}

func (l *Logger) addMask(mask string) {
	l.msg.addMask(mask)
}

func (l *Logger) Log(msg string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	fmt.Print(l.msg.Filter(msg))
}

func (l *Logger) DebugLog(msg string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	fmt.Printf("[DEBUG] %s\n", l.msg.Filter(msg))
}
