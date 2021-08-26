package kubetest

import (
	"fmt"
	"strings"
	"sync"
)

type Logger struct {
	mu  sync.Mutex
	msg *MaskedMessage
}

func NewLogger() *Logger {
	return &Logger{msg: NewMaskedMessage("", nil)}
}

func (l *Logger) AddMask(mask string) {
	l.msg.AddMask(mask)
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

type MaskedMessage struct {
	msg   string
	masks []string
	mu    sync.Mutex
}

func NewMaskedMessage(msg string, masks []string) *MaskedMessage {
	return &MaskedMessage{msg: msg, masks: masks}
}

func (m *MaskedMessage) AddMessage(msg string) {
	m.mu.Lock()
	m.msg += msg
	m.mu.Unlock()
}

func (m *MaskedMessage) AddMask(mask string) {
	m.mu.Lock()
	m.masks = append(m.masks, mask)
	m.mu.Unlock()
}

func (m *MaskedMessage) Filter(msg string) string {
	m.mu.Lock()
	masks := m.masks
	m.mu.Unlock()
	return m.mask(msg, masks)
}

func (m *MaskedMessage) mask(msg string, masks []string) string {
	maskedMsg := msg
	for _, mask := range masks {
		genMaskText := strings.Repeat("*", len(mask))
		maskedMsg = strings.Replace(maskedMsg, mask, genMaskText, -1)
	}
	return maskedMsg
}

func (m *MaskedMessage) String() string {
	m.mu.Lock()
	msg := m.msg
	masks := m.masks
	m.mu.Unlock()
	return m.mask(msg, masks)
}
